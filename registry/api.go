package registry

import (
	"fmt"

	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/dvote/httprouter"
	"go.vocdoni.io/dvote/log"
	"go.vocdoni.io/dvote/metrics"
	"go.vocdoni.io/manager/database"
	"go.vocdoni.io/manager/rpcapi"
)

type Registry struct {
	api *rpcapi.RPCAPI
	db  database.Database
	ma  *metrics.Agent
}

// NewRegistry creates a new registry handler for the Router
func NewRegistry(signer *ethereum.SignKeys, r *httprouter.HTTProuter, route string, d database.Database, ma *metrics.Agent) (*Registry, error) {
	if r == nil || d == nil {
		return nil, fmt.Errorf("invalid arguments for manager API")
	}

	api, err := rpcapi.NewAPI(signer, r, "registry Mobile", route+"/registry", nil, false)
	if err != nil {
		return nil, fmt.Errorf("could not create the manager API: %v", err)
	}
	// api := jsonrpcapi.NewSignedJRPC(signer, types.NewApiRequest, types.NewApiResponse, false)
	// rpcapi.AddNamespace("manager", api)
	// rpcapi.APIs = append(rpcapi.APIs, "manager")
	// api.AddAuthorizedAddress(signer.Address())
	// rpcapi.ManagerAPI = api
	return &Registry{api: api, db: d, ma: ma}, nil
}

// RegisterMethods registers all registry methods behind the given path
func (r *Registry) EnableAPI() error {
	log.Infof("enabling registry API")

	r.api.RegisterPublic("register", true, r.register)
	r.api.RegisterPublic("validateToken", true, r.validateToken)
	r.api.RegisterPublic("registrationStatus", true, r.registrationStatus)
	r.api.RegisterPublic("subscribe", true, r.subscribe)
	r.api.RegisterPublic("unsubscribe", true, r.unsubscribe)
	r.api.RegisterPublic("listSubscriptions", true, r.listSubscriptions)
	r.registerMetrics()
	return nil
}
