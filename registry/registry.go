package registry

import (
	"gitlab.com/vocdoni/vocdoni-manager-backend/router"
	"gitlab.com/vocdoni/vocdoni-manager-backend/types"
)

type Registry struct {
	Router *router.Router
}

// NewRegistry creates a new registry handler for the Router
func NewRegistry(r *router.Router) *Registry {
	return &Registry{Router: r}
}

// RegisterMethods registers all registry methods behind the given path
func (r *Registry) RegisterMethods(path string) error {
	r.Router.Transport.AddNamespace(path + "/registry")
	if err := r.Router.AddHandler("info", path+"/registry", r.info, false); err != nil {
		return err
	}
	return nil
}

func (r *Registry) info(request router.RouterRequest) {
	var response types.ResponseMessage
	r.Router.Transport.Send(r.Router.BuildReply(request, response))
}
