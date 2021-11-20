package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"

	"go.vocdoni.io/dvote/crypto/ethereum"
	chain "go.vocdoni.io/dvote/ethereum"
	"go.vocdoni.io/dvote/ethereum/ethevents"
	"go.vocdoni.io/dvote/httprouter"
	log "go.vocdoni.io/dvote/log"
	"go.vocdoni.io/dvote/metrics"
	"go.vocdoni.io/dvote/service"
	"go.vocdoni.io/manager/config"
	"go.vocdoni.io/manager/database"
	"go.vocdoni.io/manager/database/pgsql"
	"go.vocdoni.io/manager/notify"
	"go.vocdoni.io/manager/urlapi"
)

func newConfig() (*config.Notify, config.Error) {
	var err error
	var cfgError config.Error
	cfg := config.NewNotifyConfig()
	home, err := os.UserHomeDir()
	if err != nil {
		cfgError = config.Error{
			Critical: true,
			Message:  fmt.Sprintf("cannot get user home directory with error: %s", err),
		}
		return nil, cfgError
	}
	// flags
	flag.StringVar(&cfg.DataDir, "dataDir", home+"/.dvotenotif", "directory where data is stored")

	// global
	cfg.LogLevel = *flag.String("logLevel", "info", "Log level (debug, info, warn, error, fatal)")
	cfg.LogOutput = *flag.String("logOutput", "stdout", "Log output (stdout, stderr or filepath)")
	cfg.LogErrorFile = *flag.String("logErrorFile", "", "Log errors and warnings to a file")
	cfg.SaveConfig = *flag.Bool("saveConfig", false, "overwrites an existing config file with the CLI provided flags")
	// TODO: @jordipainan modify other components, currently just notify service
	cfg.Env = *flag.String("env", "", "environment to run on: dev or stage, main otherwise")
	cfg.DB.Host = *flag.String("dbHost", "127.0.0.1", "DB server address")
	cfg.DB.Port = *flag.Int("dbPort", 5432, "DB server port")
	cfg.DB.User = *flag.String("dbUser", "vocdoni", "DB Username")
	cfg.DB.Password = *flag.String("dbPassword", "vocdoni", "DB password")
	cfg.DB.Dbname = *flag.String("dbName", "vocdoni", "DB database name")
	cfg.DB.Sslmode = *flag.String("dbSslmode", "prefer", "DB postgres sslmode")
	// api
	cfg.API.Route = *flag.String("apiRoute", "/api", "dvote API route")
	cfg.API.ListenHost = *flag.String("apiListenHost", "127.0.0.1", "API endpoint listen address")
	cfg.API.ListenPort = *flag.Int("apiListenPort", 8000, "API endpoint http port")
	cfg.API.Ssl.Domain = *flag.String("sslDomain", "", "enable TLS secure domain with LetsEncrypt auto-generated certificate")
	// notifications
	cfg.Notifications.KeyFile = *flag.String("pushNotificationsKeyFile", "", "path to notifications service private key file")
	cfg.Notifications.Service = *flag.Int("pushNotificationsService", notify.Firebase, "push notifications service, 1: Firebase")
	//ethereum node
	cfg.Ethereum.SigningKey = *flag.String("ethSigningKey", "", "signing private Key (if not specified the Ethereum keystore will be used)")
	// ethereum events
	cfg.SubscribeOnly = *flag.Bool("subscribeOnly", true, "only subscribe to new ethereum events (do not read past log)")
	// ethereum web3
	cfg.Web3.W3External = *flag.StringArrayP("w3External", "", []string{},
		"use an external web3 endpoint instead of the local one. Supported protocols: http(s)://, ws(s):// and IPC filepath")
	cfg.Web3.ChainType = *flag.String("ethChain", "sokol", fmt.Sprintf("Ethereum blockchain to use: %s", chain.AvailableChains))
	// ipfs
	cfg.IPFS.NoInit = *flag.Bool("ipfsNoInit", false, "disables inter planetary file system support")
	cfg.IPFS.SyncKey = *flag.String("ipfsSyncKey", "", "enable IPFS cluster synchronization using the given secret key")
	cfg.IPFS.SyncPeers = *flag.StringArray("ipfsSyncPeers", []string{}, "use custom ipfsSync peers/bootnodes for accessing the DHT")
	// metrics
	cfg.Metrics.Enabled = *flag.Bool("metricsEnabled", true, "enable prometheus metrics")
	cfg.Metrics.RefreshInterval = *flag.Int("metricsRefreshInterval", 10, "metrics refresh interval in seconds")

	// parse flags
	flag.Parse()

	// setting up viper
	viper := viper.New()
	viper.AddConfigPath(cfg.DataDir)
	viper.SetConfigName("dvotenotif")
	viper.SetConfigType("yml")
	viper.SetEnvPrefix("NOTIF")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// binding flags to viper

	// global
	viper.BindPFlag("dataDir", flag.Lookup("dataDir"))
	viper.BindPFlag("logLevel", flag.Lookup("logLevel"))
	viper.BindPFlag("logErrorFile", flag.Lookup("logErrorFile"))
	viper.BindPFlag("logOutput", flag.Lookup("logOutput"))
	viper.BindPFlag("env", flag.Lookup("env"))
	viper.BindPFlag("db.host", flag.Lookup("dbHost"))
	viper.BindPFlag("db.port", flag.Lookup("dbPort"))
	viper.BindPFlag("db.user", flag.Lookup("dbUser"))
	viper.BindPFlag("db.password", flag.Lookup("dbPassword"))
	viper.BindPFlag("db.dbName", flag.Lookup("dbName"))
	viper.BindPFlag("db.sslMode", flag.Lookup("dbSslmode"))
	// api
	viper.BindPFlag("api.route", flag.Lookup("apiRoute"))
	viper.BindPFlag("api.listenHost", flag.Lookup("apiListenHost"))
	viper.BindPFlag("api.listenPort", flag.Lookup("apiListenPort"))
	viper.Set("api.ssl.dirCert", cfg.DataDir+"/tls")
	viper.BindPFlag("api.ssl.domain", flag.Lookup("sslDomain"))
	// notifications
	viper.BindPFlag("notifications.KeyFile", flag.Lookup("pushNotificationsKeyFile"))
	viper.BindPFlag("notifications.Service", flag.Lookup("pushNotificationsService"))
	// ethereum node
	viper.Set("ethereum.datadir", cfg.DataDir+"/ethereum")
	viper.BindPFlag("ethereum.signingKey", flag.Lookup("ethSigningKey"))
	viper.BindPFlag("ethereum.noWaitSync", flag.Lookup("ethNoWaitSync"))
	viper.BindPFlag("subscribeOnly", flag.Lookup("subscribeOnly"))
	// ethereum web3
	viper.BindPFlag("web3.w3External", flag.Lookup("w3External"))
	viper.BindPFlag("web3.chainType", flag.Lookup("ethChain"))
	// ipfs
	viper.Set("ipfs.ConfigPath", cfg.DataDir+"/ipfs")
	viper.BindPFlag("ipfs.NoInit", flag.Lookup("ipfsNoInit"))
	viper.BindPFlag("ipfs.SyncKey", flag.Lookup("ipfsSyncKey"))
	viper.BindPFlag("ipfs.SyncPeers", flag.Lookup("ipfsSyncPeers"))

	// metrics
	viper.BindPFlag("metrics.enabled", flag.Lookup("metricsEnabled"))
	viper.BindPFlag("metrics.refreshInterval", flag.Lookup("metricsRefreshInterval"))

	// check if config file exists
	_, err = os.Stat(cfg.DataDir + "/dvotenotif.yml")
	if os.IsNotExist(err) {
		cfgError = config.Error{
			Message: fmt.Sprintf("creating new config file in %s", cfg.DataDir),
		}
		// creting config folder if not exists
		err = os.MkdirAll(cfg.DataDir, os.ModePerm)
		if err != nil {
			cfgError = config.Error{
				Message: fmt.Sprintf("cannot create data directory: %s", err),
			}
		}
		// create config file if not exists
		if err := viper.SafeWriteConfig(); err != nil {
			cfgError = config.Error{
				Message: fmt.Sprintf("cannot write config file into config dir: %s", err),
			}
		}
	} else {
		// read config file
		err = viper.ReadInConfig()
		if err != nil {
			cfgError = config.Error{
				Message: fmt.Sprintf("cannot read loaded config file in %s: %s", cfg.DataDir, err),
			}
		}
	}
	err = viper.Unmarshal(&cfg)
	if err != nil {
		cfgError = config.Error{
			Message: fmt.Sprintf("cannot unmarshal loaded config file: %s", err),
		}
	}

	if cfg.SaveConfig {
		viper.Set("saveConfig", false)
		if err := viper.WriteConfig(); err != nil {
			cfgError = config.Error{
				Message: fmt.Sprintf("cannot overwrite config file into config dir: %s", err),
			}
		}
	}
	return cfg, cfgError
}

func main() {
	var err error
	// setup config
	// creating config and init logger
	cfg, cfgerr := newConfig()
	if cfgerr.Critical {
		panic(cfgerr.Message)
	}
	if cfg == nil {
		panic("cannot read configuration")
	}
	log.Init(cfg.LogLevel, cfg.LogOutput)
	if path := cfg.LogErrorFile; path != "" {
		if err := log.SetFileErrorLog(path); err != nil {
			log.Fatal(err)
		}
	}
	log.Debugf("initializing config: %s", cfg.String())

	// ethereum sign key
	signer := ethereum.NewSignKeys()
	if err := signer.AddHexKey(cfg.Ethereum.SigningKey); err != nil {
		log.Fatalf("invalid signing key: %s", err)
	}

	// ethereum service
	if len(cfg.Web3.W3External) == 0 {
		log.Warnf("no web3 endpoint defined")
	} else {
		log.Infof("using external ethereum endpoint: %s", cfg.Web3.W3External)
	}

	chainSpecs, specErr := chain.SpecsFor(cfg.Web3.ChainType)
	if specErr != nil {
		log.Fatal("cannot get chain specifications with the ENS registry address: %s", specErr)
	}

	// db
	var db database.Database

	// postgres with sqlx
	db, err = pgsql.New(cfg.DB)
	if err != nil {
		log.Fatal(err)
	}

	// Router
	var httpRouter httprouter.HTTProuter
	httpRouter.TLSdomain = cfg.API.Ssl.Domain
	httpRouter.TLSdirCert = cfg.API.Ssl.DirCert
	if err = httpRouter.Init(cfg.API.ListenHost, cfg.API.ListenPort); err != nil {
		log.Fatal(err)
	}

	var metricsAgent *metrics.Agent
	// Enable metrics via proxy
	if cfg.Metrics.Enabled {
		metricsAgent = metrics.NewAgent("/metrics",
			time.Duration(cfg.Metrics.RefreshInterval)*time.Second, &httpRouter)
	}

	// Rest api
	urlApi, err := urlapi.NewURLAPI(&httpRouter, cfg.API.Route, metricsAgent)
	if err != nil {
		log.Fatal(err)
	}

	// init notifications service
	var fa notify.PushNotifier
	if len(cfg.Notifications.KeyFile) > 0 {
		// create file tracker
		chainSpecs.Contracts["processes"].SetABI("processes")
		chainSpecs.Contracts["entities"].SetABI("entities")
		ipfsFileTracker := notify.NewIPFSFileTracker(cfg.IPFS,
			metricsAgent, db, chainSpecs.Contracts["entityResolver"].Address.Hex(),
			cfg.Web3.W3External[0],
			chainSpecs.Contracts["entities"].Domain,
		)
		switch cfg.Notifications.Service {
		case notify.Firebase:
			fa = notify.NewFirebaseAdmin(cfg.Notifications.KeyFile, cfg.Env, ipfsFileTracker)
			log.Info("initilizing Firebase push notifications service")
		default:
			log.Fatal("unsuported push notifications service")
		}

		if err := fa.Init(); err != nil {
			log.Fatalf("cannot init push notifications service: %s", err)
		}
	} else {
		log.Fatal("cannot start push notifications service without a key file")
	}
	log.Info("push notifications service started")

	// ethereum events service
	var evh []ethevents.EventHandler
	ctx := context.Background()
	// Handle ethereum events and notify
	evh = append(evh, fa.HandleEthereum)

	var initBlock *int64
	if !cfg.SubscribeOnly {
		initBlock = new(int64)
		if specErr != nil {
			log.Warn("cannot get chain block to start looking for events, using 0")
			*initBlock = 0
		} else {
			*initBlock = chainSpecs.StartingBlock
		}
	}

	if err := service.EthEvents(ctx, cfg.Web3.W3External, chainSpecs.Name, signer, nil, evh, []string{}); err != nil {
		log.Fatal(err)
	}

	// start notifications API
	log.Infof("enabling Notifications API methods")
	notif := notify.NewAPI(fa)
	if err := urlApi.EnableNotifyHandlers(notif); err != nil {
		log.Fatal(err)
	}

	log.Info("startup complete")

	// close if interrupt received
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	log.Warnf("received SIGTERM, exiting at %s", time.Now().Format(time.RFC850))
	os.Exit(0)
}
