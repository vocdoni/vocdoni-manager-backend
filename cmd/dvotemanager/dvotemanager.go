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
	"go.vocdoni.io/manager/database"
	"go.vocdoni.io/manager/database/pgsql"
	"go.vocdoni.io/manager/ethclient"
	"go.vocdoni.io/manager/manager"
	"go.vocdoni.io/manager/registry"
	endpoint "go.vocdoni.io/manager/services/api-endpoint"
	"go.vocdoni.io/manager/smtpclient"
	"go.vocdoni.io/manager/tokenapi"
	"go.vocdoni.io/manager/types"

	"go.vocdoni.io/dvote/crypto/ethereum"
	chain "go.vocdoni.io/dvote/ethereum"
	log "go.vocdoni.io/dvote/log"
	"go.vocdoni.io/manager/config"
)

func newConfig() (*config.Manager, config.Error) {
	var err error
	var cfgError config.Error
	cfg := config.NewManagerConfig()
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
	cfg.SigningKeys = *flag.StringArray("signingKeys", []string{}, "signing private Keys (if not specified, a new one will be created), the first one is the oracle public key")
	cfg.GatewayUrls = *flag.StringArray("gatewayUrls", []string{"https://gw1.vocdoni.net/dvote"}, "urls to use as gateway api endpoints")
	cfg.API.Route = *flag.String("apiRoute", "/api", "dvote API route")
	cfg.API.ListenHost = *flag.String("listenHost", "0.0.0.0", "API endpoint listen address")
	cfg.API.ListenPort = *flag.Int("listenPort", 8000, "API endpoint http port")
	cfg.API.Ssl.Domain = *flag.String("sslDomain", "", "enable TLS secure domain with LetsEncrypt auto-generated certificate")
	cfg.DB.Host = *flag.String("dbHost", "127.0.0.1", "DB server address")
	cfg.DB.Port = *flag.Int("dbPort", 5432, "DB server port")
	cfg.DB.User = *flag.String("dbUser", "user", "DB Username")
	cfg.DB.Password = *flag.String("dbPassword", "password", "DB password")
	cfg.DB.Dbname = *flag.String("dbName", "database", "DB database name")
	cfg.DB.Sslmode = *flag.String("dbSslmode", "prefer", "DB postgres sslmode")
	cfg.Migrate.Action = *flag.String("migrateAction", "", "Migration action (up,down,status)")
	cfg.SMTP.Host = *flag.String("smtpHost", "127.0.0.1", "SMTP server host")
	cfg.SMTP.Port = *flag.Int("smtpPort", 587, "SMTP server port")
	cfg.SMTP.User = *flag.String("smtpUser", "user", "SMTP Username")
	cfg.SMTP.Password = *flag.String("smtpPassword", "password", "SMTP password")
	cfg.SMTP.PoolSize = *flag.Int("smtpPoolSize", 4, "SMTP connection pool size")
	cfg.SMTP.Timeout = *flag.Int("smtpTimeout", 30, "SMTP send timout in seconds")
	cfg.SMTP.ValidationURL = *flag.String("smtpValidationURL", "https://vocdoni.link/validation", "URL prefix of the token validation service")
	cfg.SMTP.WebpollURL = *flag.String("smtpWebpollURL", "https://manager.vocdoni.net/processes/vote/#", "URL prefix of the token validation service")
	cfg.SMTP.Sender = *flag.String("smtpSender", "validation@bender.vocdoni.io", "SMTP Sender address")
	cfg.SMTP.SenderName = *flag.String("smtpSenderName", "Vocdoni", "Name that appears as sender identity in emails")
	cfg.SMTP.Contact = *flag.String("smtpContact", "contact@vocdoni.io", "Fallback contact email address in emails")
	cfg.EthNetwork.Name = *flag.String("ethNetworkName", "goerli", fmt.Sprintf("Ethereum blockchain to use: %s", chain.AvailableChains))
	cfg.EthNetwork.Provider = *flag.String("ethNetworkProvider", "", "Ethereum network gateway")
	cfg.EthNetwork.GasLimit = *flag.Uint64("ethNetworkGasLimit", 0, "Gas limit for sending an EVM transaction in units")
	cfg.EthNetwork.FaucetAmount = *flag.Int("ethNetworkFaucetAmount", 0*types.Finney, "Amount of eth or similar to be provided upon an entity sign up (in milliEther)")
	cfg.EthNetwork.Timeout = *flag.Duration("ethNetworkTimeout", 60*time.Second, "Timeout for ethereum transactions (default: 60s) ")
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
	viper.BindPFlag("signingKeys", flag.Lookup("signingKeys"))
	viper.BindPFlag("gatewayUrls", flag.Lookup("gatewayUrls"))
	viper.BindPFlag("api.route", flag.Lookup("apiRoute"))
	viper.BindPFlag("api.listenHost", flag.Lookup("listenHost"))
	viper.BindPFlag("api.listenPort", flag.Lookup("listenPort"))
	viper.Set("api.ssl.dirCert", cfg.DataDir+"/tls")
	viper.BindPFlag("api.ssl.domain", flag.Lookup("sslDomain"))
	viper.BindPFlag("db.host", flag.Lookup("dbHost"))
	viper.BindPFlag("db.port", flag.Lookup("dbPort"))
	viper.BindPFlag("db.user", flag.Lookup("dbUser"))
	viper.BindPFlag("db.password", flag.Lookup("dbPassword"))
	viper.BindPFlag("db.dbName", flag.Lookup("dbName"))
	viper.BindPFlag("db.sslMode", flag.Lookup("dbSslmode"))
	viper.BindPFlag("migrate.action", flag.Lookup("migrateAction"))
	viper.BindPFlag("smtp.host", flag.Lookup("smtpHost"))
	viper.BindPFlag("smtp.port", flag.Lookup("smtpPort"))
	viper.BindPFlag("smtp.user", flag.Lookup("smtpUser"))
	viper.BindPFlag("smtp.password", flag.Lookup("smtpPassword"))
	viper.BindPFlag("smtp.poolSize", flag.Lookup("smtpPoolSize"))
	viper.BindPFlag("smtp.timeOut", flag.Lookup("smtpTimeout"))
	viper.BindPFlag("smtp.validationURL", flag.Lookup("smtpValidationURL"))
	viper.BindPFlag("smtp.webpollURL", flag.Lookup("smtpWebpollURL"))
	viper.BindPFlag("smtp.sender", flag.Lookup("smtpSender"))
	viper.BindPFlag("smtp.senderName", flag.Lookup("smtpSenderName"))
	viper.BindPFlag("smtp.contact", flag.Lookup("smtpContact"))
	viper.BindPFlag("ethnetwork.name", flag.Lookup("ethNetworkName"))
	viper.BindPFlag("ethnetwork.provider", flag.Lookup("ethNetworkProvider"))
	viper.BindPFlag("ethnetwork.gasLimit", flag.Lookup("ethNetworkGasLimit"))
	viper.BindPFlag("ethnetwork.faucetAmount", flag.Lookup("ethNetworkFaucetAmount"))
	viper.BindPFlag("ethnetwork.timeout", flag.Lookup("ethNetworkTimeout"))
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

	// Generate and save signing key if nos specified
	if len(cfg.SigningKeys) == 0 {
		fmt.Println("no signing keys, generating one...")
		signer := ethereum.NewSignKeys()
		signer.Generate()
		if err != nil {
			cfgError = config.Error{
				Message: fmt.Sprintf("cannot generate signing key: %s", err),
			}
			return cfg, cfgError
		}
		_, priv := signer.HexString()
		viper.Set("signingKeys", priv)
		cfg.SigningKeys[0] = priv
		cfg.SaveConfig = true
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
	// var err error
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
	if !cfg.ValidMode() {
		log.Fatalf("invalid mode %s", cfg.Mode)
	}

	// Signer
	signer := ethereum.NewSignKeys()
	for idx, key := range cfg.SigningKeys {
		cfg.SigningKeys[idx] = strings.Trim(key, `"[]`)
	}
	if err := signer.AddHexKey(cfg.SigningKeys[0]); err != nil {
		log.Fatal(err)
	}
	pub, _ := signer.HexString()
	log.Infof("my public key: %s", pub)
	log.Infof("my address: %s", signer.AddressString())

	// Gateways
	for idx, url := range cfg.GatewayUrls {
		cfg.GatewayUrls[idx] = strings.Trim(url, `"[]`)
	}

	// vocclient.New(cfg.GatewayUrls, "")

	// WS Endpoint and Router
	ep, err := endpoint.NewEndpoint(cfg, signer)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	// Database Interface
	var db database.Database

	// Postgres with sqlx
	db, err = pgsql.New(cfg.DB)
	if err != nil {
		log.Fatal(err)
	}

	// Standalone Migrations
	if cfg.Migrate.Action != "" {
		if err := pgsql.Migrator(cfg.Migrate.Action, db); err != nil {
			log.Fatal(err)
		}
		return
	}

	// Check that all migrations are applied before proceeding
	// and if not apply them
	if err := pgsql.Migrator("upSync", db); err != nil {
		log.Fatal(err)
	}

	// Generate SMTP config object
	smtp := smtpclient.New(cfg.SMTP)
	if err := smtp.StartPool(); err != nil {
		log.Fatal(err)
	}
	defer smtp.ClosePool()

	// User registry
	if cfg.Mode == "registry" || cfg.Mode == "all" {
		log.Infof("enabling Registry API methods")
		reg := registry.NewRegistry(ep.Router, db, ep.MetricsAgent)
		if err := reg.RegisterMethods(cfg.API.Route); err != nil {
			log.Fatal(err)
		}
	}

	// Manager
	signers := make([]*ethclient.Signer, len(cfg.SigningKeys))
	privs := make([]string, len(cfg.SigningKeys))
	for idx, key := range cfg.SigningKeys {
		kpair := ethereum.NewSignKeys()
		kpair.AddHexKey(key)
		signers[idx] = &ethclient.Signer{
			SignKeys: kpair,
			Taken:    make(chan bool, 1),
		}
		_, priv := kpair.HexString()
		privs[idx] = priv
		log.Infof("Added signer %s for faucet", signers[idx].SignKeys.AddressString())
	}
	viper.Set("signingKeys", signers)
	cfg.SaveConfig = true
	var ethClient *ethclient.Eth
	if len(cfg.EthNetwork.Name) != 0 {
		ethClient, err = ethclient.New(ctx, cfg.EthNetwork, signers)
		if err != nil {
			log.Fatalf("cannot connect to ethereum endpoint: %w", err)
		}
		balance, err := ethClient.BalanceAt(context.Background(), signer.Address(), nil)
		if err != nil {
			log.Errorf("cannot get signer balance: (%v)", err)
		}
		log.Infof("my %s balance is %s", cfg.EthNetwork.Name, balance.String())
	}

	if cfg.Mode == "manager" || cfg.Mode == "all" {
		log.Infof("enabling Manager API methods")
		mgr := manager.NewManager(ep.Router, db, smtp, ethClient)
		if err := mgr.RegisterMethods(cfg.API.Route); err != nil {
			log.Fatal(err)
		}
	}

	// External token API
	if cfg.Mode == "token" || cfg.Mode == "all" {
		log.Infof("enabling Token API methods")
		tok := tokenapi.NewTokenAPI(ep.Router, db, ep.MetricsAgent)
		if err := tok.RegisterMethods(cfg.API.Route); err != nil {
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
