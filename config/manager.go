package config

import (
	"fmt"
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
}

func (m *Manager) String() string {
	return fmt.Sprintf("API: %+v,  DB: %+v, SMTP: %+v, LogLevel: %s, LogOutput: %s, LogErrorFile: %s,  Metrics: %+v, Mode: %s, DataDir: %s, SaveConfig: %v, SigningKey: %s, Migrate: %+v",
		*m.API, *m.DB, *m.SMTP, m.LogLevel, m.LogOutput, m.LogErrorFile, *m.Metrics, m.Mode, m.DataDir, m.SaveConfig, m.SigningKey, *m.Migrate)
}

func (m *Manager) ValidMode() bool {
	return Modes[m.Mode]
}

// NewManagerConfig initializes the fields in the config stuct
func NewManagerConfig() *Manager {
	return &Manager{
		API:     new(API),
		DB:      new(DB),
		Migrate: new(Migrate),
		SMTP:    new(SMTP),
		Metrics: new(MetricsCfg),
	}
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

type Migrate struct {
	// Action defines the migration action to be taken (up, down, status)
	Action string
}
