package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"strings"
	"time"

	"gitlab.com/vocdoni/go-dvote/crypto"
	"gitlab.com/vocdoni/go-dvote/crypto/ethereum"
	"gitlab.com/vocdoni/go-dvote/log"
	"gitlab.com/vocdoni/manager/manager-backend/types"
	"nhooyr.io/websocket"
)

// APIConnection holds an API websocket connection
type APIConnection struct {
	Conn *websocket.Conn
}

// NewAPIConnection starts a connection with the given endpoint address. The
// connection is closed automatically when the test or benchmark finishes.
func NewAPIConnection(addr string) *APIConnection {
	r := &APIConnection{}
	var err error
	r.Conn, _, err = websocket.Dial(context.TODO(), addr, nil)
	if err != nil {
		log.Fatal(err)
	}
	return r
}

// Request makes a request to the previously connected endpoint
func (r *APIConnection) Request(req types.MetaRequest, signer *ethereum.SignKeys) *types.MetaResponse {
	method := req.Method

	req.Timestamp = int32(time.Now().Unix())
	reqInner, err := crypto.SortedMarshalJSON(req)
	if err != nil {
		log.Fatalf("%s: %v", method, err)
	}
	var signature string
	if signer != nil {
		signature, err = signer.Sign(reqInner)
		if err != nil {
			log.Fatalf("%s: %v", method, err)
		}
	}

	reqOuter := types.RequestMessage{
		ID:          fmt.Sprintf("%d", rand.Intn(1000)),
		Signature:   signature,
		MetaRequest: reqInner,
	}
	reqBody, err := json.Marshal(reqOuter)
	if err != nil {
		log.Fatalf("%s: %v", method, err)
	}
	log.Infof("sending: %s", reqBody)
	if err := r.Conn.Write(context.TODO(), websocket.MessageText, reqBody); err != nil {
		log.Fatalf("%s: %v", method, err)
	}
	_, message, err := r.Conn.Read(context.TODO())
	log.Infof("received: %s", message)
	if err != nil {
		log.Fatalf("%s: %v", method, err)
	}
	var respOuter types.ResponseMessage
	if err := json.Unmarshal(message, &respOuter); err != nil {
		log.Fatalf("%s: %v", method, err)
	}
	if respOuter.ID != reqOuter.ID {
		log.Fatalf("%s: %v", method, "request ID doesn'tb match")
	}
	if respOuter.Signature == "" {
		log.Fatalf("%s: empty signature in response: %s", method, message)
	}
	var respInner types.MetaResponse
	if err := json.Unmarshal(respOuter.MetaResponse, &respInner); err != nil {
		log.Fatalf("%s: %v", method, err)
	}
	return &respInner
}

func main() {
	host := flag.String("host", "ws://127.0.0.1:8000/api/registry", "host to connect to")
	logLevel := flag.String("logLevel", "info", "log level <debug, info, warn, error>")
	privKey := flag.String("key", "", "private key for signature (leave blank for auto-generate)")
	eid := flag.String("eid", "", "entityID")
	method := flag.String("method", "tokenRegister", "available methods: tokenRegister")
	tokenList := flag.String("tokenList", "", "path to the file containing the tokens")
	flag.Parse()
	log.Init(*logLevel, "stdout")
	rand.Seed(time.Now().UnixNano())

	signer := new(ethereum.SignKeys)
	if *privKey != "" {
		if err := signer.AddHexKey(*privKey); err != nil {
			panic(err)
		}
	} else {
		signer.Generate()
	}
	log.Infof("connecting to %s", *host)
	c := NewAPIConnection(*host)
	defer c.Conn.Close(websocket.StatusNormalClosure, "")

	switch *method {
	case "tokenRegister":
		tokens, err := splitFile(*tokenList)
		if err != nil {
			log.Fatal(err)
		}
		if len(*eid) < 20 {
			log.Fatal("invalid entityID")
		}
		if err := tokenRegister(*eid, tokens, c); err != nil {
			log.Fatal(err)
		}
	}
}

func splitFile(filepath string) ([]string, error) {
	f, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	return strings.Split(string(f), "\n"), nil
}

func tokenRegister(eid string, tokenList []string, c *APIConnection) error {
	var req types.MetaRequest
	req.Method = "validateToken"
	req.EntityID = eid
	s := ethereum.SignKeys{}
	for _, t := range tokenList {
		s.Generate()
		req.Token = t
		resp := c.Request(req, &s)
		log.Infof("%+v", *resp)
	}
	return nil
}
