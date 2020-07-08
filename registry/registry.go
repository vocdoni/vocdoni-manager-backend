package registry

import (
	"database/sql"
	"encoding/hex"
	"fmt"

	"github.com/badoux/checkmail"
	"github.com/google/uuid"
	"gitlab.com/vocdoni/go-dvote/crypto/ethereum"
	"gitlab.com/vocdoni/go-dvote/log"
	"gitlab.com/vocdoni/go-dvote/util"
	"gitlab.com/vocdoni/manager/manager-backend/database"
	"gitlab.com/vocdoni/manager/manager-backend/router"
	"gitlab.com/vocdoni/manager/manager-backend/types"
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

func (r *Registry) send(req router.RouterRequest, resp types.MetaResponse) {
	r.Router.Transport.Send(r.Router.BuildReply(req, resp))
}

func (r *Registry) register(request router.RouterRequest) {
	var entity *types.Entity
	var err error
	var member *types.Member
	var user types.User
	var uid uuid.UUID
	var response types.MetaResponse

	if request.PubKey != "" {
		// check public key length
		if len(request.PubKey) != ethereum.PubKeyLength {
			r.Router.SendError(request, "invalid public key")
			return
		}
		// decode public key
		if user.PubKey, err = hex.DecodeString(util.TrimHex(request.PubKey)); err != nil {
			log.Warn(err)
			r.Router.SendError(request, "cannot decode public key")
			return
		}

		// check public key against message signature extracted public key
		if request.PubKey != request.SignaturePublicKey {
			log.Warnf("%s != %s", request.PubKey, request.SignaturePublicKey)
			r.Router.SendError(request, "public key does not match")
			return
		}
	} else {
		if user.PubKey, err = hex.DecodeString(request.SignaturePublicKey); err != nil {
			log.Warn(err)
			r.Router.SendError(request, "cannot decode public key")
			return
		}
	}

	// check entityId exists
	entityID, err := hex.DecodeString(util.TrimHex(request.EntityID))
	if err != nil {
		log.Warn(err)
		r.Router.SendError(request, "invalid entityId")
		return
	}
	// get entity
	if entity, err = r.db.Entity(entityID); err != nil {
		log.Warn(err)
		r.Router.SendError(request, "entity does not exist")
		return
	}
	// compare request.EntityID vs entityID fetched from db
	if string(entityID) != string(entity.ID) {
		r.Router.SendError(request, "invalid entity")
		return
	}

	// if user does not exist create
	// check user exists
	if _, err = r.db.User(user.PubKey); err != nil {
		// user does not exist, create new
		if err == sql.ErrNoRows {
			if err = r.db.AddUser(&user); err != nil {
				log.Warn(err)
				r.Router.SendError(request, "unkown error on AddUser")
				return
			}
		} else {
			log.Error(err)
			r.Router.SendError(request, "cannot query for user")
			return
		}
	}

	// either token or valid member info should be valid
	if len(request.Token) == 0 {
		if !checkMemberInfo(request.Member) {
			r.Router.SendError(request, "invalid member info")
			return
		}
		if _, err = r.db.AddMember(entityID, user.PubKey, &request.Member.MemberInfo); err != nil {
			log.Warn(err)
			r.Router.SendError(request, fmt.Sprintf("cannot create member: (%s)", err))
			return
		}
		member = request.Member
	} else {
		var token []byte
		if token, err = hex.DecodeString(util.TrimHex(request.Token)); err != nil {
			log.Warn(err)
			r.Router.SendError(request, "invalid token hexstring")
			return
		}
		if uid, err = uuid.FromBytes(token); err != nil {
			log.Warn(err)
			r.Router.SendError(request, "invalid token")
			return
		}
		member, err = r.db.Member(entityID, uid)
		if err != nil {
			log.Warn(err)
			r.Router.SendError(request, "invalid token id")
			return
		}
		if err = r.db.UpdateMember(entityID, uid, &request.Member.MemberInfo); err != nil {
			log.Warn(err)
			r.Router.SendError(request, fmt.Sprintf("cannot set member info: (%s)", err))
			return
		}
	}
	member.EntityID = entityID
	member.PubKey = user.PubKey
	log.Infof("member added %+v", member)
	r.send(request, response)
}

func (r *Registry) status(request router.RouterRequest) {
	var member *types.Member
	var response types.MetaResponse

	signaturePubKeyBytes, err := hex.DecodeString(request.SignaturePublicKey)
	if err != nil {
		log.Warn(err)
		r.Router.SendError(request, "cannot decode public key")
		return
	}

	// check if user exists
	if _, err := r.db.User(signaturePubKeyBytes); err != nil {
		if err == sql.ErrNoRows {
			r.Router.SendError(request, "user does not exist")
			return
		}
		log.Warn(err)
		r.Router.SendError(request, "cannot query for user")
		return
	}

	// decode entityID
	entityID, err := hex.DecodeString(util.TrimHex(request.EntityID))
	if err != nil {
		log.Warn(err)
		r.Router.SendError(request, "invalid entityId")
		return
	}
	// check if entity exists
	if _, err := r.db.Entity(entityID); err != nil {
		log.Warn(err)
		r.Router.SendError(request, "entity does not exist")
		return
	}

	// check if user is a member
	if member, err = r.db.MemberPubKey(entityID, signaturePubKeyBytes); err != nil {
		// user is not a member but exists
		if err == sql.ErrNoRows {
			response.Status = &types.Status{
				Registered:  false,
				NeedsUpdate: false,
			}
			r.Router.Transport.Send(r.Router.BuildReply(request, response))
			return
		}
		log.Warn(err)
		r.Router.SendError(request, "cannot query for member")
		return
	}
	// user exists and is member
	if member != nil {
		response.Status = &types.Status{
			Registered:  true,
			NeedsUpdate: false,
		}
	}
	r.Router.Transport.Send(r.Router.BuildReply(request, response))
}

func (r *Registry) subscribe(request router.RouterRequest) {
	/*
		var response types.MetaResponse
		var err error
		var token []byte
		var uid uuid.UUID
		var member *types.Member
		var user *types.User
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

		// check if user exists
		if user, err = r.db.User([]byte(request.SignaturePublicKey)); err != nil {
			log.Error(err)
			if err.Error() == "sql: no rows in result set" {
				r.Router.SendError(request, "user does not exist")
			} else {
				r.Router.SendError(request, "cannot query for user")
			}
			return
		}

		// decode entityID
		entityID, err := hex.DecodeString(request.EntityID)
		if err != nil {
			log.Warn(err)
			r.Router.SendError(request, "invalid entityId")
			return
		}
		// check if entity exists
		if _, err := r.db.Entity(entityID); err != nil {
			log.Warn(err)
			r.Router.SendError(request, "entity does not exist")
			return
		}

		// check if member exists
		if member, err = r.db.Member(entityID, uid); err != nil {
			// member does not exist
			if err.Error() == "sql: no rows in result set" {
				r.Router.SendError(request, fmt.Sprintf("member does not exist"))
				return
			}
			log.Error(err)
			r.Router.SendError(request, "cannot query for member")
			return
		}

		// not subscribed
		if !r.db.EntityHas(entityID, uid) {
			// add member
			if _, err = r.db.AddMember(entityID, user.PubKey, &member.MemberInfo); err != nil {
				log.Warn(err)
				r.Router.SendError(request, fmt.Sprintf("cannot add member: (%s)", err))
				return
			}
			r.Router.Transport.Send(r.Router.BuildReply(request, response))
			return
		}
		// already subscribed
		r.Router.SendError(request, "already subscribed")
	*/
	r.send(request, types.MetaResponse{Ok: true})
}

func (r *Registry) unsubscribe(request router.RouterRequest) {
	/*
		var response types.MetaResponse
		var err error
		var token []byte
		var uid uuid.UUID
		var member *types.Member
		var user *types.User
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

		// check if user exists
		if user, err = r.db.User([]byte(request.SignaturePublicKey)); err != nil {
			log.Error(err)
			if err.Error() == "sql: no rows in result set" {
				r.Router.SendError(request, "user does not exist")
			} else {
				r.Router.SendError(request, "cannot query for user")
			}
			return
		}

		// decode entityID
		entityID, err := hex.DecodeString(request.EntityID)
		if err != nil {
			log.Warn(err)
			r.Router.SendError(request, "invalid entityId")
			return
		}
		// check if entity exists
		if _, err := r.db.Entity(entityID); err != nil {
			log.Warn(err)
			r.Router.SendError(request, "entity does not exist")
			return
		}

		// check if member exists
		if member, err = r.db.Member(entityID, uid); err != nil {
			// member does not exist
			if err.Error() == "sql: no rows in result set" {
				r.Router.SendError(request, fmt.Sprintf("member does not exist"))
				return
			}
			log.Error(err)
			r.Router.SendError(request, "cannot query for member")
			return
		}

		// subscribed
		if r.db.EntityHas(entityID, uid) {
			// TBD: DELETE MEMBER QUERY ?
		}
		// not subscribed
		r.Router.SendError(request, "not subscribed")
	*/
	r.send(request, types.MetaResponse{Ok: true})
}

func (r *Registry) listSubscriptions(request router.RouterRequest) {
	var response types.MetaResponse
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
