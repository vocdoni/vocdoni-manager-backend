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

	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/dvote/log"
	"go.vocdoni.io/manager/database/pgsql"
	"go.vocdoni.io/manager/types"
	"go.vocdoni.io/manager/util"
)

func (m *Manager) signUp(request *types.APIrequest) (*types.APIresponse, error) {
	var entityID []byte
	var entityInfo *types.EntityInfo
	var target *types.Target
	var err error
	var response types.APIresponse

	// check public key length
	// dvoteutil.IsHexEncodedStringWithLength
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		return nil, fmt.Errorf("invalid public key")
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("cannot recover entityID")
	}

	entityInfo = &types.EntityInfo{CensusManagersAddresses: [][]byte{entityID}, Origins: []types.Origin{types.Token}}
	if request.Entity != nil {
		// For now control which EntityInfo fields end up to the DB
		entityInfo.Name = request.Entity.Name
		entityInfo.Email = request.Entity.Email
		entityInfo.Size = request.Entity.Size
		entityInfo.Type = request.Entity.Type
		entityInfo.Consented = request.Entity.Consented
	}

	// Add Entity
	if err = m.db.AddEntity(entityID, entityInfo); err != nil && !strings.Contains(err.Error(), "entities_pkey") {
		log.Errorf("cannot add entity %x to the DB: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("cannot add entity to the DB")
	}

	target = &types.Target{EntityID: entityID, Name: "all", Filters: json.RawMessage([]byte("{}"))}
	if _, err = m.db.AddTarget(entityID, target); err != nil && !strings.Contains(err.Error(), "result has no rows") {
		log.Errorf("cannot create entity's %x generic target: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("cannot create entity generic target")
	}

	entityAddress := ethcommon.BytesToAddress(entityID)
	// do not try to send tokens if ethclient is nil
	if m.eth != nil {
		// send the default amount of faucet tokens iff wallet balance is zero
		sent, err := m.eth.SendTokens(context.Background(), entityAddress, 0, 0)
		if err != nil {
			if !strings.Contains(err.Error(), "maxAcceptedBalance") {
				log.Errorf("error sending tokens to entity %s : %v", entityAddress.String(), err)
				return nil, fmt.Errorf("could not send tokens to %s", entityAddress.String())
			}
			log.Warnf("signUp not sending tokens to entity %s : %v", entityAddress.String(), err)
		}
		response.Count = int(sent.Int64())
	}

	log.Debugf("Entity: %s signUp", entityAddress.String())
	return &response, nil
}

func (m *Manager) getEntity(request *types.APIrequest) (*types.APIresponse, error) {
	var entityID []byte
	var err error
	var response types.APIresponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		return nil, fmt.Errorf("invalid public key")
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("cannot recover entityID")
	}

	if response.Entity, err = m.db.Entity(entityID); err != nil {
		if err == sql.ErrNoRows {
			log.Errorf("entity requesting its info with getEntity not found")
			return nil, fmt.Errorf("entity not found")
		}
		log.Errorf("cannot retrieve details of entity %x: (%v)", entityID, err)
		return nil, fmt.Errorf("cannot retrieve entity")
	}

	log.Infof("listing details of Entity %x", entityID)
	return &response, nil
}

func (m *Manager) updateEntity(request *types.APIrequest) (*types.APIresponse, error) {
	var entityID []byte
	var err error
	var response types.APIresponse

	// check public key length
	// dvoteutil.IsHexEncodedStringWithLength
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		return nil, fmt.Errorf("invalid public key")
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("cannot recover entityID")
	}

	if request.Entity == nil {
		log.Errorf("updateEntity with no entity data to update for %x", entityID)
		return nil, fmt.Errorf("no entity data to update")
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
		return nil, fmt.Errorf("cannot update entity")
	}

	log.Debugf("Entity: %x entityUpdate", entityID)
	return &response, nil
}

func (m *Manager) listMembers(request *types.APIrequest) (*types.APIresponse, error) {
	var entityID []byte
	var err error
	var response types.APIresponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		return nil, fmt.Errorf("invalid public key")
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("cannot recover entityID")
	}

	// check filter
	if err = checkOptions(request.ListOptions, request.Method); err != nil {
		log.Warnf("invalid filter options %x: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("invalid filter options")
	}

	// Query for members
	if response.Members, err = m.db.ListMembers(entityID, request.ListOptions); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no members found")
		}
		log.Errorf("cannot retrieve members of %x: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("cannot retrieve members")
	}

	log.Debugf("Entity: %x listMembers %d members", request.SignaturePublicKey, len(response.Members))
	return &response, nil
}

func (m *Manager) getMember(request *types.APIrequest) (*types.APIresponse, error) {
	var entityID []byte
	var err error
	var response types.APIresponse

	if request.MemberID == nil {
		log.Warnf("memberID is nil on getMember")
		return nil, fmt.Errorf("invalid memberId")
	}

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		return nil, fmt.Errorf("invalid public key")
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("cannot recover entityID")
	}

	if response.Member, err = m.db.Member(entityID, request.MemberID); err != nil {
		if err == sql.ErrNoRows {
			log.Warn("member not found")
			return nil, fmt.Errorf("member not found")
		}
		log.Errorf("cannot retrieve member %q for entity %x: (%v)", request.MemberID.String(), request.SignaturePublicKey, err)
		return nil, fmt.Errorf("cannot retrieve member")
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
		return nil, fmt.Errorf("error retrieving member targets")
	}

	log.Infof("listing member %q for Entity with public Key %x", request.MemberID.String(), request.SignaturePublicKey)
	return &response, nil
}

func (m *Manager) updateMember(request *types.APIrequest) (*types.APIresponse, error) {
	var entityID []byte
	var err error
	var response types.APIresponse

	if request.Member == nil {
		return nil, fmt.Errorf("invalid member struct")
	}

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		return nil, fmt.Errorf("invalid public key")
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("cannot recover entityID")
	}

	// If a string Member property is sent as "" then it is not updated
	if response.Count, err = m.db.UpdateMember(entityID, &request.Member.ID, &request.Member.MemberInfo); err != nil {
		log.Errorf("cannot update member %q for entity %x: (%v)", request.Member.ID.String(), request.SignaturePublicKey, err)
		return nil, fmt.Errorf("cannot update member")
	}

	log.Infof("update member %q for Entity with public Key %x", request.Member.ID.String(), request.SignaturePublicKey)
	return &response, nil
}

func (m *Manager) deleteMembers(request *types.APIrequest) (*types.APIresponse, error) {
	var entityID []byte
	var err error
	var response types.APIresponse

	if len(request.MemberIDs) == 0 {
		return nil, fmt.Errorf("invalid member list")
	}

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		return nil, fmt.Errorf("invalid public key")
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("cannot recover entityID")
	}

	response.Count, response.InvalidIDs, err = m.db.DeleteMembers(entityID, request.MemberIDs)
	if err != nil {
		log.Errorf("error deleting members for entity %x: (%v)", entityID, err)
		return nil, fmt.Errorf("error deleting members")
	}

	log.Infof("deleted %d members, found %d invalid tokens, for Entity with public Key %x", response.Count, len(response.InvalidIDs), entityID)
	return &response, nil
}

func (m *Manager) countMembers(request *types.APIrequest) (*types.APIresponse, error) {
	var entityID []byte
	var err error
	var response types.APIresponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		return nil, fmt.Errorf("invalid public key")
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("cannot recover entityID")
	}

	// Query for members
	if response.Count, err = m.db.CountMembers(entityID); err != nil {
		log.Errorf("cannot count members for %x: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("cannot count members")
	}

	log.Debugf("Entity %q countMembers: %d members", request.SignaturePublicKey, response.Count)
	return &response, nil
}

func (m *Manager) generateTokens(request *types.APIrequest) (*types.APIresponse, error) {
	var entityID []byte
	var err error
	var response types.APIresponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		return nil, fmt.Errorf("invalid public key")
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("cannot recover entityID")
	}

	if request.Amount < 1 {
		log.Warnf("invalid token amount requested by %x", request.SignaturePublicKey)
		return nil, fmt.Errorf("invalid token amount")
	}

	response.Tokens = make([]uuid.UUID, request.Amount)
	for idx := range response.Tokens {
		response.Tokens[idx] = uuid.New()
	}
	// TODO: Probably I need to initialize tokens
	if err = m.db.CreateMembersWithTokens(entityID, response.Tokens); err != nil {
		log.Errorf("could not register generated tokens for %x: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("could not register generated tokens")
	}

	log.Debugf("Entity: %x generateTokens: %d tokens", request.SignaturePublicKey, len(response.Tokens))
	return &response, nil
}

func (m *Manager) exportTokens(request *types.APIrequest) (*types.APIresponse, error) {
	var entityID []byte
	var members []types.Member
	var err error
	var response types.APIresponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		return nil, fmt.Errorf("invalid public key")
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("cannot recover entityID")
	}

	// TODO: Probably I need to initialize tokens
	if members, err = m.db.MembersTokensEmails(entityID); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no members found")
		}
		log.Errorf("could not retrieve members tokens for %x: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("could not retrieve members tokens")
	}
	response.MembersTokens = make([]types.TokenEmail, len(members))
	for idx, member := range members {
		response.MembersTokens[idx] = types.TokenEmail{Token: member.ID, Email: member.Email}
	}

	log.Debugf("Entity: %x exportTokens: %d tokens", request.SignaturePublicKey, len(members))
	return &response, nil
}

func (m *Manager) importMembers(request *types.APIrequest) (*types.APIresponse, error) {
	var entityID []byte
	var err error
	var response types.APIresponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		return nil, fmt.Errorf("invalid public key")
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("cannot recover entityID")
	}

	if len(request.MembersInfo) < 1 {
		log.Warnf("no member data provided for import members by %x", request.SignaturePublicKey)
		return nil, fmt.Errorf("no member data provided")
	}

	for idx := range request.MembersInfo {
		request.MembersInfo[idx].Origin = types.Token
	}

	// Add members
	if err = m.db.ImportMembers(entityID, request.MembersInfo); err != nil {
		log.Errorf("could not import members for %x: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("could not import members")
	}

	log.Debugf("Entity: %x importMembers: %d members", request.SignaturePublicKey, len(request.MembersInfo))
	return &response, nil
}

func (m *Manager) countTargets(request *types.APIrequest) (*types.APIresponse, error) {
	var entityID []byte
	var err error
	var response types.APIresponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		return nil, fmt.Errorf("invalid public key")
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("cannot recover entityID")
	}

	// Query for members
	if response.Count, err = m.db.CountTargets(entityID); err != nil {
		log.Errorf("cannot count targets for %x: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("cannot count targets")
	}

	log.Debugf("Entity %x countTargets: %d targets", request.SignaturePublicKey, response.Count)
	return &response, nil
}

func (m *Manager) listTargets(request *types.APIrequest) (*types.APIresponse, error) {
	var entityID []byte
	var err error
	var response types.APIresponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		return nil, fmt.Errorf("invalid public key")
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("cannot recover entityID")
	}

	// check filter
	if err = checkOptions(request.ListOptions, request.Method); err != nil {
		log.Warnf("invalid filter options %x: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("invalid filter options: (%v)", err)
	}

	// Retrieve targets
	// Implement filters in DB
	response.Targets, err = m.db.ListTargets(entityID)
	if err != nil || len(response.Targets) == 0 {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no targets found")
		}
		log.Errorf("cannot query targets for %x: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("cannot query targets")
	}

	log.Debugf("Entity: %x listTargets: %d targets", request.SignaturePublicKey, len(response.Targets))
	return &response, nil
}

func (m *Manager) getTarget(request *types.APIrequest) (*types.APIresponse, error) {
	var entityID []byte
	var err error
	var response types.APIresponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		return nil, fmt.Errorf("invalid public key")
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("cannot recover entityID")
	}

	if response.Target, err = m.db.Target(entityID, request.TargetID); err != nil {
		if err == sql.ErrNoRows {
			log.Debugf("target %q not found for %x", request.TargetID.String(), request.SignaturePublicKey)
			return nil, fmt.Errorf("target not found")
		}
		log.Errorf("could not retrieve target for %x: %+v", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("could not retrieve target")
	}

	response.Members, err = m.db.TargetMembers(entityID, request.TargetID)
	if err != nil {
		log.Warn("members for requested target could not be retrieved")
		return nil, fmt.Errorf("members for requested target could not be retrieved")
	}

	log.Debugf("Entity: %x getTarget: %s", request.SignaturePublicKey, request.TargetID.String())
	return &response, nil
}

func (m *Manager) dumpTarget(request *types.APIrequest) (*types.APIresponse, error) {
	var target *types.Target
	var entityID []byte
	var err error
	var response types.APIresponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		return nil, fmt.Errorf("invalid public key")
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("cannot recover entityID")
	}

	if target, err = m.db.Target(entityID, request.TargetID); err != nil || target.Name != "all" {
		if err == sql.ErrNoRows {
			log.Debugf("target %q not found for %x", request.TargetID.String(), request.SignaturePublicKey)
			return nil, fmt.Errorf("target not found")
		}
		log.Errorf("could not retrieve target for %x: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("could not retrieve target")
	}

	// TODO: Implement DumpTargetClaims filtered directly by target filters
	if response.Claims, err = m.db.DumpClaims(entityID); err != nil {
		if err == sql.ErrNoRows {
			log.Debugf("no claims found for %x", request.SignaturePublicKey)
			return nil, fmt.Errorf("no claims found")
		}
		log.Errorf("cannot dump claims for %x: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("cannot dump claims")
	}

	log.Debugf("Entity: %x dumpTarget: %d claims", request.SignaturePublicKey, len(response.Claims))
	return &response, nil
}

func (m *Manager) dumpCensus(request *types.APIrequest) (*types.APIresponse, error) {
	var err error
	var response types.APIresponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		return nil, fmt.Errorf("invalid public key")
	}

	// retrieve entity ID
	entityID, err := util.PubKeyToEntityID(request.SignaturePublicKey)
	if err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("cannot recover entityID")
	}

	censusID, err := util.DecodeCensusID(request.CensusID, request.SignaturePublicKey)
	if err != nil {
		log.Errorf("cannot decode census id %s for %x", request.CensusID, entityID)
		return nil, fmt.Errorf("cannot decode census id")
	}

	// TODO: Implement DumpTargetClaims filtered directly by target filters
	censusMembers, err := m.db.ExpandCensusMembers(entityID, censusID)
	if err != nil {
		log.Errorf("cannot dump claims for %q: (%v)", entityID, err)
		return nil, fmt.Errorf("cannot dump claims")
	}
	shuffledClaims := make([][]byte, len(censusMembers))
	shuffledIndexes := rand.Perm(len(censusMembers))
	for i, v := range shuffledIndexes {
		shuffledClaims[v] = censusMembers[i].DigestedPubKey
	}
	response.Claims = shuffledClaims

	log.Debugf("Entity: %x dumpCensus: %d claims", entityID, len(response.Claims))
	return &response, nil
}

func (m *Manager) sendVotingLinks(request *types.APIrequest) (*types.APIresponse, error) {

	if len(request.MemberID) == 0 || len(request.ProcessID) == 0 {
		return nil, fmt.Errorf("invalid arguments")
	}

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		return nil, fmt.Errorf("invalid public key")
	}

	// retrieve entity ID
	entityID, err := util.PubKeyToEntityID(request.SignaturePublicKey)
	if err != nil {
		log.Errorf("cannot recover %x entityID from public key: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("cannot recover entityID from public key")
	}

	entity, err := m.db.Entity(entityID)
	if err != nil {
		log.Errorf("cannot recover entity %x: (%v)", entityID, err)
		return nil, fmt.Errorf("cannot recover entity from public key")
	}

	censusID, err := util.DecodeCensusID(request.CensusID, request.SignaturePublicKey)
	if err != nil {
		log.Errorf("cannot decode census id %s for %x", request.CensusID, entityID)
		return nil, fmt.Errorf("cannot decode census id")
	}

	if request.Email != "" {
		// Individual email
		censusMember, err := m.db.EphemeralMemberInfoByEmail(entityID, censusID, request.Email)
		if err != nil {
			log.Errorf("cannot retrieve ephemeral member %s of  census %x for enity %x: (%v)", request.Email, censusID, entityID, err)
			return nil, fmt.Errorf("cannot retrieve ephemeral census member by email")
		}
		if err := m.smtp.SendVotingLink(censusMember, entity, request.ProcessID); err != nil {
			log.Errorf("could not send voting link for member %q entity: (%v)", censusMember.ID, err)
			return nil, fmt.Errorf("could not send voting link")
		}
		log.Infof("send validation links to 1 members for Entity %x", entityID)
		var response types.APIresponse
		response.Count = 1
		return &response, nil
	}

	censusMembers, err := m.db.ListEphemeralMemberInfo(entityID, censusID)
	if err != nil {
		log.Errorf("cannot retrieve ephemeral members of  census %x for enity %x: (%v)", censusID, entityID, err)
		return nil, fmt.Errorf("cannot retrieve ephemeral census members")
	}

	var response types.APIresponse
	if len(censusMembers) == 0 {
		response.Count = 0
		return &response, nil
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
		return nil, fmt.Errorf("inconsistency in number of sent emails and errors")
	}
	if len(errors) == len(censusMembers) {
		log.Errorf("no validation email was sent %v", errors)
		return nil, fmt.Errorf("could not send emails")
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
				return nil, fmt.Errorf("sent emails but could not assign tag")
			}
		} else {
			log.Infof("send validation links to %d members, skipped %d invalid IDs and %d errors , for Entity %x\nErrors: %v", response.Count, len(response.InvalidIDs), len(errors), entityID, errors)
			log.Errorf("error retreiving Pending tag:  %v", err)
			return nil, fmt.Errorf("sent emails but could not assign tag")
		}
	}
	_, _, err = m.db.AddTagToMembers(entityID, successUUIDs, tag.ID)
	if err != nil {
		log.Infof("send validation links to %d members, skipped %d invalid IDs and %d errors , for Entity %x\nErrors: %v", response.Count, len(response.InvalidIDs), len(errors), entityID, errors)
		log.Errorf("error assinging Pending tag:  %v", err)
		return nil, fmt.Errorf("sent emails but could not assign tag")
	}

	log.Infof("send validation links to %d members, skipped %d invalid IDs and %d errors , for Entity %x\nErrors: %v", response.Count, len(response.InvalidIDs), len(errors), entityID, errors)
	return &response, nil
}

func (m *Manager) addCensus(request *types.APIrequest) (*types.APIresponse, error) {
	var entityID []byte
	var err error
	var response types.APIresponse

	if len(request.TargetID) == 0 {
		log.Debugf("invalid target id %q for %x", request.TargetID.String(), request.SignaturePublicKey)
		return nil, fmt.Errorf("invalid target id")
	}

	if len(request.CensusID) == 0 {
		log.Debugf("invalid census id %q for %x", request.CensusID, request.SignaturePublicKey)
		return nil, fmt.Errorf("invalid census id")
	}

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		return nil, fmt.Errorf("invalid public key")
	}

	// retrieve entity ID
	entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey)
	if err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("cannot recover entityID")
	}

	var censusID []byte
	if censusID, err = util.DecodeCensusID(request.CensusID, request.SignaturePublicKey); err != nil {
		return nil, fmt.Errorf(err.Error())
	}

	// size, err := m.db.AddCensusWithMembers(entityID, censusID, request.TargetID, request.Census)
	if err := m.db.AddCensus(entityID, censusID, request.TargetID, request.Census); err != nil {
		log.Errorf("cannot add census %q  for: %q: (%v)", request.CensusID, entityID, err)
		return nil, fmt.Errorf("cannot add census")
	}

	log.Debugf("Entity: %x addCensus: %s  ", entityID, request.CensusID)
	return &response, nil
}

func (m *Manager) updateCensus(request *types.APIrequest) (*types.APIresponse, error) {
	// TODO Handle invalid claims
	var entityID []byte
	var err error
	var response types.APIresponse

	if len(request.CensusID) == 0 {
		log.Debugf("invalid census id %q for %x", request.CensusID, request.SignaturePublicKey)
		return nil, fmt.Errorf("invalid census id")
	}

	if request.Census == nil {
		log.Debugf("invalid census info for census %q for entity %x", request.CensusID, request.SignaturePublicKey)
		return nil, fmt.Errorf("invalid census info")
	}

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		return nil, fmt.Errorf("invalid public key")
	}

	// retrieve entity ID
	entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey)
	if err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("cannot recover entityID")
	}

	var censusID []byte
	if censusID, err = util.DecodeCensusID(request.CensusID, request.SignaturePublicKey); err != nil {
		return nil, fmt.Errorf(err.Error())
	}

	if request.InvalidClaims != nil && len(request.InvalidClaims) > 0 {
		log.Warnf("invalid claims: %v", request.InvalidClaims)
	}

	if response.Count, err = m.db.UpdateCensus(entityID, censusID, request.Census); err != nil {
		log.Errorf("cannot update census %q for %x: (%v)", request.CensusID, entityID, err)
		return nil, fmt.Errorf("cannot update census")
	}

	log.Debugf("Entity: %x updateCensus: %s \n %v", entityID, request.CensusID, request.Census)
	return &response, nil
}

func (m *Manager) getCensus(request *types.APIrequest) (*types.APIresponse, error) {
	var entityID []byte
	var err error
	var response types.APIresponse

	if len(request.CensusID) == 0 {
		log.Debugf("invalid census id %q for %s", request.CensusID, request.SignaturePublicKey)
		return nil, fmt.Errorf("invalid census id")
	}

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		return nil, fmt.Errorf("invalid public key")
	}

	// retrieve entity ID
	entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey)
	if err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("cannot recover entityID")
	}

	var censusID []byte
	if censusID, err = util.DecodeCensusID(request.CensusID, request.SignaturePublicKey); err != nil {
		log.Errorf("cannot decode census id %s for %x", request.CensusID, request.SignaturePublicKey)
		return nil, fmt.Errorf("cannot decode census id")
	}

	response.Census, err = m.db.Census(entityID, censusID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Debugf("census %q not found for %x", request.CensusID, request.SignaturePublicKey)
			return nil, fmt.Errorf("census not found")
		}
		log.Errorf("error in retrieving censuses for entity %x: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("cannot query for censuses")
	}

	response.Target, err = m.db.Target(entityID, &response.Census.TargetID)
	if err != nil {
		log.Warn("census target not found")
		return nil, fmt.Errorf("census target not found")
	}

	log.Debugf("Entity: %x getCensus:%s", request.SignaturePublicKey, request.CensusID)
	return &response, nil
}

func (m *Manager) countCensus(request *types.APIrequest) (*types.APIresponse, error) {
	var entityID []byte
	var err error
	var response types.APIresponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		return nil, fmt.Errorf("invalid public key")
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("cannot recover entityID")
	}

	// Query for members
	if response.Count, err = m.db.CountCensus(entityID); err != nil {
		log.Errorf("cannot count censuses for %x: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("cannot count censuses")
	}

	log.Debugf("Entity %x countCensus: %d censuses", request.SignaturePublicKey, response.Count)
	return &response, nil
}

func (m *Manager) listCensus(request *types.APIrequest) (*types.APIresponse, error) {
	var entityID []byte
	var err error
	var response types.APIresponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		return nil, fmt.Errorf("invalid public key")
	}

	// retrieve entity ID
	entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey)
	if err != nil {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		return nil, fmt.Errorf("invalid public key")
	}

	// check filter
	if err := checkOptions(request.ListOptions, request.Method); err != nil {
		log.Warnf("invalid filter options %x: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("invalid filter options")
	}

	// Query for members
	// TODO Implement listCensus in Db that supports filters
	response.Censuses, err = m.db.ListCensus(entityID, request.ListOptions)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no censuses found")
		}
		log.Errorf("error in retrieving censuses for entity %x: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("cannot query for censuses")
	}
	log.Debugf("Entity: %x listCensuses: %d censuses", request.SignaturePublicKey, len(response.Censuses))
	return &response, nil
}

func (m *Manager) deleteCensus(request *types.APIrequest) (*types.APIresponse, error) {
	var entityID []byte
	var err error
	var response types.APIresponse

	if len(request.CensusID) == 0 {
		log.Debugf("invalid census id %q for %x", request.CensusID, request.SignaturePublicKey)
		return nil, fmt.Errorf("invalid census id")
	}

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		return nil, fmt.Errorf("invalid public key")
	}

	// retrieve entity ID
	entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey)
	if err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("cannot recover entityID")
	}

	var censusID []byte
	if censusID, err = util.DecodeCensusID(request.CensusID, request.SignaturePublicKey); err != nil {
		log.Errorf("cannot decode census id %x for %s", request.CensusID, request.SignaturePublicKey)
		return nil, fmt.Errorf("cannot decode census id")
	}

	err = m.db.DeleteCensus(entityID, censusID)
	if err != nil && err != sql.ErrNoRows {
		log.Errorf("error deleting census %s for entity %x: (%v)", request.CensusID, entityID, err)
		return nil, fmt.Errorf("cannot delete census")
	}

	log.Debugf("Entity: %x deleteCensus:%s", entityID, request.CensusID)
	return &response, nil
}

func (m *Manager) sendValidationLinks(request *types.APIrequest) (*types.APIresponse, error) {

	if len(request.MemberID) == 0 {
		return nil, fmt.Errorf("invalid arguments")
	}

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		return nil, fmt.Errorf("invalid public key")
	}

	// retrieve entity ID
	entityID, err := util.PubKeyToEntityID(request.SignaturePublicKey)
	if err != nil {
		log.Errorf("cannot recover %x entityID from public key: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("cannot recover entityID from public key")
	}

	entity, err := m.db.Entity(entityID)
	if err != nil {
		log.Errorf("cannot recover %x entity: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("cannot recover entity from public key")
	}

	var response types.APIresponse
	var members []types.Member
	members, response.InvalidIDs, err = m.db.Members(entityID, request.MemberIDs)
	if err != nil {
		log.Errorf("cannot retrieve members for entity %x: (%v)", entityID, err)
		return nil, fmt.Errorf("cannot retrieve member")
	}

	if len(members) == 0 {
		response.Count = 0
		return &response, nil
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
			}
			if err := m.smtp.SendValidationLink(&member, entity); err != nil {
				log.Errorf("could not send validation link for member %q entity: (%v)", member.ID, err)
				ec <- fmt.Errorf("member %s error  %v", member.ID, err)
				wg.Done()
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
		return nil, fmt.Errorf("inconsistency in number of sent emails and errors")
	}
	if len(errors) == len(members) {
		log.Errorf("no validation email was sent %v", errors)
		return nil, fmt.Errorf("could not send emails")
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
				return nil, fmt.Errorf("sent emails but could not assign tag")
			}
		} else {
			log.Infof("send validation links to %d members, skipped %d invalid IDs, %d duplicates and %d errors , for Entity %x\nErrors: %v", response.Count, len(response.InvalidIDs), duplicates, len(errors), entityID, errors)
			log.Errorf("error retreiving Pending tag:  %v", err)
			return nil, fmt.Errorf("sent emails but could not assign tag")
		}
	}
	_, _, err = m.db.AddTagToMembers(entityID, successUUIDs, tag.ID)
	if err != nil {
		log.Infof("send validation links to %d members, skipped %d invalid IDs, %d duplicates and %d errors , for Entity %x\nErrors: %v", response.Count, len(response.InvalidIDs), duplicates, len(errors), entityID, errors)
		log.Errorf("error assinging Pending tag:  %v", err)
		return nil, fmt.Errorf("sent emails but could not assign tag")
	}

	log.Infof("send validation links to %d members, skipped %d invalid IDs, %d duplicates and %d errors , for Entity %x\nErrors: %v", response.Count, len(response.InvalidIDs), duplicates, len(errors), entityID, errors)
	return &response, nil
}

func (m *Manager) createTag(request *types.APIrequest) (*types.APIresponse, error) {
	var entityID []byte
	var err error
	var response types.APIresponse

	if request.TagName == "" {
		log.Debug("createTag with empty tag")
		return nil, fmt.Errorf("invalid tag name")
	}

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		return nil, fmt.Errorf("invalid public key")
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("cannot recover entityID")
	}

	response.Tag = &types.Tag{
		Name: request.TagName,
	}

	if response.Tag.ID, err = m.db.AddTag(entityID, request.TagName); err != nil {
		log.Errorf("cannot create tag '%s' for entity %x: (%v)", request.TagName, entityID, err)
		return nil, fmt.Errorf("cannot create tag ")
	}

	log.Infof("created tag with id %d and name '%s' for Entity %x", response.Tag.ID, response.Tag.Name, entityID)
	return &response, nil
}

func (m *Manager) listTags(request *types.APIrequest) (*types.APIresponse, error) {
	var entityID []byte
	var err error
	var response types.APIresponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		return nil, fmt.Errorf("invalid public key")
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("cannot recover entityID")
	}

	// Query for members
	if response.Tags, err = m.db.ListTags(entityID); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no tags found")
		}
		log.Errorf("cannot retrieve tags of %x: (%v)", entityID, err)
		return nil, fmt.Errorf("cannot retrieve tags")
	}

	log.Debugf("Entity: %x list %d tags", entityID, len(response.Tags))
	return &response, nil
}

func (m *Manager) deleteTag(request *types.APIrequest) (*types.APIresponse, error) {
	var entityID []byte
	var err error
	var response types.APIresponse

	if request.TagID == 0 {
		log.Debug("deleteTag with empty tag")
		return nil, fmt.Errorf("invalid tag id")
	}

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		return nil, fmt.Errorf("invalid public key")
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("cannot recover entityID")
	}

	if err = m.db.DeleteTag(entityID, request.TagID); err != nil {
		log.Errorf("cannot delete tag %d for entity %x: (%v)", request.TagID, entityID, err)
		return nil, fmt.Errorf("cannot delete tag ")
	}

	log.Infof("delete tag with id %d for Entity %x", request.TagID, entityID)
	return &response, nil
}

func (m *Manager) addTag(request *types.APIrequest) (*types.APIresponse, error) {
	var entityID []byte
	var err error
	var response types.APIresponse

	if request.TagID == 0 || len(request.MemberIDs) == 0 {
		log.Debug("addTag invalid arguments")
		return nil, fmt.Errorf("invalid arguments")
	}

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		return nil, fmt.Errorf("invalid public key")
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("cannot recover entityID")
	}

	response.Count, response.InvalidIDs, err = m.db.AddTagToMembers(entityID, request.MemberIDs, request.TagID)
	if err != nil {
		log.Errorf("cannot add tag %d to members for entity %x: (%v)", request.TagID, entityID, err)
		return nil, fmt.Errorf("cannot add tag ")
	}

	log.Infof("added tag with id %d to %d, with %d invalid IDs, members of Entity %x", request.TagID, response.Count, len(response.InvalidIDs), entityID)
	return &response, nil
}

func (m *Manager) removeTag(request *types.APIrequest) (*types.APIresponse, error) {
	var entityID []byte
	var err error
	var response types.APIresponse

	if request.TagID == 0 || len(request.MemberIDs) == 0 {
		log.Debug("removeTag invalid arguments")
		return nil, fmt.Errorf("invalid arguments")
	}

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		return nil, fmt.Errorf("invalid public key")
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("cannot recover entityID")
	}

	response.Count, response.InvalidIDs, err = m.db.RemoveTagFromMembers(entityID, request.MemberIDs, request.TagID)
	if err != nil {
		log.Errorf("cannot remove tag %d from members for entity %x: (%v)", request.TagID, entityID, err)
		return nil, fmt.Errorf("cannot remove tag ")
	}

	log.Infof("removed tag with id %d from %d members, with %d invalid IDs, of Entity %x", request.TagID, response.Count, len(response.InvalidIDs), entityID)
	return &response, nil
}

func (m *Manager) adminEntityList(request *types.APIrequest) (*types.APIresponse, error) {
	var entityID []byte
	var err error
	var response types.APIresponse

	// check public key length
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %x", request.SignaturePublicKey)
		return nil, fmt.Errorf("invalid public key")
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %x entityID: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("cannot recover entityID")
	}
	log.Debugf("%s", ethcommon.BytesToAddress((entityID)).String())
	if ethcommon.BytesToAddress((entityID)).String() != "0xCc41C6545234ac63F11c47bC282f89Ca77aB9945" {
		log.Warnf("invalid auth: (%v)", request.SignaturePublicKey)
		return nil, fmt.Errorf("invalid auth")
	}

	// Query for members
	if response.Entities, err = m.db.AdminEntityList(); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no entities found")
		}
		log.Errorf("cannot retrieve entities: (%v)", err)
		return nil, fmt.Errorf("cannot retrieve entities")
	}

	log.Debugf("Entity: %x adminEntityList %d entities", request.SignaturePublicKey, len(response.Entities))
	return &response, nil
}

func (m *Manager) requestGas(request *types.APIrequest) (*types.APIresponse, error) {
	var entityID []byte
	var err error
	var response types.APIresponse

	if m.eth == nil {
		log.Errorf("cannot request for tokens, ethereum client is nil")
		return nil, fmt.Errorf("internal error")
	}
	// check public key length
	// dvoteutil.IsHexEncodedStringWithLength
	if len(request.SignaturePublicKey) != ethereum.PubKeyLengthBytes {
		log.Warnf("invalid public key: %s", request.SignaturePublicKey)
		return nil, fmt.Errorf("invalid public key")
	}

	// retrieve entity ID
	if entityID, err = util.PubKeyToEntityID(request.SignaturePublicKey); err != nil {
		log.Errorf("cannot recover %q entityID: (%v)", request.SignaturePublicKey, err)
		return nil, fmt.Errorf("cannot recover entityID")
	}

	entityAddress := ethcommon.BytesToAddress(entityID)

	// check entity exists
	if _, err := m.db.Entity(entityID); err != nil {
		if err == sql.ErrNoRows {
			log.Errorf("entity not found")
			return nil, fmt.Errorf("entity not found")
		}
		log.Errorf("cannot retrieve details of entity %x: (%v)", entityID, err)
		return nil, fmt.Errorf("cannot retrieve entity")
	}

	sent, err := m.eth.SendTokens(context.Background(), entityAddress, 0, 0)
	if err != nil {
		log.Errorf("error sending tokens to entity %s : %v", entityAddress.String(), err)
		return nil, fmt.Errorf("error sending tokens")
	}

	response.Count = int(sent.Int64())
	return &response, nil
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
