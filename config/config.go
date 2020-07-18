package config

var Modes = map[string]bool{
	"registry": true,
	"manager":  true,
	"all":      true,
}

type Manager struct {
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

func (m *Manager) ValidMode() bool {
	return Modes[m.Mode]
}

// NewConfig initializes the fields in the config stuct
func NewConfig() *Manager {
	return &Manager{
		API:     new(API),
		DB:      new(DB),
		Migrate: new(Migrate),
		Metrics: new(MetricsCfg),
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
