package testcommon

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"gitlab.com/vocdoni/go-dvote/crypto"
	"gitlab.com/vocdoni/go-dvote/crypto/ethereum"
	"gitlab.com/vocdoni/go-dvote/log"
	"gitlab.com/vocdoni/manager/manager-backend/types"
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
func (r *APIConnection) Request(req types.MetaRequest, signer *ethereum.SignKeys) *types.MetaResponse {
	r.tb.Helper()
	method := req.Method

	req.Timestamp = int32(time.Now().Unix())
	reqInner, err := crypto.SortedMarshalJSON(req)
	if err != nil {
		r.tb.Fatalf("%s: %v", method, err)
	}
	var signature string
	if signer != nil {
		signature, err = signer.Sign(reqInner)
		if err != nil {
			r.tb.Fatalf("%s: %v", method, err)
		}
	}

	reqOuter := types.RequestMessage{
		ID:          fmt.Sprintf("%d", rand.Intn(1000)),
		Signature:   signature,
		MetaRequest: reqInner,
	}
	reqBody, err := json.Marshal(reqOuter)
	if err != nil {
		r.tb.Fatalf("%s: %v", method, err)
	}

	log.Infof("request: %s", reqBody)
	if err := r.Conn.Write(context.TODO(), websocket.MessageText, reqBody); err != nil {
		r.tb.Fatalf("%s: %v", method, err)
	}
	_, message, err := r.Conn.Read(context.TODO())
	if err != nil {
		r.tb.Fatalf("%s: %v", method, err)
	}
	log.Infof("response: %s", message)
	var respOuter types.ResponseMessage
	if err := json.Unmarshal(message, &respOuter); err != nil {
		r.tb.Fatalf("%s: %v", method, err)
	}
	if respOuter.ID != reqOuter.ID {
		r.tb.Fatalf("%s: %v", method, "request ID doesn'tb match")
	}
	if respOuter.Signature == "" {
		r.tb.Fatalf("%s: empty signature in response: %s", method, message)
	}
	var respInner types.MetaResponse
	if err := json.Unmarshal(respOuter.MetaResponse, &respInner); err != nil {
		r.tb.Fatalf("%s: %v", method, err)
	}
	return &respInner
}
