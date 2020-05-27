package registry

import (
	"github.com/badoux/checkmail"
	"gitlab.com/vocdoni/vocdoni-manager-backend/database"
	"gitlab.com/vocdoni/vocdoni-manager-backend/router"
	"gitlab.com/vocdoni/vocdoni-manager-backend/types"
)

type Registry struct {
	Router *router.Router
	db     database.Database
}

// NewRegistry creates a new registry handler for the Router
func NewRegistry(r *router.Router, d database.Database) *Registry {
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

func (r *Registry) send(req router.RouterRequest, resp types.ResponseMessage) {
	r.Router.Transport.Send(r.Router.BuildReply(req, resp))
}

func (r *Registry) register(request router.RouterRequest) {
	var entity *types.Entity
	var err error
	var response types.ResponseMessage
	// check entityId exist
	if entity, err = r.db.Entity(request.EntityID); err != nil {
		r.Router.SendError(request, err.Error())
		return
	}
	if !checkMemberInfo(request.Member) {
		r.Router.SendError(request, "invalid member info")
		return
	}
	member, err := r.db.Member(request.Member.ID)
	if err != nil {
		r.Router.SendError(request, "invalid id")
		return

	}
	if member.EntityID != entity.ID {
		r.Router.SendError(request, "invalid entity")
		return
	}
	// check token or check form fields
	r.send(request, response)
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
	return true
}
