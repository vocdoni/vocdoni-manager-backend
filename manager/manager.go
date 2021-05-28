package manager

import (
	"context"
	"database/sql"
	"encoding/json"
	"math/rand"
	"sync"

	"fmt"
	"reflect"
	"strings"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"go.vocdoni.io/manager/ethclient"

	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/dvote/log"
	"go.vocdoni.io/dvote/multirpc/transports"
	"go.vocdoni.io/manager/database"
	"go.vocdoni.io/manager/database/pgsql"
	"go.vocdoni.io/manager/router"
	"go.vocdoni.io/manager/smtpclient"
	"go.vocdoni.io/manager/types"
	"go.vocdoni.io/manager/util"
)

type Manager struct {
	Router *router.Router
	db     database.Database
	smtp   *smtpclient.SMTP
	faucet *ethclient.Faucet
}

// NewManager creates a new registry handler for the Router
func NewManager(r *router.Router,
	d database.Database,
	s *smtpclient.SMTP,
	ethfaucet *ethclient.Faucet) *Manager {
	return &Manager{
		Router: r,
		db:     d,
		smtp:   s,
		faucet: ethfaucet,
	}
}

// RegisterMethods registers all registry methods behind the given path
func (m *Manager) RegisterMethods(path string) error {
	var transport transports.Transport
	if tr, ok := m.Router.Transports["httpws"]; ok {
		transport = tr
	} else if tr, ok = m.Router.Transports["ws"]; ok {
		transport = tr
	} else if tr, ok = m.Router.Transports["http"]; ok {
		transport = tr
	} else {
		return fmt.Errorf("no compatible transports found (ws or http)")
	}

	log.Infof("adding namespace manager %s", path+"/manager")
	transport.AddNamespace(path + "/manager")
	if err := m.Router.AddHandler("getInfo", path+"/manager", m.Router.Info, false, true); err != nil {
		return err
	}
	if err := m.Router.AddHandler("signUp", path+"/manager", m.signUp, false, false); err != nil {
		return err
	}
	if err := m.Router.AddHandler("getEntity", path+"/manager", m.getEntity, false, false); err != nil {
		return err
	}
	if err := m.Router.AddHandler("updateEntity", path+"/manager", m.updateEntity, false, false); err != nil {
		return err
	}
	if err := m.Router.AddHandler("countMembers", path+"/manager", m.countMembers, false, false); err != nil {
		return err
	}
	if err := m.Router.AddHandler("listMembers", path+"/manager", m.listMembers, false, false); err != nil {
		return err
	}
	if err := m.Router.AddHandler("getMember", path+"/manager", m.getMember, false, false); err != nil {
		return err
	}
	if err := m.Router.AddHandler("updateMember", path+"/manager", m.updateMember, false, false); err != nil {
		return err
	}
	if err := m.Router.AddHandler("deleteMembers", path+"/manager", m.deleteMembers, false, false); err != nil {
		return err
	}
	if err := m.Router.AddHandler("generateTokens", path+"/manager", m.generateTokens, false, false); err != nil {
		return err
	}
	if err := m.Router.AddHandler("exportTokens", path+"/manager", m.exportTokens, false, false); err != nil {
		return err
	}
	if err := m.Router.AddHandler("importMembers", path+"/manager", m.importMembers, false, false); err != nil {
		return err
	}
	if err := m.Router.AddHandler("countTargets", path+"/manager", m.countTargets, false, false); err != nil {
		return err
	}
	if err := m.Router.AddHandler("listTargets", path+"/manager", m.listTargets, false, false); err != nil {
		return err
	}
	if err := m.Router.AddHandler("getTarget", path+"/manager", m.getTarget, false, false); err != nil {
		return err
	}
	if err := m.Router.AddHandler("dumpTarget", path+"/manager", m.dumpTarget, false, false); err != nil {
		return err
	}
	if err := m.Router.AddHandler("dumpCensus", path+"/manager", m.dumpCensus, false, false); err != nil {
		return err
	}
	if err := m.Router.AddHandler("addCensus", path+"/manager", m.addCensus, false, false); err != nil {
		return err
	}
	if err := m.Router.AddHandler("updateCensus", path+"/manager", m.updateCensus, false, false); err != nil {
		return err
	}
	if err := m.Router.AddHandler("getCensus", path+"/manager", m.getCensus, false, false); err != nil {
		return err
	}
	if err := m.Router.AddHandler("countCensus", path+"/manager", m.countCensus, false, false); err != nil {
		return err
	}
	if err := m.Router.AddHandler("listCensus", path+"/manager", m.listCensus, false, false); err != nil {
		return err
	}
	if err := m.Router.AddHandler("deleteCensus", path+"/manager", m.deleteCensus, false, false); err != nil {
		return err
	}
	if err := m.Router.AddHandler("sendValidationLinks", path+"/manager", m.sendValidationLinks, false, false); err != nil {
		return err
	}
	if err := m.Router.AddHandler("sendVotingLinks", path+"/manager", m.sendVotingLinks, false, false); err != nil {
		return err
	}
	if err := m.Router.AddHandler("createTag", path+"/manager", m.createTag, false, false); err != nil {
		return err
	}
	if err := m.Router.AddHandler("listTags", path+"/manager", m.listTags, false, false); err != nil {
		return err
	}
	if err := m.Router.AddHandler("deleteTag", path+"/manager", m.deleteTag, false, false); err != nil {
		return err
	}
	if err := m.Router.AddHandler("addTag", path+"/manager", m.addTag, false, false); err != nil {
		return err
	}
	if err := m.Router.AddHandler("removeTag", path+"/manager", m.removeTag, false, false); err != nil {
		return err
	}
	if m.faucet != nil {
		// do not expose this endpoint if the manager does not have the faucet
		if err := m.Router.AddHandler("requestGas", path+"/manager", m.requestGas, false, false); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) send(req *router.RouterRequest, resp *types.MetaResponse) {
	if req == nil || req.MessageContext == nil || resp == nil {
		log.Errorf("message context or request is nil, cannot send reply message")
		return
	}
	req.Send(m.Router.BuildReply(req, resp))
}

func (m *Manager) signUp(request router.RouterRequest) {
	var entityID []byte
	var entityInfo *types.EntityInfo
	var target *types.Target
	var err error
	var response types.MetaResponse

	// check public key length
	// dvoteutil.IsHexEncodedStringWithLength
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot recover entityID")
		return
	}

	entityInfo = &types.EntityInfo{CensusManagersAddresses: [][]byte{entityID}, Origins: []types.Origin{types.Token}}
	if request.Entity != nil {
		// For now control which EntityInfo fields end up to the DB
		entityInfo.Name = request.Entity.Name
		entityInfo.Email = request.Entity.Email
	}

	// Add Entity
	if err = m.db.AddEntity(entityID, entityInfo); err != nil && !strings.Contains(err.Error(), "entities_pkey") {
		log.Errorf("cannot add entity %x to the DB: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot add entity to the DB")
		return
	}

	target = &types.Target{EntityID: entityID, Name: "all", Filters: json.RawMessage([]byte("{}"))}
	if _, err = m.db.AddTarget(entityID, target); err != nil && !strings.Contains(err.Error(), "result has no rows") {
		log.Errorf("cannot create entity's %x generic target: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot create entity generic target")
		return
	}

	entityAddress := ethcommon.BytesToAddress(entityID)
	// do not try to send tokens if no faucet
	if m.faucet != nil {
		// send the default amount of faucet tokens iff wallet balance is zero
		sent, err := m.faucet.SendTokens(context.Background(), entityAddress)
		if err != nil {
			if !strings.Contains(err.Error(), "maxAcceptedBalance") {
				log.Errorf("error sending tokens to entity %s : %v", entityAddress.String(), err)
				m.Router.SendError(request, "could not send tokens to empty wallet")
				return
			}
			log.Warnf("signUp not sending tokens to entity %s : %v", entityAddress.String(), err)
		}
		response.Count = int(sent.Int64())
	}

	log.Debugf("Entity: %s signUp", entityAddress.String())
	m.send(&request, &response)
}

func (m *Manager) getEntity(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot recover entityID")
		return
	}

	if response.Entity, err = m.db.Entity(entityID); err != nil {
		if err == sql.ErrNoRows {
			log.Errorf("entity requesting its info with getEntity not found")
			m.Router.SendError(request, "entity not found")
			return
		}
		log.Errorf("cannot retrieve details of entity %x: (%v)", entityID, err)
		m.Router.SendError(request, "cannot retrieve entity")
		return
	}

	log.Infof("listing details of Entity %x", entityID)
	m.send(&request, &response)
}

func (m *Manager) updateEntity(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	// check public key length
	// dvoteutil.IsHexEncodedStringWithLength
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot recover entityID")
		return
	}

	if request.Entity == nil {
		log.Errorf("updateEntity with no entity data to update for %x", entityID)
		m.Router.SendError(request, "no entity data to update")
		return
	}

	entityInfo := &types.EntityInfo{
		Name:  request.Entity.Name,
		Email: request.Entity.Email,
		// Initialize values to accept empty spaces from the UI
		CallbackURL:    "",
		CallbackSecret: "",
	}
	if len(request.Entity.CallbackURL) > 0 {
		entityInfo.CallbackURL = request.Entity.CallbackURL
	}
	if len(request.Entity.CallbackSecret) > 0 {
		entityInfo.CallbackSecret = request.Entity.CallbackSecret
	}

	// Add Entity
	if response.Count, err = m.db.UpdateEntity(entityID, entityInfo); err != nil {
		log.Errorf("cannot update entity %x to the DB: (%v)", entityID, err)
		m.Router.SendError(request, "cannot update entity")
		return
	}

	log.Debugf("Entity: %x entityUpdate", entityID)
	m.send(&request, &response)
}

func (m *Manager) listMembers(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot recover entityID")
		return
	}

	// check filter
	if err = checkOptions(request.ListOptions, request.Method); err != nil {
		log.Warnf("invalid filter options %x: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "invalid filter options")
		return
	}

	// Query for members
	if response.Members, err = m.db.ListMembers(entityID, request.ListOptions); err != nil {
		if err == sql.ErrNoRows {
			m.Router.SendError(request, "no members found")
			return
		}
		log.Errorf("cannot retrieve members of %x: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot retrieve members")
		return
	}

	log.Debugf("Entity: %x listMembers %d members", request.SignaturePublicKey, len(response.Members))
	m.send(&request, &response)
}

func (m *Manager) getMember(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	if request.MemberID == nil {
		log.Warnf("memberID is nil on getMember")
		m.Router.SendError(request, "invalid memberId")
		return
	}

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot recover entityID")
		return
	}

	if response.Member, err = m.db.Member(entityID, request.MemberID); err != nil {
		if err == sql.ErrNoRows {
			log.Warn("member not found")
			m.Router.SendError(request, "member not found")
			return
		}
		log.Errorf("cannot retrieve member %q for entity %x: (%v)", request.MemberID.String(), request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot retrieve member")
		return
	}

	// TODO: Change when targets are implemented
	var targets []types.Target
	targets, err = m.db.ListTargets(entityID)
	if err == sql.ErrNoRows || len(targets) == 0 {
		log.Warnf("no targets found for member %q of entity %x", request.MemberID.String(), request.SignaturePublicKey)
		response.Target = &types.Target{}
	} else if err == nil {
		response.Target = &targets[0]
	} else {
		log.Errorf("error retrieving member %q targets for entity %x: (%v)", request.MemberID.String(), request.SignaturePublicKey, err)
		m.Router.SendError(request, "error retrieving member targets")
		return
	}

	log.Infof("listing member %q for Entity with public Key %x", request.MemberID.String(), request.SignaturePublicKey)
	m.send(&request, &response)
}

func (m *Manager) updateMember(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	if request.Member == nil {
		m.Router.SendError(request, "invalid member struct")
	}

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot recover entityID")
		return
	}

	// If a string Member property is sent as "" then it is not updated
	if response.Count, err = m.db.UpdateMember(entityID, &request.Member.ID, &request.Member.MemberInfo); err != nil {
		log.Errorf("cannot update member %q for entity %x: (%v)", request.Member.ID.String(), request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot update member")
		return
	}

	log.Infof("update member %q for Entity with public Key %x", request.Member.ID.String(), request.SignaturePublicKey)
	m.send(&request, &response)
}

func (m *Manager) deleteMembers(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	if len(request.MemberIDs) == 0 {
		m.Router.SendError(request, "invalid member list")
		return
	}

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot recover entityID")
		return
	}

	response.Count, response.InvalidIDs, err = m.db.DeleteMembers(entityID, request.MemberIDs)
	if err != nil {
		log.Errorf("error deleting members for entity %x: (%v)", entityID, err)
		m.Router.SendError(request, "error deleting members")
		return
	}

	log.Infof("deleted %d members, found %d invalid tokens, for Entity with public Key %x", response.Count, len(response.InvalidIDs), entityID)
	m.send(&request, &response)
}

func (m *Manager) countMembers(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot recover entityID")
		return
	}

	// Query for members
	if response.Count, err = m.db.CountMembers(entityID); err != nil {
		log.Errorf("cannot count members for %x: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot count members")
		return
	}

	log.Debugf("Entity %q countMembers: %d members", request.SignaturePublicKey, response.Count)
	m.send(&request, &response)
}

func (m *Manager) generateTokens(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot recover entityID")
		return
	}

	if request.Amount < 1 {
		log.Warnf("invalid token amount requested by %x", request.SignaturePublicKey)
		m.Router.SendError(request, "invalid token amount")
		return
	}

	response.Tokens = make([]uuid.UUID, request.Amount)
	for idx := range response.Tokens {
		response.Tokens[idx] = uuid.New()
	}
	// TODO: Probably I need to initialize tokens
	if err = m.db.CreateMembersWithTokens(entityID, response.Tokens); err != nil {
		log.Errorf("could not register generated tokens for %x: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "could not register generated tokens")
		return
	}

	log.Debugf("Entity: %x generateTokens: %d tokens", request.SignaturePublicKey, len(response.Tokens))
	m.send(&request, &response)
}

func (m *Manager) exportTokens(request router.RouterRequest) {
	var entityID []byte
	var members []types.Member
	var err error
	var response types.MetaResponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot recover entityID")
		return
	}

	// TODO: Probably I need to initialize tokens
	if members, err = m.db.MembersTokensEmails(entityID); err != nil {
		if err == sql.ErrNoRows {
			m.Router.SendError(request, "no members found")
			return
		}
		log.Errorf("could not retrieve members tokens for %x: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, err.Error())
		return
	}
	response.MembersTokens = make([]types.TokenEmail, len(members))
	for idx, member := range members {
		response.MembersTokens[idx] = types.TokenEmail{Token: member.ID, Email: member.Email}
	}

	log.Debugf("Entity: %x exportTokens: %d tokens", request.SignaturePublicKey, len(members))
	m.send(&request, &response)
}

func (m *Manager) importMembers(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot recover entityID")
		return
	}

	if len(request.MembersInfo) < 1 {
		log.Warnf("no member data provided for import members by %x", request.SignaturePublicKey)
		m.Router.SendError(request, "no member data provided")
		return
	}

	for idx := range request.MembersInfo {
		request.MembersInfo[idx].Origin = types.Token
	}

	// Add members
	if err = m.db.ImportMembers(entityID, request.MembersInfo); err != nil {
		log.Errorf("could not import members for %x: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, err.Error())
		return
	}

	log.Debugf("Entity: %x importMembers: %d members", request.SignaturePublicKey, len(request.MembersInfo))
	m.send(&request, &response)
}

func (m *Manager) countTargets(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot recover entityID")
		return
	}

	// Query for members
	if response.Count, err = m.db.CountTargets(entityID); err != nil {
		log.Errorf("cannot count targets for %x: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot count targets")
		return
	}

	log.Debugf("Entity %x countTargets: %d targets", request.SignaturePublicKey, response.Count)
	m.send(&request, &response)
}

func (m *Manager) listTargets(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot recover entityID")
		return
	}

	// check filter
	if err = checkOptions(request.ListOptions, request.Method); err != nil {
		log.Warnf("invalid filter options %x: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, err.Error())
		return
	}

	// Retrieve targets
	// Implement filters in DB
	response.Targets, err = m.db.ListTargets(entityID)
	if err != nil || len(response.Targets) == 0 {
		if err == sql.ErrNoRows {
			m.Router.SendError(request, "no targets found")
			return
		}
		log.Errorf("cannot query targets for %x: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot query targets")
		return
	}

	log.Debugf("Entity: %x listTargets: %d targets", request.SignaturePublicKey, len(response.Targets))
	m.send(&request, &response)
}

func (m *Manager) getTarget(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot recover entityID")
		return
	}

	if response.Target, err = m.db.Target(entityID, request.TargetID); err != nil {
		if err == sql.ErrNoRows {
			log.Debugf("target %q not found for %x", request.TargetID.String(), request.SignaturePublicKey)
			m.Router.SendError(request, "target not found")
			return
		}
		log.Errorf("could not retrieve target for %x: %+v", request.SignaturePublicKey, err)
		m.Router.SendError(request, "could not retrieve target")
		return
	}

	response.Members, err = m.db.TargetMembers(entityID, request.TargetID)
	if err != nil {
		log.Warn("members for requested target could not be retrieved")
		m.Router.SendError(request, "members for requested target could not be retrieved")
		return
	}

	log.Debugf("Entity: %x getTarget: %s", request.SignaturePublicKey, request.TargetID.String())
	m.send(&request, &response)
}

func (m *Manager) dumpTarget(request router.RouterRequest) {
	var target *types.Target
	var entityID []byte
	var err error
	var response types.MetaResponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot recover entityID")
		return
	}

	if target, err = m.db.Target(entityID, request.TargetID); err != nil || target.Name != "all" {
		if err == sql.ErrNoRows {
			log.Debugf("target %q not found for %x", request.TargetID.String(), request.SignaturePublicKey)
			m.Router.SendError(request, "target not found")
			return
		}
		log.Errorf("could not retrieve target for %x: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "could not retrieve target")
		return
	}

	// TODO: Implement DumpTargetClaims filtered directly by target filters
	if response.Claims, err = m.db.DumpClaims(entityID); err != nil {
		if err == sql.ErrNoRows {
			log.Debugf("no claims found for %x", request.SignaturePublicKey)
			m.Router.SendError(request, "no claims found")
			return
		}
		log.Errorf("cannot dump claims for %x: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot dump claims")
		return
	}

	log.Debugf("Entity: %x dumpTarget: %d claims", request.SignaturePublicKey, len(response.Claims))
	m.send(&request, &response)
}

func (m *Manager) dumpCensus(request router.RouterRequest) {
	var err error
	var response types.MetaResponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	entityID, err := util.PubKeyToEntityID(request.SignaturePublicKey)
	if err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot recover entityID")
		return
	}

	censusID, err := util.DecodeCensusID(request.CensusID, request.SignaturePublicKey)
	if err != nil {
		log.Errorf("cannot decode census id %s for %x", request.CensusID, entityID)
		m.Router.SendError(request, "cannot decode census id")
		return
	}

	// TODO: Implement DumpTargetClaims filtered directly by target filters
	censusMembers, err := m.db.ExpandCensusMembers(entityID, censusID)
	if err != nil {
		log.Errorf("cannot dump claims for %q: (%v)", entityID, err)
		m.Router.SendError(request, "cannot dump claims")
		return
	}
	shuffledClaims := make([][]byte, len(censusMembers))
	shuffledIndexes := rand.Perm(len(censusMembers))
	for i, v := range shuffledIndexes {
		shuffledClaims[v] = censusMembers[i].DigestedPubKey
	}
	response.Claims = shuffledClaims

	log.Debugf("Entity: %x dumpCensus: %d claims", entityID, len(response.Claims))
	m.send(&request, &response)
}

func (m *Manager) sendVotingLinks(request router.RouterRequest) {

	if len(request.MemberID) == 0 || len(request.ProcessID) == 0 {
		m.Router.SendError(request, "invalid arguments")
		return
	}

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	entityID, err := util.PubKeyToEntityID(request.SignaturePublicKey)
	if err != nil {
		log.Errorf("cannot recover %x entityID from public key: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot recover entityID from public key")
		return
	}

	entity, err := m.db.Entity(entityID)
	if err != nil {
		log.Errorf("cannot recover entity %x: (%v)", entityID, err)
		m.Router.SendError(request, "cannot recover entity from public key")
		return
	}

	censusID, err := util.DecodeCensusID(request.CensusID, request.SignaturePublicKey)
	if err != nil {
		log.Errorf("cannot decode census id %s for %x", request.CensusID, entityID)
		m.Router.SendError(request, "cannot decode census id")
		return
	}

	if request.Email != "" {
		// Individual email
		censusMember, err := m.db.EphemeralMemberInfoByEmail(entityID, censusID, request.Email)
		if err != nil {
			log.Errorf("cannot retrieve ephemeral member %s of  census %x for enity %x: (%v)", request.Email, censusID, entityID, err)
			m.Router.SendError(request, "cannot retrieve ephemeral census member by email")
			return
		}
		if err := m.smtp.SendVotingLink(censusMember, entity, request.ProcessID); err != nil {
			log.Errorf("could not send voting link for member %q entity: (%v)", censusMember.ID, err)
			m.Router.SendError(request, "could not send voting link")
			return
		}
		log.Infof("send validation links to 1 members for Entity %x", entityID)
		var response types.MetaResponse
		response.Count = 1
		m.send(&request, &response)
		return
	}

	censusMembers, err := m.db.ListEphemeralMemberInfo(entityID, censusID)
	if err != nil {
		log.Errorf("cannot retrieve ephemeral members of  census %x for enity %x: (%v)", censusID, entityID, err)
		m.Router.SendError(request, "cannot retrieve ephemeral census members")
		return
	}

	var response types.MetaResponse
	if len(censusMembers) == 0 {
		response.Count = 0
		m.send(&request, &response)
	}
	// send concurrently emails
	processID := request.ProcessID
	var wg sync.WaitGroup
	wg.Add(len(censusMembers))
	ec := make(chan error, len(censusMembers))
	sc := make(chan uuid.UUID, len(censusMembers))
	for _, member := range censusMembers {
		go func(member types.EphemeralMemberInfo) {
			if err := m.smtp.SendVotingLink(&member, entity, processID); err != nil {
				log.Errorf("could not send voting link for member %q entity: (%v)", member.ID, err)
				ec <- fmt.Errorf("member %s error  %v", member.ID, err)
				wg.Done()
				return
			}
			sc <- member.ID
			wg.Done()
		}(member)
	}
	wg.Wait()
	close(ec)
	close(sc)
	// get results
	var successUUIDs []uuid.UUID
	for uid := range sc {
		successUUIDs = append(successUUIDs, uid)
	}
	response.Count = int(len(successUUIDs))
	var errors []error
	for err := range ec {
		errors = append(errors, err)
	}
	if len(errors)+response.Count != len(censusMembers) {
		log.Errorf("inconsistency in number of sent emails and errors")
		m.Router.SendError(request, "inconsistency in number of sent emails and errors")
	}
	if len(errors) == len(censusMembers) {
		log.Errorf("no validation email was sent %v", errors)
		m.Router.SendError(request, "could not send emails")
		return
	}
	if len(errors) > 0 {
		response.Message = fmt.Sprintf("%d where found:\n%v", len(errors), errors)
	}

	// add tag PendingValidation to sucessful members
	tagName := "VoteEmailSent"
	tag, err := m.db.TagByName(entityID, tagName)
	if err != nil {
		if err == sql.ErrNoRows {
			tag = &types.Tag{}
			tag.ID, err = m.db.AddTag(entityID, tagName)
			if err != nil {
				log.Infof("send validation links to %d members, skipped %d invalid IDs and %d errors , for Entity %x\nErrors: %v", response.Count, len(response.InvalidIDs), len(errors), entityID, errors)
				log.Errorf("error creating Pending tag:  %v", err)
				m.Router.SendError(request, "sent emails but could not assign tag")
				return
			}
		} else {
			log.Infof("send validation links to %d members, skipped %d invalid IDs and %d errors , for Entity %x\nErrors: %v", response.Count, len(response.InvalidIDs), len(errors), entityID, errors)
			log.Errorf("error retreiving Pending tag:  %v", err)
			m.Router.SendError(request, "sent emails but could not assign tag")
			return
		}
	}
	_, _, err = m.db.AddTagToMembers(entityID, successUUIDs, tag.ID)
	if err != nil {
		log.Infof("send validation links to %d members, skipped %d invalid IDs and %d errors , for Entity %x\nErrors: %v", response.Count, len(response.InvalidIDs), len(errors), entityID, errors)
		log.Errorf("error assinging Pending tag:  %v", err)
		m.Router.SendError(request, "sent emails but could not assign tag")
	}

	log.Infof("send validation links to %d members, skipped %d invalid IDs and %d errors , for Entity %x\nErrors: %v", response.Count, len(response.InvalidIDs), len(errors), entityID, errors)
	m.send(&request, &response)
}

func (m *Manager) addCensus(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	if len(request.TargetID) == 0 {
		log.Debugf("invalid target id %q for %x", request.TargetID.String(), request.SignaturePublicKey)
		m.Router.SendError(request, "invalid target id")
		return
	}

	if len(request.CensusID) == 0 {
		log.Debugf("invalid census id %q for %x", request.CensusID, request.SignaturePublicKey)
		m.Router.SendError(request, "invalid census id")
		return
	}

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey)
	if err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot recover entityID")
		return
	}

	var censusID []byte
	if censusID, err = util.DecodeCensusID(request.CensusID, request.SignaturePublicKey); err != nil {
		m.Router.SendError(request, err.Error())
		return
	}

	// size, err := m.db.AddCensusWithMembers(entityID, censusID, request.TargetID, request.Census)
	if err := m.db.AddCensus(entityID, censusID, request.TargetID, request.Census); err != nil {
		log.Errorf("cannot add census %q  for: %q: (%v)", request.CensusID, entityID, err)
		m.Router.SendError(request, "cannot add census")
		return
	}

	log.Debugf("Entity: %x addCensus: %s  ", entityID, request.CensusID)
	m.send(&request, &response)
}

func (m *Manager) updateCensus(request router.RouterRequest) {
	// TODO Handle invalid claims
	var entityID []byte
	var err error
	var response types.MetaResponse

	if len(request.CensusID) == 0 {
		log.Debugf("invalid census id %q for %x", request.CensusID, request.SignaturePublicKey)
		m.Router.SendError(request, "invalid census id")
		return
	}

	if request.Census == nil {
		log.Debugf("invalid census info for census %q for entity %x", request.CensusID, request.SignaturePublicKey)
		m.Router.SendError(request, "invalid census info")
		return
	}

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey)
	if err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot recover entityID")
		return
	}

	var censusID []byte
	if censusID, err = util.DecodeCensusID(request.CensusID, request.SignaturePublicKey); err != nil {
		m.Router.SendError(request, err.Error())
		return
	}

	if request.InvalidClaims != nil && len(request.InvalidClaims) > 0 {
		log.Warnf("invalid claims: %v", request.InvalidClaims)
	}

	if response.Count, err = m.db.UpdateCensus(entityID, censusID, request.Census); err != nil {
		log.Errorf("cannot update census %q for %x: (%v)", request.CensusID, entityID, err)
		m.Router.SendError(request, "cannot update census")
		return
	}

	log.Debugf("Entity: %x updateCensus: %s \n %v", entityID, request.CensusID, request.Census)
	m.send(&request, &response)
}

func (m *Manager) getCensus(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	if len(request.CensusID) == 0 {
		log.Debugf("invalid census id %q for %s", request.CensusID, request.SignaturePublicKey)
		m.Router.SendError(request, "invalid census id")
		return
	}

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey)
	if err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot recover entityID")
		return
	}

	var censusID []byte
	if censusID, err = util.DecodeCensusID(request.CensusID, request.SignaturePublicKey); err != nil {
		log.Errorf("cannot decode census id %s for %x", request.CensusID, request.SignaturePublicKey)
		m.Router.SendError(request, "cannot decode census id")
		return
	}

	response.Census, err = m.db.Census(entityID, censusID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Debugf("census %q not found for %x", request.CensusID, request.SignaturePublicKey)
			m.Router.SendError(request, "census not found")
			return
		}
		log.Errorf("error in retrieving censuses for entity %x: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot query for censuses")
		return
	}

	response.Target, err = m.db.Target(entityID, &response.Census.TargetID)
	if err != nil {
		log.Warn("census target not found")
		m.Router.SendError(request, "census target not found")
		return
	}

	log.Debugf("Entity: %x getCensus:%s", request.SignaturePublicKey, request.CensusID)
	m.send(&request, &response)
}

func (m *Manager) countCensus(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot recover entityID")
		return
	}

	// Query for members
	if response.Count, err = m.db.CountCensus(entityID); err != nil {
		log.Errorf("cannot count censuses for %x: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot count censuses")
		return
	}

	log.Debugf("Entity %x countCensus: %d censuses", request.SignaturePublicKey, response.Count)
	m.send(&request, &response)
}

func (m *Manager) listCensus(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey)
	if err != nil {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		m.Router.SendError(request, "invalid public key")
		return
	}

	// check filter
	if err := checkOptions(request.ListOptions, request.Method); err != nil {
		log.Warnf("invalid filter options %x: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "invalid filter options")
		return
	}

	// Query for members
	// TODO Implement listCensus in Db that supports filters
	response.Censuses, err = m.db.ListCensus(entityID, request.ListOptions)
	if err != nil {
		if err == sql.ErrNoRows {
			m.Router.SendError(request, "no censuses found")
			return
		}
		log.Errorf("error in retrieving censuses for entity %x: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot query for censuses")
		return
	}
	log.Debugf("Entity: %x listCensuses: %d censuses", request.SignaturePublicKey, len(response.Censuses))
	m.send(&request, &response)
}

func (m *Manager) deleteCensus(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	if len(request.CensusID) == 0 {
		log.Debugf("invalid census id %q for %x", request.CensusID, request.SignaturePublicKey)
		m.Router.SendError(request, "invalid census id")
		return
	}

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey)
	if err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot recover entityID")
		return
	}

	var censusID []byte
	if censusID, err = util.DecodeCensusID(request.CensusID, request.SignaturePublicKey); err != nil {
		log.Errorf("cannot decode census id %x for %s", request.CensusID, request.SignaturePublicKey)
		m.Router.SendError(request, "cannot decode census id")
		return
	}

	err = m.db.DeleteCensus(entityID, censusID)
	if err != nil && err != sql.ErrNoRows {
		log.Errorf("error deleting census %s for entity %x: (%v)", request.CensusID, entityID, err)
		m.Router.SendError(request, "cannot delete census")
		return
	}

	log.Debugf("Entity: %x deleteCensus:%s", entityID, request.CensusID)
	m.send(&request, &response)
}

func (m *Manager) sendValidationLinks(request router.RouterRequest) {

	if len(request.MemberID) == 0 {
		m.Router.SendError(request, "invalid arguments")
		return
	}

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	entityID, err := util.PubKeyToEntityID(request.SignaturePublicKey)
	if err != nil {
		log.Errorf("cannot recover %x entityID from public key: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot recover entityID from public key")
		return
	}

	entity, err := m.db.Entity(entityID)
	if err != nil {
		log.Errorf("cannot recover %x entity: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot recover entity from public key")
		return
	}

	var response types.MetaResponse
	var members []types.Member
	members, response.InvalidIDs, err = m.db.Members(entityID, request.MemberIDs)
	if err != nil {
		log.Errorf("cannot retrieve members for entity %x: (%v)", entityID, err)
		m.Router.SendError(request, "cannot retrieve member")
		return
	}

	if len(members) == 0 {
		response.Count = 0
		m.send(&request, &response)
	}
	// send concurrently emails
	var wg sync.WaitGroup
	wg.Add(len(members))
	ec := make(chan error, len(members))
	sc := make(chan uuid.UUID, len(members))
	for _, member := range members {
		go func(member types.Member) {
			if member.PubKey != nil {
				log.Errorf("member %s is already validated at  %s", member.ID.String(), member.Verified)
				ec <- fmt.Errorf("member %s is already validated at  %s", member.ID.String(), member.Verified)
				wg.Done()
				return
			}
			if err := m.smtp.SendValidationLink(&member, entity); err != nil {
				log.Errorf("could not send validation link for member %q entity: (%v)", member.ID, err)
				ec <- fmt.Errorf("member %s error  %v", member.ID, err)
				wg.Done()
				return
			}
			sc <- member.ID
			wg.Done()
		}(member)
	}
	wg.Wait()
	close(ec)
	close(sc)
	// get results
	var successUUIDs []uuid.UUID
	for uid := range sc {
		successUUIDs = append(successUUIDs, uid)
	}
	response.Count = int(len(successUUIDs))
	var errors []error
	for err := range ec {
		errors = append(errors, err)
	}
	if len(errors)+response.Count != len(members) {
		log.Errorf("inconsistency in number of sent emails and errors")
		m.Router.SendError(request, "inconsistency in number of sent emails and errors")
	}
	if len(errors) == len(members) {
		log.Errorf("no validation email was sent %v", errors)
		m.Router.SendError(request, "could not send emails")
		return
	}
	if len(errors) > 0 {
		response.Message = fmt.Sprintf("%d where found:\n%v", len(errors), errors)
	}
	duplicates := len(request.MemberIDs) - len(members) - len(response.InvalidIDs)

	// add tag PendingValidation to sucessful members
	tagName := "PendingValidation"
	tag, err := m.db.TagByName(entityID, tagName)
	if err != nil {
		if err == sql.ErrNoRows {
			tag = &types.Tag{}
			tag.ID, err = m.db.AddTag(entityID, tagName)
			if err != nil {
				log.Infof("send validation links to %d members, skipped %d invalid IDs, %d duplicates and %d errors , for Entity %x\nErrors: %v", response.Count, len(response.InvalidIDs), duplicates, len(errors), entityID, errors)
				log.Errorf("error creating Pending tag:  %v", err)
				m.Router.SendError(request, "sent emails but could not assign tag")
				return
			}
		} else {
			log.Infof("send validation links to %d members, skipped %d invalid IDs, %d duplicates and %d errors , for Entity %x\nErrors: %v", response.Count, len(response.InvalidIDs), duplicates, len(errors), entityID, errors)
			log.Errorf("error retreiving Pending tag:  %v", err)
			m.Router.SendError(request, "sent emails but could not assign tag")
			return
		}
	}
	_, _, err = m.db.AddTagToMembers(entityID, successUUIDs, tag.ID)
	if err != nil {
		log.Infof("send validation links to %d members, skipped %d invalid IDs, %d duplicates and %d errors , for Entity %x\nErrors: %v", response.Count, len(response.InvalidIDs), duplicates, len(errors), entityID, errors)
		log.Errorf("error assinging Pending tag:  %v", err)
		m.Router.SendError(request, "sent emails but could not assign tag")
	}

	log.Infof("send validation links to %d members, skipped %d invalid IDs, %d duplicates and %d errors , for Entity %x\nErrors: %v", response.Count, len(response.InvalidIDs), duplicates, len(errors), entityID, errors)
	m.send(&request, &response)
}

func (m *Manager) createTag(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	if request.TagName == "" {
		log.Debug("createTag with empty tag")
		m.Router.SendError(request, "invalid tag name")
		return
	}

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot recover entityID")
		return
	}

	response.Tag = &types.Tag{
		Name: request.TagName,
	}

	if response.Tag.ID, err = m.db.AddTag(entityID, request.TagName); err != nil {
		log.Errorf("cannot create tag '%s' for entity %x: (%v)", request.TagName, entityID, err)
		m.Router.SendError(request, "cannot create tag ")
		return
	}

	log.Infof("created tag with id %d and name '%s' for Entity %x", response.Tag.ID, response.Tag.Name, entityID)
	m.send(&request, &response)
}

func (m *Manager) listTags(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot recover entityID")
		return
	}

	// Query for members
	if response.Tags, err = m.db.ListTags(entityID); err != nil {
		if err == sql.ErrNoRows {
			m.Router.SendError(request, "no tags found")
			return
		}
		log.Errorf("cannot retrieve tags of %x: (%v)", entityID, err)
		m.Router.SendError(request, "cannot retrieve tags")
		return
	}

	log.Debugf("Entity: %x list %d tags", entityID, len(response.Tags))
	m.send(&request, &response)
}

func (m *Manager) deleteTag(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	if request.TagID == 0 {
		log.Debug("deleteTag with empty tag")
		m.Router.SendError(request, "invalid tag id")
		return
	}

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot recover entityID")
		return
	}

	if err = m.db.DeleteTag(entityID, request.TagID); err != nil {
		log.Errorf("cannot delete tag %d for entity %x: (%v)", request.TagID, entityID, err)
		m.Router.SendError(request, "cannot delete tag ")
		return
	}

	log.Infof("delete tag with id %d for Entity %x", request.TagID, entityID)
	m.send(&request, &response)
}

func (m *Manager) addTag(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	if request.TagID == 0 || len(request.MemberIDs) == 0 {
		log.Debug("addTag invalid arguments")
		m.Router.SendError(request, "invalid arguments")
		return
	}

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot recover entityID")
		return
	}

	response.Count, response.InvalidIDs, err = m.db.AddTagToMembers(entityID, request.MemberIDs, request.TagID)
	if err != nil {
		log.Errorf("cannot add tag %d to members for entity %x: (%v)", request.TagID, entityID, err)
		m.Router.SendError(request, "cannot add tag ")
		return
	}

	log.Infof("added tag with id %d to %d, with %d invalid IDs, members of Entity %x", request.TagID, response.Count, len(response.InvalidIDs), entityID)
	m.send(&request, &response)
}

func (m *Manager) removeTag(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	if request.TagID == 0 || len(request.MemberIDs) == 0 {
		log.Debug("removeTag invalid arguments")
		m.Router.SendError(request, "invalid arguments")
		return
	}

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot recover entityID")
		return
	}

	response.Count, response.InvalidIDs, err = m.db.RemoveTagFromMembers(entityID, request.MemberIDs, request.TagID)
	if err != nil {
		log.Errorf("cannot remove tag %d from members for entity %x: (%v)", request.TagID, entityID, err)
		m.Router.SendError(request, "cannot remove tag ")
		return
	}

	log.Infof("removed tag with id %d from %d members, with %d invalid IDs, of Entity %x", request.TagID, response.Count, len(response.InvalidIDs), entityID)
	m.send(&request, &response)
}

func (m *Manager) requestGas(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	if m.faucet == nil {
		log.Errorf("cannot request for tokens, no faucet found")
		m.Router.SendError(request, "internal error")
		return
	}
	// check public key length
	// dvoteutil.IsHexEncodedStringWithLength
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %s", request.SignaturePublicKey)
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %q entityID: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot recover entityID")
		return
	}

	entityAddress := ethcommon.BytesToAddress(entityID)

	// check entity exists
	if _, err := m.db.Entity(entityID); err != nil {
		if err == sql.ErrNoRows {
			log.Errorf("entity not found")
			m.Router.SendError(request, "entity not found")
			return
		}
		log.Errorf("cannot retrieve details of entity %x: (%v)", entityID, err)
		m.Router.SendError(request, "cannot retrieve entity")
		return
	}

	sent, err := m.faucet.SendTokens(context.Background(), entityAddress)
	if err != nil {
		log.Errorf("error sending tokens to entity %s : %v", entityAddress.String(), err)
		m.Router.SendError(request, "error sending tokens")
		return
	}

	response.Count = int(sent.Int64())
	m.send(&request, &response)
}

func checkOptions(filter *types.ListOptions, method string) error {
	if filter == nil {
		return nil
	}
	// Check skip and count
	if filter.Skip < 0 || filter.Count < 0 {
		return fmt.Errorf("invalid skip/count")
	}
	var t reflect.Type
	// check method
	switch method {
	case "listMembers":
		t = reflect.TypeOf(types.MemberInfo{})
	case "listCensus":
		t = reflect.TypeOf(types.CensusInfo{})
	default:
		return fmt.Errorf("invalid method")
	}
	// Check sortby
	if len(filter.SortBy) > 0 {
		_, found := t.FieldByName(strings.Title(filter.SortBy))
		if !found {
			return fmt.Errorf("invalid filter field")
		}
		// sqli guard
		protectedOrderField := pgsql.ToOrderBySQLi(filter.SortBy)
		if protectedOrderField == -1 {
			return fmt.Errorf("invalid sort by field on query: %s", filter.SortBy)
		}
		// Check order
		if len(filter.Order) > 0 && !(filter.Order == "ascend" || filter.Order == "descend") {
			return fmt.Errorf("invalid filter order")
		}

	} else if len(filter.Order) > 0 {
		// Also check that order does not make sense without sortby
		return fmt.Errorf("invalid filter order")
	}
	return nil
}
