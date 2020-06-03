package registry

import (
	"encoding/hex"
	"fmt"

	"github.com/badoux/checkmail"
	"github.com/google/uuid"
	"gitlab.com/vocdoni/go-dvote/crypto/ethereum"
	"gitlab.com/vocdoni/go-dvote/crypto/snarks"
	"gitlab.com/vocdoni/go-dvote/log"
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
	var member *types.Member
	var user types.User
	var uid uuid.UUID
	var response types.ResponseMessage

	// check public key
	if len(request.PubKey) != ethereum.PubKeyLength {
		r.Router.SendError(request, "invalid public key")
		return
	}
	if user.PubKey, err = hex.DecodeString(request.PubKey); err != nil {
		log.Warn(err)
		r.Router.SendError(request, "canot decode public key")
		return
	}

	if request.PubKey != request.SignaturePublicKey {
		log.Warnf("%s != %s", request.PubKey, request.SignaturePublicKey)
		r.Router.SendError(request, "public key does not match")
		return
	}

	var u *types.User
	if u, err = r.db.User(user.PubKey); err != nil {
		log.Error(err)
		r.Router.SendError(request, "cannot query for user")
		return
	}
	if u == nil {
		user.DigestedPubKey = snarks.Poseidon.Hash(user.PubKey)
		if err = r.db.AddUser(&user); err != nil {
			log.Error(err)
			r.Router.SendError(request, "unkown error on AddUser")
			return
		}
	}

	// check entityId exist
	entityID, err := hex.DecodeString(request.EntityID)
	if err != nil {
		log.Warn(err)
		r.Router.SendError(request, "wrong entityId")
		return
	}
	if entity, err = r.db.Entity(entityID); err != nil {
		log.Warn(err)
		r.Router.SendError(request, "entity does not exist")
		return
	}
	if string(entityID) != string(entity.ID) {
		r.Router.SendError(request, "invalid entity")
		return
	}

	// either token or valid member info should be valid
	if len(request.Token) == 0 {
		if !checkMemberInfo(request.Member) {
			r.Router.SendError(request, "invalid member info")
			return
		}
		r.db.AddUser(&user)
		if err = r.db.AddMember(entityID, user.PubKey, &request.Member.MemberInfo); err != nil {
			log.Warn(err)
			r.Router.SendError(request, fmt.Sprintf("cannot create member: (%s)", err))
			return
		}
	} else {
		var token []byte
		if token, err = hex.DecodeString(request.Token); err != nil {
			log.Warn(err)
			r.Router.SendError(request, "invalid token hexstring")
			return
		}
		if uid, err = uuid.FromBytes(token); err != nil {
			log.Warn(err)
			r.Router.SendError(request, "invalid token")
			return
		}
		member, err = r.db.Member(uid)
		if err != nil {
			log.Warn(err)
			r.Router.SendError(request, "invalid token id")
			return
		}
		// TO-DO set MemberInfo
	}
	log.Infof("member created: %+v", *member)
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
