package testcommon

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
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
	tb      testing.TB
	WS      *websocket.Conn
	HTTP    *http.Client
	Address string
}

// NewAPIConnection starts a connection with the given endpoint address. The
// connection is closed automatically when the test or benchmark finishes.
func NewAPIConnection(addr string, tb testing.TB) (*APIConnection, error) {
	r := &APIConnection{tb: tb, Address: addr}
	var err error
	r.WS, _, err = websocket.Dial(context.TODO(), addr, nil)
	if err != nil {
		return nil, err
	}
	r.tb.Cleanup(func() { r.WS.Close(websocket.StatusNormalClosure, "") })
	return r, nil
}

// NewHTTPapiConnection starts a connection with the given endpoint address. The
// connection is closed automatically when the test or benchmark finishes.
func NewHTTPapiConnection(addr string, tb testing.TB) (*APIConnection, error) {
	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    10 * time.Second,
		DisableCompression: true,
	}
	r := &APIConnection{tb: tb, Address: addr, HTTP: &http.Client{Transport: tr, Timeout: time.Second * 2}}

	r.tb.Cleanup(func() { r.HTTP.CloseIdleConnections() })
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
	var message []byte
	if r.WS != nil {
		if err := r.WS.Write(context.TODO(), websocket.MessageText, reqBody); err != nil {
			r.tb.Fatalf("%s: %v", method, err)
		}
		_, message, err = r.WS.Read(context.TODO())
		if err != nil {
			r.tb.Fatalf("%s: %v", method, err)
		}
	}
	if r.HTTP != nil {
		resp, err := r.HTTP.Post(r.Address, "application/json", bytes.NewBuffer(reqBody))
		if err != nil {
			r.tb.Fatalf("%s: %v", method, err)
		}
		message, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			r.tb.Fatalf("%s: %v", method, err)
		}
		resp.Body.Close()
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
