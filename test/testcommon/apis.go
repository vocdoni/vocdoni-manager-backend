package testcommon

import (
	"gitlab.com/vocdoni/go-dvote/crypto/ethereum"
	"gitlab.com/vocdoni/go-dvote/log"
	"gitlab.com/vocdoni/vocdoni-manager-backend/config"
	"gitlab.com/vocdoni/vocdoni-manager-backend/database"
	"gitlab.com/vocdoni/vocdoni-manager-backend/database/pgsql"
	"gitlab.com/vocdoni/vocdoni-manager-backend/database/testdb"
	"gitlab.com/vocdoni/vocdoni-manager-backend/registry"

	endpoint "gitlab.com/vocdoni/vocdoni-manager-backend/services/api-endpoint"
)

type TestAPI struct {
	DB     database.Database
	EP     *endpoint.EndPoint
	Port   int
	Signer *ethereum.SignKeys
}

// Start creates a new database connection and API endpoint for testing.
// If dbc is nill the testdb will be used.
// If route is nill, then the websockets API won't be initialized
func (t *TestAPI) Start(dbc *config.DB, route string) error {
	log.Init("info", "stdout")
	var err error
	if route != "" {
		// Signer
		t.Signer = new(ethereum.SignKeys)
		t.Signer.Generate()

		cfg := &config.Manager{
			API: &config.API{
				Route:      route,
				ListenPort: t.Port,
				ListenHost: "127.0.0.1",
			},
		}

		// WS Endpoint and Router
		if t.EP, err = endpoint.NewEndpoint(cfg, t.Signer); err != nil {
			return err
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
		log.Infof("enabling Registry API methods")
		reg := registry.NewRegistry(t.EP.Router, t.DB)
		if err := reg.RegisterMethods(route); err != nil {
			log.Fatal(err)
		}
	}
	return nil
}
