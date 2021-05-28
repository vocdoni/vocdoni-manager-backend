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
	SigningKey string
	// Migration options
	Migrate *Migrate
	// Web3 connection options
	EthNetwork *EthereumCfg
	// Faucet config
	Faucet *FaucetCfg
}

func (m *Manager) String() string {
	return fmt.Sprintf("API: %+v,  DB: %+v, SMTP: %+v, LogLevel: %s, LogOutput: %s, LogErrorFile: %s,  Metrics: %+v, Mode: %s, DataDir: %s, SaveConfig: %v, SigningKey: %s,  SMTP: %v, Migrate: %+v, Eth: %v",
		*m.API, *m.DB, *m.SMTP, m.LogLevel, m.LogOutput, m.LogErrorFile, *m.Metrics, m.Mode, m.DataDir, m.SaveConfig, m.SigningKey, *m.SMTP, *m.Migrate, *m.EthNetwork)
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
		EthNetwork: new(EthereumCfg),
		Faucet:     new(FaucetCfg),
	}
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

type EthereumCfg struct {
	// Name is the Ethereum Network Name
	// currently supported: "mainnet", "sokol", goerli", "xdai",
	// more info at:
	// https://github.com/vocdoni/vocdoni-node/blob/8b5a1fbc161603b96831fed7b0748190afff0bff/chain/blockchains.go
	Name string
	// DialAddress is the URL to connect with
	DialAddress string
	// Timeout applied to the ethereum transactions
	Timeout time.Duration
}

type FaucetCfg struct {
	// Amount amount to send
	Amount int64
	// GasPrice gas price for sending a transactions
	GasPrice int64
	// GasLimit gas limit for sending a transactions
	GasLimit int64
	// Signers list of signers for the signers pool
	Signers []string
	// MaxBalance is the maximum amount an address can have
	// in order to receive more faucet funds
	MaxBalance int64
}
