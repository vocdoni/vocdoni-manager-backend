// Package router provides the routing and entry point for the go-dvote API
package router

import (
	"encoding/json"
	"fmt"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"

	"gitlab.com/vocdoni/go-dvote/crypto"
	"gitlab.com/vocdoni/go-dvote/crypto/ethereum"
	"gitlab.com/vocdoni/go-dvote/log"
	"gitlab.com/vocdoni/go-dvote/net"
	dvote "gitlab.com/vocdoni/go-dvote/types"
	"gitlab.com/vocdoni/manager/manager-backend/types"
)

type registeredMethod struct {
	public        bool
	skipSignature bool
	handler       func(RouterRequest)
}

type RouterRequest struct {
	types.MetaRequest
	dvote.MessageContext

	method             string
	id                 string
	Authenticated      bool
	Address            ethcommon.Address
	SignaturePublicKey string
	private            bool
}

// Router holds a router object
type Router struct {
	Transports map[string]net.Transport
	methods    map[string]registeredMethod
	inbound    <-chan dvote.Message
	signer     *ethereum.SignKeys
}

// NewRouter creates a router multiplexer instance
func NewRouter(inbound <-chan dvote.Message, transports map[string]net.Transport, signer *ethereum.SignKeys) *Router {
	r := new(Router)
	r.methods = make(map[string]registeredMethod)
	r.inbound = inbound
	r.Transports = transports
	r.signer = signer
	return r
}

// InitRouter sets up a Router object which can then be used to route requests
func InitRouter(inbound <-chan dvote.Message, transports map[string]net.Transport, signer *ethereum.SignKeys) *Router {
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
		log.Debugf("received: %s\n\t%+v", msg.Data, request)
		go method.handler(request)
	}
}

func (r *Router) getRequest(namespace string, payload []byte, context dvote.MessageContext) (request RouterRequest, err error) {
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
		// TBD: remove when everything is compressed only
		//	if request.SignaturePublicKey, err = ethereum.CompressPubKey(request.SignaturePublicKey); err != nil {
		//		return request, err
		//	}
		if len(request.SignaturePublicKey) == 0 {
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

func (r *Router) BuildReply(request *RouterRequest, resp *types.MetaResponse) dvote.Message {
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
		return dvote.Message{
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
		return dvote.Message{
			TimeStamp: int32(time.Now().Unix()),
			Context:   request.MessageContext,
			Data:      []byte(err.Error()),
		}
	}
	log.Debugf("response: %s", respData)
	return dvote.Message{
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
		msg := dvote.Message{
			TimeStamp: int32(time.Now().Unix()),
			Context:   request.MessageContext,
			Data:      data,
		}
		request.Send(msg)
	}
}
