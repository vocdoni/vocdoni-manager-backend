package manager

import (
	"database/sql"
	"encoding/hex"
	"encoding/json"

	"fmt"
	"reflect"
	"strings"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"

	"gitlab.com/vocdoni/go-dvote/crypto/ethereum"
	"gitlab.com/vocdoni/go-dvote/log"
	dvoteUtil "gitlab.com/vocdoni/go-dvote/util"
	"gitlab.com/vocdoni/manager/manager-backend/database"
	"gitlab.com/vocdoni/manager/manager-backend/database/pgsql"
	"gitlab.com/vocdoni/manager/manager-backend/router"
	"gitlab.com/vocdoni/manager/manager-backend/types"
	"gitlab.com/vocdoni/manager/manager-backend/util"
)

type Manager struct {
	Router *router.Router
	db     database.Database
}

// NewManager creates a new registry handler for the Router
func NewManager(r *router.Router, d database.Database) *Manager {
	return &Manager{Router: r, db: d}
}

// RegisterMethods registers all registry methods behind the given path
func (m *Manager) RegisterMethods(path string) error {
	m.Router.Transport.AddNamespace(path + "/manager")
	if err := m.Router.AddHandler("signUp", path+"/manager", m.signUp, false, false); err != nil {
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
	if err := m.Router.AddHandler("deleteMember", path+"/manager", m.deleteMember, false, false); err != nil {
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
	if err := m.Router.AddHandler("addCensus", path+"/manager", m.addCensus, false, false); err != nil {
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
	return nil
}

func (m *Manager) send(req router.RouterRequest, resp types.MetaResponse) {
	m.Router.Transport.Send(m.Router.BuildReply(req, resp))
}

func (m *Manager) signUp(request router.RouterRequest) {
	var entityID []byte
	var entityInfo *types.EntityInfo
	var entityAddress ethcommon.Address
	var target *types.Target
	var err error
	var response types.MetaResponse

	// check public key length
	// dvoteUtil.IsHexEncodedStringWithLength
	if len(request.SignaturePublicKey) != ethereum.PubKeyLength && len(request.SignaturePublicKey) != ethereum.PubKeyLengthUncompressed {
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

	// retrieve entity Address
	if entityAddress, err = util.PubKeyToAddress(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover entity %q address: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot recover entity address")
		return
	}
	// TODO: Receive from API census Managers addresses during signUp
	entityAddressBytes, err := hex.DecodeString(dvoteUtil.TrimHex(entityAddress.String()))
	if err != nil {
		log.Errorf("cannot decode entity address: %s", err)
		m.Router.SendError(request, "cannot add entity to the DB")
	}
	entityInfo = &types.EntityInfo{Address: entityAddressBytes, CensusManagersAddresses: [][]byte{entityAddressBytes}, Origins: []types.Origin{types.Token}}
	// Add Entity
	if err = m.db.AddEntity(entityID, entityInfo); err != nil {
		log.Errorf("cannot add entity %q to the DB: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot add entity to the DB")
		return
	}

	target = &types.Target{EntityID: entityID, Name: "all", Filters: json.RawMessage([]byte("{}"))}
	if _, err = m.db.AddTarget(entityID, target); err != nil {
		log.Errorf("cannot create entity's %q generic target: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot create entity generic target")
		return
	}

	log.Debugf("Entity: %q signUp", request.SignaturePublicKey)
	m.send(request, response)
}

func (m *Manager) listMembers(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLength && len(request.SignaturePublicKey) != ethereum.PubKeyLengthUncompressed {
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

	// check filter
	if err = checkOptions(request.ListOptions, request.Method); err != nil {
		log.Warnf("invalid filter options %q: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "invalid filter options")
		return
	}

	// Query for members
	if response.Members, err = m.db.ListMembers(entityID, request.ListOptions); err != nil {
		if err == sql.ErrNoRows {
			m.Router.SendError(request, "no members found")
			return
		}
		log.Errorf("cannot retrieve members of %q: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot retrieve members")
		return
	}

	log.Debugf("Entity: %q listMembers", request.SignaturePublicKey)
	m.send(request, response)
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
	if len(request.SignaturePublicKey) != ethereum.PubKeyLength && len(request.SignaturePublicKey) != ethereum.PubKeyLengthUncompressed {
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

	if response.Member, err = m.db.Member(entityID, request.MemberID); err != nil {
		if err == sql.ErrNoRows {
			log.Warn("member not found")
			m.Router.SendError(request, "member not found")
			return
		}
		log.Errorf("cannot retrieve member %q for entity %q: (%v)", request.MemberID, request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot retrieve member")
		return
	}

	// TODO: Change when targets are implemented
	var targets []types.Target
	targets, err = m.db.ListTargets(entityID)
	if err == sql.ErrNoRows || len(targets) == 0 {
		log.Warnf("no targets found for member %q of entity %s", request.MemberID, request.SignaturePublicKey)
		response.Target = &types.Target{}
	} else if err == nil {
		response.Target = &targets[0]
	} else {
		log.Errorf("error retrieving member %q targets for entity %q: (%v)", request.MemberID, request.SignaturePublicKey, err)
		m.Router.SendError(request, "error retrieving member targets")
		return
	}

	log.Infof("listing member %q for Entity with public Key %s", request.MemberID.String(), request.SignaturePublicKey)
	m.send(request, response)
}

func (m *Manager) updateMember(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	if request.Member == nil {
		m.Router.SendError(request, "invalid member struct")
	}

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLength && len(request.SignaturePublicKey) != ethereum.PubKeyLengthUncompressed {
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %q entityID: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot recover entityID")
		return
	}

	// If a string Member property is sent as "" then it is not updated
	if err = m.db.UpdateMember(entityID, &request.Member.ID, &request.Member.MemberInfo); err != nil {
		log.Errorf("cannot update member %q for entity %q: (%v)", request.Member.ID.String(), request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot update member")
		return
	}

	log.Infof("update member %q for Entity with public Key %s", request.Member.ID.String(), request.SignaturePublicKey)
	m.send(request, response)
}

func (m *Manager) deleteMember(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	if request.MemberID == nil || *request.MemberID == uuid.Nil {
		m.Router.SendError(request, "invalid member ID")
		return
	}

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLength && len(request.SignaturePublicKey) != ethereum.PubKeyLengthUncompressed {
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %q entityID: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot recover entityID")
		return
	}

	if err = m.db.DeleteMember(entityID, request.MemberID); err != nil {
		log.Errorf("cannot delete member %q for entity %q: (%v)", request.MemberID, request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot delete member")
		return
	}

	log.Infof("deleted member %q for Entity with public Key %s", request.MemberID.String(), request.SignaturePublicKey)
	m.send(request, response)
}

func (m *Manager) countMembers(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLength && len(request.SignaturePublicKey) != ethereum.PubKeyLengthUncompressed {
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %q entityID: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot recover entityID")
		return
	}

	// Query for members
	if response.Count, err = m.db.CountMembers(entityID); err != nil {
		log.Errorf("cannot count members for %q: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot count members")
		return
	}

	log.Debugf("Entity %q countMembers: %d members", request.SignaturePublicKey, response.Count)
	m.send(request, response)
}

func (m *Manager) generateTokens(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLength && len(request.SignaturePublicKey) != ethereum.PubKeyLengthUncompressed {
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

	if request.Amount < 1 {
		log.Warnf("invalid token amount requested by %s", request.SignaturePublicKey)
		m.Router.SendError(request, "invalid token amount")
		return
	}

	response.Tokens = make([]uuid.UUID, request.Amount)
	for idx := range response.Tokens {
		response.Tokens[idx] = uuid.New()
	}
	// TODO: Probably I need to initialize tokens
	if err = m.db.CreateMembersWithTokens(entityID, response.Tokens); err != nil {
		log.Errorf("could not register generated tokens for %q: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "could not register generated tokens")
		return
	}

	log.Debugf("Entity: %q generateTokens: %d tokens", request.SignaturePublicKey, len(response.Tokens))
	m.send(request, response)
}

func (m *Manager) exportTokens(request router.RouterRequest) {
	var entityID []byte
	var members []types.Member
	var err error
	var response types.MetaResponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLength && len(request.SignaturePublicKey) != ethereum.PubKeyLengthUncompressed {
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

	// TODO: Probably I need to initialize tokens
	if members, err = m.db.MembersTokensEmails(entityID); err != nil {
		if err == sql.ErrNoRows {
			m.Router.SendError(request, "no members found")
			return
		}
		log.Errorf("could not retrieve members tokens for %q: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, err.Error())
		return
	}
	response.MembersTokens = make([]types.TokenEmail, len(members))
	for idx, member := range members {
		response.MembersTokens[idx] = types.TokenEmail{Token: member.ID, Email: member.Email}
	}

	log.Debugf("Entity: %q exportTokens: %d tokens", request.SignaturePublicKey, len(members))
	m.send(request, response)
}

func (m *Manager) importMembers(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLength && len(request.SignaturePublicKey) != ethereum.PubKeyLengthUncompressed {
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

	if len(request.MembersInfo) < 1 {
		log.Warnf("no member data provided for import members by %s", request.SignaturePublicKey)
		m.Router.SendError(request, "no member data provided")
		return
	}

	for idx := range request.MembersInfo {
		request.MembersInfo[idx].Origin = types.Token
	}

	// Add members
	if err = m.db.ImportMembers(entityID, request.MembersInfo); err != nil {
		log.Errorf("could not import members for %q: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, err.Error())
		return
	}

	log.Debugf("Entity: %q importMembers: %d members", request.SignaturePublicKey, len(request.MembersInfo))
	m.send(request, response)
}

func (m *Manager) countTargets(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLength && len(request.SignaturePublicKey) != ethereum.PubKeyLengthUncompressed {
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %q entityID: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot recover entityID")
		return
	}

	// Query for members
	if response.Count, err = m.db.CountTargets(entityID); err != nil {
		log.Errorf("cannot count targets for %q: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot count targets")
		return
	}

	log.Debugf("Entity %q countTargets: %d targets", request.SignaturePublicKey, response.Count)
	m.send(request, response)
}

func (m *Manager) listTargets(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLength && len(request.SignaturePublicKey) != ethereum.PubKeyLengthUncompressed {
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

	// check filter
	if err = checkOptions(request.ListOptions, request.Method); err != nil {
		log.Warnf("invalid filter options %q: (%v)", request.SignaturePublicKey, err)
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
		log.Errorf("cannot query targets for %q: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot query targets")
		return
	}

	log.Debugf("Entity: %q listTargets: %d targets", request.SignaturePublicKey, len(response.Targets))
	m.send(request, response)
}

func (m *Manager) getTarget(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLength && len(request.SignaturePublicKey) != ethereum.PubKeyLengthUncompressed {
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

	if response.Target, err = m.db.Target(entityID, request.TargetID); err != nil {
		if err == sql.ErrNoRows {
			log.Debugf("target %q not found for %s", request.TargetID, request.SignaturePublicKey)
			m.Router.SendError(request, "target not found")
			return
		}
		log.Errorf("could not retrieve target for %q: %+v", request.SignaturePublicKey, err)
		m.Router.SendError(request, "could not retrieve target")
		return
	}

	response.Members, err = m.db.TargetMembers(entityID, request.TargetID)
	if err != nil {
		log.Warn("members for requested target could not be retrieved")
		m.Router.SendError(request, "members for requested target could not be retrieved")
		return
	}

	log.Debugf("Entity: %q getTarget: %s", request.SignaturePublicKey, request.TargetID.String())
	m.send(request, response)
}

func (m *Manager) dumpTarget(request router.RouterRequest) {
	var target *types.Target
	var entityID []byte
	var err error
	var response types.MetaResponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLength && len(request.SignaturePublicKey) != ethereum.PubKeyLengthUncompressed {
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

	if target, err = m.db.Target(entityID, request.TargetID); err != nil || target.Name != "all" {
		if err == sql.ErrNoRows {
			log.Debugf("target %q not found for %s", request.TargetID, request.SignaturePublicKey)
			m.Router.SendError(request, "target not found")
			return
		}
		log.Errorf("could not retrieve target for %q: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "could not retrieve target")
		return
	}

	// TODO: Implement DumpTargetClaims filtered directly by target filters
	if response.Claims, err = m.db.DumpClaims(entityID); err != nil {
		if err == sql.ErrNoRows {
			log.Debugf("no claims found for %s", request.SignaturePublicKey)
			m.Router.SendError(request, "no claims found")
			return
		}
		log.Errorf("cannot dump claims for %q: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot dump claims")
		return
	}

	log.Debugf("Entity: %q dumpTarget: %d claims", request.SignaturePublicKey, len(response.Claims))
	m.send(request, response)
}

func (m *Manager) addCensus(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	if len(request.TargetID) == 0 {
		log.Debugf("invalid target id %q for %s", request.TargetID, request.SignaturePublicKey)
		m.Router.SendError(request, "invalid target id")
		return
	}

	if len(request.CensusID) == 0 {
		log.Debugf("invalid census id %q for %s", request.CensusID, request.SignaturePublicKey)
		m.Router.SendError(request, "invalid census id")
		return
	}

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLength && len(request.SignaturePublicKey) != ethereum.PubKeyLengthUncompressed {
		log.Warnf("invalid public key: %s", request.SignaturePublicKey)
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey)
	if err != nil {
		log.Errorf("cannot recover %q entityID: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot recover entityID")
		return
	}

	var censusID []byte
	if censusID, err = util.DecodeCensusID(request.CensusID, request.SignaturePublicKey); err != nil {
		m.Router.SendError(request, err.Error())
		return
	}

	size, err := m.db.AddCensusWithMembers(entityID, censusID, request.TargetID, request.Census)
	if err != nil {
		log.Errorf("cannot add census %q  target members for: %q: (%v)", request.CensusID, request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot add census members")
		return
	}

	log.Debugf("Entity: %q addCensus: %s  %d members", request.SignaturePublicKey, request.CensusID, size)
	log.Infof("addCensus")
	m.send(request, response)
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
	if len(request.SignaturePublicKey) != ethereum.PubKeyLength && len(request.SignaturePublicKey) != ethereum.PubKeyLengthUncompressed {
		log.Warnf("invalid public key: %s", request.SignaturePublicKey)
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey)
	if err != nil {
		log.Errorf("cannot recover %q entityID: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot recover entityID")
		return
	}

	var censusID []byte
	if censusID, err = util.DecodeCensusID(request.CensusID, request.SignaturePublicKey); err != nil {
		log.Errorf("cannot decode census id %s for %s", request.CensusID, request.SignaturePublicKey)
		m.Router.SendError(request, "cannot decode census id")
		return
	}

	response.Census, err = m.db.Census(entityID, censusID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Debugf("census %q not found for %s", request.CensusID, request.SignaturePublicKey)
			m.Router.SendError(request, "census not found")
			return
		}
		log.Errorf("error in retrieving censuses for entity %q: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot query for censuses")
		return
	}

	response.Target, err = m.db.Target(entityID, &response.Census.TargetID)
	if err != nil {
		log.Warn("census target not found")
		m.Router.SendError(request, "census target not found")
		return
	}

	log.Debugf("Entity: %q getCensus:%s", request.SignaturePublicKey, request.CensusID)
	m.send(request, response)
}

func (m *Manager) countCensus(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLength && len(request.SignaturePublicKey) != ethereum.PubKeyLengthUncompressed {
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %q entityID: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot recover entityID")
		return
	}

	// Query for members
	if response.Count, err = m.db.CountCensus(entityID); err != nil {
		log.Errorf("cannot count censuses for %q: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot count censuses")
		return
	}

	log.Debugf("Entity %q countCensus: %d censuses", request.SignaturePublicKey, response.Count)
	m.send(request, response)
}

func (m *Manager) listCensus(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLength && len(request.SignaturePublicKey) != ethereum.PubKeyLengthUncompressed {
		log.Warnf("invalid public key: %s", request.SignaturePublicKey)
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey)
	if err != nil {
		log.Warnf("invalid public key: %s", request.SignaturePublicKey)
		m.Router.SendError(request, "invalid public key")
		return
	}

	// check filter
	if err := checkOptions(request.ListOptions, request.Method); err != nil {
		log.Warnf("invalid filter options %q: (%v)", request.SignaturePublicKey, err)
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
		log.Errorf("error in retrieving censuses for entity %q: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot query for censuses")
		return
	}
	log.Debugf("Entity: %q listCensuses: %d censuses", request.SignaturePublicKey, len(response.Censuses))
	m.send(request, response)
}

func (m *Manager) deleteCensus(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	if len(request.CensusID) == 0 {
		log.Debugf("invalid census id %q for %s", request.CensusID, request.SignaturePublicKey)
		m.Router.SendError(request, "invalid census id")
		return
	}

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLength && len(request.SignaturePublicKey) != ethereum.PubKeyLengthUncompressed {
		log.Warnf("invalid public key: %s", request.SignaturePublicKey)
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey)
	if err != nil {
		log.Errorf("cannot recover %q entityID: (%v)", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot recover entityID")
		return
	}

	var censusID []byte
	if censusID, err = util.DecodeCensusID(request.CensusID, request.SignaturePublicKey); err != nil {
		log.Errorf("cannot decode census id %q for %s", request.CensusID, request.SignaturePublicKey)
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
	m.send(request, response)
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
