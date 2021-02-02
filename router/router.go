// Package router provides the routing and entry point for the go-dvote API
package router

import (
	"encoding/json"
	"fmt"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"

	psload "github.com/shirou/gopsutil/load"
	psmem "github.com/shirou/gopsutil/mem"
	psnet "github.com/shirou/gopsutil/net"
	"github.com/vocdoni/multirpc/transports"
	"go.vocdoni.io/dvote/crypto"
	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/dvote/log"
	"go.vocdoni.io/manager/types"
)

const (
	healthMemMax   = 100
	healthLoadMax  = 10
	healthSocksMax = 10000
)

type registeredMethod struct {
	public        bool
	skipSignature bool
	handler       func(RouterRequest)
}

type RouterRequest struct {
	types.MetaRequest
	transports.MessageContext

	method             string
	id                 string
	Authenticated      bool
	Address            ethcommon.Address
	SignaturePublicKey []byte
	private            bool
}

// Router holds a router object
type Router struct {
	Transports map[string]transports.Transport
	methods    map[string]registeredMethod
	inbound    <-chan transports.Message
	signer     *ethereum.SignKeys
}

// NewRouter creates a router multiplexer instance
func NewRouter(inbound <-chan transports.Message, transports map[string]transports.Transport, signer *ethereum.SignKeys) *Router {
	r := new(Router)
	r.methods = make(map[string]registeredMethod)
	r.inbound = inbound
	r.Transports = transports
	r.signer = signer
	return r
}

// InitRouter sets up a Router object which can then be used to route requests
func InitRouter(inbound <-chan transports.Message, transports map[string]transports.Transport, signer *ethereum.SignKeys) *Router {
	return NewRouter(inbound, transports, signer)
}

// AddHandler adds a new function handler for serving a specific method identified by name
func (r *Router) AddHandler(method, namespace string, handler func(RouterRequest), private, skipSignature bool) error {
	if private {
		return r.registerPrivate(namespace, method, handler)
	}
	return r.registerPublic(namespace, method, handler, skipSignature)
}

// Route routes requests through the Router object
func (r *Router) Route() {
	if len(r.methods) == 0 {
		log.Warnf("router methods are not properly initialized: %+v", r)
		return
	}
	for {
		msg := <-r.inbound
		request, err := r.getRequest(msg.Namespace, msg.Data, msg.Context)
		if err != nil {
			go r.SendError(request, err.Error())
			continue
		}

		method := r.methods[msg.Namespace+request.method]
		if !method.skipSignature && !request.Authenticated {
			go r.SendError(request, "invalid authentication")
			continue
		}
		log.Infof("api method %s/%s", msg.Namespace, request.method)
		if len(msg.Data) < 10000 {
			log.Debugf("received: %s\n\t%+v", msg.Data, request)
		}
		go method.handler(request)
	}
}

func (r *Router) getRequest(namespace string, payload []byte, context transports.MessageContext) (request RouterRequest, err error) {
	// First unmarshal the outer layer, to obtain the request ID, the signed
	// request, and the signature.
	var reqOuter types.RequestMessage
	if err := json.Unmarshal(payload, &reqOuter); err != nil {
		return request, err
	}
	request.id = reqOuter.ID
	request.MessageContext = context

	var reqInner types.MetaRequest
	if err := json.Unmarshal(reqOuter.MetaRequest, &reqInner); err != nil {
		return request, err
	}
	request.MetaRequest = reqInner
	request.method = reqInner.Method
	if request.method == "" {
		return request, fmt.Errorf("method is empty")
	}

	method, ok := r.methods[namespace+request.method]
	if !ok {
		return request, fmt.Errorf("method not valid (%s)", request.method)
	}

	if !method.skipSignature {
		if len(reqOuter.Signature) < 64 {
			return request, fmt.Errorf("no signature provided")
		}
		if request.SignaturePublicKey, err = ethereum.PubKeyFromSignature(reqOuter.MetaRequest, reqOuter.Signature); err != nil {
			return request, err
		}
		if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
			return request, fmt.Errorf("could not extract public key from signature")
		}
		if request.Address, err = ethereum.AddrFromPublicKey(request.SignaturePublicKey); err != nil {
			return request, err
		}
		request.private = !method.public

		// If private method, check authentication
		if method.public {
			request.Authenticated = true
		} else {
			if r.signer.Authorized[request.Address] {
				request.Authenticated = true
			}
		}
	}
	return request, err
}

func (r *Router) BuildReply(request *RouterRequest, resp *types.MetaResponse) transports.Message {
	// Add any last fields to the inner response, and marshal it with sorted
	// fields for signing.
	resp.Ok = true
	resp.Request = request.id
	resp.Timestamp = int32(time.Now().Unix())
	respInner, err := crypto.SortedMarshalJSON(resp)
	if err != nil {
		// This should never happen. If it does, return a very simple
		// plaintext error, and log the error.
		log.Error(err)
		return transports.Message{
			TimeStamp: int32(time.Now().Unix()),
			Context:   request.MessageContext,
			Data:      []byte(err.Error()),
		}
	}

	// Sign the marshaled inner response.
	signature, err := r.signer.Sign(respInner)
	if err != nil {
		log.Error(err)
		// continue without the signature
	}

	// Build the outer response with the already-marshaled inner response
	// and its signature.
	respOuter := types.ResponseMessage{
		ID:           request.id,
		Signature:    signature,
		MetaResponse: respInner,
	}
	// We don't need to use crypto.SortedMarshalJSON here, since we don't
	// sign these bytes.
	respData, err := json.Marshal(respOuter)
	if err != nil {
		// This should never happen. If it does, return a very simple
		// plaintext error, and log the error.
		log.Error(err)
		return transports.Message{
			TimeStamp: int32(time.Now().Unix()),
			Context:   request.MessageContext,
			Data:      []byte(err.Error()),
		}
	}
	if len(respData) < 10000 {
		log.Debugf("response: %s", respData)
	}
	return transports.Message{
		TimeStamp: int32(time.Now().Unix()),
		Context:   request.MessageContext,
		Data:      respData,
	}
}

func (r *Router) registerPrivate(namespace, method string, handler func(RouterRequest)) error {
	if _, ok := r.methods[namespace+method]; ok {
		return fmt.Errorf("duplicate method %s for namespace %s", method, namespace)
	}
	r.methods[namespace+method] = registeredMethod{handler: handler}
	return nil
}

func (r *Router) registerPublic(namespace, method string, handler func(RouterRequest), skipSignature bool) error {
	if _, ok := r.methods[namespace+method]; ok {
		return fmt.Errorf("duplicate method %s for namespace %s", method, namespace)
	}
	r.methods[namespace+method] = registeredMethod{public: true, handler: handler, skipSignature: skipSignature}
	return nil
}

func (r *Router) SendError(request RouterRequest, errMsg string) {
	log.Warn(errMsg)

	// Add any last fields to the inner response, and marshal it with sorted
	// fields for signing.
	response := types.MetaResponse{
		Request:   request.id,
		Timestamp: int32(time.Now().Unix()),
	}
	response.SetError(errMsg)
	respInner, err := crypto.SortedMarshalJSON(response)
	if err != nil {
		log.Error(err)
		return
	}

	// Sign the marshaled inner response.
	signature, err := r.signer.Sign(respInner)
	if err != nil {
		log.Error(err)
		// continue without the signature
	}

	respOuter := types.ResponseMessage{
		ID:           request.id,
		Signature:    signature,
		MetaResponse: respInner,
	}
	if request.MessageContext != nil {
		data, err := json.Marshal(respOuter)
		if err != nil {
			log.Warnf("error marshaling response body: %s", err)
		}
		msg := transports.Message{
			TimeStamp: int32(time.Now().Unix()),
			Context:   request.MessageContext,
			Data:      data,
		}
		request.Send(msg)
	}
}

func (r *Router) Info(request RouterRequest) {
	var response types.MetaResponse
	response.APIList = []string{"registry"}
	response.Request = request.id
	if health, err := getHealth(); err == nil {
		response.Health = health
	} else {
		response.Health = -1
		log.Errorf("cannot get health status: (%s)", err)
	}
	request.Send(r.BuildReply(&request, &response))
}

// Health is a number between 0 and 99 that represents the status of the node, as bigger the better
// The formula ued to calculate health is: 100* (1- ( Sum(weight[0..1] * value/value_max) ))
// Weight is a number between 0 and 1 used to give a specific weight to a value. The sum of all weights used must be equals to 1
//  so 0.2*value1 + 0.8*value2 would give 20% of weight to value1 and 80% of weight to value2
// Each value must be represented as a number between 0 and 1. To this aim the value might be divided by its maximum value
//  so if the mettered value is cpuLoad, a maximum must be defined in order to give a normalized value between 0 and 1
//   i.e cpuLoad=2 and maxCpuLoad=10. The value is: 2/10 (where cpuLoad<10) = 0.2
// The last operation includes the reverse of the values, so 1- (result).
//   And its *100 multiplication and trunking in order to provide a natural number between 0 and 99
func getHealth() (int32, error) {
	v, err := psmem.VirtualMemory()
	if err != nil {
		return 0, err
	}
	memUsed := v.UsedPercent
	l, err := psload.Avg()
	if err != nil {
		return 0, err
	}
	load15 := l.Load15
	n, err := psnet.Connections("tcp")
	if err != nil {
		return 0, err
	}
	sockets := float64(len(n))

	// ensure maximums are not overflow
	if memUsed > healthMemMax {
		memUsed = healthMemMax
	}
	if load15 > healthLoadMax {
		load15 = healthLoadMax
	}
	if sockets > healthSocksMax {
		sockets = healthSocksMax
	}
	result := int32((1 - (0.33*(memUsed/healthMemMax) +
		0.33*(load15/healthLoadMax) +
		0.33*(sockets/healthSocksMax))) * 100)
	if result < 0 || result >= 100 {
		return 0, fmt.Errorf("expected health to be between 0 and 99: %d", result)
	}
	return result, nil
}
