package config

type Manager struct {
	// API api config options
	API *API
	// LogLevel logging level
	LogLevel string
	// LogOutput logging output
	LogOutput string
	// ErrorLogFile for logging warning, error and fatal messages
	LogErrorFile string
	// DataDir path where the gateway files will be stored
	DataDir string
	// SaveConfig overwrites the config file with the CLI provided flags
	SaveConfig bool
}

// NewConfig initializes the fields in the config stuct
func NewConfig() *Manager {
	return &Manager{
		API: new(API),
	}
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
