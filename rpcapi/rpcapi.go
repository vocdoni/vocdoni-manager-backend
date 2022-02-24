package rpcapi

import (
	"github.com/ethereum/go-ethereum/common"
	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/dvote/httprouter"
	"go.vocdoni.io/dvote/httprouter/jsonrpcapi"
	"go.vocdoni.io/dvote/log"
	"go.vocdoni.io/dvote/metrics"
	"go.vocdoni.io/manager/types"
)

const MaxListSize = 64

type Handler = func(*types.APIrequest) (*types.APIresponse, error)
type RPCAPI struct {
	PrivateCalls uint64
	PublicCalls  uint64
	APIs         []string

	router       *httprouter.HTTProuter
	rpcAPI       *jsonrpcapi.SignedJRPC
	methods      map[string]Handler
	signer       *ethereum.SignKeys
	metricsagent *metrics.Agent
	allowPrivate bool
}

func NewAPI(signer *ethereum.SignKeys, router *httprouter.HTTProuter, name, endpoint string,
	metricsagent *metrics.Agent, allowPrivate bool) (*RPCAPI, error) {

	api := new(RPCAPI)
	api.signer = signer
	api.metricsagent = metricsagent
	api.allowPrivate = allowPrivate
	api.methods = make(map[string]Handler, 128)
	api.rpcAPI = jsonrpcapi.NewSignedJRPC(signer, types.NewApiRequest, types.NewApiResponse, allowPrivate)
	api.rpcAPI.AddAuthorizedAddress(signer.Address())
	api.router = router
	api.APIs = []string{name}
	router.AddNamespace(name, api.rpcAPI)

	api.RegisterPublic("getInfo", false, api.info)
	if metricsagent != nil {
		api.registerMetrics(metricsagent)
	}

	router.AddPrivateHandler(name, endpoint, "POST", api.route)

	return api, nil
}

func (a *RPCAPI) RegisterPrivate(method string, h Handler) {
	a.rpcAPI.RegisterMethod(method, true, false)
	a.methods[method] = h
}

func (a *RPCAPI) RegisterPublic(method string, requireSignature bool, h Handler) {
	a.rpcAPI.RegisterMethod(method, false, !requireSignature)
	a.methods[method] = h
}

func (a *RPCAPI) AuthorizedAddress(addr *common.Address) bool {
	if addr == nil {
		return false
	}
	if !a.allowPrivate {
		return false
	}
	// Warning: if allowPrivate is true but no authorized addresses,
	// we allow any address
	if len(a.signer.Authorized) == 0 {
		return true
	}
	return a.signer.Authorized[*addr]
}

func (a *RPCAPI) route(msg httprouter.Message) {
	request := msg.Data.(*jsonrpcapi.SignedJRPCdata)
	apiMsg := request.Message.(*types.APIrequest)
	apiMsg.SignaturePublicKey = request.SignaturePublicKey
	// apiMsg.SetAddress(&request.Address)
	method := a.methods[apiMsg.GetMethod()]
	apiMsgResponse, err := method(apiMsg)
	if err != nil {
		a.rpcAPI.SendError(
			request.ID,
			err.Error(),
			msg.Context,
		)
		return
	}
	apiMsgResponse.Ok = true
	data, err := jsonrpcapi.BuildReply(a.signer, apiMsgResponse, request.ID)
	if err != nil {
		log.Errorf("cannot build reply for method %s: %v", apiMsg.GetMethod(), err)
		return
	}
	if err := msg.Context.Send(data, 200); err != nil {
		log.Warnf("cannot send api response: %v", err)
	}
}

func (a *RPCAPI) AddNamespace(endpoint string, api *jsonrpcapi.SignedJRPC) {
	a.router.AddNamespace(endpoint, api)
}

func (a *RPCAPI) AddMethod(method string, h Handler) {
	a.methods[method] = h
}
