package config

import (
	"fmt"

	"go.vocdoni.io/dvote/config"
)

type Notify struct {
	// API api config options
	API *API
	// Database connection options
	DB *DB
	// LogLevel logging level
	LogLevel string
	// LogOutput logging output
	LogOutput string
	// ErrorLogFile for logging warning, error and fatal messages
	LogErrorFile string
	// Ethereum subscription
	SubscribeOnly bool
	// Metrics config options
	Metrics *MetricsCfg
	// DataDir path where the gateway files will be stored
	DataDir string
	// SaveConfig overwrites the config file with the CLI provided flags
	SaveConfig bool
	// SigningKey is the ECDSA hexString private key for signing messages
	SigningKey string
	// Env {dev, stage, default: main}
	Env string
	// Notifications
	Notifications *Notifications
	// Ethereum node config
	Ethereum *config.EthCfg
	// Web3 endpoint config
	Web3 *config.W3Cfg
	// IPFS config options
	IPFS *config.IPFSCfg
}

func (n *Notify) String() string {
	return fmt.Sprintf("API: %+v,  DB: %+v, LogLevel: %s, LogOutput: %s, LogErrorFile: %s,  Metrics: %+v, DataDir: %s, SaveConfig: %v, SigningKey: %s, Notifications: %+v, Web3: %+v, EthereumEvents: %+v, IPFS: %+v",
		*n.API, *n.DB, n.LogLevel, n.LogOutput, n.LogErrorFile, *n.Metrics, n.DataDir, n.SaveConfig, n.SigningKey, *n.Notifications, *n.Ethereum, *n.Web3, *n.IPFS)
	// return fmt.Sprintf("API: %+v,  DB: %+v, LogLevel: %s, LogOutput: %s, LogErrorFile: %s,  Metrics: %+v, DataDir: %s, SaveConfig: %v, SigningKey: %s, Notifications: %+v, Ethereum: %+v, Web3: %+v, EthereumEvents: %+v, IPFS: %+v",
	// *n.API, *n.DB, n.LogLevel, n.LogOutput, n.LogErrorFile, *n.Metrics, n.DataDir, n.SaveConfig, n.SigningKey, *n.Notifications, *n.Ethereum, *n.Web3, *n.EthereumEvents, *n.IPFS)
}

// NewNotifyConfig initializes the fields in the config stuct
func NewNotifyConfig() *Notify {
	return &Notify{
		API:           new(API),
		DB:            new(DB),
		Metrics:       new(MetricsCfg),
		Notifications: new(Notifications),
		Ethereum:      new(config.EthCfg),
		Web3:          new(config.W3Cfg),
		// EthereumEvents: new(config.EthEventCfg),
		IPFS: new(config.IPFSCfg),
	}
}

type Notifications struct {
	Service int
	KeyFile string
}
