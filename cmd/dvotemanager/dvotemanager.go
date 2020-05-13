package main

import (
	"fmt"
	"os"

	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"

	log "gitlab.com/vocdoni/go-dvote/log"
	"gitlab.com/vocdoni/vocdoni-manager-backend/config"
	endpoint "gitlab.com/vocdoni/vocdoni-manager-backend/services/api-endpoint"
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
	cfg.LogLevel = *flag.String("logLevel", "info", "Log level (debug, info, warn, error, fatal)")
	cfg.LogOutput = *flag.String("logOutput", "stdout", "Log output (stdout, stderr or filepath)")
	cfg.LogErrorFile = *flag.String("logErrorFile", "", "Log errors and warnings to a file")
	cfg.SaveConfig = *flag.Bool("saveConfig", false, "overwrites an existing config file with the CLI provided flags")
	cfg.API.Route = *flag.String("apiRoute", "/api", "dvote API route")
	cfg.API.ListenHost = *flag.String("listenHost", "0.0.0.0", "API endpoint listen address")
	cfg.API.ListenPort = *flag.Int("listenPort", 8000, "API endpoint http port")
	cfg.API.Ssl.Domain = *flag.String("sslDomain", "", "enable TLS secure domain with LetsEncrypt auto-generated certificate")

	// parse flags
	flag.Parse()

	// setting up viper
	viper := viper.New()
	viper.AddConfigPath(cfg.DataDir)
	viper.SetConfigName("dvotemanager")
	viper.SetConfigType("yml")

	// binding flags to viper

	// global
	viper.BindPFlag("dataDir", flag.Lookup("dataDir"))
	viper.BindPFlag("logLevel", flag.Lookup("logLevel"))
	viper.BindPFlag("logErrorFile", flag.Lookup("logErrorFile"))
	viper.BindPFlag("logOutput", flag.Lookup("logOutput"))

	viper.BindPFlag("api.route", flag.Lookup("apiRoute"))
	viper.BindPFlag("api.listenHost", flag.Lookup("listenHost"))
	viper.BindPFlag("api.listenPort", flag.Lookup("listenPort"))
	viper.Set("api.ssl.dirCert", cfg.DataDir+"/tls")
	viper.BindPFlag("api.ssl.domain", flag.Lookup("sslDomain"))

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
	log.Debugf("initializing config %+v", *cfg)

	_, err := endpoint.NewEndpoint(cfg)
	if err != nil {
		log.Fatal(err)
	}
}
