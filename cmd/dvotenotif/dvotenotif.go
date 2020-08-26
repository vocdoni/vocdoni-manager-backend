package main

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/auth"
	"gitlab.com/vocdoni/go-dvote/chain"
	"gitlab.com/vocdoni/go-dvote/chain/ethevents"
	"gitlab.com/vocdoni/go-dvote/crypto/ethereum"
	log "gitlab.com/vocdoni/go-dvote/log"
	"gitlab.com/vocdoni/go-dvote/service"
	"gitlab.com/vocdoni/manager/manager-backend/config"
	endpoint "gitlab.com/vocdoni/manager/manager-backend/services/api-endpoint"
	"golang.org/x/net/context"
)

func newConfig() (*config.Manager, config.Error) {
	var err error
	var cfgError config.Error
	cfg := config.NewConfig()
	home, err := os.UserHomeDir()
	if err != nil {
		cfgError = config.Error{
			Critical: true,
			Message:  fmt.Sprintf("cannot get user home directory with error: %s", err),
		}
		return nil, cfgError
	}
	// flags
	flag.StringVar(&cfg.DataDir, "dataDir", home+"/.dvotemanager", "directory where data is stored")
	cfg.Mode = *flag.String("mode", "all", fmt.Sprintf("operation mode: %s", func() (modes []string) {
		for m := range config.Modes {
			modes = append(modes, m)
		}
		return
	}()))

	cfg.LogLevel = *flag.String("logLevel", "info", "Log level (debug, info, warn, error, fatal)")
	cfg.LogOutput = *flag.String("logOutput", "stdout", "Log output (stdout, stderr or filepath)")
	cfg.LogErrorFile = *flag.String("logErrorFile", "", "Log errors and warnings to a file")
	cfg.SaveConfig = *flag.Bool("saveConfig", false, "overwrites an existing config file with the CLI provided flags")
	cfg.DB.Host = *flag.String("dbHost", "127.0.0.1", "DB server address")
	cfg.DB.Port = *flag.Int("dbPort", 5432, "DB server port")
	cfg.DB.User = *flag.String("dbUser", "user", "DB Username")
	cfg.DB.Password = *flag.String("dbPassword", "password", "DB password")
	cfg.DB.Dbname = *flag.String("dbName", "database", "DB database name")
	cfg.DB.Sslmode = *flag.String("dbSslmode", "prefer", "DB postgres sslmode")
	cfg.Notifications.FirebaseKeyFile = *flag.String("firebaseKey", "", "firebase json file private key")
	cfg.Ethereum.SigningKey = *flag.String("ethSigningKey", "", "signing private Key (if not specified the Ethereum keystore will be used)")

	// metrics
	cfg.Metrics.Enabled = *flag.Bool("metricsEnabled", true, "enable prometheus metrics")
	cfg.Metrics.RefreshInterval = *flag.Int("metricsRefreshInterval", 10, "metrics refresh interval in seconds")

	// parse flags
	flag.Parse()

	// setting up viper
	viper := viper.New()
	viper.AddConfigPath(cfg.DataDir)
	viper.SetConfigName("dvotemanager")
	viper.SetConfigType("yml")
	viper.SetEnvPrefix("DVOTE")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// binding flags to viper

	// global
	viper.BindPFlag("dataDir", flag.Lookup("dataDir"))
	viper.BindPFlag("logLevel", flag.Lookup("logLevel"))
	viper.BindPFlag("logErrorFile", flag.Lookup("logErrorFile"))
	viper.BindPFlag("logOutput", flag.Lookup("logOutput"))
	viper.BindPFlag("db.host", flag.Lookup("dbHost"))
	viper.BindPFlag("db.port", flag.Lookup("dbPort"))
	viper.BindPFlag("db.user", flag.Lookup("dbUser"))
	viper.BindPFlag("db.password", flag.Lookup("dbPassword"))
	viper.BindPFlag("db.dbName", flag.Lookup("dbName"))
	viper.BindPFlag("db.sslMode", flag.Lookup("dbSslmode"))
	viper.BindPFlag("notifications.firebaseKeyFile", flag.Lookup("firebaseKey"))
	viper.BindPFlag("ethereum.ethSigningKey", flag.Lookup("ethSingningKey"))

	// metrics
	viper.BindPFlag("metrics.enabled", flag.Lookup("metricsEnabled"))
	viper.BindPFlag("metrics.refreshInterval", flag.Lookup("metricsRefreshInterval"))

	// check if config file exists
	_, err = os.Stat(cfg.DataDir + "/dvotemanager.yml")
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
	log.Debugf("initializing config %+v %+v %+v", *cfg, *cfg.API, *cfg.DB)
	if !cfg.ValidMode() {
		log.Fatalf("invalid mode %s", cfg.Mode)
	}

	ctx := context.Background()
	opt := option.WithCredentialsFile(cfg.Notifications.FirebaseKeyFile)
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		log.Fatal(err)
	}
	client, err := app.Auth(ctx)
	if err != nil {
		log.Fatal(err)
	}

	token, err := client.CustomToken(ctx, "pepe2")
	if err != nil {
		log.Errorf("cannot create new token: (%s)", err)
	}
	log.Infof("new token: %s", token)

	user := &auth.UserToCreate{}
	user.UID("pepe3")
	user.DisplayName("Pepe Tres")

	_, err = client.CreateUser(ctx, user)
	if err != nil {
		log.Error(err)
	}

	ur, err := client.GetUser(ctx, "pepe2")
	if err != nil {
		log.Error(err)
	}
	log.Warnf("%+v", ur.UserInfo)

	it := client.Users(ctx, "")
	for {
		if a, err := it.Next(); err == iterator.Done {
			break
		} else {
			if err != nil {
				log.Fatal(err)
			}
			log.Infof("%+v", *a.UserRecord.UserInfo)
		}
	}

	// ethereum sign key
	signer := ethereum.NewSignKeys()
	if err := signer.AddHexKey(cfg.Ethereum.SigningKey); err != nil {
		log.Fatal(err)
	}

	// create and init proxy
	ep, err := endpoint.NewEndpoint(cfg, signer)
	if err != nil {
		log.Fatal(err)
	}

	// ethereum service
	var node *chain.EthChainContext
	if len(cfg.Notifications.FirebaseKeyFile) > 0 {
		node, err = service.Ethereum(cfg.Ethereum, cfg.Web3, ep.Proxy, signer, ep.MetricsAgent)
		if err != nil {
			log.Fatal(err)
		}
	}
	// wait ethereum node to be ready
	if !cfg.Ethereum.NoWaitSync {
		requiredPeers := 2
		if len(cfg.Web3.W3External) > 0 {
			requiredPeers = 1
		}
		for {
			if info, err := node.SyncInfo(); err == nil && info.Synced && info.Peers >= requiredPeers && info.Height > 0 {
				log.Infof("ethereum blockchain synchronized (%+v)", info)
				break
			}
			time.Sleep(time.Second * 5)
		}
	}
	// ethereum events service
	if !cfg.Ethereum.NoWaitSync {
		var evh []ethevents.EventHandler
		var w3uri string
		switch {
		case cfg.Web3.W3External == "":
			// If local ethereum node enabled, use the Go-Ethereum websockets endpoint
			w3uri = "ws://" + net.JoinHostPort(cfg.Web3.RPCHost, fmt.Sprintf("%d", cfg.Web3.RPCPort))
		case strings.HasPrefix(cfg.Web3.W3External, "ws"):
			w3uri = cfg.Web3.W3External
		case strings.HasSuffix(cfg.Web3.W3External, "ipc"):
			w3uri = cfg.Web3.W3External

		default:
			log.Fatal("web3 external must be websocket or IPC for event subscription")
		}

		// @jordipainan TODO: Handle events
		//evh = append(evh, ethevents.HandleVochainOracle)

		var initBlock *int64
		if !cfg.EthEventConfig.SubscribeOnly {
			initBlock = new(int64)
			chainSpecs, err := chain.SpecsFor(cfg.Ethereum.ChainType)
			if err != nil {
				log.Warn("cannot get chain block to start looking for events, using 0")
				*initBlock = 0
			} else {
				*initBlock = chainSpecs.StartingBlock
			}
		}
		if err := service.EthEvents(cfg.Ethereum.ProcessDomain, w3uri, cfg.Ethereum.ChainType, initBlock, nil, signer, nil, evh); err != nil {
			log.Fatal(err)
		}
	}
}
