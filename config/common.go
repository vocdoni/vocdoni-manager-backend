package config

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

// MetricsCfg initializes the metrics config
type MetricsCfg struct {
	Enabled         bool
	RefreshInterval int
}
