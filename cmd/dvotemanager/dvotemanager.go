package main

import (
	"fmt"
	"os"

	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"

	log "gitlab.com/vocdoni/go-dvote/log"
	"gitlab.com/vocdoni/vocdoni-manager-backend/config"
)

func newConfig() (*config.Manager, config.Error) {
	var err error
	var cfgError config.Error
	globalCfg := config.NewConfig()
	home, err := os.UserHomeDir()
	if err != nil {
		cfgError = config.Error{
			Critical: true,
			Message:  fmt.Sprintf("cannot get user home directory with error: %s", err),
		}
		return nil, cfgError
	}

	// global
	flag.StringVar(&globalCfg.DataDir, "dataDir", home+"/.dvotemanager", "directory where data is stored")
	globalCfg.LogLevel = *flag.String("logLevel", "info", "Log level (debug, info, warn, error, fatal)")
	globalCfg.LogOutput = *flag.String("logOutput", "stdout", "Log output (stdout, stderr or filepath)")
	globalCfg.LogErrorFile = *flag.String("logErrorFile", "", "Log errors and warnings to a file")
	globalCfg.SaveConfig = *flag.Bool("saveConfig", false, "overwrites an existing config file with the CLI provided flags")

	globalCfg.API.Route = *flag.String("apiRoute", "/dvote", "dvote API route")
	globalCfg.API.ListenHost = *flag.String("listenHost", "0.0.0.0", "API endpoint listen address")
	globalCfg.API.ListenPort = *flag.Int("listenPort", 9090, "API endpoint http port")
	// ssl
	globalCfg.API.Ssl.Domain = *flag.String("sslDomain", "", "enable TLS secure domain with LetsEncrypt auto-generated certificate (listenPort=443 is required)")

	// parse flags
	flag.Parse()

	// setting up viper
	viper := viper.New()
	viper.AddConfigPath(globalCfg.DataDir)
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
	viper.Set("api.ssl.dirCert", globalCfg.DataDir+"/tls")
	viper.BindPFlag("api.ssl.domain", flag.Lookup("sslDomain"))

	// check if config file exists
	_, err = os.Stat(globalCfg.DataDir + "/dvotemanager.yml")
	if os.IsNotExist(err) {
		cfgError = config.Error{
			Message: fmt.Sprintf("creating new config file in %s", globalCfg.DataDir),
		}
		// creting config folder if not exists
		err = os.MkdirAll(globalCfg.DataDir, os.ModePerm)
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
				Message: fmt.Sprintf("cannot read loaded config file in %s: %s", globalCfg.DataDir, err),
			}
		}
	}
	err = viper.Unmarshal(&globalCfg)
	if err != nil {
		cfgError = config.Error{
			Message: fmt.Sprintf("cannot unmarshal loaded config file: %s", err),
		}
	}

	if globalCfg.SaveConfig {
		viper.Set("saveConfig", false)
		if err := viper.WriteConfig(); err != nil {
			cfgError = config.Error{
				Message: fmt.Sprintf("cannot overwrite config file into config dir: %s", err),
			}
		}
	}

	return globalCfg, cfgError
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

}
