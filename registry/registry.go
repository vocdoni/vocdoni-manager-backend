package registry

import (
	"github.com/badoux/checkmail"
	"gitlab.com/vocdoni/vocdoni-manager-backend/database"
	"gitlab.com/vocdoni/vocdoni-manager-backend/router"
	"gitlab.com/vocdoni/vocdoni-manager-backend/types"
)

type Registry struct {
	Router *router.Router
	db     *database.Database
}

// NewRegistry creates a new registry handler for the Router
func NewRegistry(r *router.Router, d *database.Database) *Registry {
	return &Registry{Router: r, db: d}
}

// RegisterMethods registers all registry methods behind the given path
func (r *Registry) RegisterMethods(path string) error {
	r.Router.Transport.AddNamespace(path + "/registry")
	if err := r.Router.AddHandler("register", path+"/registry", r.register, false); err != nil {
		return err
	}
	if err := r.Router.AddHandler("status", path+"/registry", r.status, false); err != nil {
		return err
	}
	if err := r.Router.AddHandler("subscribe", path+"/registry", r.subscribe, false); err != nil {
		return err
	}
	if err := r.Router.AddHandler("unsubscribe", path+"/registry", r.unsubscribe, false); err != nil {
		return err
	}
	if err := r.Router.AddHandler("listSubscriptions", path+"/registry", r.listSubscriptions, false); err != nil {
		return err
	}
	return nil
}

func (r *Registry) register(request router.RouterRequest) {
	// check entityId exist
	r.db.Entity(request.EntityID)
	// check entityId registration mechanism (token or form)
	// check token or check form fields
	var response types.ResponseMessage
	r.Router.Transport.Send(r.Router.BuildReply(request, response))
}

func (r *Registry) status(request router.RouterRequest) {
	var response types.ResponseMessage
	r.Router.Transport.Send(r.Router.BuildReply(request, response))
}

func (r *Registry) subscribe(request router.RouterRequest) {
	var response types.ResponseMessage
	r.Router.Transport.Send(r.Router.BuildReply(request, response))
}

func (r *Registry) unsubscribe(request router.RouterRequest) {
	var response types.ResponseMessage
	r.Router.Transport.Send(r.Router.BuildReply(request, response))
}

func (r *Registry) listSubscriptions(request router.RouterRequest) {
	var response types.ResponseMessage
	r.Router.Transport.Send(r.Router.BuildReply(request, response))
}

// ===== helpers =======

func checkMemberInfo(m *types.Member) bool {
	// TBD: check valid dateOfBirth
	if m == nil {
		return false
	}
	if err := checkmail.ValidateFormat(m.Email); err != nil {
		return false
	}
	if err := checkmail.ValidateHost(m.Email); err != nil {
		return false
	}
	return true
}
