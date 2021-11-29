package registry

import (
	"bytes"
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/badoux/checkmail"
	"github.com/google/uuid"
	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/dvote/httprouter"
	"go.vocdoni.io/dvote/httprouter/bearerstdapi"
	"go.vocdoni.io/dvote/log"

	"go.vocdoni.io/manager/database"
	"go.vocdoni.io/manager/types"
	"go.vocdoni.io/manager/util"
)

type Registry struct {
	db database.Database
}

// NewRegistry creates a new registry handler for the Router
func NewRegistry(d database.Database) *Registry {
	return &Registry{db: d}
}

func (r *Registry) Register(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var err error
	var entityID []byte
	var signaturePubKey []byte
	var member *types.Member
	var memberInfo *types.MemberInfo
	var user types.User
	var uid uuid.UUID
	var response types.MetaResponse

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	user.PubKey = signaturePubKey

	if _, err := r.db.Entity(entityID); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("register: invalid entity ID %x", entityID)
		}
		return fmt.Errorf("register: error retrieving entity %x", entityID)
	}

	if err = util.DecodeJsonMessage(memberInfo, "memberInfo", ctx); err != nil {
		return err
	}

	// either token or valid member info should be valid
	if !checkMemberInfo(memberInfo) {
		return fmt.Errorf("register: invalid member info %v", memberInfo)
	}
	if uid, err = r.db.AddMember(entityID, user.PubKey, memberInfo); err != nil {
		return fmt.Errorf("cannot create member: (%s)", err)
	}
	member = &types.Member{ID: uid, PubKey: user.PubKey, EntityID: entityID, MemberInfo: *memberInfo}
	log.Infof("new member added %+v for entity %s", *member, entityID)
	return util.SendResponse(response, ctx)
}

func (r *Registry) ValidateToken(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var uid uuid.UUID
	var entityID []byte
	var signaturePubKey []byte
	var err error
	var response types.MetaResponse

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}

	log.Debugf("got validateToken request with pubKey %x", signaturePubKey)

	token := ctx.URLParam("token")
	// either token or valid member info should be valid
	if len(token) == 0 {
		return fmt.Errorf("empty token validation for entity %s", entityID)
	}
	if uid, err = uuid.Parse(token); err != nil {
		return fmt.Errorf("invalid token id format %s for entity %s: (%v)", token, entityID, err)
	}
	// check entityId exists
	entity, err := r.db.Entity(entityID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("trying to validate token  %s for non-existing combination entity %s", token, entityID)

		}
		return fmt.Errorf("error retrieving entity (%q) to validate token (%q): (%q)", entityID, token, err)
	}
	member, err := r.db.Member(entityID, &uid)
	if err != nil {
		if err == sql.ErrNoRows { // token does not exist
			return fmt.Errorf("using non-existing combination of token  %s and entity %s: (%v)", token, entityID, err)
		}
		return fmt.Errorf("error retrieving member (%q) for entity (%q): (%q)", token, entityID, err)
	}

	// 1.
	if bytes.Equal(member.PubKey, signaturePubKey) {
		return fmt.Errorf("pubKey (%q) with token  (%q)  already registered for entity (%q): (%q)", fmt.Sprintf("%x", member.PubKey), token, entityID, err)
	} else if len(member.PubKey) != 0 {
		return fmt.Errorf("pubKey (%q) with token  (%q)  already registered for entity (%q): (%q)", fmt.Sprintf("%x", member.PubKey), token, entityID, err)
	}

	// if len(member.PubKey) != 0 {
	// 	if string(member.PubKey) == string(requestPubKey) {
	// 		RegistryRequests.With(prometheus.Labels{"method": "validateToken_error_registered"}).Inc()
	// 		log.Warnf("pubKey (%q) with token  (%q)  already registered for entity (%q): (%q)", fmt.Sprintf("%x", member.PubKey), request.Token, request.EntityID, err)
	// 		r.Router.SendError(request, "already registered")
	// 		return
	// 	}
	// 	if user, err := r.db.User(member.PubKey); err != nil {
	// 		if err == sql.ErrNoRows {
	// 			//
	// 		} else {
	// 			log.Warnf("error retrieving user with pubkey (%q) and token (%q) for entity (%q): (%q)", fmt.Sprintf("%x", member.PubKey), request.Token, request.EntityID, err)
	// 			r.Router.SendError(request, "error retrieving token")
	// 			return
	// 		}

	// 	} else {
	// 		if string(user.PubKey) == string(member.PubKey) {
	// 			log.Warnf("error trying to reuse token  (%q)  from different pubkey (%x) and for entity (%q): (%q)", uid, fmt.Sprintf("%x", member.PubKey), request.EntityID, err)
	// 			RegistryRequests.With(prometheus.Labels{"method": "validateToken_error_token_duplicate"}).Inc()
	// 			r.Router.SendError(request, "duplicate user")
	// 		} else {
	// 			log.Warnf("UNEXPECTED: error retrieving user with pubkey (%q) and token (%q) for entity (%q): (%q)", fmt.Sprintf("%x", member.PubKey), request.Token, request.EntityID, err)
	// 			r.Router.SendError(request, "error retrieving token")

	// 		}
	// 		return
	// 	}
	// }

	if err = r.db.RegisterMember(entityID, signaturePubKey, &uid); err != nil {
		log.Warnf("cannot register member for entity %s: (%v)", entityID, err)
		msg := "invalidToken"
		// if err.Error() == "duplicate user" {
		// 	msg = "duplicate user"
		// }
		return fmt.Errorf(msg)
	}
	log.Debugf("new user registered with pubKey: %x", signaturePubKey)

	_, err = url.ParseRequestURI(entity.CallbackURL)
	if err == nil {
		go callback(entity.CallbackURL, entity.CallbackSecret, "register", uid)
	} else {
		log.Debugf("no callback URL defined for (%x)", entityID)
	}

	// remove pedning tag if exists
	tagName := "PendingValidation"
	tag, err := r.db.TagByName(entityID, tagName)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Errorf("error retrieving PendingValidationTag: (%v)", err)
		}
	}
	if tag != nil {
		if _, _, err := r.db.RemoveTagFromMembers(entityID, []uuid.UUID{member.ID}, tag.ID); err != nil {
			log.Errorf("error removing pendingValidationTag from member %s : (%v)", member.ID.String(), err)
		}
	}

	log.Infof("token %s validated for Entity %x", token, entityID)

	return util.SendResponse(response, ctx)
}

// callback: /callback?authHash={AUTH}&event={EVENT}&ts={TIMESTAMP}&token={TOKEN}
// TBD: do not allow localhost or private networks, that would open a possible attack vector
func callback(callbackURL, secret, event string, uid uuid.UUID) error {
	client := &http.Client{Timeout: time.Second * 5} // 5 seconds should be enough
	ts := fmt.Sprintf("%d", int32(time.Now().Unix()))
	h := ethereum.HashRaw([]byte(event + ts + uid.String() + secret))
	callbackURL = strings.ReplaceAll(callbackURL, "{TOKEN}", uid.String())
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

func (r *Registry) RegistrationStatus(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var err error
	var entityID []byte
	var signaturePubKey []byte
	var member *types.Member
	var response types.MetaResponse

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	log.Debugf("got registrationStatus request with pubKey %x", signaturePubKey)

	// check if entity exists
	if _, err := r.db.Entity(entityID); err != nil {
		log.Warn(err)
		return fmt.Errorf("entity does not exist")
	}

	// check if user is a member
	if member, err = r.db.MemberPubKey(entityID, signaturePubKey); err != nil {
		// user is not a member but exists
		if err == sql.ErrNoRows {
			response.Status = &types.Status{
				Registered:  false,
				NeedsUpdate: false,
			}
			return util.SendResponse(response, ctx)
		}
		log.Warn(err)
		return fmt.Errorf("cannot query for member")
	}
	// user exists and is member
	if member != nil {
		response.Status = &types.Status{
			Registered:  true,
			NeedsUpdate: false,
		}
	}
	return util.SendResponse(response, ctx)
}

func (r *Registry) Subscribe(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
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
	return util.SendResponse(types.MetaResponse{Ok: true}, ctx)
}

func (r *Registry) Unsubscribe(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
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
	return util.SendResponse(types.MetaResponse{Ok: true}, ctx)
}

func (r *Registry) ListSubscriptions(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var response types.MetaResponse
	return util.SendResponse(response, ctx)
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
