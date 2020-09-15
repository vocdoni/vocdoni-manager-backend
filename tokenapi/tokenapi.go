package tokenapi

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gitlab.com/vocdoni/go-dvote/crypto/ethereum"
	"gitlab.com/vocdoni/go-dvote/log"
	"gitlab.com/vocdoni/go-dvote/metrics"
	"gitlab.com/vocdoni/go-dvote/net"
	"gitlab.com/vocdoni/go-dvote/util"
	"gitlab.com/vocdoni/manager/manager-backend/database"
	"gitlab.com/vocdoni/manager/manager-backend/router"
	"gitlab.com/vocdoni/manager/manager-backend/types"
)

// AuthWindowSeconds is the time window (in seconds) that the tokenapi Auth tolerates
const AuthWindowSeconds = 5

// TokenAPI is a handler for external token managmement
type TokenAPI struct {
	Router *router.Router
	db     database.Database
	ma     *metrics.Agent
}

// NewTokenAPI creates a new token API handler for the Router
func NewTokenAPI(r *router.Router, d database.Database, ma *metrics.Agent) *TokenAPI {
	return &TokenAPI{Router: r, db: d, ma: ma}
}

// RegisterMethods registers all tokenAPI methods behind the given path
func (t *TokenAPI) RegisterMethods(path string) error {
	var transport net.Transport
	if tr, ok := t.Router.Transports["httpws"]; ok {
		transport = tr
	} else if tr, ok = t.Router.Transports["http"]; ok {
		transport = tr
	} else if tr, ok = t.Router.Transports["ws"]; ok {
		transport = tr
	} else {
		return fmt.Errorf("no compatible transports found (ws or http)")
	}

	log.Infof("adding namespace token %s", path+"/token")
	transport.AddNamespace(path + "/token")
	if err := t.Router.AddHandler("revoke", path+"/token", t.revoke, false, true); err != nil {
		return err
	}
	if err := t.Router.AddHandler("status", path+"/token", t.status, false, true); err != nil {
		return err
	}
	if err := t.Router.AddHandler("generate", path+"/token", t.generate, false, true); err != nil {
		return err
	}
	return nil
}

func (t *TokenAPI) send(req *router.RouterRequest, resp *types.MetaResponse) {
	if req == nil || req.MessageContext == nil || resp == nil {
		log.Errorf("message context or request is nil, cannot send reply message")
		return
	}
	req.Send(t.Router.BuildReply(req, resp))
}

func checkAuth(fields []string, timestamp int32, auth string) bool {
	if len(fields) == 0 {
		return false
	}
	current := int32(time.Now().Unix())

	if timestamp > current+AuthWindowSeconds || timestamp < current-AuthWindowSeconds {
		log.Warnf("timestamp out of window")
		return false
	}
	toHash := ""
	for _, f := range fields {
		toHash += f
	}
	thisAuth := hex.EncodeToString(ethereum.HashRaw([]byte(toHash)))
	return thisAuth == util.TrimHex(auth)
}

func (t *TokenAPI) getSecret(entityID []byte) (string, error) {
	entity, err := t.db.Entity(entityID)
	if err != nil {
		return "", err
	}
	return entity.CallbackSecret, nil
}

func (t *TokenAPI) revoke(request router.RouterRequest) {
	if len(request.EntityID) == 0 {
		log.Warnf("trying to revoke token %q for null entity %s", request.Token, request.EntityID)
		t.Router.SendError(request, "invalid entityId")
		return

	}
	// check entityId exists
	entityID, err := hex.DecodeString(util.TrimHex(request.EntityID))
	if err != nil {
		log.Warnf("trying to revoke token %q but cannot decode entityId %q : (%v)", request.Token, request.EntityID, err)
		t.Router.SendError(request, "invalid entityId")
		return
	}
	// either token or valid member info should be valid
	if len(request.Token) == 0 {
		log.Warnf("empty token validation for entity %s", request.EntityID)
		t.Router.SendError(request, "invalid token")
		return
	}
	var uid uuid.UUID
	if uid, err = uuid.Parse(request.Token); err != nil {
		log.Warnf("invalid token id format %s for entity %s: (%v)", request.Token, request.EntityID, err)
		t.Router.SendError(request, "invalid token format")
		return
	}

	secret, err := t.getSecret(entityID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Warnf("invalid authentication: trying to validate token  %q for non-existing combination with entity %s", request.Token, request.EntityID)
			t.Router.SendError(request, "invalid authentication")
			return

		}
		log.Warnf("invalid authentication: error retrieving entity (%q) to validate token (%q): (%v)", request.EntityID, request.Token, err)
		t.Router.SendError(request, "invalid authentication")
		return
	}
	if !checkAuth(
		[]string{request.EntityID, request.Method, fmt.Sprintf("%d", request.Timestamp), request.Token, secret},
		request.Timestamp,
		request.AuthHash) {
		log.Warnf("invalid authentication: checkAuth error for entity (%q) to validate token (%q): (%v)", request.EntityID, request.Token, err)
		t.Router.SendError(request, "invalid authentication")
		return
	}
	if err = t.db.DeleteMember(entityID, &uid); err != nil {
		log.Warnf("database error: could not delete token (%q) for entity (%q): (%v)", request.Token, request.EntityID, err)
		t.Router.SendError(request, "could not delete member")
		return
	}

	log.Infof("deleted member with token (%q) for entity (%s)", request.Token, request.EntityID)
	var resp types.MetaResponse
	t.send(&request, &resp)
}

func (t *TokenAPI) status(request router.RouterRequest) {
	var resp types.MetaResponse

	if len(request.EntityID) == 0 {
		log.Warnf("trying to revoke token %q for null entity %s", request.Token, request.EntityID)
		t.Router.SendError(request, "invalid entityId")
		return

	}
	// check entityId exists
	entityID, err := hex.DecodeString(util.TrimHex(request.EntityID))
	if err != nil {
		log.Warnf("trying retrieve status of token %q but cannot decode entityId %q : (%v)", request.Token, request.EntityID, err)
		t.Router.SendError(request, "invalid entityId")
		return
	}
	// either token or valid member info should be valid
	if len(request.Token) == 0 {
		log.Warnf("empty token validation for entity %s", request.EntityID)
		t.Router.SendError(request, "invalid token")
		return
	}
	var uid uuid.UUID
	if uid, err = uuid.Parse(request.Token); err != nil {
		log.Warnf("invalid token id format %s for entity %s: (%v)", request.Token, request.EntityID, err)
		t.Router.SendError(request, "invalid token format")
		return
	}

	secret, err := t.getSecret(entityID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Warnf("invalid authentication: trying to validate token  %q for non-existing combination with entity %s", request.Token, request.EntityID)
			t.Router.SendError(request, "invalid authentication")
			return

		}
		log.Warnf("invalid authentication: error retrieving entity (%q) to validate token (%q): (%v)", request.EntityID, request.Token, err)
		t.Router.SendError(request, "invalid authentication")
		return
	}
	if !checkAuth(
		[]string{request.EntityID, request.Method, fmt.Sprintf("%d", request.Timestamp), request.Token, secret},
		request.Timestamp,
		request.AuthHash) {
		log.Warnf("invalid authentication: checkAuth error for entity (%q) to validate token (%q): (%v)", request.EntityID, request.Token, err)
		t.Router.SendError(request, "invalid authentication")
		return
	}
	member, err := t.db.Member(entityID, &uid)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Warnf("invalid token: trying to get status for token  (%q) for non-existing combination with entity %s", request.Token, request.EntityID)

		}
		log.Warnf("database error: trying to get status for token (%q) for entity (%q): (%v)", request.Token, request.EntityID, err)
		resp.TokenStatus = "invalid"
		t.send(&request, &resp)
		return
	}

	if len(hex.EncodeToString(member.PubKey)) != ethereum.PubKeyLength {
		log.Debugf("status for member with token (%q) for entity (%s): ", request.Token, request.EntityID)
		resp.TokenStatus = "available"
		t.send(&request, &resp)
		return
	}

	resp.TokenStatus = "registered"
	t.send(&request, &resp)
}

func (t *TokenAPI) generate(request router.RouterRequest) {
	var response types.MetaResponse

	if len(request.EntityID) == 0 {
		log.Warnf("trying to generate tokens for null entity %s", request.EntityID)
		t.Router.SendError(request, "invalid entityId")
		return

	}
	// check entityId exists
	entityID, err := hex.DecodeString(util.TrimHex(request.EntityID))
	if err != nil {
		log.Warnf("trying to generate tokens but cannot decode entityId %q : (%v)", request.EntityID, err)
		t.Router.SendError(request, "invalid entityId")
		return
	}

	secret, err := t.getSecret(entityID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Warnf("invalid authentication: trying to generate tokens for non-existing combination with entity %s", request.EntityID)
			t.Router.SendError(request, "invalid authentication")
			return

		}
		log.Warnf("invalid authentication: error retrieving entity (%q) to generate tokens: (%v)", request.EntityID, err)
		t.Router.SendError(request, "invalid authentication")
		return
	}
	if !checkAuth(
		[]string{fmt.Sprintf("%d", request.Amount), request.EntityID, request.Method, fmt.Sprintf("%d", request.Timestamp), secret},
		request.Timestamp,
		request.AuthHash) {
		log.Warnf("invalid authentication: checkAuth error for entity (%q) to generate tokens: (%v)", request.EntityID, err)
		t.Router.SendError(request, "invalid authentication")
		return
	}

	if request.Amount < 1 {
		log.Warnf("invalid token amount requested by %s", request.EntityID)
		t.Router.SendError(request, "invalid token amount")
		return
	}

	for i := 0; i < request.Amount; i++ {
		response.Tokens = append(response.Tokens, uuid.New())
	}
	// TODO: Probably I need to initialize tokens
	if err = t.db.CreateMembersWithTokens(entityID, response.Tokens); err != nil {
		log.Errorf("could not create members with generated tokens for %q: (%v)", request.SignaturePublicKey, err)
		t.Router.SendError(request, "could not generate tokens")
		return
	}

	log.Debugf("Entity: %q generateTokens: %d tokens", request.SignaturePublicKey, len(response.Tokens))
	t.send(&request, &response)
}
