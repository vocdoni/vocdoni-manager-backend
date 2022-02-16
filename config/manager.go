package config

import (
	"fmt"
	"time"
)

var Modes = map[string]bool{
	"registry":      true,
	"manager":       true,
	"token":         true,
	"notifications": true,
	"all":           true,
}

type Manager struct {
	// API api config options
	API *API
	// Database connection options
	DB *DB
	// SMTP options
	SMTP *SMTP
	// LogLevel logging level
	LogLevel string
	// LogOutput logging output
	LogOutput string
	// ErrorLogFile for logging warning, error and fatal messages
	LogErrorFile string
	// Metrics config options
	Metrics *MetricsCfg
	// Mode is the main operation mode
	Mode string
	// DataDir path where the gateway files will be stored
	DataDir string
	// SaveConfig overwrites the config file with the CLI provided flags
	SaveConfig bool
	// SigningKey is the ECDSA hexString private key for signing messages
	SigningKeys []string
	// Migration options
	Migrate *Migrate
	// Web3 connection options
	EthNetwork *EthNetwork
	// Hubsport integration config
	Hubspot *Hubspot
}

func (m *Manager) String() string {
	return fmt.Sprintf("API: %+v,  DB: %+v, SMTP: %+v, LogLevel: %s, LogOutput: %s, LogErrorFile: %s,  Metrics: %+v, Mode: %s, DataDir: %s, SaveConfig: %v, SigningKey: %s,  SMTP: %v, Migrate: %+v, Eth: %v",
		*m.API, *m.DB, *m.SMTP, m.LogLevel, m.LogOutput, m.LogErrorFile, *m.Metrics, m.Mode, m.DataDir, m.SaveConfig, m.SigningKeys, *m.SMTP, *m.Migrate, *m.EthNetwork)
}

func (m *Manager) ValidMode() bool {
	return Modes[m.Mode]
}

// NewManagerConfig initializes the fields in the config stuct
func NewManagerConfig() *Manager {
	return &Manager{
		API:        new(API),
		DB:         new(DB),
		Migrate:    new(Migrate),
		SMTP:       new(SMTP),
		Metrics:    new(MetricsCfg),
		EthNetwork: new(EthNetwork),
		Hubspot:    new(Hubspot),
	}
}

type Hubspot struct {
	ApiKey  string
	BaseUrl string
	Enabled bool
}

type SMTP struct {
	Host          string
	Port          int
	User          string
	Password      string
	PoolSize      int
	Timeout       int
	ValidationURL string
	Sender        string
	SenderName    string
	Contact       string
	WebpollURL    string
}

type Migrate struct {
	// Action defines the migration action to be taken (up, down, status)
	Action string
}

type EthNetwork struct {
	// NetworkName is the Ethereum Network Name
	// currently supported: "mainnet", "sokol", goerli", "xdai",
	// more info in:
	// https://github.com/vocdoni/vocdoni-node/blob/8b5a1fbc161603b96831fed7b0748190afff0bff/chain/blockchains.go
	Name string
	// Provider is the Ethereum gateway host
	Provider string
	// GasLimit is the deafult gas limit for sending an EVM transaction
	GasLimit uint64
	// FaucetAmount is the default amount of xdai/gas to send to entities
	// 1 XDAI/ETH (as xDAI is the native token for xDAI chain)
	FaucetAmount int
	// Timeout applied to the ethereum transactions
	Timeout time.Duration
}
