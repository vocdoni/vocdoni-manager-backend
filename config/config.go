package config

import "gitlab.com/vocdoni/go-dvote/config"

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
	// Notifications
	Notifications *Notifications
	// Ethereum node config
	Ethereum *config.EthCfg
	// Web3 endpoint config
	Web3 *config.W3Cfg
	// EthereumEvents ethereum even subscription config options
	EthereumEvents *config.EthEventCfg
	// IPFS config options
	IPFS *config.IPFSCfg
}

func (m *Manager) ValidMode() bool {
	return Modes[m.Mode]
}

// NewConfig initializes the fields in the config stuct
func NewConfig() *Manager {
	return &Manager{
		API:            new(API),
		DB:             new(DB),
		Migrate:        new(Migrate),
		SMTP:           new(SMTP),
		Metrics:        new(MetricsCfg),
		Notifications:  new(Notifications),
		Ethereum:       new(config.EthCfg),
		Web3:           new(config.W3Cfg),
		EthereumEvents: new(config.EthEventCfg),
		IPFS:           new(config.IPFSCfg),
	}
}

type DB struct {
	Host     string
	Port     int
	User     string
	Password string
	Dbname   string
	Sslmode  string
}

type SMTP struct {
	Host          string
	Port          int
	User          string
	Password      string
	PoolSize      int
	ValidationURL string
	Sender        string
	SenderName    string
	Contact       string
}

type API struct {
	// Route is the URL router where the API will be served
	Route string
	// ListenPort port where the API server will listen on
	ListenPort int
	// ListenHost host where the API server will listen on
	ListenHost string
	// Ssl tls related config options
	Ssl struct {
		Domain  string
		DirCert string
	}
}

type Error struct {
	// Critical indicates if the error encountered is critical and the app must be stopped
	Critical bool
	// Message error message
	Message string
}

type Migrate struct {
	// Action defines the migration action to be taken (up, down, status)
	Action string
}

// MetricsCfg initializes the metrics config
type MetricsCfg struct {
	Enabled         bool
	RefreshInterval int
}

type Notifications struct {
	Service int
	KeyFile string
}
