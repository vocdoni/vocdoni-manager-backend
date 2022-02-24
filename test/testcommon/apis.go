package testcommon

import (
	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/dvote/httprouter"
	"go.vocdoni.io/dvote/log"
	"go.vocdoni.io/manager/config"
	"go.vocdoni.io/manager/database"
	"go.vocdoni.io/manager/database/pgsql"
	"go.vocdoni.io/manager/database/testdb"
	"go.vocdoni.io/manager/manager"
	"go.vocdoni.io/manager/registry"
	"go.vocdoni.io/manager/smtpclient"
	"go.vocdoni.io/manager/tokenapi"
)

type TestAPI struct {
	DB     database.Database
	Router *httprouter.HTTProuter
	Port   int
	Signer *ethereum.SignKeys
}

// Start creates a new database connection and API endpoint for testing.
// If dbc is nill the testdb will be used.
// If route is nill, then the websockets API won't be initialized
func (t *TestAPI) Start(dbc *config.DB, route string) error {
	log.Init("info", "stdout")
	var err error
	var cfg *config.Manager
	var signer *ethereum.SignKeys
	if route != "" {
		// Signer
		signer = ethereum.NewSignKeys()
		signer.Generate()
		t.Signer = signer

		cfg = &config.Manager{
			API: &config.API{
				Route:      route,
				ListenPort: t.Port,
				ListenHost: "127.0.0.1",
			},
		}

	}
	if dbc != nil {
		// Postgres with sqlx
		if t.DB, err = pgsql.New(dbc); err != nil {
			return err
		}
	} else {
		// Mock database
		if t.DB, err = testdb.New(); err != nil {
			return err
		}
	}

	if route != "" {
		log.Infof("enabling API methods")

		var httpRouter httprouter.HTTProuter
		if err = httpRouter.Init(cfg.API.ListenHost, cfg.API.ListenPort); err != nil {
			log.Fatal(err)
		}

		smtpConfig := &config.SMTP{
			User:          "coby.rippin@ethereal.email",
			Password:      "HmjWVQ86X3Q6nKBR3u",
			Host:          "smtp.ethereal.email",
			Port:          587,
			ValidationURL: "https://vocdoni.link/validation",
			WebpollURL:    "https://webpoll.vocdoni.net",
			Sender:        "coby.rippin@ethereal.email",
			Timeout:       7,
			PoolSize:      4,
		}
		s := smtpclient.New(smtpConfig)
		if err := s.StartPool(); err != nil {
			log.Fatal(err)
		}
		// defer s.ClosePool()

		mg, err := manager.NewManager(signer, &httpRouter, "/api", t.DB, s, nil)
		if err != nil {
			log.Fatal(err)
		}

		if err := mg.EnableAPI(); err != nil {
			log.Fatal(err)
		}

		r, err := registry.NewRegistry(t.Signer, &httpRouter, "/api", t.DB, nil)
		if err != nil {
			log.Fatal(err)
		}

		if err := r.EnableAPI(); err != nil {
			log.Fatal(err)
		}

		ta, err := tokenapi.NewTokenAPI(&httpRouter, "/api", t.DB, nil)
		if err != nil {
			log.Fatal(err)
		}

		if err := ta.EnableAPI(); err != nil {
			log.Fatal(err)
		}
	}
	return nil
}
