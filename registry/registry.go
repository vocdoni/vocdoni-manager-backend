package registry

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/badoux/checkmail"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/vocdoni/go-dvote/crypto/ethereum"
	"gitlab.com/vocdoni/go-dvote/log"
	"gitlab.com/vocdoni/go-dvote/util"
	"gitlab.com/vocdoni/manager/manager-backend/database"
	"gitlab.com/vocdoni/manager/manager-backend/router"
	"gitlab.com/vocdoni/manager/manager-backend/services/metrics"
	"gitlab.com/vocdoni/manager/manager-backend/types"
)

type Registry struct {
	Router *router.Router
	db     database.Database
	ma     *metrics.Agent
}

// NewRegistry creates a new registry handler for the Router
func NewRegistry(r *router.Router, d database.Database, ma *metrics.Agent) *Registry {
	return &Registry{Router: r, db: d, ma: ma}
}

// RegisterMethods registers all registry methods behind the given path
func (r *Registry) RegisterMethods(path string) error {
	r.Router.Transport.AddNamespace(path + "/registry")
	if err := r.Router.AddHandler("register", path+"/registry", r.register, false); err != nil {
		return err
	}
	if err := r.Router.AddHandler("validateToken", path+"/registry", r.validateToken, false); err != nil {
		return err
	}
	if err := r.Router.AddHandler("registrationStatus", path+"/registry", r.registrationStatus, false); err != nil {
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
	r.registerMetrics()
	return nil
}

func (r *Registry) send(req router.RouterRequest, resp types.MetaResponse) {
	r.Router.Transport.Send(r.Router.BuildReply(req, resp))
}

func (r *Registry) register(request router.RouterRequest) {
	var err error
	var member *types.Member
	var user types.User
	var uid uuid.UUID
	var response types.MetaResponse
	// increase stats counter
	RegistryRequests.With(prometheus.Labels{"method": "register"}).Inc()

	if user.PubKey, err = hex.DecodeString(request.SignaturePublicKey); err != nil {
		log.Warn(err)
		r.Router.SendError(request, "cannot decode public key")
		return
	}

	// check entityId exists
	entityID, err := hex.DecodeString(util.TrimHex(request.EntityID))
	if err != nil {
		log.Warn(err)
		r.Router.SendError(request, "invalid entityId")
		return
	}

	// either token or valid member info should be valid
	if !checkMemberInfo(request.MemberInfo) {
		r.Router.SendError(request, "invalid member info")
		return
	}
	if uid, err = r.db.AddMember(entityID, user.PubKey, request.MemberInfo); err != nil {
		log.Warn(err)
		r.Router.SendError(request, fmt.Sprintf("cannot create member: (%s)", err))
		return
	}
	member = &types.Member{ID: uid, PubKey: user.PubKey, EntityID: entityID, MemberInfo: *request.MemberInfo}

	log.Infof("new member added %+v for entity %s", *member, request.EntityID)
	r.send(request, response)

	// increase stats counter
	RegistryRequests.With(prometheus.Labels{"method": "register_success"}).Inc()
}

func (r *Registry) validateToken(request router.RouterRequest) {
	var err error
	var user types.User
	var uid uuid.UUID
	var response types.MetaResponse

	// increase stats counter
	RegistryRequests.With(prometheus.Labels{"method": "validateToken"}).Inc()

	if user.PubKey, err = hex.DecodeString(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot decode user public key: (%v)", err)
		r.Router.SendError(request, "cannot decode user public key")
		return
	}

	// check entityId exists
	entityID, err := hex.DecodeString(util.TrimHex(request.EntityID))
	if err != nil {
		log.Warnf("invalid entityId %s : (%v)", request.EntityID, err)
		r.Router.SendError(request, "invalid entityId")
		return
	}

	// either token or valid member info should be valid
	if len(request.Token) == 0 {
		log.Warnf("empty token validation for entity %s", request.EntityID)
		r.Router.SendError(request, "invalid token")
		return
	}
	if uid, err = uuid.Parse(request.Token); err != nil {
		log.Warnf("invalid token id format %s for entity %s: (%v)", request.Token, request.EntityID, err)
		r.Router.SendError(request, "invalid token format")
		return
	}
	entity, err := r.db.Entity(entityID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Warnf("trying to validate token  %s for non-existing combination entity %s", uid, request.EntityID)
			r.Router.SendError(request, "invalid entity id")
			return

		}
		log.Warnf("error retrieving entity (%q) to validate token (%q): (%q)", request.EntityID, uid, err)
		r.Router.SendError(request, "error retrieving entity")
		return
	}
	_, err = r.db.Member(entityID, &uid)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Warnf("using non-existing combination of token  %s and entity %s: (%v)", uid, request.EntityID, err)
			r.Router.SendError(request, "invalid token id")
			return
		}
		log.Warnf("error retrieving member (%q) for entity (%q): (%q)", uid, request.EntityID, err)
		r.Router.SendError(request, "error retrieving token")
		return
	}
	if err = r.db.RegisterMember(entityID, user.PubKey, &uid); err != nil {
		log.Warnf("cannot register member for entity %s: (%v)", request.EntityID, err)
		msg := "cannot register member"
		if err.Error() == "duplicate user" {
			msg = "duplicate user"
		}
		r.Router.SendError(request, msg)
		return
	}

	_, err = url.ParseRequestURI(entity.CallbackURL)
	if err == nil {
		go callback(entity.CallbackURL, entity.CallbackSecret, "register", uid)
	} else {
		log.Debugf("no callback URL defined for (%x)", entityID)
	}

	log.Infof("token %s validated for Entity %x", request.Token, entityID)
	r.send(request, response)

	// increase stats counter
	RegistryRequests.With(prometheus.Labels{"method": "validateToken_sucess"}).Inc()
}

// callback example: /callback?id=63c93e6f-5326-407b-960a-f796036eca5f?ts=1594912052?auth=c4a998ec01f45b8d3939090eb155e1a4038a59996e40fb5c03e58ff0cabb7528
// TBD: do not allow localhost or private networks, that would open a possible attack vector
func callback(callbackURL, secret, event string, uid uuid.UUID) error {
	client := &http.Client{Timeout: time.Second * 5} // 5 seconds should be enough
	ts := fmt.Sprintf("%d", time.Now().Unix())
	h := ethereum.HashRaw([]byte(secret + event + uid.String() + ts))
	callbackURL = strings.ReplaceAll(callbackURL, "{ID}", uid.String())
	callbackURL = strings.ReplaceAll(callbackURL, "{TIMESTAMP}", ts)
	callbackURL = strings.ReplaceAll(callbackURL, "{AUTH}", fmt.Sprintf("%x", h))
	callbackURL = strings.ReplaceAll(callbackURL, "{EVENT}", event)
	result, err := client.Get(callbackURL)
	if err != nil {
		log.Warnf("callback GET (%s) error: (%s)", callbackURL, err)
	}
	log.Debugf("gallback Get Result: (%v)", result)
	return err
}

func (r *Registry) registrationStatus(request router.RouterRequest) {
	var member *types.Member
	var response types.MetaResponse

	// increase stats counter
	RegistryRequests.With(prometheus.Labels{"method": "status"}).Inc()

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
		// increase stats counter
		RegistryRequests.With(prometheus.Labels{"method": "status_exists"}).Inc()
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

func checkMemberInfo(m *types.MemberInfo) bool {
	// TBD: check valid dateOfBirth
	if m == nil {
		return false
	}

	if err := checkmail.ValidateFormat(m.Email); err != nil {
		return false
	}
	return true
}
