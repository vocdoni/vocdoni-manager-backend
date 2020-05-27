package testcommon

import (
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

func (t *TestAPI) Start(host, route string, port int) error {
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
	if t.EP, err = endpoint.NewEndpoint(cfg, t.Signer); err != nil {
		return err
	}

	// Mock database
	if t.DB, err = testdb.New("host", 1234, "user", "password", "dbname", "sslmode"); err != nil {
		return err
	}
	return nil
}
