package tokenapi

import (
	"bytes"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/dvote/httprouter"
	"go.vocdoni.io/dvote/httprouter/bearerstdapi"
	"go.vocdoni.io/dvote/log"
	dvoteutil "go.vocdoni.io/dvote/util"
	"go.vocdoni.io/manager/database"
	"go.vocdoni.io/manager/types"
	"go.vocdoni.io/manager/util"
)

// AuthWindowSeconds is the time window (in seconds) that the tokenapi Auth tolerates
const AuthWindowSeconds = 300

// TokenAPI is a handler for external token managmement
type TokenAPI struct {
	db database.Database
}

// NewTokenAPI creates a new token API handler for the Router
func NewTokenAPI(d database.Database) *TokenAPI {
	return &TokenAPI{db: d}
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

func (t *TokenAPI) Revoke(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var entityID []byte
	var token string
	var timestamp int32
	var authHash string
	var method string
	var err error
	var response types.MetaResponse
	if entityID, _, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	token = ctx.URLParam("token")
	// either token or valid member info should be valid
	if len(token) == 0 {
		return fmt.Errorf("empty token validation for entity %s", entityID)
	}

	uid, err := uuid.Parse(token)
	if err != nil {
		return fmt.Errorf("invalid token id format %s for entity %s:(%v)", token, entityID, err)
	}

	secret, err := t.getSecret(entityID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("revoke: invalid entity id %s", entityID)

		}
		return fmt.Errorf("revoke: database error error retrieving entity (%q) : (%v)", entityID, err)
	}

	if err = util.DecodeJsonMessage(&timestamp, "timestamp", ctx); err != nil {
		return err
	}
	authHash = ctx.URLParam("authHash")
	method = ctx.URLParam("method")

	if !checkAuth(
		timestamp, authHash,
		entityID, method, fmt.Sprintf("%d", timestamp), token, secret) {
		return fmt.Errorf("invalid authentication: checkAuth error for entity (%q) to validate token (%q):(%v)", entityID, token, err)
	}
	if err = t.db.DeleteMember(entityID, &uid); err != nil {
		return fmt.Errorf("database error: could not delete token %q for entity %q: (%v)", token, entityID, err)
	}

	log.Infof("deleted member with token (%q) for entity (%s)", token, entityID)
	return util.SendResponse(response, ctx)
}

func (t *TokenAPI) Status(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var entityID []byte
	var token string
	var timestamp int32
	var authHash string
	var method string
	var err error
	var response types.MetaResponse

	if entityID, _, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}

	token = ctx.URLParam("token")
	// either token or valid member info should be valid
	if len(token) == 0 {
		return fmt.Errorf("empty token validation for entity %s", entityID)
	}
	uid, err := uuid.Parse(token)
	if err != nil {
		return fmt.Errorf("invalid token id format %s for entity %s:(%v)", token, entityID, err)
	}

	secret, err := t.getSecret(entityID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("revoke: invalid entity id %s", entityID)

		}
		return fmt.Errorf("revoke: database error error retrieving entity (%q) : (%v)", entityID, err)
	}

	if err = util.DecodeJsonMessage(&timestamp, "timestamp", ctx); err != nil {
		return err
	}
	authHash = ctx.URLParam("authHash")
	method = ctx.URLParam("method")
	if !checkAuth(
		timestamp, authHash,
		entityID, method, fmt.Sprintf("%d", timestamp), token, secret) {
		return fmt.Errorf("invalid authentication: checkAuth error for entity (%q) to validate token (%q):(%v)", entityID, token, err)
	}
	member, err := t.db.Member(entityID, &uid)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("invalid token: trying to get status for token  (%q) for non-existing combination with entity %s", token, entityID)

		}
		log.Errorf("database error: trying to get status for token (%q) for entity (%q): (%v)", token, entityID, err)
		response.TokenStatus = "invalid"
		return util.SendResponse(response, ctx)
	}

	if len(member.PubKey) != ethereum.PubKeyLengthBytes {
		log.Debugf("status for member with token (%q) for entity (%s): ", token, entityID)
		response.TokenStatus = "available"
		return util.SendResponse(response, ctx)
	}

	response.TokenStatus = "registered"
	return util.SendResponse(response, ctx)
}

func (t *TokenAPI) Generate(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var entityID []byte
	var signaturePubKey []byte
	var token string
	var timestamp int32
	var authHash string
	var method string
	var amount int
	var err error
	var response types.MetaResponse

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	token = ctx.URLParam("token")
	// either token or valid member info should be valid
	if len(token) == 0 {
		return fmt.Errorf("empty token validation for entity %s", entityID)
	}

	secret, err := t.getSecret(entityID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("revoke: invalid entity id %s", entityID)

		}
		return fmt.Errorf("revoke: database error error retrieving entity (%q) : (%v)", entityID, err)
	}
	if err = util.DecodeJsonMessage(&timestamp, "timestamp", ctx); err != nil {
		return err
	}
	authHash = ctx.URLParam("authHash")
	method = ctx.URLParam("method")

	if !checkAuth(
		timestamp, authHash,
		entityID, method, fmt.Sprintf("%d", timestamp), token, secret) {
		return fmt.Errorf("invalid authentication: checkAuth error for entity (%q) to validate token (%q):(%v)", entityID, token, err)
	}

	if err = util.DecodeJsonMessage(&amount, "amount", ctx); err != nil {
		return err
	}
	if amount < 1 {
		return fmt.Errorf("invalid token amount requested by %s", entityID)
	}

	for i := 0; i < amount; i++ {
		response.Tokens = append(response.Tokens, uuid.New())
	}
	// TODO: Probably I need to initialize tokens
	if err = t.db.CreateMembersWithTokens(entityID, response.Tokens); err != nil {
		return fmt.Errorf("could not create members with generated tokens for %q: (%v)", signaturePubKey, err)
	}

	log.Debugf("Entity: %x generateTokens: %d tokens", signaturePubKey, len(response.Tokens))
	return util.SendResponse(response, ctx)
}

func (t *TokenAPI) ImportKeysBulk(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var entityID []byte
	var signaturePubKey []byte
	var timestamp int32
	var token string
	var authHash string
	var method string
	var keys []string
	var err error
	var response types.MetaResponse

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	token = ctx.URLParam("token")
	// either token or valid member info should be valid
	if len(token) == 0 {
		return fmt.Errorf("empty token validation for entity %s", entityID)
	}
	secret, err := t.getSecret(entityID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("revoke: invalid entity id %s", entityID)

		}
		return fmt.Errorf("revoke: database error error retrieving entity (%q) : (%v)", entityID, err)
	}

	if err = util.DecodeJsonMessage(&timestamp, "timestamp", ctx); err != nil {
		return err
	}
	authHash = ctx.URLParam("authHash")
	method = ctx.URLParam("method")

	if !checkAuth(
		timestamp, authHash,
		entityID, method, fmt.Sprintf("%d", timestamp), token, secret) {
		return fmt.Errorf("invalid authentication: checkAuth error for entity (%q) to validate token (%q):(%v)", entityID, token, err)
	}

	if err = util.DecodeJsonMessage(&keys, "keys", ctx); err != nil {
		return err
	}
	members := make([]types.Member, len(keys))
	for i, claim := range keys {
		if members[i].PubKey, err = hex.DecodeString(dvoteutil.TrimHex(claim)); err != nil {
			return fmt.Errorf("importKeysBulk: error decoding claim (%s): (%v)", claim, err)
		}
	}

	if err = t.db.AddMemberBulk(entityID, members); err != nil {
		log.Errorf("importKeysBulk: could not import provided keys for %s: (%v)", entityID, err)
	}

	log.Debugf("Entity: %x importKeysBulk: %d tokens", signaturePubKey, len(keys))
	return util.SendResponse(response, ctx)
}

func (t *TokenAPI) ListKeys(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var entityID []byte
	var signaturePubKey []byte
	var timestamp int32
	var token string
	var authHash string
	var method string
	var listOptions *types.ListOptions
	var keys []string
	var err error
	var response types.MetaResponse

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	token = ctx.URLParam("token")
	// either token or valid member info should be valid
	if len(token) == 0 {
		return fmt.Errorf("empty token validation for entity %s", entityID)
	}
	secret, err := t.getSecret(entityID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("revoke: invalid entity id %s", entityID)

		}
		return fmt.Errorf("revoke: database error error retrieving entity (%q) : (%v)", entityID, err)
	}

	if err = util.DecodeJsonMessage(&timestamp, "timestamp", ctx); err != nil {
		return err
	}
	if err = util.DecodeJsonMessage(listOptions, "listOptions", ctx); err != nil {
		return err
	}
	authHash = ctx.URLParam("authHash")
	method = ctx.URLParam("method")

	if !checkAuth(
		timestamp, authHash,
		entityID, method, fmt.Sprintf("%d", timestamp), token, secret) {
		return fmt.Errorf("invalid authentication: checkAuth error for entity (%q) to validate token (%q):(%v)", entityID, token, err)
	}

	// check filter
	if err = checkOptions(listOptions, method); err != nil {
		return fmt.Errorf("invalid filter options %q: (%v)", signaturePubKey, err)
	}

	// Query for members
	members, err := t.db.ListMembers(entityID, listOptions)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("no members found")
		}
		return fmt.Errorf("cannot retrieve members of %q: (%v)", signaturePubKey, err)
	}
	for _, member := range members {
		if len(member.PubKey) > 0 {
			keys = append(keys, fmt.Sprintf("%x", member.PubKey))
		}

	}

	response.Keys = keys
	log.Debugf("Entity: %x listKeys, dump %d keys from %d members", entityID, len(keys), len(members))
	return util.SendResponse(response, ctx)
}

func (t *TokenAPI) DeleteKeys(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var entityID []byte
	var signaturePubKey []byte
	var timestamp int32
	var token string
	var authHash string
	var method string
	var listOptions *types.ListOptions
	var keys []string
	var err error
	var response types.MetaResponse

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	token = ctx.URLParam("token")
	// either token or valid member info should be valid
	if len(token) == 0 {
		return fmt.Errorf("empty token validation for entity %s", entityID)
	}
	secret, err := t.getSecret(entityID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("revoke: invalid entity id %s", entityID)

		}
		return fmt.Errorf("revoke: database error error retrieving entity (%q) : (%v)", entityID, err)
	}

	if err = util.DecodeJsonMessage(&timestamp, "timestamp", ctx); err != nil {
		return err
	}
	if err = util.DecodeJsonMessage(listOptions, "listOptions", ctx); err != nil {
		return err
	}
	authHash = ctx.URLParam("authHash")
	method = ctx.URLParam("method")

	if !checkAuth(
		timestamp, authHash,
		entityID, method, fmt.Sprintf("%d", timestamp), token, secret) {
		return fmt.Errorf("invalid authentication: checkAuth error for entity (%q) to validate token (%q):(%v)", entityID, token, err)
	}

	//convert keys to []byte
	if err = util.DecodeJsonMessage(&keys, "keys", ctx); err != nil {
		return err
	}
	keysArray := make([][]byte, len(keys))
	for i, key := range keys {
		keyBytes, err := hex.DecodeString(key)
		if err != nil {
			return fmt.Errorf("deleteKeys: error decoding key %s for entity %s", key, entityID)
		}
		keysArray[i] = keyBytes
	}
	// Call db query
	deleted, invalidKeysBytes, err := t.db.DeleteMembersByKeys(entityID, keysArray)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("no members found")
		}
		return fmt.Errorf("deleteKeys: cannot delete members of %q: (%v)", signaturePubKey, err)
	}
	response.Count = deleted

	// convert invalid Keys to strings
	for _, key := range invalidKeysBytes {
		response.InvalidKeys = append(response.InvalidKeys, fmt.Sprintf("%x", key))
	}
	log.Debugf("Entity: %x deleteKeys: %d deleted, %d invalid and %d duplicate keys", entityID, deleted, len(invalidKeysBytes), len(keys)-deleted-len(invalidKeysBytes))
	return util.SendResponse(response, ctx)
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
