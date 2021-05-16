package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"

	"go.vocdoni.io/dvote/crypto"
	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/dvote/log"
	dvotetypes "go.vocdoni.io/dvote/types"
	"go.vocdoni.io/manager/types"
	"nhooyr.io/websocket"
)

// APIConnection holds an API websocket connection
type APIConnection struct {
	Addr string
	WS   *websocket.Conn
	HTTP *http.Client
}

// NewAPIConnection starts a connection with the given endpoint address. The
// connection is closed automatically when the test or benchmark finishes.
func NewAPIConnection(addr string) *APIConnection {
	r := &APIConnection{Addr: addr}
	var err error
	if strings.HasPrefix(addr, "ws") {
		r.WS, _, err = websocket.Dial(context.TODO(), addr, nil)
		if err != nil {
			log.Fatal(err)
		}
		return r
	} else if strings.HasPrefix(addr, "http") {
		tr := &http.Transport{
			MaxIdleConns:       10,
			IdleConnTimeout:    10 * time.Second,
			DisableCompression: false,
		}
		r.HTTP = &http.Client{Transport: tr, Timeout: time.Second * 20}
	} else {
		log.Fatalf("address is not websockets nor http: %s", addr)
	}
	log.Info("client ready")
	return r
}

func (r *APIConnection) Close() error {
	var err error
	if r.WS != nil {
		err = r.WS.Close(websocket.StatusNormalClosure, "")
	}
	if r.HTTP != nil {
		r.HTTP.CloseIdleConnections()
	}
	return err
}

// Request makes a request to the previously connected endpoint
func (r *APIConnection) Request(req types.MetaRequest, signer *ethereum.SignKeys) *types.MetaResponse {
	method := req.Method

	req.Timestamp = int32(time.Now().Unix())
	reqInner, err := crypto.SortedMarshalJSON(req)
	if err != nil {
		log.Fatalf("%s: %v", method, err)
	}
	var signature dvotetypes.HexBytes
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
	message := []byte{}
	if r.WS != nil {
		tctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()
		if err := r.WS.Write(tctx, websocket.MessageText, reqBody); err != nil {
			log.Fatalf("%s: %v", method, err)
		}
		_, message, err = r.WS.Read(tctx)
		if err != nil {
			log.Fatalf("%s: %v", method, err)
		}
	}
	if r.HTTP != nil {
		resp, err := r.HTTP.Post(r.Addr, "application/json", bytes.NewBuffer(reqBody))
		if err != nil {
			log.Fatalf(err.Error())
		}
		message, err = io.ReadAll(resp.Body)
		if err != nil {
			log.Fatalf(err.Error())

		}
		resp.Body.Close()
	}
	// if err := r.Conn.Write(context.TODO(), websocket.MessageText, reqBody); err != nil {
	// 	log.Fatalf("%s: %v", method, err)
	// }
	// _, message, err := r.Conn.Read(context.TODO())
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
	if len(respOuter.Signature) == 0 {
		log.Fatalf("%s: empty signature in response: %s", method, message)
	}
	var respInner types.MetaResponse
	if err := json.Unmarshal(respOuter.MetaResponse, &respInner); err != nil {
		log.Fatalf("%s: %v", method, err)
	}
	return &respInner
}

func printNice(resp *types.MetaResponse) {
	v := reflect.ValueOf(*resp)
	typeOfS := v.Type()
	output := "\n"
	var val reflect.Value
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).Type().Name() == "bool" || v.Field(i).Type().Name() == "int64" || !v.Field(i).IsZero() {
			if v.Field(i).Kind() == reflect.Ptr {
				val = v.Field(i).Elem()
			} else {
				val = v.Field(i)
			}
			output += fmt.Sprintf("%v: %v\n", typeOfS.Field(i).Name, val)
		}
	}
	fmt.Print(output + "\n")
}

func processLine(input []byte) types.MetaRequest {
	var req types.MetaRequest
	err := json.Unmarshal(input, &req)
	if err != nil {
		panic(err)
	}
	return req
}

func main() {
	host := flag.String("host", "http://127.0.0.1:9000/api/manager", "host to connect to")
	logLevel := flag.String("logLevel", "error", "log level <debug, info, warn, error>")
	privKey := flag.String("key", "", "private key for signature (leave blank for auto-generate)")
	flag.Parse()
	log.Init(*logLevel, "stdout")
	rand.Seed(time.Now().UnixNano())

	signer := ethereum.NewSignKeys()
	if *privKey != "" {
		if err := signer.AddHexKey(*privKey); err != nil {
			panic(err)
		}
	} else {
		signer.Generate()
		_, priv := signer.HexString()
		log.Debugf("privKey %s", priv)
	}
	log.Infof("connecting to %s", *host)
	c := NewAPIConnection(*host)
	defer c.Close()
	var req types.MetaRequest
	reader := bufio.NewReader(os.Stdin)
	for {
		line, _, err := reader.ReadLine()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
		if len(line) < 7 || strings.HasPrefix(string(line), "#") {
			continue
		}
		req = processLine(line)

		resp := c.Request(req, signer)
		if !resp.Ok {
			printNice(resp)
		} else {
			printNice(resp)
		}

	}
}
