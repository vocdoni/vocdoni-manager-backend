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
	dvoteutil "go.vocdoni.io/dvote/util"
	"go.vocdoni.io/manager/types"
)

// AuthWindowSeconds is the time window (in seconds) that the tokenapi Auth tolerates
const AuthWindowSeconds = 300

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

func (t *TokenAPI) revoke(request *types.APIrequest) (*types.APIresponse, error) {
	if len(request.EntityID) == 0 {
		log.Errorf("trying to revoke token %q for null entity %s", request.Token, request.EntityID)
		return nil, fmt.Errorf("invalid entityID")

	}

	// either token or valid member info should be valid
	if len(request.Token) == 0 {
		log.Errorf("empty token validation for entity %s", request.EntityID)
		return nil, fmt.Errorf("invalid token")
	}

	uid, err := uuid.Parse(request.Token)
	if err != nil {
		log.Errorf("invalid token id format %s for entity %s:(%v)", request.Token, request.EntityID, err)
		return nil, fmt.Errorf("invalid token format")
	}

	secret, err := t.getSecret(request.EntityID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Errorf("revoke: invalid entity id %s", request.EntityID)
			return nil, fmt.Errorf("invalid entityID")

		}
		log.Errorf("revoke: database error error retrieving entity (%q) : (%v)", request.EntityID, err)
		return nil, fmt.Errorf("error retrieving entity")
	}
	if !checkAuth(
		request.Timestamp, request.AuthHash,
		request.EntityID, request.Method, fmt.Sprintf("%d", request.Timestamp), request.Token, secret) {
		log.Errorf("invalid authentication: checkAuth error for entity (%q) to validate token (%q):(%v)", request.EntityID, request.Token, err)
		return nil, fmt.Errorf("invalid authentication")
	}
	if err = t.db.DeleteMember(request.EntityID, &uid); err != nil {
		log.Errorf("database error: could not delete token %q for entity %q: (%v)", request.Token, request.EntityID, err)
		return nil, fmt.Errorf("could not delete member")
	}

	log.Infof("deleted member with token (%q) for entity (%s)", request.Token, request.EntityID)
	return &types.APIresponse{Ok: true}, nil
}

func (t *TokenAPI) status(request *types.APIrequest) (*types.APIresponse, error) {
	var resp types.APIresponse

	if len(request.EntityID) == 0 {
		log.Errorf("invalid entity %s", request.EntityID)
		return nil, fmt.Errorf("invalid entityID")
	}

	// either token or valid member info should be valid
	if len(request.Token) == 0 {
		log.Errorf("empty token validation for entity %s", request.EntityID)
		return nil, fmt.Errorf("invalid token")
	}
	uid, err := uuid.Parse(request.Token)
	if err != nil {
		log.Errorf("invalid token id format %s for entity %s:(%v)", request.Token, request.EntityID, err)
		return nil, fmt.Errorf("invalid token format")
	}

	secret, err := t.getSecret(request.EntityID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Errorf("status: invalid entity id %s", request.EntityID)
			return nil, fmt.Errorf("invalid entityID")

		}
		log.Errorf("status: database error error retrieving entity (%q) : (%v)", request.EntityID, err)
		return nil, fmt.Errorf("error retrieving entity")
	}
	if !checkAuth(
		request.Timestamp, request.AuthHash,
		request.EntityID, request.Method, fmt.Sprintf("%d", request.Timestamp), request.Token, secret) {
		log.Errorf("invalid authentication: checkAuth error for entity (%q) to validate token (%q): (%v)", request.EntityID, request.Token, err)
		return nil, fmt.Errorf("invalid authentication")
	}
	member, err := t.db.Member(request.EntityID, &uid)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Errorf("invalid token: trying to get status for token  (%q) for non-existing combination with entity %s", request.Token, request.EntityID)

		}
		log.Errorf("database error: trying to get status for token (%q) for entity (%q): (%v)", request.Token, request.EntityID, err)
		resp.TokenStatus = "invalid"
		return &resp, nil
	}

	if len(member.PubKey) != ethereum.PubKeyLengthBytes {
		log.Debugf("status for member with token (%q) for entity (%s): ", request.Token, request.EntityID)
		resp.TokenStatus = "available"
		return &resp, nil
	}

	resp.TokenStatus = "registered"
	return &resp, nil
}

func (t *TokenAPI) generate(request *types.APIrequest) (*types.APIresponse, error) {
	var response types.APIresponse

	if len(request.EntityID) == 0 {
		log.Errorf("trying to generate tokens for null entity %s", request.EntityID)
		return nil, fmt.Errorf("invalid entityId")

	}

	secret, err := t.getSecret(request.EntityID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Errorf("generate: invalid entity id %s", request.EntityID)
			return nil, fmt.Errorf("invalid entityID")

		}
		log.Errorf("generate: database error error retrieving entity (%q) : (%v)", request.EntityID, err)
		return nil, fmt.Errorf("error retrieving entity")
	}
	if !checkAuth(
		request.Timestamp, request.AuthHash,
		fmt.Sprintf("%d", request.Amount), request.EntityID, request.Method, fmt.Sprintf("%d", request.Timestamp), secret) {
		log.Errorf("invalid authentication: checkAuth error for entity (%q) to generate tokens: (%v)", request.EntityID, err)
		return nil, fmt.Errorf("invalid authentication")
	}

	if request.Amount < 1 {
		log.Errorf("invalid token amount requested by %s", request.EntityID)
		return nil, fmt.Errorf("invalid token amount")
	}

	for i := 0; i < request.Amount; i++ {
		response.Tokens = append(response.Tokens, uuid.New())
	}
	// TODO: Probably I need to initialize tokens
	if err = t.db.CreateMembersWithTokens(request.EntityID, response.Tokens); err != nil {
		log.Errorf("could not create members with generated tokens for %q: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("could not generate tokens")
	}

	log.Debugf("Entity: %x generateTokens: %d tokens", request.SignaturePublicKey, len(response.Tokens))
	return &response, nil
}

func (t *TokenAPI) importKeysBulk(request *types.APIrequest) (*types.APIresponse, error) {
	var response types.APIresponse

	if len(request.EntityID) == 0 || len(request.Keys) == 0 {
		return nil, fmt.Errorf("invalid arguments")
	}

	secret, err := t.getSecret(request.EntityID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Errorf("importKeysBulk: invalid entity id %s", request.EntityID)
			return nil, fmt.Errorf("invalid entityID")

		}
		log.Errorf("importKeysBulk: database error error retrieving entity (%q) : (%v)", request.EntityID, err)
		return nil, fmt.Errorf("error retrieving entity")
	}
	if !checkAuth(
		request.Timestamp, request.AuthHash,
		request.Keys, request.EntityID, request.Method, fmt.Sprintf("%d", request.Timestamp), secret) {
		log.Errorf("importKeysBulk invalid authentication: checkAuth error for entity (%q)", request.EntityID)
		return nil, fmt.Errorf("invalid authentication")
	}

	members := make([]types.Member, len(request.Keys))
	for i, claim := range request.Keys {
		if members[i].PubKey, err = hex.DecodeString(dvoteutil.TrimHex(claim)); err != nil {
			log.Errorf("importKeysBulk: error decoding claim (%s): (%v)", claim, err)
			return nil, fmt.Errorf("error decoding claim (%s)", claim)
		}
	}

	if err = t.db.AddMemberBulk(request.EntityID, members); err != nil {
		log.Errorf("importKeysBulk: could not import provided keys for %s: (%v)", request.EntityID, err)
		return nil, fmt.Errorf("could not import keys")
	}

	log.Debugf("Entity: %x importKeysBulk: %d tokens", request.SignaturePublicKey, len(request.Keys))
	return &response, nil
}

func (t *TokenAPI) listKeys(request *types.APIrequest) (*types.APIresponse, error) {
	var response types.APIresponse

	if len(request.EntityID) == 0 {
		return nil, fmt.Errorf("invalid arguments")
	}

	secret, err := t.getSecret(request.EntityID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Errorf("invalid entity id: trying to validate token  %q for non-existing entity %s", request.Token, request.EntityID)
			return nil, fmt.Errorf("invalid entityID")

		}
		log.Errorf("database error: error retrieving entity (%q) to validate token (%q): (%v)", request.EntityID, request.Token, err)
		return nil, fmt.Errorf("error retrieving entity")
	}
	if !checkAuth(
		request.Timestamp, request.AuthHash,
		request.EntityID, request.ListOptions, request.Method, fmt.Sprintf("%d", request.Timestamp), secret) {
		log.Errorf("importKeysBulk invalid authentication: checkAuth error for entity (%v)", request.EntityID)
		return nil, fmt.Errorf("invalid authentication")
	}

	// check filter
	if err = checkOptions(request.ListOptions, request.Method); err != nil {
		log.Errorf("invalid filter options %q: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("invalid filter options")
	}

	// Query for members
	members, err := t.db.ListMembers(request.EntityID, request.ListOptions)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no members found")
		}
		log.Errorf("cannot retrieve members of %q: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("cannot retrieve members")
	}
	var keys []string
	for _, member := range members {
		if len(member.PubKey) > 0 {
			keys = append(keys, fmt.Sprintf("%x", member.PubKey))
		}

	}

	response.Keys = keys
	log.Debugf("Entity: %x listKeys, dump %d keys from %d members", request.EntityID, len(keys), len(members))
	return &response, nil
}

func (t *TokenAPI) deleteKeys(request *types.APIrequest) (*types.APIresponse, error) {
	var response types.APIresponse

	if len(request.EntityID) == 0 {
		return nil, fmt.Errorf("invalid arguments")
	}

	secret, err := t.getSecret(request.EntityID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Errorf("deleteKeys: invalid entity id %s", request.EntityID)
			return nil, fmt.Errorf("invalid entityID")

		}
		log.Errorf("deleteKeys: database error error retrieving entity (%q) : (%v)", request.EntityID, err)
		return nil, fmt.Errorf("error retrieving entity")
	}
	if !checkAuth(
		request.Timestamp, request.AuthHash,
		request.EntityID, request.Keys, request.Method, fmt.Sprintf("%d", request.Timestamp), secret) {
		log.Errorf("deleteKeys: invalid authentication: checkAuth error for entity (%v)", request.EntityID)
		return nil, fmt.Errorf("invalid authentication")
	}

	//convert keys to []byte
	keys := make([][]byte, len(request.Keys))
	for i, key := range request.Keys {
		keyBytes, err := hex.DecodeString(key)
		if err != nil {
			log.Errorf("deleteKeys: error decoding key %s for entity %s", key, request.EntityID)
			return nil, fmt.Errorf("error decoding keys")
		}
		keys[i] = keyBytes
	}
	// Call db query
	deleted, invalidKeysBytes, err := t.db.DeleteMembersByKeys(request.EntityID, keys)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no members found")
		}
		log.Errorf("deleteKeys: cannot delete members of %q: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("cannot delete members")
	}
	response.Count = deleted

	// convert invalid Keys to strings
	for _, key := range invalidKeysBytes {
		response.InvalidKeys = append(response.InvalidKeys, fmt.Sprintf("%x", key))
	}
	log.Debugf("Entity: %x deleteKeys: %d deleted, %d invalid and %d duplicate keys", request.EntityID, deleted, len(invalidKeysBytes), len(request.Keys)-deleted-len(invalidKeysBytes))
	return &response, nil
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
