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
	"github.com/prometheus/client_golang/prometheus"
	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/dvote/log"

	"go.vocdoni.io/manager/types"
)

func (r *Registry) register(request *types.APIrequest) (*types.APIresponse, error) {
	var err error
	var member *types.Member
	var user types.User
	var uid uuid.UUID
	var response types.APIresponse
	// increase stats counter
	RegistryRequests.With(prometheus.Labels{"method": "register"}).Inc()

	user.PubKey = request.SignaturePublicKey

	// check entityId exists
	entityID := request.EntityID
	if _, err := r.db.Entity(request.EntityID); err != nil {
		if err == sql.ErrNoRows {
			log.Warnf("register: invalid entity ID %x", entityID)
			return nil, fmt.Errorf("invalid entityID")
		}
		log.Errorf("register: error retrieving entity %x", entityID)
		return nil, fmt.Errorf("error retrieving entity")
	}

	// either token or valid member info should be valid
	if !checkMemberInfo(request.MemberInfo) {
		log.Warnf("register: invalid member info %v", request.MemberInfo)
		return nil, fmt.Errorf("invalid member info")
	}
	if uid, err = r.db.AddMember(entityID, user.PubKey, request.MemberInfo); err != nil {
		log.Error(err)
		return nil, fmt.Errorf(fmt.Sprintf("cannot create member: (%s)", err))
	}
	member = &types.Member{ID: uid, PubKey: user.PubKey, EntityID: entityID, MemberInfo: *request.MemberInfo}

	log.Infof("new member added %+v for entity %s", *member, request.EntityID)
	// increase stats counter
	RegistryRequests.With(prometheus.Labels{"method": "register_success"}).Inc()

	return &response, nil

}

func (r *Registry) validateToken(request *types.APIrequest) (*types.APIresponse, error) {
	var uid uuid.UUID
	var response types.APIresponse

	// increase stats counter
	RegistryRequests.With(prometheus.Labels{"method": "validateToken"}).Inc()
	log.Debugf("got validateToken request with pubKey %x", request.SignaturePublicKey)

	// either token or valid member info should be valid
	if len(request.Token) == 0 {
		log.Warnf("empty token validation for entity %s", request.EntityID)
		return nil, fmt.Errorf("invalid token")
	}
	var err error
	if uid, err = uuid.Parse(request.Token); err != nil {
		log.Warnf("invalid token id format %s for entity %s: (%v)", request.Token, request.EntityID, err)
		return nil, fmt.Errorf("invalid token format")
	}
	// check entityId exists
	entity, err := r.db.Entity(request.EntityID)
	if err != nil {
		if err == sql.ErrNoRows {
			RegistryRequests.With(prometheus.Labels{"method": "validateToken_error_entity"}).Inc()
			log.Warnf("trying to validate token  %s for non-existing combination entity %s", request.Token, request.EntityID)
			return nil, fmt.Errorf("invalid entity id")

		}
		log.Warnf("error retrieving entity (%q) to validate token (%q): (%q)", request.EntityID, request.Token, err)
		return nil, fmt.Errorf("error retrieving entity")
	}
	member, err := r.db.Member(request.EntityID, &uid)
	if err != nil {
		if err == sql.ErrNoRows { // token does not exist
			RegistryRequests.With(prometheus.Labels{"method": "validateToken_error_invalid_token"}).Inc()
			log.Warnf("using non-existing combination of token  %s and entity %s: (%v)", request.Token, request.EntityID, err)
			return nil, fmt.Errorf("invalid token id")
		}
		log.Warnf("error retrieving member (%q) for entity (%q): (%q)", request.Token, request.EntityID, err)
		return nil, fmt.Errorf("error retrieving token")
	}

	// 1.
	if bytes.Equal(member.PubKey, request.SignaturePublicKey) {
		RegistryRequests.With(prometheus.Labels{"method": "validateToken_error_already_registered"}).Inc()
		log.Warnf("pubKey (%q) with token  (%q)  already registered for entity (%q): (%q)", fmt.Sprintf("%x", member.PubKey), request.Token, request.EntityID, err)
		return nil, fmt.Errorf("duplicate user already registered")
	} else if len(member.PubKey) != 0 {
		RegistryRequests.With(prometheus.Labels{"method": "validateToken_error_reused_token"}).Inc()
		log.Warnf("pubKey (%q) with token  (%q)  already registered for entity (%q): (%q)", fmt.Sprintf("%x", member.PubKey), request.Token, request.EntityID, err)
		return nil, fmt.Errorf("invalid token")
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

	if err = r.db.RegisterMember(request.EntityID, request.SignaturePublicKey, &uid); err != nil {
		log.Warnf("cannot register member for entity %s: (%v)", request.EntityID, err)
		msg := "invalidToken"
		// if err.Error() == "duplicate user" {
		// 	msg = "duplicate user"
		// }
		return nil, fmt.Errorf(msg)
	}
	log.Debugf("new user registered with pubKey: %x", request.SignaturePublicKey)

	_, err = url.ParseRequestURI(entity.CallbackURL)
	if err == nil {
		go callback(entity.CallbackURL, entity.CallbackSecret, "register", uid)
	} else {
		log.Debugf("no callback URL defined for (%x)", request.EntityID)
	}

	// remove pedning tag if exists
	tagName := "PendingValidation"
	tag, err := r.db.TagByName(request.EntityID, tagName)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Errorf("error retrieving PendingValidationTag: (%v)", err)
		}
	}
	if tag != nil {
		if _, _, err := r.db.RemoveTagFromMembers(request.EntityID, []uuid.UUID{member.ID}, tag.ID); err != nil {
			log.Errorf("error removing pendingValidationTag from member %s : (%v)", member.ID.String(), err)
		}
	}

	log.Infof("token %s validated for Entity %x", request.Token, request.EntityID)
	// increase stats counter
	RegistryRequests.With(prometheus.Labels{"method": "validateToken_sucess"}).Inc()

	return &response, nil

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

func (r *Registry) registrationStatus(request *types.APIrequest) (*types.APIresponse, error) {
	var member *types.Member
	var response types.APIresponse

	// increase stats counter
	RegistryRequests.With(prometheus.Labels{"method": "status"}).Inc()
	log.Debugf("got registrationStatus request with pubKey %x", request.SignaturePublicKey)

	// check if entity exists
	if _, err := r.db.Entity(request.EntityID); err != nil {
		log.Warn(err)

		return nil, fmt.Errorf("entity does not exist")
	}

	// check if user is a member
	var err error
	if member, err = r.db.MemberPubKey(request.EntityID, request.SignaturePublicKey); err != nil {
		// user is not a member but exists
		if err == sql.ErrNoRows {
			response.Status = &types.Status{
				Registered:  false,
				NeedsUpdate: false,
			}
			RegistryRequests.With(prometheus.Labels{"method": "status_not_registered"}).Inc()
			return &response, nil
		}
		log.Warn(err)
		return nil, fmt.Errorf("cannot query for member")
	}
	// user exists and is member
	if member != nil {
		// increase stats counter
		RegistryRequests.With(prometheus.Labels{"method": "status_registered"}).Inc()
		response.Status = &types.Status{
			Registered:  true,
			NeedsUpdate: false,
		}
	}
	return &response, nil
}

func (r *Registry) subscribe(request *types.APIrequest) (*types.APIresponse, error) {
	/*
		var response types.APIresponse
		var err error
		var token []byte
		var uid uuid.UUID
		var member *types.Member
		var user *types.User
		if token, err = hex.DecodeString(request.Token); err != nil {
			log.Warn(err)
			return nil, fmt.Errorf("invalid token hexstring")
			return
		}
		if uid, err = uuid.FromBytes(token); err != nil {
			log.Warn(err)
			return nil, fmt.Errorf("invalid token")
			return
		}

		// check if user exists
		if user, err = r.db.User([]byte(request.SignaturePublicKey)); err != nil {
			log.Error(err)
			if err.Error() == "sql: no rows in result set" {
				return nil, fmt.Errorf("user does not exist")
			} else {
				return nil, fmt.Errorf("cannot query for user")
			}
			return
		}

		// decode entityID
		entityID, err := hex.DecodeString(request.EntityID)
		if err != nil {
			log.Warn(err)
			return nil, fmt.Errorf("invalid entityId")
			return
		}
		// check if entity exists
		if _, err := r.db.Entity(entityID); err != nil {
			log.Warn(err)
			return nil, fmt.Errorf("entity does not exist")
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
			return nil, fmt.Errorf("cannot query for member")
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
		return nil, fmt.Errorf("already subscribed")
	*/
	return &types.APIresponse{
		Ok: true,
	}, nil
}

func (r *Registry) unsubscribe(request *types.APIrequest) (*types.APIresponse, error) {
	/*
		var response types.APIresponse
		var err error
		var token []byte
		var uid uuid.UUID
		var member *types.Member
		var user *types.User
		if token, err = hex.DecodeString(request.Token); err != nil {
			log.Warn(err)
			return nil, fmt.Errorf("invalid token hexstring")
			return
		}
		if uid, err = uuid.FromBytes(token); err != nil {
			log.Warn(err)
			return nil, fmt.Errorf("invalid token")
			return
		}

		// check if user exists
		if user, err = r.db.User([]byte(request.SignaturePublicKey)); err != nil {
			log.Error(err)
			if err.Error() == "sql: no rows in result set" {
				return nil, fmt.Errorf("user does not exist")
			} else {
				return nil, fmt.Errorf("cannot query for user")
			}
			return
		}

		// decode entityID
		entityID, err := hex.DecodeString(request.EntityID)
		if err != nil {
			log.Warn(err)
			return nil, fmt.Errorf("invalid entityId")
			return
		}
		// check if entity exists
		if _, err := r.db.Entity(entityID); err != nil {
			log.Warn(err)
			return nil, fmt.Errorf("entity does not exist")
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
			return nil, fmt.Errorf("cannot query for member")
			return
		}

		// subscribed
		if r.db.EntityHas(entityID, uid) {
			// TBD: DELETE MEMBER QUERY ?
		}
		// not subscribed
		return nil, fmt.Errorf("not subscribed")
	*/
	return &types.APIresponse{
		Ok: true,
	}, nil
}

func (r *Registry) listSubscriptions(request *types.APIrequest) (*types.APIresponse, error) {
	return &types.APIresponse{
		Ok: true,
	}, nil
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
