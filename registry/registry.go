package registry

import (
	"gitlab.com/vocdoni/vocdoni-manager-backend/router"
	"gitlab.com/vocdoni/vocdoni-manager-backend/types"
)

type Registry struct {
	Router *router.Router
}

func NewRegistry(r *router.Router) *Registry {
	return &Registry{Router: r}
}

func (r *Registry) RegisterMethods() error {
	if err := r.Router.AddHandler("info", r.info, false); err != nil {
		return err
	}
	return nil
}

func (r *Registry) info(request router.RouterRequest) {
	var response types.ResponseMessage
	r.Router.Transport.Send(r.Router.BuildReply(request, response))
}
