package tokenapi

import (
	"fmt"

	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/dvote/httprouter"
	"go.vocdoni.io/dvote/log"
	"go.vocdoni.io/dvote/metrics"
	"go.vocdoni.io/manager/database"
	"go.vocdoni.io/manager/rpcapi"
)

// TokenAPI is a handler for external token managmement
type TokenAPI struct {
	api *rpcapi.RPCAPI
	db  database.Database
	ma  *metrics.Agent
}

// NewTokenAPI creates a new token API handler for the Router
func NewTokenAPI(r *httprouter.HTTProuter, route string, d database.Database, ma *metrics.Agent) (*TokenAPI, error) {
	if r == nil || d == nil {
		return nil, fmt.Errorf("invalid arguments for manager API")
	}

	signer := ethereum.NewSignKeys()
	signer.Generate()
	api, err := rpcapi.NewAPI(signer, r, "tokenapi", route+"/token", ma, false)
	if err != nil {
		return nil, fmt.Errorf("could not create the manager API: %v", err)
	}
	return &TokenAPI{api: api, db: d, ma: ma}, nil
}

// RegisterMethods registers all tokenAPI methods behind the given path
func (t *TokenAPI) EnableAPI() error {
	log.Infof("enabling tokenAPI")

	t.api.RegisterPublic("revoke", false, t.revoke)
	t.api.RegisterPublic("status", false, t.status)
	t.api.RegisterPublic("generate", false, t.generate)
	t.api.RegisterPublic("importKeysBulk", false, t.importKeysBulk)
	t.api.RegisterPublic("listKeys", false, t.listKeys)
	t.api.RegisterPublic("deleteKeys", false, t.deleteKeys)
	return nil
}
