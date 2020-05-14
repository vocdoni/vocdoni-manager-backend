// Package router provides the routing and entry point for the go-dvote API
package router

import (
	"encoding/json"
	"errors"
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
	authenticated bool
	address       string
	context       dvote.MessageContext
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
		log.Warnf("received namespace: %s", msg.Namespace)
		request, err := r.getRequest(msg.Namespace, msg.Data, msg.Context)
		if !request.authenticated && err != nil {
			go r.sendError(request, err.Error())
			continue
		}
		method, ok := r.methods[msg.Namespace+request.method]
		if !ok {
			errMsg := fmt.Sprintf("router has no method %s/%s", msg.Namespace, request.method)
			go r.sendError(request, errMsg)
			continue
		}
		if !method.public && !request.authenticated {
			errMsg := fmt.Sprintf("authentication is required for %s/%s", msg.Namespace, request.method)
			go r.sendError(request, errMsg)
			continue
		}

		log.Infof("api method %s/%s", msg.Namespace, request.method)
		log.Debugf("received: %+v", request.MetaRequest)

		go method.handler(request)
	}
}

// semi-unmarshalls message, returns method name
func (r *Router) getRequest(namespace string, payload []byte, context dvote.MessageContext) (request RouterRequest, err error) {
	var msgStruct types.RequestMessage
	request.context = context
	err = json.Unmarshal(payload, &msgStruct)
	if err != nil {
		return request, err
	}
	request.MetaRequest = msgStruct.MetaRequest
	request.id = msgStruct.ID
	request.method = msgStruct.Method
	if request.method == "" {
		return request, errors.New("method is empty")
	}
	method, ok := r.methods[namespace+request.method]
	if !ok {
		return request, fmt.Errorf("method not valid (%s)", request.method)
	}
	if method.public {
		request.private = false
		request.authenticated = true
		request.address = "00000000000000000000"
	} else {
		request.private = true
		request.authenticated, request.address, err = r.signer.VerifyJSONsender(msgStruct.MetaRequest, msgStruct.Signature)
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
			Context:   request.context,
			Data:      []byte(err.Error()),
		}
	}
	log.Debugf("response: %s", respData)
	return dvote.Message{
		TimeStamp: int32(time.Now().Unix()),
		Context:   request.context,
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

func (r *Router) sendError(request RouterRequest, errMsg string) {
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
	if request.context != nil {
		data, err := json.Marshal(response)
		if err != nil {
			log.Warnf("error marshaling response body: %s", err)
		}
		msg := dvote.Message{
			TimeStamp: int32(time.Now().Unix()),
			Context:   request.context,
			Data:      data,
		}
		r.Transport.Send(msg)
	}
}
