package testcommon

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"gitlab.com/vocdoni/go-dvote/crypto/signature"
	"gitlab.com/vocdoni/go-dvote/log"
	"gitlab.com/vocdoni/vocdoni-manager-backend/types"
	"nhooyr.io/websocket"
)

// APIConnection holds an API websocket connection
type APIConnection struct {
	tb   testing.TB
	Conn *websocket.Conn
}

// NewAPIConnection starts a connection with the given endpoint address. The
// connection is closed automatically when the test or benchmark finishes.
func NewAPIConnection(addr string, tb testing.TB) (*APIConnection, error) {
	r := &APIConnection{tb: tb}
	var err error
	r.Conn, _, err = websocket.Dial(context.TODO(), addr, nil)
	if err != nil {
		return nil, err
	}
	r.tb.Cleanup(func() { r.Conn.Close(websocket.StatusNormalClosure, "") })
	return r, nil
}

// Request makes a request to the previously connected endpoint
func (r *APIConnection) Request(req types.MetaRequest, signer *signature.SignKeys) *types.MetaResponse {
	r.tb.Helper()
	method := req.Method

	var cmReq types.RequestMessage
	cmReq.MetaRequest = req
	cmReq.ID = fmt.Sprintf("%d", rand.Intn(1000))
	cmReq.Timestamp = int32(time.Now().Unix())
	if signer != nil {
		var err error
		cmReq.Signature, err = signer.SignJSON(cmReq.MetaRequest)
		if err != nil {
			r.tb.Fatalf("%s: %v", method, err)
		}
	}
	rawReq, err := json.Marshal(cmReq)
	if err != nil {
		r.tb.Fatalf("%s: %v", method, err)
	}
	log.Infof("request: %s", rawReq)
	if err := r.Conn.Write(context.TODO(), websocket.MessageText, rawReq); err != nil {
		r.tb.Fatalf("%s: %v", method, err)
	}
	_, message, err := r.Conn.Read(context.TODO())
	if err != nil {
		r.tb.Fatalf("%s: %v", method, err)
	}
	log.Infof("response: %s", message)
	var cmRes types.ResponseMessage
	if err := json.Unmarshal(message, &cmRes); err != nil {
		r.tb.Fatalf("%s: %v", method, err)
	}
	if cmRes.ID != cmReq.ID {
		r.tb.Fatalf("%s: %v", method, "request ID doesn'tb match")
	}
	if cmRes.Signature == "" {
		r.tb.Fatalf("%s: empty signature in response: %s", method, message)
	}
	return &cmRes.MetaResponse
}
