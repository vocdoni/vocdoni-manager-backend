package manager

import (
	"fmt"

	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/dvote/httprouter"
	"go.vocdoni.io/dvote/log"
	"go.vocdoni.io/manager/database"
	"go.vocdoni.io/manager/ethclient"
	"go.vocdoni.io/manager/rpcapi"
	"go.vocdoni.io/manager/smtpclient"
)

type Manager struct {
	api    *rpcapi.RPCAPI
	signer *ethereum.SignKeys
	db     database.Database
	smtp   *smtpclient.SMTP
	eth    *ethclient.Eth
}

func NewManager(signer *ethereum.SignKeys, router *httprouter.HTTProuter, route string, db database.Database, smtp *smtpclient.SMTP, eth *ethclient.Eth) (*Manager, error) {
	if signer == nil || db == nil {
		return nil, fmt.Errorf("invalid arguments for manager API")
	}

	api, err := rpcapi.NewAPI(signer, router, "registry", route+"/manager", nil, false)
	if err != nil {
		return nil, fmt.Errorf("could not create the manager API: %v", err)
	}
	// api := jsonrpcapi.NewSignedJRPC(signer, types.NewApiRequest, types.NewApiResponse, false)
	// rpcapi.AddNamespace("manager", api)
	// rpcapi.APIs = append(rpcapi.APIs, "manager")
	// api.AddAuthorizedAddress(signer.Address())
	// rpcapi.ManagerAPI = api
	return &Manager{
		api:    api,
		signer: signer,
		db:     db,
		smtp:   smtp,
		eth:    eth,
	}, nil
}

func (m *Manager) EnableAPI() error {
	log.Infof("enabling manager API")

	m.api.RegisterPublic("signUp", true, m.signUp)
	m.api.RegisterPublic("getEntity", true, m.getEntity)
	m.api.RegisterPublic("updateEntity", true, m.updateEntity)
	m.api.RegisterPublic("adminEntityList", true, m.adminEntityList)
	m.api.RegisterPublic("countMembers", true, m.countMembers)
	m.api.RegisterPublic("listMembers", true, m.listMembers)
	m.api.RegisterPublic("getMember", true, m.getMember)
	m.api.RegisterPublic("updateMember", true, m.updateMember)
	m.api.RegisterPublic("deleteMembers", true, m.deleteMembers)
	m.api.RegisterPublic("generateTokens", true, m.generateTokens)
	m.api.RegisterPublic("exportTokens", true, m.exportTokens)
	m.api.RegisterPublic("importMembers", true, m.importMembers)
	m.api.RegisterPublic("countTargets", true, m.countTargets)
	m.api.RegisterPublic("listTargets", true, m.listTargets)
	m.api.RegisterPublic("getTarget", true, m.getTarget)
	m.api.RegisterPublic("dumpTarget", true, m.dumpTarget)
	m.api.RegisterPublic("dumpCensus", true, m.dumpCensus)
	m.api.RegisterPublic("addCensus", true, m.addCensus)
	m.api.RegisterPublic("updateCensus", true, m.updateCensus)
	m.api.RegisterPublic("getCensus", true, m.getCensus)
	m.api.RegisterPublic("countCensus", true, m.countCensus)
	m.api.RegisterPublic("listCensus", true, m.listCensus)
	m.api.RegisterPublic("deleteCensus", true, m.deleteCensus)
	m.api.RegisterPublic("createTag", true, m.createTag)
	m.api.RegisterPublic("listTags", true, m.listTags)
	m.api.RegisterPublic("deleteTag", true, m.deleteTag)
	m.api.RegisterPublic("addTag", true, m.addTag)
	m.api.RegisterPublic("removeTag", true, m.removeTag)
	if m.eth != nil {
		// do not expose this endpoint if the manager does not have an ethereum client
		m.api.RegisterPublic("requestGas", true, m.requestGas)
	} else {
		log.Warn("No eth connection provided for manager API")
	}

	if m.smtp != nil {
		m.api.RegisterPublic("sendValidationLinks", true, m.sendValidationLinks)
		m.api.RegisterPublic("sendVotingLinks", true, m.sendVotingLinks)
	} else {
		log.Warn("No smtp server connection provided for manager API")
	}

	return nil
}
