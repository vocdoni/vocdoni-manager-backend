package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"

	"gitlab.com/vocdoni/go-dvote/chain"
	"gitlab.com/vocdoni/go-dvote/chain/ethevents"
	"gitlab.com/vocdoni/go-dvote/crypto/ethereum"
	log "gitlab.com/vocdoni/go-dvote/log"
	"gitlab.com/vocdoni/go-dvote/service"
	"gitlab.com/vocdoni/manager/manager-backend/config"
	"gitlab.com/vocdoni/manager/manager-backend/database"
	"gitlab.com/vocdoni/manager/manager-backend/database/pgsql"
	"gitlab.com/vocdoni/manager/manager-backend/notify"
	endpoint "gitlab.com/vocdoni/manager/manager-backend/services/api-endpoint"
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
	flag.StringVar(&cfg.DataDir, "dataDir", home+"/.dvotenotif", "directory where data is stored")
	cfg.Mode = *flag.String("mode", "all", fmt.Sprintf("operation mode: %s", func() (modes []string) {
		for m := range config.Modes {
			modes = append(modes, m)
		}
		return
	}()))

	// global
	cfg.LogLevel = *flag.String("logLevel", "info", "Log level (debug, info, warn, error, fatal)")
	cfg.LogOutput = *flag.String("logOutput", "stdout", "Log output (stdout, stderr or filepath)")
	cfg.LogErrorFile = *flag.String("logErrorFile", "", "Log errors and warnings to a file")
	cfg.SaveConfig = *flag.Bool("saveConfig", false, "overwrites an existing config file with the CLI provided flags")
	cfg.DB.Host = *flag.String("dbHost", "127.0.0.1", "DB server address")
	cfg.DB.Port = *flag.Int("dbPort", 5432, "DB server port")
	cfg.DB.User = *flag.String("dbUser", "vocdoni", "DB Username")
	cfg.DB.Password = *flag.String("dbPassword", "vocdoni", "DB password")
	cfg.DB.Dbname = *flag.String("dbName", "vocdoni", "DB database name")
	cfg.DB.Sslmode = *flag.String("dbSslmode", "prefer", "DB postgres sslmode")
	// notifications
	cfg.Notifications.KeyFile = *flag.String("pushNotificationsKeyFile", "", "path to notifications service private key file")
	cfg.Notifications.Service = *flag.Int("pushNotificationsService", notify.Firebase, "push notifications service, 1: Firebase")
	//ethereum node
	cfg.Ethereum.SigningKey = *flag.String("ethSigningKey", "", "signing private Key (if not specified the Ethereum keystore will be used)")
	cfg.Ethereum.ChainType = *flag.String("ethChain", "goerli", fmt.Sprintf("Ethereum blockchain to use: %s", chain.AvailableChains))
	cfg.Ethereum.LightMode = *flag.Bool("ethChainLightMode", false, "synchronize Ethereum blockchain in light mode")
	cfg.Ethereum.NodePort = *flag.Int("ethNodePort", 30303, "Ethereum p2p node port to use")
	cfg.Ethereum.BootNodes = *flag.StringArray("ethBootNodes", []string{}, "Ethereum p2p custom bootstrap nodes (enode://<pubKey>@<ip>[:port])")
	cfg.Ethereum.TrustedPeers = *flag.StringArray("ethTrustedPeers", []string{}, "Ethereum p2p trusted peer nodes (enode://<pubKey>@<ip>[:port])")
	cfg.Ethereum.ProcessDomain = *flag.String("ethProcessDomain", "voting-process.vocdoni.eth", "voting contract ENS domain")
	cfg.Ethereum.NoWaitSync = *flag.Bool("ethNoWaitSync", false, "do not wait for Ethereum to synchronize (for testing only)")
	// ethereum events
	cfg.EthereumEvents.CensusSync = *flag.Bool("ethCensusSync", true, "automatically import new census published on the smart contract")
	cfg.EthereumEvents.SubscribeOnly = *flag.Bool("ethSubscribeOnly", true, "only subscribe to new ethereum events (do not read past log)")
	// ethereum web3
	cfg.Web3.W3External = *flag.String("w3External", "", "use an external web3 endpoint instead of the local one. Supported protocols: http(s)://, ws(s):// and IPC filepath")
	cfg.Web3.Enabled = *flag.Bool("w3Enabled", true, "if true, a web3 public endpoint will be enabled")
	cfg.Web3.Route = *flag.String("w3Route", "/web3", "web3 endpoint API route")
	cfg.Web3.RPCPort = *flag.Int("w3RPCPort", 9091, "web3 RPC port")
	cfg.Web3.RPCHost = *flag.String("w3RPCHost", "127.0.0.1", "web3 RPC host")
	// ipfs
	cfg.IPFS.NoInit = *flag.Bool("ipfsNoInit", false, "disables inter planetary file system support")
	cfg.IPFS.SyncKey = *flag.String("ipfsSyncKey", "", "enable IPFS cluster synchronization using the given secret key")
	cfg.IPFS.SyncPeers = *flag.StringArray("ipfsSyncPeers", []string{}, "use custom ipfsSync peers/bootnodes for accessing the DHT")
	// db migrations
	cfg.Migrate.Action = *flag.String("migrateAction", "", "Migration action (up,down,status)")
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
	viper.BindPFlag("db.host", flag.Lookup("dbHost"))
	viper.BindPFlag("db.port", flag.Lookup("dbPort"))
	viper.BindPFlag("db.user", flag.Lookup("dbUser"))
	viper.BindPFlag("db.password", flag.Lookup("dbPassword"))
	viper.BindPFlag("db.dbName", flag.Lookup("dbName"))
	viper.BindPFlag("db.sslMode", flag.Lookup("dbSslmode"))
	// notifications
	viper.BindPFlag("notifications.KeyFile", flag.Lookup("pushNotificationsKeyFile"))
	viper.BindPFlag("notifications.Service", flag.Lookup("pushNotificationsService"))
	// ethereum node
	viper.Set("ethereum.datadir", cfg.DataDir+"/ethereum")
	viper.BindPFlag("ethereum.signingKey", flag.Lookup("ethSigningKey"))
	viper.BindPFlag("ethereum.chainType", flag.Lookup("ethChain"))
	viper.BindPFlag("ethereum.lightMode", flag.Lookup("ethChainLightMode"))
	viper.BindPFlag("ethereum.nodePort", flag.Lookup("ethNodePort"))
	viper.BindPFlag("ethereum.bootNodes", flag.Lookup("ethBootNodes"))
	viper.BindPFlag("ethereum.trustedPeers", flag.Lookup("ethTrustedPeers"))
	viper.BindPFlag("ethereum.processDomain", flag.Lookup("ethProcessDomain"))
	viper.BindPFlag("ethereum.noWaitSync", flag.Lookup("ethNoWaitSync"))
	viper.BindPFlag("ethereumEvents.censusSync", flag.Lookup("ethCensusSync"))
	viper.BindPFlag("ethereumEvents.subscribeOnly", flag.Lookup("ethSubscribeOnly"))
	// ethereum web3
	viper.BindPFlag("web3.w3External", flag.Lookup("w3External"))
	viper.BindPFlag("web3.route", flag.Lookup("w3Route"))
	viper.BindPFlag("web3.enabled", flag.Lookup("w3Enabled"))
	viper.BindPFlag("web3.RPCPort", flag.Lookup("w3RPCPort"))
	viper.BindPFlag("web3.RPCHost", flag.Lookup("w3RPCHost"))
	// ipfs
	viper.Set("ipfs.ConfigPath", cfg.DataDir+"/ipfs")
	viper.BindPFlag("ipfs.NoInit", flag.Lookup("ipfsNoInit"))
	viper.BindPFlag("ipfs.SyncKey", flag.Lookup("ipfsSyncKey"))
	viper.BindPFlag("ipfs.SyncPeers", flag.Lookup("ipfsSyncPeers"))
	// db migrations
	viper.BindPFlag("migrate.action", flag.Lookup("migrateAction"))
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

	// ethereum sign key
	signer := ethereum.NewSignKeys()
	if err := signer.AddHexKey(cfg.Ethereum.SigningKey); err != nil {
		log.Fatalf("invalid signing key: %s", err)
	}

	// create and init proxy
	ep, err := endpoint.NewEndpoint(cfg, signer)
	if err != nil {
		log.Fatal(err)
	}

	// ethereum service
	var node *chain.EthChainContext
	if cfg.Web3.W3External == "" {
		node, err = service.Ethereum(cfg.Ethereum, cfg.Web3, ep.Proxy, signer, ep.MetricsAgent)
		if err != nil {
			log.Fatal(err)
		}
	}
	// wait ethereum node to be ready if local node
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

	// db
	var db database.Database

	// postgres with sqlx
	db, err = pgsql.New(cfg.DB)
	if err != nil {
		log.Fatal(err)
	}

	// standalone Migrations
	if cfg.Migrate.Action != "" {
		if err := pgsql.Migrator(cfg.Migrate.Action, db); err != nil {
			log.Fatal(err)
		}
		return
	}

	// check that all migrations are applied before proceeding
	// and if not apply them
	if err := pgsql.Migrator("upSync", db); err != nil {
		log.Fatal(err)
	}

	// init notifications service
	var fa notify.PushNotifier
	if len(cfg.Notifications.KeyFile) > 0 {
		// create file tracker
		ipfsFileTracker := notify.NewIPFSFileTracker(cfg.IPFS, ep.MetricsAgent, db)
		switch cfg.Notifications.Service {
		case notify.Firebase:
			fa = notify.NewFirebaseAdmin(cfg.Notifications.KeyFile, ipfsFileTracker)
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

	// Handle ethereum events and notify
	evh = append(evh, fa.HandleEthereum)

	var initBlock *int64
	if !cfg.EthereumEvents.SubscribeOnly {
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

	// start notifications API
	if cfg.Mode == "notifications" || cfg.Mode == "all" {
		log.Infof("enabling Notifications API methods")
		notif := notify.NewAPI(ep.Router, fa, ep.MetricsAgent)
		if err := notif.RegisterMethods(cfg.API.Route); err != nil {
			log.Fatal(err)
		}
	}

	// Only start routing once we have registered all methods. Otherwise we
	// have a data race.
	go ep.Router.Route()

	log.Info("startup complete")

	// close if interrupt received
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	log.Warnf("received SIGTERM, exiting at %s", time.Now().Format(time.RFC850))
	os.Exit(0)
}
