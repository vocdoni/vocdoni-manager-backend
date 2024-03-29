package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strings"
	"time"

	flag "github.com/spf13/pflag"

	"go.vocdoni.io/dvote/crypto"
	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/dvote/log"
	"go.vocdoni.io/manager/types"
	"go.vocdoni.io/manager/util"
	"nhooyr.io/websocket"
)

// APIConnection holds an API websocket connection
type APIConnection struct {
	WS      *websocket.Conn
	HTTP    *http.Client
	Address string
}

// NewWSapiConnection starts a connection with the given endpoint address. The
// connection is closed automatically when the test or benchmark finishes.
func NewWSapiConnection(addr string) (*APIConnection, error) {
	r := &APIConnection{}
	var err error
	r.WS, _, err = websocket.Dial(context.TODO(), addr, nil)
	if err != nil {
		return nil, err
	}
	//caller must do: defer c.WS.Close(websocket.StatusNormalClosure, "")
	return r, nil
}

// NewHTTPapiConnection starts a connection with the given endpoint address. The
// connection is closed automatically when the test or benchmark finishes.
func NewHTTPapiConnection(addr string) (*APIConnection, error) {
	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    10 * time.Second,
		DisableCompression: true,
	}
	r := &APIConnection{Address: addr, HTTP: &http.Client{Transport: tr, Timeout: time.Second * 2}}

	return r, nil
}

// Request makes a request to the previously connected endpoint
func (r *APIConnection) Request(req types.APIrequest, signer *ethereum.SignKeys) *types.APIresponse {
	method := req.Method

	req.Timestamp = int32(time.Now().Unix())
	reqInner, err := crypto.SortedMarshalJSON(req)
	if err != nil {
		log.Fatalf("%s: %v", method, err)
	}
	var signature types.HexBytes
	if signer != nil {
		signature, err = signer.SignVocdoniMsg(reqInner)
		if err != nil {
			log.Fatalf("%s: %v", method, err)
		}
	}

	reqOuter := types.RequestMessage{
		ID:         fmt.Sprintf("%d", rand.Intn(1000)),
		Signature:  signature,
		MessageAPI: reqInner,
	}
	reqBody, err := json.Marshal(reqOuter)
	if err != nil {
		log.Fatalf("%s: %v", method, err)
	}
	log.Debugf("sending: %s", reqBody)

	var message []byte
	if r.WS != nil {
		if err := r.WS.Write(context.TODO(), websocket.MessageText, reqBody); err != nil {
			log.Fatalf("%s: %v", method, err)
		}
		_, message, err = r.WS.Read(context.TODO())
	}
	if r.HTTP != nil {
		resp, err := r.HTTP.Post(r.Address, "application/json", bytes.NewBuffer(reqBody))
		if err != nil {
			log.Fatalf("%s: %v", method, err)
		}
		message, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatalf("%s: %v", method, err)
		}
		resp.Body.Close()
	}

	log.Debugf("received: %s", message)
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
	if len(respOuter.Signature) == 0 {
		log.Fatalf("%s: empty signature in response: %s", method, message)
	}
	var respInner types.APIresponse
	if err := json.Unmarshal(respOuter.MessageAPI, &respInner); err != nil {
		log.Fatalf("%s: %v", method, err)
	}
	return &respInner
}

func usage() {
	flag.PrintDefaults()
	fmt.Println(`Usage:
	// Generates a set of tokens for a new random entity (info of entity in log) and stores them in a new folder in /tmp
	go run cmd/managertest/managertest.go --method=generateTokens  --usersNumber=10
	// Generates a set of tokens for an existing entity (info of entity in log) and stores them in a new folder in /tmp
	go run cmd/managertest/managertest.go --method=generateTokens  --usersNumber=1 --entityKey=bb240
	// Get registerd status for list of  private Keys
	go run cmd/managertest/managertest.go --method=registrationStatus  --privKeys=/tmp/privKeys  --eid=e1245f...
	// Get registerd status for random entity and random private Keys
	go run cmd/managertest/managertest.go --method=registrationStatus  --usersNumber=10
	// Validate tokens for given entity, keys and tokens (any of those will be autogenerated when ommited)
	go run cmd/managertest/managertest.go --method=validateToken  --privKeys=/tmp/rivKeys --tokens=/tmp/tokens --eid=e1245f...
	// Perform entire registration flow for random entity and random keys (registrationStatus -> validateToken -> registrationStatus)
	go run cmd/managertest/managertest.go --method=registrationFlow  --usersNumber=10
	// Perform entire registration flow for given entity, keys and tokens (any of those will be autogenerated when ommited)
	go run cmd/managertest/managertest.go --method=registrationFlow  -privKeys=/tmp/privKeys --tokens=/tmp/tokens  --eid=e1245f...`)
}

func main() {
	host := flag.String("host", "127.0.0.1:8000", "host to connect to")
	apiRoute := flag.String("route", "/api", "base api route for queries")
	tls := flag.Bool("tls", false, "use TLS connection for the api")
	logLevel := flag.String("logLevel", "debug", "log level <debug, info, warn, error>")
	entityKey := flag.String("entityKey", "", "private key for signature (leave blank for auto-generate)")
	eid := flag.String("eid", "", "entityID")
	method := flag.String("method", "validateToken", " <registrationStatus, validateToken, generateTokens, registrationFlow>")
	usersNumber := flag.Int("usersNumber", 0, "number of keys to generate")
	tokenList := flag.String("tokens", "", "path to the file containing the tokens")
	privKeyList := flag.String("privKeys", "", "path to the file containing the user public keys")
	flag.Usage = usage
	flag.Parse()
	log.Init(*logLevel, "stdout")
	rand.Seed(time.Now().UnixNano())

	var keys, tokens []string
	var dir string
	var entityID types.HexBytes
	entityID, err := hex.DecodeString(*eid)
	if err != nil {
		log.Errorf("error decoding entity id:  %v", err)
	}

	if len(*privKeyList) > 0 {
		keys, err = splitFile(*privKeyList)
		if err != nil {
			log.Fatal(err)
		}
	} else if *usersNumber > 0 {
		dir, err = ioutil.TempDir("/tmp", "managertest*")
		if err != nil {
			log.Fatalf("error creating temp dir (%s)", err)
		}
		log.Info("Generating keys and exiting")
		keys = generateKeys(*usersNumber, dir)
	} else {
		log.Fatal("invalid arguments for keys generation")
	}
	// Create connections => http->registry / ws->manager
	cregHost := fmt.Sprintf("%s%s/registry", *host, *apiRoute)
	cmgrHost := fmt.Sprintf("%s%s/manager", *host, *apiRoute)
	if *tls {
		cregHost = "https://" + cregHost
		cmgrHost = "wss://" + cmgrHost
	} else {
		cregHost = "http://" + cregHost
		cmgrHost = "ws://" + cmgrHost
	}
	log.Infof("connecting to %s", cregHost)
	creg, err := NewHTTPapiConnection(cregHost)
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("connecting to %s", cmgrHost)
	cmgr, err := NewWSapiConnection(cmgrHost)
	if err != nil {
		log.Fatal(err)
	}
	defer cmgr.WS.Close(websocket.StatusNormalClosure, "")

	switch *method {
	case "generateTokens":
		if *usersNumber <= 0 {
			log.Fatal("invalid users number")
		}
		if len(*entityKey) == 0 {
			signer := ethereum.NewSignKeys()
			signer.Generate()
			_, *entityKey = signer.HexString()
			*eid = fmt.Sprintf("%x", signer.Address().Bytes())
			pub := signer.PublicKey()
			log.Infof("entity pub key: %x\n entity id %s", pub, *eid)
		}
		generateTokens(cmgr, *usersNumber, *entityKey, dir)

	case "registrationStatus":
		if *usersNumber > 0 {
			if len(*entityKey) == 0 {
				signer := ethereum.NewSignKeys()
				signer.Generate()
				_, *entityKey = signer.HexString()
				*eid = fmt.Sprintf("%x", signer.Address().Bytes())
				pub := signer.PublicKey()
				log.Infof("entity pub key: %x\n entity priv key %s\n entity id %s", pub, *entityKey, *eid)
			}
		}
		if len(*eid) < 20 {
			log.Fatal("invalid entityID")
		}
		if len(keys) == 0 {
			log.Fatal("No keys provided")
		}
		if err := registrationStatus(entityID, keys, creg); err != nil {
			log.Fatal(err)
		}
	case "validateToken":
		if len(*tokenList) > 0 {
			tokens, err = splitFile(*tokenList)
			if err != nil {
				log.Fatal(err)
			}
		} else if *usersNumber > 0 {
			if len(*entityKey) == 0 {
				generateEntity(entityKey, eid)
			}

			tokens = generateTokens(cmgr, *usersNumber, *entityKey, dir)
		} else {
			log.Fatal("invalid arguments for token generation")
		}
		if len(*eid) < 20 {
			log.Fatal("invalid entityID")
		}
		if len(keys) > 0 && len(keys) != len(tokens) {
			log.Fatal("Mismatch on keys and tokens size")
		}
		if err := validateToken(entityID, tokens, keys, creg); err != nil {
			log.Fatal(err)
		}
	case "registrationFlow":
		if len(*tokenList) > 0 {
			tokens, err = splitFile(*tokenList)
			if err != nil {
				log.Fatal(err)
			}
		} else if *usersNumber > 0 {
			if len(*entityKey) == 0 {
				generateEntity(entityKey, eid)
			}
			tokens = generateTokens(cmgr, *usersNumber, *entityKey, dir)
		} else {
			log.Fatal("invalid arguments for token generation")
		}
		if len(*eid) < 20 {
			log.Fatal("invalid entityID")
		}
		if len(keys) > 0 && len(keys) != len(tokens) {
			log.Fatal("Mismatch on keys and tokens size")
		}
		if err := registerFlow(entityID, tokens, keys, creg); err != nil {
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

func generateEntity(entityKey *string, eid *string) {
	signer := ethereum.NewSignKeys()
	signer.Generate()
	_, priv := signer.HexString()
	pub := signer.PublicKey()
	if entityID, err := util.PubKeyToEntityID(pub); err != nil {
		log.Errorf("cannot calculate entityID: (%v)", err)
	} else {
		*eid = fmt.Sprintf("%x", entityID)
	}
	*entityKey = priv
	log.Infof("entity pub key: %s\n entity priv key %s\n entity id %s", pub, priv, *eid)
}

func generateKeys(n int, dir string) []string {
	signer := ethereum.NewSignKeys()
	var keys []string
	for i := 0; i < n; i++ {
		signer.Generate()
		_, priv := signer.HexString()
		keys = append(keys, priv)
		// keys = keys + priv + "\n"
	}
	byteKeys := []byte(strings.Join(keys, "\n"))
	if len(dir) == 0 {
		var err error
		dir, err = ioutil.TempDir("/tmp", "managertest*")
		if err != nil {
			log.Fatalf("error creating temp dir (%s)", err)
		}
	}
	if err := ioutil.WriteFile(dir+"/privKeys", byteKeys, 0644); err != nil {
		log.Fatalf("error writting privKeys file (%s)", err)
	}
	log.Infof("stored keys in %s", dir+"/privKeys")
	return keys
}

func generateTokens(c *APIConnection, n int, entityKey, dir string) []string {
	signer := ethereum.NewSignKeys()
	if entityKey != "" {
		if err := signer.AddHexKey(entityKey); err != nil {
			panic(err)
		}
	} else {
		signer.Generate()
		pub, priv := signer.HexString()
		log.Infof("entity pub key: %s\n entity priv key", pub, priv)
	}
	var req types.APIrequest
	req.Method = "signUp"
	resp := c.Request(req, signer)
	if !resp.Ok {
		log.Warnf("error during entity signUp: (%s)", resp.Message)
	}
	log.Debugf("%+v", *resp)
	req.Method = "generateTokens"
	req.Amount = n
	resp = c.Request(req, signer)
	if !resp.Ok || len(resp.Tokens) != n {
		log.Fatal("error generating entity tokens")
	}
	var tokens []string
	for _, token := range resp.Tokens {
		tokens = append(tokens, "\""+token.String()+"\"")
	}
	byteKeys := []byte(strings.Join(tokens, "\n"))
	if len(dir) == 0 {
		var err error
		dir, err = ioutil.TempDir("/tmp", "managertest*")
		if err != nil {
			log.Fatalf("error creating temp dir (%s)", err)
		}
	}
	if err := ioutil.WriteFile(dir+"/tokens", byteKeys, 0644); err != nil {
		log.Fatalf("error writting tokens file (%s)", err)
	}
	log.Infof("stored tokens in %s", dir+"/tokens")
	return tokens
}

func registrationStatus(eid []byte, privKeyList []string, c *APIConnection) error {
	var req types.APIrequest
	req.Method = "registrationStatus"
	req.EntityID = eid
	s := ethereum.NewSignKeys()
	for _, key := range privKeyList {
		s.AddHexKey(key)
		resp := c.Request(req, s)
		if !resp.Ok {
			log.Warnf("recieved error (%s)", resp.Message)
		}
	}
	return nil
}

func validateToken(eid []byte, tokenList, privKeyList []string, c *APIConnection) error {
	var req types.APIrequest
	req.Method = "validateToken"
	req.EntityID = eid
	s := ethereum.NewSignKeys()
	if len(privKeyList) > 0 {
		for idx, t := range tokenList {
			s.AddHexKey(privKeyList[idx])
			req.Token = t
			resp := c.Request(req, s)
			if !resp.Ok {
				log.Warnf("recieved error (%s)", resp.Message)
			}
		}
	} else {
		for _, t := range tokenList {
			s.Generate()
			req.Token = t
			resp := c.Request(req, s)
			if !resp.Ok {
				log.Warnf("recieved error (%s)", resp.Message)
			}
		}
	}

	return nil
}

func registerFlow(eid []byte, tokenList, privKeyList []string, c *APIConnection) error {
	var req types.APIrequest
	req.EntityID = eid
	s := ethereum.NewSignKeys()
	for idx, t := range tokenList {
		if len(privKeyList) > 0 {
			s.AddHexKey(privKeyList[idx])
		} else {
			s.Generate()
		}
		req.Method = "registrationStatus"
		resp := c.Request(req, s)
		if resp.Ok && resp.Status != nil && resp.Status.Registered {
			log.Errorf("privKey %s already registered", privKeyList[idx])
		}
		req.Method = "validateToken"
		req.Token = t
		resp = c.Request(req, s)
		if !resp.Ok {
			log.Errorf("could not retrieve validate token (%q) for privKey (%q)", t, privKeyList[idx])
		}
		req.Method = "registrationStatus"
		req.Token = ""
		resp = c.Request(req, s)
		if !resp.Ok {
			log.Errorf("could not retrieve status for privKey (%s)", privKeyList[idx])
		}
		if !resp.Status.Registered {
			log.Errorf("privKey %s was not registered correctly", privKeyList[idx])
		}
	}

	return nil
}
