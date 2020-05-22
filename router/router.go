// Package router provides the routing and entry point for the go-dvote API
package router

import (
	"encoding/json"
	"fmt"
	"time"

	"gitlab.com/vocdoni/go-dvote/crypto/signature"
	"gitlab.com/vocdoni/go-dvote/log"
	"gitlab.com/vocdoni/go-dvote/net"
	dvote "gitlab.com/vocdoni/go-dvote/types"
	"gitlab.com/vocdoni/vocdoni-manager-backend/types"
)

type registeredMethod struct {
	public  bool
	handler func(RouterRequest)
}

type RouterRequest struct {
	types.MetaRequest

	method        string
	id            string
	Authenticated bool
	Address       string
	PublicKey     string
	Context       dvote.MessageContext
	private       bool
}

// Router holds a router object
type Router struct {
	Transport net.Transport
	methods   map[string]registeredMethod
	inbound   <-chan dvote.Message
	signer    *signature.SignKeys
}

// NewRouter creates a router multiplexer instance
func NewRouter(inbound <-chan dvote.Message, transport net.Transport, signer *signature.SignKeys) *Router {
	r := new(Router)
	r.methods = make(map[string]registeredMethod)
	r.inbound = inbound
	r.Transport = transport
	r.signer = signer
	return r
}

// InitRouter sets up a Router object which can then be used to route requests
func InitRouter(inbound <-chan dvote.Message, transport net.Transport, signer *signature.SignKeys) *Router {
	return NewRouter(inbound, transport, signer)
}

// AddHandler adds a new function handler for serving a specific method identified by name
func (r *Router) AddHandler(method, namespace string, handler func(RouterRequest), private bool) error {
	if private {
		return r.registerPrivate(namespace, method, handler)
	}
	return r.registerPublic(namespace, method, handler)
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
		if !request.Authenticated {
			go r.SendError(request, "invalid authentication")
			continue
		}
		method := r.methods[msg.Namespace+request.method]
		log.Infof("api method %s/%s", msg.Namespace, request.method)
		log.Debugf("received: %+v", request.MetaRequest)
		go method.handler(request)
	}
}

// semi-unmarshalls message, returns method name
func (r *Router) getRequest(namespace string, payload []byte, context dvote.MessageContext) (request RouterRequest, err error) {
	var msgStruct types.RequestMessage
	request.Context = context
	err = json.Unmarshal(payload, &msgStruct)
	if err != nil {
		return request, err
	}
	request.MetaRequest = msgStruct.MetaRequest
	request.id = msgStruct.ID
	request.method = msgStruct.Method
	if request.method == "" {
		return request, fmt.Errorf("method is empty")
	}
	if len(msgStruct.Signature) < 64 {
		return request, fmt.Errorf("no signature provided")
	}
	method, ok := r.methods[namespace+request.method]
	if !ok {
		return request, fmt.Errorf("method not valid (%s)", request.method)
	}

	// Extract publicKey and address from signature
	msg, err := json.Marshal(msgStruct.MetaRequest)
	if err != nil {
		return request, fmt.Errorf("unable to marshal message to sign: %s", msg)
	}
	if request.PublicKey, err = signature.PubKeyFromSignature(msg, msgStruct.Signature); err != nil {
		return request, err
	}
	if len(request.PublicKey) == 0 {
		return request, fmt.Errorf("could not extract public key from signature")
	}
	if request.Address, err = signature.AddrFromPublicKey(request.PublicKey); err != nil {
		return request, err
	}
	request.private = !method.public

	// If private method, check authentication
	if method.public {
		request.Authenticated = true
	} else {
		for _, addr := range r.signer.Authorized {
			if fmt.Sprintf("%x", addr) == request.Address {
				request.Authenticated = true
				break
			}
		}

	}
	return request, err
}

func (r *Router) BuildReply(request RouterRequest, response types.ResponseMessage) dvote.Message {
	response.ID = request.id
	response.Ok = true
	response.Request = request.id
	response.Timestamp = int32(time.Now().Unix())
	var err error
	response.Signature, err = r.signer.SignJSON(response.MetaResponse)
	if err != nil {
		log.Error(err)
		// continue without the signature
	}
	respData, err := json.Marshal(response)
	if err != nil {
		// This should never happen. If it does, return a very simple
		// plaintext error, and log the error.
		log.Error(err)
		return dvote.Message{
			TimeStamp: int32(time.Now().Unix()),
			Context:   request.Context,
			Data:      []byte(err.Error()),
		}
	}
	log.Debugf("response: %s", respData)
	return dvote.Message{
		TimeStamp: int32(time.Now().Unix()),
		Context:   request.Context,
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

func (r *Router) registerPublic(namespace, method string, handler func(RouterRequest)) error {
	if _, ok := r.methods[namespace+method]; ok {
		return fmt.Errorf("duplicate method %s for namespace %s", method, namespace)
	}
	r.methods[namespace+method] = registeredMethod{public: true, handler: handler}
	return nil
}

func (r *Router) SendError(request RouterRequest, errMsg string) {
	log.Warn(errMsg)
	var err error
	var response types.ResponseMessage
	response.ID = request.id
	response.MetaResponse.Request = request.id
	response.MetaResponse.Timestamp = int32(time.Now().Unix())
	response.MetaResponse.SetError(errMsg)
	response.Signature, err = r.signer.SignJSON(response.MetaResponse)
	if err != nil {
		log.Error(err)
	}
	if request.Context != nil {
		data, err := json.Marshal(response)
		if err != nil {
			log.Warnf("error marshaling response body: %s", err)
		}
		msg := dvote.Message{
			TimeStamp: int32(time.Now().Unix()),
			Context:   request.Context,
			Data:      data,
		}
		r.Transport.Send(msg)
	}
}
