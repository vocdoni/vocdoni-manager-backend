package test

import (
	"testing"

	"gitlab.com/vocdoni/go-dvote/crypto/signature"
	"gitlab.com/vocdoni/go-dvote/log"
	"gitlab.com/vocdoni/vocdoni-manager-backend/config"
	"gitlab.com/vocdoni/vocdoni-manager-backend/database"
	"gitlab.com/vocdoni/vocdoni-manager-backend/database/testdb"

	endpoint "gitlab.com/vocdoni/vocdoni-manager-backend/services/api-endpoint"
)

type TestAPI struct {
	DB     database.Database
	EP     *endpoint.EndPoint
	Signer *signature.SignKeys
}

func (t *TestAPI) Start(tb testing.TB, host, route string, port int) {
	log.Init("info", "stdout")
	var err error
	// Signer
	t.Signer = new(signature.SignKeys)
	t.Signer.Generate()

	cfg := &config.Manager{
		API: &config.API{
			Route:      route,
			ListenPort: port,
			ListenHost: host,
		},
	}
	// WS Endpoint and Router
	t.EP, err = endpoint.NewEndpoint(cfg, t.Signer)
	if err != nil {
		tb.Fatal(err)
	}

	// Mock database
	t.DB, err = testdb.New("host", 1234, "user", "password", "dbname", "sslmode")
	if err != nil {
		tb.Fatal(err)
	}

}
