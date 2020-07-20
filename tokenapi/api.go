package tokenapi

import (
	"encoding/hex"
	"fmt"
	"time"

	"gitlab.com/vocdoni/go-dvote/crypto/ethereum"
	"gitlab.com/vocdoni/go-dvote/log"
	"gitlab.com/vocdoni/manager/manager-backend/database"
	"gitlab.com/vocdoni/manager/manager-backend/router"
	"gitlab.com/vocdoni/manager/manager-backend/services/metrics"
	"gitlab.com/vocdoni/manager/manager-backend/types"
)

const authWindowSeconds = 3

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
	t.Router.Transport.AddNamespace(path + "/token")
	if err := t.Router.AddHandler("revoke", path+"/token", t.revoke, false); err != nil {
		return err
	}
	if err := t.Router.AddHandler("status", path+"/token", t.status, false); err != nil {
		return err
	}
	if err := t.Router.AddHandler("generate", path+"/token", t.generate, false); err != nil {
		return err
	}
	return nil
}

func (t *TokenAPI) send(req router.RouterRequest, resp types.MetaResponse) {
	t.Router.Transport.Send(t.Router.BuildReply(req, resp))
}

func (t *TokenAPI) checkAuth(fields []string, timestamp int32, auth string) bool {
	if len(fields) == 0 {
		return false
	}
	current := int32(time.Now().Unix())
	if timestamp+authWindowSeconds > current || timestamp-authWindowSeconds < current {
		log.Warnf("timestamp out of window")
		return false
	}
	toHash := ""
	for _, f := range fields {
		toHash += f
	}
	thisAuth := hex.EncodeToString(ethereum.HashRaw([]byte(toHash)))
	return thisAuth == auth
}

func (t *TokenAPI) getSecret(entityID string) (string, error) {
	return "get secret from DB", nil
}

func (t *TokenAPI) revoke(request router.RouterRequest) {
	secret, err := t.getSecret(request.EntityID)
	if err != nil {
		log.Warn(err)
		t.Router.SendError(request, "not authorized")
	}
	if !t.checkAuth(
		[]string{request.EntityID, fmt.Sprintf("%d", request.Timestamp), request.Token, secret},
		request.Timestamp,
		request.HashAuth) {
		t.Router.SendError(request, "invalid authentication")
		return
	}
	var resp types.MetaResponse
	t.send(request, resp)
}

func (t *TokenAPI) status(request router.RouterRequest) {
	var resp types.MetaResponse
	t.send(request, resp)
}

func (t *TokenAPI) generate(request router.RouterRequest) {
	var resp types.MetaResponse
	t.send(request, resp)
}
