package manager

import (
	"database/sql"
	"encoding/json"

	"fmt"
	"reflect"
	"strings"

	"github.com/google/uuid"
	"gitlab.com/vocdoni/go-dvote/crypto/ethereum"
	"gitlab.com/vocdoni/go-dvote/log"
	"gitlab.com/vocdoni/vocdoni-manager-backend/database"
	"gitlab.com/vocdoni/vocdoni-manager-backend/router"
	"gitlab.com/vocdoni/vocdoni-manager-backend/types"
	"gitlab.com/vocdoni/vocdoni-manager-backend/util"
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
	if err := m.Router.AddHandler("signUp", path+"/manager", m.signUp, false); err != nil {
		return err
	}
	if err := m.Router.AddHandler("countMembers", path+"/manager", m.countMembers, false); err != nil {
		return err
	}
	if err := m.Router.AddHandler("listMembers", path+"/manager", m.listMembers, false); err != nil {
		return err
	}
	if err := m.Router.AddHandler("getMember", path+"/manager", m.getMember, false); err != nil {
		return err
	}
	if err := m.Router.AddHandler("updateMember", path+"/manager", m.updateMember, false); err != nil {
		return err
	}
	if err := m.Router.AddHandler("deleteMember", path+"/manager", m.deleteMember, false); err != nil {
		return err
	}
	if err := m.Router.AddHandler("generateTokens", path+"/manager", m.generateTokens, false); err != nil {
		return err
	}
	if err := m.Router.AddHandler("exportTokens", path+"/manager", m.exportTokens, false); err != nil {
		return err
	}
	if err := m.Router.AddHandler("importMembers", path+"/manager", m.importMembers, false); err != nil {
		return err
	}
	if err := m.Router.AddHandler("countTargets", path+"/manager", m.countTargets, false); err != nil {
		return err
	}
	if err := m.Router.AddHandler("listTargets", path+"/manager", m.listTargets, false); err != nil {
		return err
	}
	if err := m.Router.AddHandler("getTarget", path+"/manager", m.getTarget, false); err != nil {
		return err
	}
	if err := m.Router.AddHandler("dumpTarget", path+"/manager", m.dumpTarget, false); err != nil {
		return err
	}
	if err := m.Router.AddHandler("addCensus", path+"/manager", m.addCensus, false); err != nil {
		return err
	}
	if err := m.Router.AddHandler("getCensus", path+"/manager", m.getCensus, false); err != nil {
		return err
	}
	if err := m.Router.AddHandler("countCensus", path+"/manager", m.countCensus, false); err != nil {
		return err
	}
	if err := m.Router.AddHandler("listCensus", path+"/manager", m.listCensus, false); err != nil {
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
	var entityAddress []byte
	var target *types.Target
	var err error
	var response types.MetaResponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLength {
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Warn(err)
		m.Router.SendError(request, err.Error())
		return
	}

	// retrieve entity Address
	if entityAddress, err = util.PubKeyToAddress(request.SignaturePublicKey); err != nil {
		log.Error(err)
		m.Router.SendError(request, err.Error())
		return
	}
	// TODO: Receive from API census Managers addresses during signUp
	entityInfo = &types.EntityInfo{Address: entityAddress, CensusManagersAddresses: [][]byte{entityAddress}, Origins: []types.Origin{types.Token}}
	// Add Entity
	if err = m.db.AddEntity(entityID, entityInfo); err != nil {
		log.Error(err)
		m.Router.SendError(request, err.Error())
		return
	}

	target = &types.Target{EntityID: entityID, Name: "all", Filters: json.RawMessage([]byte("{}"))}
	if _, err = m.db.AddTarget(entityID, target); err != nil {
		log.Error("error creating entities generic target")
		m.Router.SendError(request, "error creating entities generic target")
		return
	}

	log.Infof("Added Entity with public Key %s", request.SignaturePublicKey)
	m.send(request, response)
}

func (m *Manager) listMembers(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLength {
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Warn(err)
		m.Router.SendError(request, err.Error())
		return
	}

	// check filter
	if err = checkOptions(request.ListOptions); err != nil {
		log.Warn(err)
		m.Router.SendError(request, err.Error())
		return
	}

	// Query for members
	if response.Members, err = m.db.ListMembers(entityID, request.ListOptions); err != nil {
		if err == sql.ErrNoRows {
			m.Router.SendError(request, "no members found")
			return
		}
		log.Error(err)
		m.Router.SendError(request, "cannot query for members")
		return
	}

	log.Info("listMembers")
	m.send(request, response)
}

func (m *Manager) getMember(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLength {
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Warn(err)
		m.Router.SendError(request, err.Error())
		return
	}

	if response.Member, err = m.db.Member(entityID, request.MemberID); err != nil {
		if err == sql.ErrNoRows {

			log.Warn("member not found")
			m.Router.SendError(request, "member not found")
			return
		}
		log.Error("cannot retrieve member %s for entity %s : %+v", request.MemberID, request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot retrieve member")
		return
	}

	// TODO: Change when targets are implemented
	var targets []types.Target
	targets, err = m.db.ListTargets(entityID)
	if err == sql.ErrNoRows || len(targets) == 0 {
		log.Warn("no targets found for member %s of entity %s", request.MemberID, request.SignaturePublicKey)
		response.Target = &types.Target{}
	} else if err == nil {
		log.Warn("Hi")
		response.Target = &targets[0]
	} else {
		log.Errorf("error retrieving member %s targets for entity %s : %+v", request.MemberID, request.SignaturePublicKey, err)
		m.Router.SendError(request, "error retrieving member targets")
		return
	}

	log.Infof("listing member %s for Entity with public Key %s", request.MemberID.String(), request.SignaturePublicKey)
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
	if len(request.SignaturePublicKey) != ethereum.PubKeyLength {
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Warn(err)
		m.Router.SendError(request, err.Error())
		return
	}

	// If a string Member property is sent as "" then it is not updated
	if err = m.db.UpdateMember(entityID, request.Member.ID, &request.Member.MemberInfo); err != nil {
		log.Error("cannot update member %s for entity %s : %+v", request.Member.ID, request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot update member")
		return
	}

	log.Infof("update member %s for Entity with public Key %s", request.Member.ID.String(), request.SignaturePublicKey)
	m.send(request, response)
}

func (m *Manager) deleteMember(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	if request.MemberID == uuid.Nil {
		m.Router.SendError(request, "invalid member ID")
		return
	}

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLength {
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Warn(err)
		m.Router.SendError(request, err.Error())
		return
	}

	if err = m.db.DeleteMember(entityID, request.MemberID); err != nil {
		log.Error("cannot delete member %s for entity %s : %+v", request.MemberID, request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot delete member")
		return
	}

	log.Infof("deleted member %s for Entity with public Key %s", request.MemberID.String(), request.SignaturePublicKey)
	m.send(request, response)
}

func (m *Manager) countMembers(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLength {
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Warn(err)
		m.Router.SendError(request, err.Error())
		return
	}

	// Query for members
	if response.Count, err = m.db.CountMembers(entityID); err != nil {
		log.Errorf("cannot count members for %s : %+v", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot count members")
		return
	}

	log.Debugf("Entity %s countMembers: %d members", request.SignaturePublicKey, response.Count)
	m.send(request, response)
}

func (m *Manager) generateTokens(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLength {
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Warn(err)
		m.Router.SendError(request, err.Error())
		return
	}

	if request.Amount < 1 {
		log.Warn("invalid request arguments")
		m.Router.SendError(request, "invalid request arguments")
		return
	}

	response.Tokens = make([]uuid.UUID, request.Amount)
	for idx := range response.Tokens {
		response.Tokens[idx] = uuid.New()
	}
	// TODO: Probably I need to initialize tokens
	if err = m.db.CreateMembersWithTokens(entityID, response.Tokens); err != nil {
		log.Error("could not register generated tokens")
		m.Router.SendError(request, "could not register generated tokens")
		return
	}

	log.Infof("generate %d tokens for %s", len(response.Tokens), entityID)
	m.send(request, response)
}

func (m *Manager) exportTokens(request router.RouterRequest) {
	var entityID []byte
	var members []types.Member
	var err error
	var response types.MetaResponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLength {
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Warn(err)
		m.Router.SendError(request, err.Error())
		return
	}

	// TODO: Probably I need to initialize tokens
	if members, err = m.db.MembersTokensEmails(entityID); err != nil {
		if err == sql.ErrNoRows {
			m.Router.SendError(request, "no members found")
			return
		}
		log.Error(err)
		m.Router.SendError(request, err.Error())
		return
	}
	response.MembersTokens = make([]types.TokenEmail, len(members))
	for idx, member := range members {
		response.MembersTokens[idx] = types.TokenEmail{Token: member.ID, Email: member.Email}
	}

	log.Infof("retrieved %d tokens and their emails for Entiy %s", len(members), entityID)
	m.send(request, response)
}

func (m *Manager) importMembers(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLength {
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Warn(err)
		m.Router.SendError(request, err.Error())
		return
	}

	if len(request.MembersInfo) < 1 {
		log.Warn("importMembers: no member data provided")
		m.Router.SendError(request, "no member data provided")
		return
	}

	for idx := range request.MembersInfo {
		request.MembersInfo[idx].Origin = types.Token
	}

	// Add members
	if err = m.db.ImportMembers(entityID, request.MembersInfo); err != nil {
		log.Error(err)
		m.Router.SendError(request, err.Error())
		return
	}

	log.Infof("imported Members for Entity with public Key %s", request.SignaturePublicKey)
	m.send(request, response)
}

func (m *Manager) countTargets(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLength {
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Warn(err)
		m.Router.SendError(request, err.Error())
		return
	}

	// Query for members
	if response.Count, err = m.db.CountTargets(entityID); err != nil {
		log.Errorf("cannot count targets for %s : %+v", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot count targets")
		return
	}

	log.Debugf("Entity %s countTargets: %d targets", request.SignaturePublicKey, response.Count)
	m.send(request, response)
}

func (m *Manager) listTargets(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLength {
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Warn(err)
		m.Router.SendError(request, err.Error())
		return
	}

	// check filter
	if err = checkOptions(request.ListOptions); err != nil {
		log.Warn(err)
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
		log.Error(err)
		m.Router.SendError(request, "cannot query for targets")
		return
	}

	log.Infof("listing targets for Entity with public Key %s", request.SignaturePublicKey)
	m.send(request, response)
}

func (m *Manager) getTarget(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLength {
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Warn(err)
		m.Router.SendError(request, err.Error())
		return
	}

	response.Target, err = m.db.Target(entityID, request.TargetID)
	if err != nil {
		log.Warn("requested target not found")
		m.Router.SendError(request, "requested target not found")
		return
	}

	response.Members, err = m.db.TargetMembers(entityID, request.TargetID)
	if err != nil {
		log.Warn("members for requested target could not be retrieved")
		m.Router.SendError(request, "members for requested target could not be retrieved")
		return
	}

	log.Infof("listing target %s for Entity with public Key %s", request.TargetID.String(), request.SignaturePublicKey)
	m.send(request, response)
}

func (m *Manager) dumpTarget(request router.RouterRequest) {
	var target *types.Target
	var entityID []byte
	var err error
	var response types.MetaResponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLength {
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Warn("invalid target id")
		m.Router.SendError(request, "invalid target id")
		return
	}

	if target, err = m.db.Target(entityID, request.TargetID); err != nil || target.Name != "all" {
		log.Warn("requested target not found")
		m.Router.SendError(request, "requested target not found")
		return
	}

	// TODO: Implement DumpTargetClaims filtered directly by target filters
	if response.Claims, err = m.db.DumpClaims(entityID); err != nil {
		log.Warn(err)
		m.Router.SendError(request, err.Error())
		return
	}

	log.Infof("listing  %d claims for Entity with public Key %s", len(response.Claims), request.SignaturePublicKey)
	m.send(request, response)
}

func (m *Manager) addCensus(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	if len(request.TargetID) == 0 {
		m.Router.SendError(request, "invalid target id")
		return
	}

	if len(request.CensusID) == 0 {
		m.Router.SendError(request, "invalid census id")
		return
	}

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLength {
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey)
	if err != nil {
		log.Warn(err)
		m.Router.SendError(request, err.Error())
		return
	}

	var censusID []byte
	if censusID, err = util.DecodeCensusID(request.CensusID, request.SignaturePublicKey); err != nil {
		m.Router.SendError(request, err.Error())
		return
	}

	target, err := m.db.Target(entityID, request.TargetID)
	if err != nil && target != nil {
		log.Warnf("entity with pubKey %s requested to add census with invalid target id", request.SignaturePublicKey)
		m.Router.SendError(request, "invalid target id")
		return
	}

	err = m.db.AddCensus(entityID, censusID, request.TargetID, request.Census)
	if err != nil {
		log.Error(err)
		m.Router.SendError(request, err.Error())
		return
	}
	log.Debugf("Entity:%s addCensus:%s", request.SignaturePublicKey, request.CensusID)
	log.Infof("addCensus")
	m.send(request, response)
}

func (m *Manager) getCensus(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	if len(request.CensusID) == 0 {
		m.Router.SendError(request, "invalid census id")
		return
	}

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLength {
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey)
	if err != nil {
		log.Warn(err)
		m.Router.SendError(request, err.Error())
		return
	}

	if len(request.CensusID) == 0 {
		m.Router.SendError(request, "invalid census id")
		return
	}

	var censusID []byte
	if censusID, err = util.DecodeCensusID(request.CensusID, request.SignaturePublicKey); err != nil {
		m.Router.SendError(request, err.Error())
		return
	}

	response.Census, err = m.db.Census(entityID, censusID)
	if err != nil {
		log.Error(err)
		m.Router.SendError(request, err.Error())
		return
	}

	response.Target, err = m.db.Target(entityID, response.Census.TargetID)
	if err != nil {
		log.Warn("census target not found")
		m.Router.SendError(request, "census target not found")
		return
	}

	log.Debugf("Entity:%s getCensus:%s", request.SignaturePublicKey, request.CensusID)
	log.Infof("getCensus")
	m.send(request, response)
}

func (m *Manager) countCensus(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLength {
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Warn(err)
		m.Router.SendError(request, err.Error())
		return
	}

	// Query for members
	if response.Count, err = m.db.CountCensus(entityID); err != nil {
		log.Errorf("cannot count censuses for %s : %+v", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot count censuses")
		return
	}

	log.Debugf("Entity %s countCensus: %d censuses", request.SignaturePublicKey, response.Count)
	m.send(request, response)
}

func (m *Manager) listCensus(request router.RouterRequest) {
	var entityID []byte
	var err error
	var response types.MetaResponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLength {
		m.Router.SendError(request, "invalid public key")
		return
	}

	// retrieve entity ID
	entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey)
	if err != nil {
		log.Warn(err)
		m.Router.SendError(request, err.Error())
		return
	}

	// check filter
	err = checkOptions(request.ListOptions)
	if err != nil {
		log.Warn(err)
		m.Router.SendError(request, err.Error())
		return
	}

	// Query for members
	// TODO Implement listCensus in Db that supports filters
	response.Censuses, err = m.db.ListCensus(entityID)
	if err != nil {
		if err == sql.ErrNoRows {
			m.Router.SendError(request, "no censuses found")
			return
		}
		log.Errorf("error in retrieving censuses for entity %s : %+v", request.SignaturePublicKey, err)
		m.Router.SendError(request, "cannot query for censuses")
		return
	}
	log.Debugf("Entity:%s listCensuses", request.SignaturePublicKey)
	log.Info("listCensuses")
	m.send(request, response)
}

func checkOptions(filter *types.ListOptions) error {
	if filter == nil {
		return nil
	}
	// Check skip and count
	if filter.Skip < 0 || filter.Count < 0 {
		return fmt.Errorf("invalid filter options")
	}
	// Check sortby
	t := reflect.TypeOf(types.MemberInfo{})
	if len(filter.SortBy) > 0 {
		_, found := t.FieldByName(strings.Title(filter.SortBy))
		if !found {
			return fmt.Errorf("invalid filter options")
		}
		// Check order
		if len(filter.Order) > 0 && !(filter.Order == "asc" || filter.Order == "desc") {
			return fmt.Errorf("invalid filter options")
		}

	} else if len(filter.Order) > 0 {
		// Also check that order does not make sense without sortby
		return fmt.Errorf("invalid filter options")
	}
	return nil

}
