package tokenapi

import (
	"bytes"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/dvote/log"
	"go.vocdoni.io/dvote/metrics"
	"go.vocdoni.io/dvote/net"
	dvoteutil "go.vocdoni.io/dvote/util"
	"go.vocdoni.io/manager/database"
	"go.vocdoni.io/manager/router"
	"go.vocdoni.io/manager/types"
)

// AuthWindowSeconds is the time window (in seconds) that the tokenapi Auth tolerates
const AuthWindowSeconds = 300

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
	if err := t.Router.AddHandler("importKeysBulk", path+"/token", t.importKeysBulk, false, true); err != nil {
		return err
	}
	if err := t.Router.AddHandler("listKeys", path+"/token", t.listKeys, false, true); err != nil {
		return err
	}
	if err := t.Router.AddHandler("deleteKeys", path+"/token", t.deleteKeys, false, true); err != nil {
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

func checkAuth(timestamp int32, auth string, fields ...interface{}) bool {
	if len(fields) == 0 {
		return false
	}
	current := int32(time.Now().Unix())

	if timestamp > current+AuthWindowSeconds || timestamp < current-AuthWindowSeconds {
		log.Warnf("timestamp out of window")
		return false
	}
	var toHash bytes.Buffer
	for _, f := range fields {
		switch v := f.(type) {
		case string:
			toHash.WriteString(v)
		case []string:
			for _, key := range v {
				toHash.WriteString(key)
			}
		case types.ListOptions:
			toHash.WriteString(fmt.Sprintf("%d%d%s%s", v.Skip, v.Count, v.Order, v.SortBy))
		case []byte:
			toHash.Write(v)
		case types.HexBytes:
			toHash.Write(v)
		}
	}
	thisAuth := hex.EncodeToString(ethereum.HashRaw(toHash.Bytes()))
	return thisAuth == dvoteutil.TrimHex(auth)
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
		log.Errorf("trying to revoke token %q for null entity %s", request.Token, request.EntityID)
		t.Router.SendError(request, "invalid entityID")
		return

	}

	// either token or valid member info should be valid
	if len(request.Token) == 0 {
		log.Errorf("empty token validation for entity %s", request.EntityID)
		t.Router.SendError(request, "invalid token")
		return
	}

	uid, err := uuid.Parse(request.Token)
	if err != nil {
		log.Errorf("invalid token id format %s for entity %s:(%v)", request.Token, request.EntityID, err)
		t.Router.SendError(request, "invalid token format")
		return
	}

	secret, err := t.getSecret(request.EntityID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Errorf("revoke: invalid entity id %s", request.EntityID)
			t.Router.SendError(request, "invalid entityID")
			return

		}
		log.Errorf("revoke: database error error retrieving entity (%q) : (%v)", request.EntityID, err)
		t.Router.SendError(request, "error retrieving entity")
		return
	}
	if !checkAuth(
		request.Timestamp, request.AuthHash,
		request.EntityID, request.Method, fmt.Sprintf("%d", request.Timestamp), request.Token, secret) {
		log.Errorf("invalid authentication: checkAuth error for entity (%q) to validate token (%q):(%v)", request.EntityID, request.Token, err)
		t.Router.SendError(request, "invalid authentication")
		return
	}
	if err = t.db.DeleteMember(request.EntityID, &uid); err != nil {
		log.Errorf("database error: could not delete token %q for entity %q: (%v)", request.Token, request.EntityID, err)
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
		log.Errorf("invalid entity %s", request.EntityID)
		t.Router.SendError(request, "invalid entityID")
		return

	}

	// either token or valid member info should be valid
	if len(request.Token) == 0 {
		log.Errorf("empty token validation for entity %s", request.EntityID)
		t.Router.SendError(request, "invalid token")
		return
	}
	uid, err := uuid.Parse(request.Token)
	if err != nil {
		log.Errorf("invalid token id format %s for entity %s:(%v)", request.Token, request.EntityID, err)
		t.Router.SendError(request, "invalid token format")
		return
	}

	secret, err := t.getSecret(request.EntityID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Errorf("status: invalid entity id %s", request.EntityID)
			t.Router.SendError(request, "invalid entityID")
			return

		}
		log.Errorf("status: database error error retrieving entity (%q) : (%v)", request.EntityID, err)
		t.Router.SendError(request, "error retrieving entity")
		return
	}
	if !checkAuth(
		request.Timestamp, request.AuthHash,
		request.EntityID, request.Method, fmt.Sprintf("%d", request.Timestamp), request.Token, secret) {
		log.Errorf("invalid authentication: checkAuth error for entity (%q) to validate token (%q): (%v)", request.EntityID, request.Token, err)
		t.Router.SendError(request, "invalid authentication")
		return
	}
	member, err := t.db.Member(request.EntityID, &uid)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Errorf("invalid token: trying to get status for token  (%q) for non-existing combination with entity %s", request.Token, request.EntityID)

		}
		log.Errorf("database error: trying to get status for token (%q) for entity (%q): (%v)", request.Token, request.EntityID, err)
		resp.TokenStatus = "invalid"
		t.send(&request, &resp)
		return
	}

	if len(member.PubKey) != ethereum.PubKeyLengthBytes {
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
		log.Errorf("trying to generate tokens for null entity %s", request.EntityID)
		t.Router.SendError(request, "invalid entityId")
		return

	}

	secret, err := t.getSecret(request.EntityID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Errorf("generate: invalid entity id %s", request.EntityID)
			t.Router.SendError(request, "invalid entityID")
			return

		}
		log.Errorf("generate: database error error retrieving entity (%q) : (%v)", request.EntityID, err)
		t.Router.SendError(request, "error retrieving entity")
		return
	}
	if !checkAuth(
		request.Timestamp, request.AuthHash,
		fmt.Sprintf("%d", request.Amount), request.EntityID, request.Method, fmt.Sprintf("%d", request.Timestamp), secret) {
		log.Errorf("invalid authentication: checkAuth error for entity (%q) to generate tokens: (%v)", request.EntityID, err)
		t.Router.SendError(request, "invalid authentication")
		return
	}

	if request.Amount < 1 {
		log.Errorf("invalid token amount requested by %s", request.EntityID)
		t.Router.SendError(request, "invalid token amount")
		return
	}

	for i := 0; i < request.Amount; i++ {
		response.Tokens = append(response.Tokens, uuid.New())
	}
	// TODO: Probably I need to initialize tokens
	if err = t.db.CreateMembersWithTokens(request.EntityID, response.Tokens); err != nil {
		log.Errorf("could not create members with generated tokens for %q: (%v)", request.SignaturePublicKey, err)
		t.Router.SendError(request, "could not generate tokens")
		return
	}

	log.Debugf("Entity: %x generateTokens: %d tokens", request.SignaturePublicKey, len(response.Tokens))
	t.send(&request, &response)
}

func (t *TokenAPI) importKeysBulk(request router.RouterRequest) {
	var response types.MetaResponse

	if len(request.EntityID) == 0 || len(request.Keys) == 0 {
		t.Router.SendError(request, "invalid arguments")
		return
	}

	secret, err := t.getSecret(request.EntityID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Errorf("importKeysBulk: invalid entity id %s", request.EntityID)
			t.Router.SendError(request, "invalid entityID")
			return

		}
		log.Errorf("importKeysBulk: database error error retrieving entity (%q) : (%v)", request.EntityID, err)
		t.Router.SendError(request, "error retrieving entity")
		return
	}
	if !checkAuth(
		request.Timestamp, request.AuthHash,
		request.Keys, request.EntityID, request.Method, fmt.Sprintf("%d", request.Timestamp), secret) {
		log.Errorf("importKeysBulk invalid authentication: checkAuth error for entity (%q)", request.EntityID)
		t.Router.SendError(request, "invalid authentication")
		return
	}

	members := make([]types.Member, len(request.Keys))
	for i, claim := range request.Keys {
		if members[i].PubKey, err = hex.DecodeString(dvoteutil.TrimHex(claim)); err != nil {
			log.Errorf("importKeysBulk: error decoding claim (%s): (%v)", claim, err)
			t.Router.SendError(request, fmt.Sprintf("error decoding claim (%s)", claim))
			return
		}
	}

	if err = t.db.AddMemberBulk(request.EntityID, members); err != nil {
		log.Errorf("importKeysBulk: could not import provided keys for %s: (%v)", request.EntityID, err)
		t.Router.SendError(request, "could not import keys")
		return
	}

	log.Debugf("Entity: %x importKeysBulk: %d tokens", request.SignaturePublicKey, len(request.Keys))
	t.send(&request, &response)
}

func (t *TokenAPI) listKeys(request router.RouterRequest) {
	var response types.MetaResponse

	if len(request.EntityID) == 0 {
		t.Router.SendError(request, "invalid arguments")
		return
	}

	secret, err := t.getSecret(request.EntityID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Errorf("invalid entity id: trying to validate token  %q for non-existing entity %s", request.Token, request.EntityID)
			t.Router.SendError(request, "invalid entityID")
			return

		}
		log.Errorf("database error: error retrieving entity (%q) to validate token (%q): (%v)", request.EntityID, request.Token, err)
		t.Router.SendError(request, "error retrieving entity")
		return
	}
	if !checkAuth(
		request.Timestamp, request.AuthHash,
		request.EntityID, request.ListOptions, request.Method, fmt.Sprintf("%d", request.Timestamp), secret) {
		log.Errorf("importKeysBulk invalid authentication: checkAuth error for entity (%v)", request.EntityID)
		t.Router.SendError(request, "invalid authentication")
		return
	}

	// check filter
	if err = checkOptions(request.ListOptions, request.Method); err != nil {
		log.Errorf("invalid filter options %q: (%v)", request.SignaturePublicKey, err)
		t.Router.SendError(request, "invalid filter options")
		return
	}

	// Query for members
	members, err := t.db.ListMembers(request.EntityID, request.ListOptions)
	if err != nil {
		if err == sql.ErrNoRows {
			t.Router.SendError(request, "no members found")
			return
		}
		log.Errorf("cannot retrieve members of %q: (%v)", request.SignaturePublicKey, err)
		t.Router.SendError(request, "cannot retrieve members")
		return
	}
	var keys []string
	for _, member := range members {
		if len(member.PubKey) > 0 {
			keys = append(keys, fmt.Sprintf("%x", member.PubKey))
		}

	}

	response.Keys = keys
	log.Debugf("Entity: %x listKeys, dump %d keys from %d members", request.EntityID, len(keys), len(members))
	t.send(&request, &response)
}

func (t *TokenAPI) deleteKeys(request router.RouterRequest) {
	var response types.MetaResponse

	if len(request.EntityID) == 0 {
		t.Router.SendError(request, "invalid arguments")
		return
	}

	secret, err := t.getSecret(request.EntityID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Errorf("deleteKeys: invalid entity id %s", request.EntityID)
			t.Router.SendError(request, "invalid entityID")
			return

		}
		log.Errorf("deleteKeys: database error error retrieving entity (%q) : (%v)", request.EntityID, err)
		t.Router.SendError(request, "error retrieving entity")
		return
	}
	if !checkAuth(
		request.Timestamp, request.AuthHash,
		request.EntityID, request.Keys, request.Method, fmt.Sprintf("%d", request.Timestamp), secret) {
		log.Errorf("deleteKeys: invalid authentication: checkAuth error for entity (%v)", request.EntityID)
		t.Router.SendError(request, "invalid authentication")
		return
	}

	//convert keys to []byte
	keys := make([][]byte, len(request.Keys))
	for i, key := range request.Keys {
		keyBytes, err := hex.DecodeString(key)
		if err != nil {
			log.Errorf("deleteKeys: error decoding key %s for entity %s", key, request.EntityID)
			t.Router.SendError(request, "error decoding keys")
		}
		keys[i] = keyBytes
	}
	// Call db query
	deleted, invalidKeysBytes, err := t.db.DeleteMembersByKeys(request.EntityID, keys)
	if err != nil {
		if err == sql.ErrNoRows {
			t.Router.SendError(request, "no members found")
			return
		}
		log.Errorf("deleteKeys: cannot delete members of %q: (%v)", request.SignaturePublicKey, err)
		t.Router.SendError(request, "cannot delete members")
		return
	}
	response.Count = deleted

	// convert invalid Keys to strings
	for _, key := range invalidKeysBytes {
		response.InvalidKeys = append(response.InvalidKeys, fmt.Sprintf("%x", key))
	}
	log.Debugf("Entity: %x deleteKeys: %d deleted, %d invalid and %d duplicate keys", request.EntityID, deleted, len(invalidKeysBytes), len(request.Keys)-deleted-len(invalidKeysBytes))
	t.send(&request, &response)
}

func checkOptions(filter *types.ListOptions, method string) error {
	if filter == nil {
		return nil
	}
	// Check skip and count
	if filter.Skip < 0 || filter.Count < 0 {
		return fmt.Errorf("invalid skip/count")
	}
	// Check sortby and order
	if filter.Order != "" || filter.SortBy != "" {
		return fmt.Errorf("invalid filter parameters")
	}
	return nil
}
