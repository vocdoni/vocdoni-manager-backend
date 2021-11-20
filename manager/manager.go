package manager

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"math/rand"
	"sync"

	"fmt"
	"reflect"
	"strings"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"go.vocdoni.io/manager/ethclient"

	"go.vocdoni.io/dvote/httprouter"
	"go.vocdoni.io/dvote/httprouter/bearerstdapi"
	"go.vocdoni.io/dvote/log"
	"go.vocdoni.io/manager/database"
	"go.vocdoni.io/manager/database/pgsql"
	"go.vocdoni.io/manager/smtpclient"
	"go.vocdoni.io/manager/types"
	"go.vocdoni.io/manager/util"
)

type Manager struct {
	db   database.Database
	smtp *smtpclient.SMTP
	eth  *ethclient.Eth
}

// NewManager creates a new registry handler for the Router
func NewManager(d database.Database, s *smtpclient.SMTP, ethclient *ethclient.Eth) *Manager {
	return &Manager{db: d, smtp: s, eth: ethclient}
}

func (m *Manager) HasEthClient() bool {
	return m.eth != nil
}

func (m *Manager) SignUp(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var entityID []byte
	var signaturePubKey []byte
	var entityInfo *types.EntityInfo
	var target *types.Target
	var err error
	var response types.MetaResponse

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	var newEntity *types.EntityInfo
	if err := util.DecodeJsonMessage(newEntity, "entity", ctx); err != nil {
		return fmt.Errorf("cannot recover entity: %v", err)
	}

	entityInfo = &types.EntityInfo{CensusManagersAddresses: [][]byte{entityID}, Origins: []types.Origin{types.Token}}
	if newEntity != nil {
		// For now control which EntityInfo fields end up to the DB
		entityInfo.Name = newEntity.Name
		entityInfo.Email = newEntity.Email
		entityInfo.Size = newEntity.Size
		entityInfo.Type = newEntity.Type
	}

	// Add Entity
	if err = m.db.AddEntity(entityID, entityInfo); err != nil && !strings.Contains(err.Error(), "entities_pkey") {
		return fmt.Errorf("cannot add entity %x to the DB: (%v)", signaturePubKey, err)
	}

	target = &types.Target{EntityID: entityID, Name: "all", Filters: json.RawMessage([]byte("{}"))}
	if _, err = m.db.AddTarget(entityID, target); err != nil && !strings.Contains(err.Error(), "result has no rows") {
		return fmt.Errorf("cannot create entity's %x generic target: (%v)", signaturePubKey, err)
	}

	entityAddress := ethcommon.BytesToAddress(entityID)
	// do not try to send tokens if ethclient is nil
	if m.eth != nil {
		// send the default amount of faucet tokens iff wallet balance is zero
		sent, err := m.eth.SendTokens(context.Background(), entityAddress, 0, 0)
		if err != nil {
			if !strings.Contains(err.Error(), "maxAcceptedBalance") {
				return fmt.Errorf("error sending tokens to entity %s : %v", entityAddress.String(), err)
			}
			log.Warnf("signUp not sending tokens to entity %s : %v", entityAddress.String(), err)
		}
		response.Count = int(sent.Int64())
	}

	log.Debugf("Entity: %s signUp", entityAddress.String())
	return util.SendResponse(response, ctx)
}

func (m *Manager) GetEntity(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var entityID []byte
	var err error
	var response types.MetaResponse

	if entityID, _, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	if response.Entity, err = m.db.Entity(entityID); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("entity requesting its info with getEntity not found")
		}
		return fmt.Errorf("cannot retrieve details of entity %x: (%v)", entityID, err)
	}

	log.Infof("listing details of Entity %x", entityID)
	return util.SendResponse(response, ctx)
}

func (m *Manager) UpdateEntity(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var entityID []byte
	var err error
	var response types.MetaResponse

	if entityID, _, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}

	var newEntity *types.EntityInfo
	entityBytes, err := base64.StdEncoding.DecodeString(ctx.URLParam("entity"))
	if err != nil {
		return fmt.Errorf("cannot decode json string: (%s): %v", ctx.URLParam("entity"), err)
	}
	if err = json.Unmarshal(entityBytes, newEntity); err != nil {
		return fmt.Errorf("cannot recover entity or no data available for entity %s: %v", entityID, err)
	}

	entityInfo := &types.EntityInfo{
		Name:  newEntity.Name,
		Email: newEntity.Email,
		// Initialize values to accept empty spaces from the UI
		CallbackURL:    "",
		CallbackSecret: "",
	}
	if len(newEntity.CallbackURL) > 0 {
		entityInfo.CallbackURL = newEntity.CallbackURL
	}
	if len(newEntity.CallbackSecret) > 0 {
		entityInfo.CallbackSecret = newEntity.CallbackSecret
	}

	// Add Entity
	if response.Count, err = m.db.UpdateEntity(entityID, entityInfo); err != nil {
		return fmt.Errorf("cannot update entity %x to the DB: (%v)", entityID, err)
	}

	log.Debugf("Entity: %x entityUpdate", entityID)
	return util.SendResponse(response, ctx)
}

func (m *Manager) ListMembers(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var entityID []byte
	var signaturePubKey []byte
	var err error
	var response types.MetaResponse

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}

	var listOptions *types.ListOptions
	if err = util.DecodeJsonMessage(listOptions, "listOptions", ctx); err != nil {
		return err
	}

	// check filter
	if err = checkOptions(listOptions, ctx.URLParam("method")); err != nil {
		return fmt.Errorf("invalid filter options %x: (%v)", signaturePubKey, err)
	}

	// Query for members
	if response.Members, err = m.db.ListMembers(entityID, listOptions); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("no members found")
		}
		return fmt.Errorf("cannot retrieve members of %x: (%v)", signaturePubKey, err)
	}

	log.Debugf("Entity: %x listMembers %d members", signaturePubKey, len(response.Members))
	return util.SendResponse(response, ctx)
}

func (m *Manager) GetMember(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var entityID []byte
	var signaturePubKey []byte
	var memberID uuid.UUID
	var err error
	var response types.MetaResponse

	if len(ctx.URLParam("memberID")) == 0 {
		return fmt.Errorf("memberID is nil on getMember")
	}

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}

	if memberID, err = uuid.Parse(ctx.URLParam("memberID")); err != nil {
		return fmt.Errorf("cannot decode memberID: (%s): %v", ctx.URLParam("memberID"), err)
	}
	if response.Member, err = m.db.Member(entityID, &memberID); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("member not found")
		}
		return fmt.Errorf("cannot retrieve member %q for entity %x: (%v)", ctx.URLParam("memberID"), signaturePubKey, err)
	}

	// TODO: Change when targets are implemented
	var targets []types.Target
	targets, err = m.db.ListTargets(entityID)
	if err == sql.ErrNoRows || len(targets) == 0 {
		log.Warnf("no targets found for member %q of entity %x", memberID.String(), signaturePubKey)
		response.Target = &types.Target{}
	} else if err == nil {
		response.Target = &targets[0]
	} else {
		return fmt.Errorf("error retrieving member %q targets for entity %x: (%v)", memberID.String(), signaturePubKey, err)
	}

	log.Infof("listing member %q for Entity with public Key %x", memberID.String(), signaturePubKey)
	return util.SendResponse(response, ctx)
}

func (m *Manager) UpdateMember(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var entityID []byte
	var signaturePubKey []byte
	var member *types.Member
	var err error
	var response types.MetaResponse

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}

	if err = util.DecodeJsonMessage(member, "member", ctx); err != nil {
		return err
	}

	// If a string Member property is sent as "" then it is not updated
	if response.Count, err = m.db.UpdateMember(entityID, &member.ID, &member.MemberInfo); err != nil {
		return fmt.Errorf("cannot update member %q for entity %x: (%v)", member.ID.String(), signaturePubKey, err)
	}

	log.Infof("update member %q for Entity with public Key %x", member.ID.String(), signaturePubKey)
	return util.SendResponse(response, ctx)
}

func (m *Manager) DeleteMembers(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var entityID []byte
	var memberIDs []uuid.UUID
	var err error
	var response types.MetaResponse

	if entityID, _, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	if err := util.DecodeJsonMessage(&memberIDs, "memberIDs", ctx); err != nil {
		return err
	}
	response.Count, response.InvalidIDs, err = m.db.DeleteMembers(entityID, memberIDs)
	if err != nil {
		return fmt.Errorf("error deleting members for entity %x: (%v)", entityID, err)
	}

	log.Infof("deleted %d members, found %d invalid tokens, for Entity with public Key %x", response.Count, len(response.InvalidIDs), entityID)
	return util.SendResponse(response, ctx)
}

func (m *Manager) CountMembers(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var entityID []byte
	var signaturePubKey []byte
	var err error
	var response types.MetaResponse

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	// Query for members
	if response.Count, err = m.db.CountMembers(entityID); err != nil {
		return fmt.Errorf("cannot count members for %x: (%v)", signaturePubKey, err)
	}

	log.Debugf("Entity %q countMembers: %d members", signaturePubKey, response.Count)
	return util.SendResponse(response, ctx)
}

func (m *Manager) GenerateTokens(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var entityID []byte
	var signaturePubKey []byte
	var amount int
	var err error
	var response types.MetaResponse

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	if err = util.DecodeJsonMessage(&amount, "amount", ctx); err != nil {
		return err
	}
	if amount < 1 {
		return fmt.Errorf("invalid token amount requested by %x", signaturePubKey)
	}

	response.Tokens = make([]uuid.UUID, amount)
	for idx := range response.Tokens {
		response.Tokens[idx] = uuid.New()
	}
	// TODO: Probably I need to initialize tokens
	if err = m.db.CreateMembersWithTokens(entityID, response.Tokens); err != nil {
		return fmt.Errorf("could not register generated tokens for %x: (%v)", signaturePubKey, err)
	}

	log.Debugf("Entity: %x generateTokens: %d tokens", signaturePubKey, len(response.Tokens))
	return util.SendResponse(response, ctx)
}

func (m *Manager) ExportTokens(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var entityID []byte
	var signaturePubKey []byte
	var members []types.Member
	var err error
	var response types.MetaResponse

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	// TODO: Probably I need to initialize tokens
	if members, err = m.db.MembersTokensEmails(entityID); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("no members found")
		}
		return fmt.Errorf("could not retrieve members tokens for %x: (%v)", signaturePubKey, err)
	}
	response.MembersTokens = make([]types.TokenEmail, len(members))
	for idx, member := range members {
		response.MembersTokens[idx] = types.TokenEmail{Token: member.ID, Email: member.Email}
	}

	log.Debugf("Entity: %x exportTokens: %d tokens", signaturePubKey, len(members))
	return util.SendResponse(response, ctx)
}

func (m *Manager) ImportMembers(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var entityID []byte
	var signaturePubKey []byte
	var membersInfo []types.MemberInfo
	var err error
	var response types.MetaResponse

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	if err = util.DecodeJsonMessage(&membersInfo, "membersInfo", ctx); err != nil {
		return err
	}
	if len(membersInfo) < 1 {
		return fmt.Errorf("no member data provided for import members by %x", signaturePubKey)
	}

	for idx := range membersInfo {
		membersInfo[idx].Origin = types.Token
	}

	// Add members
	if err = m.db.ImportMembers(entityID, membersInfo); err != nil {
		return fmt.Errorf("could not import members for %x: (%v)", signaturePubKey, err)
	}

	log.Debugf("Entity: %x importMembers: %d members", signaturePubKey, len(membersInfo))
	return util.SendResponse(response, ctx)
}

func (m *Manager) CountTargets(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var entityID []byte
	var signaturePubKey []byte
	var err error
	var response types.MetaResponse

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	// Query for members
	if response.Count, err = m.db.CountTargets(entityID); err != nil {
		return fmt.Errorf("cannot count targets for %x: (%v)", signaturePubKey, err)
	}

	log.Debugf("Entity %x countTargets: %d targets", signaturePubKey, response.Count)
	return util.SendResponse(response, ctx)
}

func (m *Manager) ListTargets(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var entityID []byte
	var signaturePubKey []byte
	var listOptions *types.ListOptions
	var err error
	var response types.MetaResponse

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	if err = util.DecodeJsonMessage(listOptions, "listOptions", ctx); err != nil {
		return err
	}
	// check filter
	if err = checkOptions(listOptions, ctx.URLParam("method")); err != nil {
		return fmt.Errorf("invalid filter options %x: (%v)", signaturePubKey, err)
	}

	// Retrieve targets
	// Implement filters in DB
	response.Targets, err = m.db.ListTargets(entityID)
	if err != nil || len(response.Targets) == 0 {
		if err == sql.ErrNoRows {
			return fmt.Errorf("no targets found")
		}
		return fmt.Errorf("cannot query targets for %x: (%v)", signaturePubKey, err)
	}

	log.Debugf("Entity: %x listTargets: %d targets", signaturePubKey, len(response.Targets))
	return util.SendResponse(response, ctx)
}

func (m *Manager) GetTarget(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var entityID []byte
	var signaturePubKey []byte
	var targetID *uuid.UUID
	var err error
	var response types.MetaResponse

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	if err = util.DecodeJsonMessage(targetID, "targetID", ctx); err != nil {
		return err
	}
	if response.Target, err = m.db.Target(entityID, targetID); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("target %q not found for %x", targetID.String(), signaturePubKey)
		}
		return fmt.Errorf("could not retrieve target for %x: %+v", signaturePubKey, err)
	}

	response.Members, err = m.db.TargetMembers(entityID, targetID)
	if err != nil {
		return fmt.Errorf("members for requested target could not be retrieved")
	}
	log.Debugf("Entity: %x getTarget: %s", signaturePubKey, targetID.String())
	return util.SendResponse(response, ctx)
}

func (m *Manager) DumpTarget(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var target *types.Target
	var signaturePubKey []byte
	var entityID []byte
	var targetID *uuid.UUID
	var err error
	var response types.MetaResponse

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	if err = util.DecodeJsonMessage(targetID, "targetID", ctx); err != nil {
		return err
	}
	if target, err = m.db.Target(entityID, targetID); err != nil || target.Name != "all" {
		if err == sql.ErrNoRows {
			return fmt.Errorf("target %q not found for %x", targetID.String(), signaturePubKey)
		}
		return fmt.Errorf("could not retrieve target for %x: (%v)", signaturePubKey, err)
	}

	// TODO: Implement DumpTargetClaims filtered directly by target filters
	if response.Claims, err = m.db.DumpClaims(entityID); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("no claims found for %x", signaturePubKey)
		}
		return fmt.Errorf("cannot dump claims for %x: (%v)", signaturePubKey, err)
	}

	log.Debugf("Entity: %x dumpTarget: %d claims", signaturePubKey, len(response.Claims))
	return util.SendResponse(response, ctx)
}

func (m *Manager) DumpCensus(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var err error
	var entityID []byte
	var censusID []byte
	var signaturePubKey []byte
	var response types.MetaResponse

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	censusID, err = util.DecodeCensusID(ctx.URLParam("censusID"), signaturePubKey)
	if err != nil {
		return fmt.Errorf("cannot decode census id %s for %x", ctx.URLParam("censusID"), entityID)
	}

	// TODO: Implement DumpTargetClaims filtered directly by target filters
	censusMembers, err := m.db.ExpandCensusMembers(entityID, censusID)
	if err != nil {
		return fmt.Errorf("cannot dump claims for %q: (%v)", entityID, err)
	}
	shuffledClaims := make([][]byte, len(censusMembers))
	shuffledIndexes := rand.Perm(len(censusMembers))
	for i, v := range shuffledIndexes {
		shuffledClaims[v] = censusMembers[i].DigestedPubKey
	}
	response.Claims = shuffledClaims

	log.Debugf("Entity: %x dumpCensus: %d claims", entityID, len(response.Claims))
	return util.SendResponse(response, ctx)
}

func (m *Manager) SendVotingLinks(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var err error
	var entityID []byte
	var processID types.HexBytes
	var memberID *uuid.UUID
	var signaturePubKey []byte
	var response types.MetaResponse

	if err = util.DecodeJsonMessage(memberID, "memberID", ctx); err != nil {
		return err
	}
	if err = util.DecodeJsonMessage(&processID, "processID", ctx); err != nil {
		return err
	}
	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	entity, err := m.db.Entity(entityID)
	if err != nil {
		return fmt.Errorf("cannot recover entity %x: (%v)", entityID, err)
	}

	censusID, err := util.DecodeCensusID(ctx.URLParam("censusID"), signaturePubKey)
	if err != nil {
		return fmt.Errorf("cannot decode census id %s for %x", ctx.URLParam("censusID"), entityID)
	}

	email := ctx.URLParam("email")
	if email != "" {
		// Individual email
		censusMember, err := m.db.EphemeralMemberInfoByEmail(entityID, censusID, email)
		if err != nil {
			return fmt.Errorf("cannot retrieve ephemeral member %s of  census %x for enity %x: (%v)", email, censusID, entityID, err)
		}
		if err := m.smtp.SendVotingLink(censusMember, entity, processID); err != nil {
			return fmt.Errorf("could not send voting link for member %q entity: (%v)", censusMember.ID, err)
		}
		log.Infof("send validation links to 1 members for Entity %x", entityID)
		response.Count = 1
		return util.SendResponse(response, ctx)
	}

	censusMembers, err := m.db.ListEphemeralMemberInfo(entityID, censusID)
	if err != nil {
		return fmt.Errorf("cannot retrieve ephemeral members of  census %x for enity %x: (%v)", censusID, entityID, err)
	}

	if len(censusMembers) == 0 {
		response.Count = 0
		return util.SendResponse(response, ctx)
	}
	// send concurrently emails
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
		return fmt.Errorf("inconsistency in number of sent emails and errors")
	}
	if len(errors) == len(censusMembers) {
		return fmt.Errorf("no validation email was sent %v", errors)
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
				return fmt.Errorf("error creating Pending tag:  %v", err)
			}
		} else {
			log.Infof("send validation links to %d members, skipped %d invalid IDs and %d errors , for Entity %x\nErrors: %v", response.Count, len(response.InvalidIDs), len(errors), entityID, errors)
			return fmt.Errorf("error retreiving Pending tag:  %v", err)
		}
	}
	_, _, err = m.db.AddTagToMembers(entityID, successUUIDs, tag.ID)
	if err != nil {
		log.Infof("send validation links to %d members, skipped %d invalid IDs and %d errors , for Entity %x\nErrors: %v", response.Count, len(response.InvalidIDs), len(errors), entityID, errors)
		return fmt.Errorf("error assinging Pending tag:  %v", err)
	}

	log.Infof("send validation links to %d members, skipped %d invalid IDs and %d errors , for Entity %x\nErrors: %v", response.Count, len(response.InvalidIDs), len(errors), entityID, errors)
	return util.SendResponse(response, ctx)
}

func (m *Manager) AddCensus(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var signaturePubKey []byte
	var targetID *uuid.UUID
	var entityID []byte
	var censusID []byte
	var census *types.CensusInfo
	var err error
	var response types.MetaResponse

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	if err = util.DecodeJsonMessage(targetID, "targetID", ctx); err != nil {
		return err
	}
	if len(targetID) == 0 {
		return fmt.Errorf("invalid target id %q for %x", targetID.String(), signaturePubKey)
	}
	censusID, err = util.DecodeCensusID(ctx.URLParam("censusID"), signaturePubKey)
	if err != nil {
		return fmt.Errorf("cannot decode census id %s for %x", ctx.URLParam("censusID"), entityID)
	}
	if len(censusID) == 0 {
		return fmt.Errorf("invalid census id %q for %x", censusID, signaturePubKey)
	}
	if err = util.DecodeJsonMessage(census, "census", ctx); err != nil {
		return err
	}
	if census == nil {
		return fmt.Errorf("invalid census info for census %q for entity %x", censusID, signaturePubKey)
	}
	// size, err := m.db.AddCensusWithMembers(entityID, censusID, request.TargetID, request.Census)
	if err := m.db.AddCensus(entityID, censusID, targetID, census); err != nil {
		return fmt.Errorf("cannot add census %q  for: %q: (%v)", censusID, entityID, err)
	}

	log.Debugf("Entity: %x addCensus: %s  ", entityID, censusID)
	return util.SendResponse(response, ctx)
}

func (m *Manager) UpdateCensus(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	// TODO Handle invalid claims
	var signaturePubKey []byte
	var entityID []byte
	var censusID []byte
	var invalidClaims [][]byte
	var census *types.CensusInfo
	var err error
	var response types.MetaResponse

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	censusID, err = util.DecodeCensusID(ctx.URLParam("censusID"), signaturePubKey)
	if err != nil {
		return fmt.Errorf("cannot decode census id %s for %x", ctx.URLParam("censusID"), entityID)
	}
	if len(censusID) == 0 {
		return fmt.Errorf("invalid census id %q for %x", censusID, signaturePubKey)
	}

	if err = util.DecodeJsonMessage(census, "census", ctx); err != nil {
		return err
	}
	if census == nil {
		return fmt.Errorf("invalid census info for census %q for entity %x", censusID, signaturePubKey)
	}
	if err = util.DecodeJsonMessage(&invalidClaims, "invalidClaims", ctx); err != nil {
		return err
	}

	if invalidClaims != nil && len(invalidClaims) > 0 {
		return fmt.Errorf("invalid claims: %v", invalidClaims)
	}

	if response.Count, err = m.db.UpdateCensus(entityID, censusID, census); err != nil {
		return fmt.Errorf("cannot update census %q for %x: (%v)", censusID, entityID, err)
	}

	log.Debugf("Entity: %x updateCensus: %s \n %v", entityID, censusID, census)
	return util.SendResponse(response, ctx)
}

func (m *Manager) GetCensus(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var signaturePubKey []byte
	var entityID []byte
	var censusID []byte
	var err error
	var response types.MetaResponse

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}

	censusID, err = util.DecodeCensusID(ctx.URLParam("censusID"), signaturePubKey)
	if err != nil {
		return fmt.Errorf("cannot decode census id %s for %x", ctx.URLParam("censusID"), entityID)
	}
	if len(censusID) == 0 {
		return fmt.Errorf("invalid census id %q for %x", censusID, signaturePubKey)
	}

	response.Census, err = m.db.Census(entityID, censusID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("census %q not found for %x", censusID, signaturePubKey)
		}
		return fmt.Errorf("error in retrieving censuses for entity %x: (%v)", signaturePubKey, err)
	}

	response.Target, err = m.db.Target(entityID, &response.Census.TargetID)
	if err != nil {
		return fmt.Errorf("census target not found")
	}

	log.Debugf("Entity: %x getCensus:%s", signaturePubKey, censusID)
	return util.SendResponse(response, ctx)
}

func (m *Manager) CountCensus(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var signaturePubKey []byte
	var entityID []byte
	var err error
	var response types.MetaResponse

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	// Query for members
	if response.Count, err = m.db.CountCensus(entityID); err != nil {
		return fmt.Errorf("cannot count censuses for %x: (%v)", signaturePubKey, err)
	}

	log.Debugf("Entity %x countCensus: %d censuses", signaturePubKey, response.Count)
	return util.SendResponse(response, ctx)
}

func (m *Manager) ListCensus(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var entityID []byte
	var signaturePubKey []byte
	var listOptions *types.ListOptions
	var err error
	var response types.MetaResponse

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	if err = util.DecodeJsonMessage(listOptions, "listOptions", ctx); err != nil {
		return err
	}

	// check filter
	if err = checkOptions(listOptions, ctx.URLParam("method")); err != nil {
		return fmt.Errorf("invalid filter options %x: (%v)", signaturePubKey, err)
	}

	// Query for members
	// TODO Implement listCensus in Db that supports filters
	response.Censuses, err = m.db.ListCensus(entityID, listOptions)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("no censuses found")
		}
		return fmt.Errorf("error in retrieving censuses for entity %x: (%v)", signaturePubKey, err)
	}
	log.Debugf("Entity: %x listCensuses: %d censuses", signaturePubKey, len(response.Censuses))
	return util.SendResponse(response, ctx)
}

func (m *Manager) DeleteCensus(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var signaturePubKey []byte
	var entityID []byte
	var censusID []byte
	var err error
	var response types.MetaResponse

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	censusID, err = util.DecodeCensusID(ctx.URLParam("censusID"), signaturePubKey)
	if err != nil {
		return fmt.Errorf("cannot decode census id %s for %x", ctx.URLParam("censusID"), entityID)
	}
	if len(censusID) == 0 {
		return fmt.Errorf("invalid census id %q for %x", censusID, signaturePubKey)
	}

	err = m.db.DeleteCensus(entityID, censusID)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("error deleting census %s for entity %x: (%v)", censusID, entityID, err)
	}

	log.Debugf("Entity: %x deleteCensus:%s", entityID, censusID)
	return util.SendResponse(response, ctx)
}

func (m *Manager) SendValidationLinks(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var entityID []byte
	var signaturePubKey []byte
	var memberIDs []uuid.UUID
	var members []types.Member
	var response types.MetaResponse
	var err error

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}

	entity, err := m.db.Entity(entityID)
	if err != nil {
		return fmt.Errorf("cannot recover %x entity: (%v)", signaturePubKey, err)
	}

	if err := util.DecodeJsonMessage(&memberIDs, "memberIDs", ctx); err != nil {
		return err
	}

	members, response.InvalidIDs, err = m.db.Members(entityID, memberIDs)
	if err != nil {
		return fmt.Errorf("cannot retrieve members for entity %x: (%v)", entityID, err)
	}

	if len(members) == 0 {
		response.Count = 0
		return util.SendResponse(response, ctx)
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
		return fmt.Errorf("inconsistency in number of sent emails and errors")
	}
	if len(errors) == len(members) {
		return fmt.Errorf("no validation email was sent %v", errors)
	}
	if len(errors) > 0 {
		response.Message = fmt.Sprintf("%d where found:\n%v", len(errors), errors)
	}
	duplicates := len(memberIDs) - len(members) - len(response.InvalidIDs)

	// add tag PendingValidation to sucessful members
	tagName := "PendingValidation"
	tag, err := m.db.TagByName(entityID, tagName)
	if err != nil {
		if err == sql.ErrNoRows {
			tag = &types.Tag{}
			tag.ID, err = m.db.AddTag(entityID, tagName)
			if err != nil {
				log.Infof("send validation links to %d members, skipped %d invalid IDs, %d duplicates and %d errors , for Entity %x\nErrors: %v", response.Count, len(response.InvalidIDs), duplicates, len(errors), entityID, errors)
				return fmt.Errorf("error creating Pending tag:  %v", err)
			}
		} else {
			log.Infof("send validation links to %d members, skipped %d invalid IDs, %d duplicates and %d errors , for Entity %x\nErrors: %v", response.Count, len(response.InvalidIDs), duplicates, len(errors), entityID, errors)
			log.Errorf("error retreiving Pending tag:  %v", err)
			return fmt.Errorf("sent emails but could not assign tag")
		}
	}
	_, _, err = m.db.AddTagToMembers(entityID, successUUIDs, tag.ID)
	if err != nil {
		log.Infof("send validation links to %d members, skipped %d invalid IDs, %d duplicates and %d errors , for Entity %x\nErrors: %v", response.Count, len(response.InvalidIDs), duplicates, len(errors), entityID, errors)
		log.Errorf("error assinging Pending tag:  %v", err)
		return fmt.Errorf("sent emails but could not assign tag")
	}

	log.Infof("send validation links to %d members, skipped %d invalid IDs, %d duplicates and %d errors , for Entity %x\nErrors: %v", response.Count, len(response.InvalidIDs), duplicates, len(errors), entityID, errors)
	return util.SendResponse(response, ctx)
}

func (m *Manager) CreateTag(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var entityID []byte
	var err error
	var response types.MetaResponse

	if ctx.URLParam("tagName") == "" {
		log.Debug("createTag with empty tag")
		return fmt.Errorf("invalid tag name")
	}

	if entityID, _, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	response.Tag = &types.Tag{
		Name: ctx.URLParam("tagName"),
	}

	if response.Tag.ID, err = m.db.AddTag(entityID, ctx.URLParam("tagName")); err != nil {
		return fmt.Errorf("cannot create tag '%s' for entity %x: (%v)", ctx.URLParam("tagName"), entityID, err)
	}

	log.Infof("created tag with id %d and name '%s' for Entity %x", response.Tag.ID, response.Tag.Name, entityID)
	return util.SendResponse(response, ctx)
}

func (m *Manager) ListTags(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var entityID []byte
	var err error
	var response types.MetaResponse

	if entityID, _, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	// Query for members
	if response.Tags, err = m.db.ListTags(entityID); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("no tags found")
		}
		return fmt.Errorf("cannot retrieve tags of %x: (%v)", entityID, err)
	}

	log.Debugf("Entity: %x list %d tags", entityID, len(response.Tags))
	return util.SendResponse(response, ctx)
}

func (m *Manager) DeleteTag(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var entityID []byte
	var tagID int32
	var err error
	var response types.MetaResponse

	if err = util.DecodeJsonMessage(&tagID, "tagID", ctx); err != nil {
		return err
	}
	if tagID == 0 {
		log.Debug("deleteTag with empty tag")
		return fmt.Errorf("invalid tag id")
	}

	if entityID, _, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	if err = m.db.DeleteTag(entityID, tagID); err != nil {
		return fmt.Errorf("cannot delete tag %d for entity %x: (%v)", tagID, entityID, err)
	}

	log.Infof("delete tag with id %d for Entity %x", tagID, entityID)
	return util.SendResponse(response, ctx)
}

func (m *Manager) AddTag(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var entityID []byte
	var tagID int32
	var memberIDs []uuid.UUID
	var err error
	var response types.MetaResponse

	if err = util.DecodeJsonMessage(&tagID, "tagID", ctx); err != nil {
		return err
	}
	if tagID == 0 {
		log.Debug("deleteTag with empty tag")
		return fmt.Errorf("invalid tag id")
	}
	if err := util.DecodeJsonMessage(&memberIDs, "memberIDs", ctx); err != nil {
		return err
	}

	if entityID, _, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	response.Count, response.InvalidIDs, err = m.db.AddTagToMembers(entityID, memberIDs, tagID)
	if err != nil {
		return fmt.Errorf("cannot add tag %d to members for entity %x: (%v)", tagID, entityID, err)
	}

	log.Infof("added tag with id %d to %d, with %d invalid IDs, members of Entity %x", tagID, response.Count, len(response.InvalidIDs), entityID)
	return util.SendResponse(response, ctx)
}

func (m *Manager) RemoveTag(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var entityID []byte
	var tagID int32
	var memberIDs []uuid.UUID
	var err error
	var response types.MetaResponse

	if err = util.DecodeJsonMessage(&tagID, "tagID", ctx); err != nil {
		return err
	}
	if tagID == 0 {
		log.Debug("deleteTag with empty tag")
		return fmt.Errorf("invalid tag id")
	}
	if err := util.DecodeJsonMessage(&memberIDs, "memberIDs", ctx); err != nil {
		return err
	}

	if entityID, _, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	response.Count, response.InvalidIDs, err = m.db.RemoveTagFromMembers(entityID, memberIDs, tagID)
	if err != nil {
		return fmt.Errorf("cannot remove tag %d from members for entity %x: (%v)", tagID, entityID, err)
	}

	log.Infof("removed tag with id %d from %d members, with %d invalid IDs, of Entity %x", tagID, response.Count, len(response.InvalidIDs), entityID)
	return util.SendResponse(response, ctx)
}

func (m *Manager) AdminEntityList(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var signaturePubKey []byte
	var entityID []byte
	var err error
	var response types.MetaResponse

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	log.Debugf("%s", ethcommon.BytesToAddress((entityID)).String())
	if ethcommon.BytesToAddress((entityID)).String() != "0xCc41C6545234ac63F11c47bC282f89Ca77aB9945" {
		log.Warnf("invalid auth: (%v)", signaturePubKey)
		return fmt.Errorf("invalid auth")
	}

	// Query for members
	if response.Entities, err = m.db.AdminEntityList(); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("no entities found")
		}
		return fmt.Errorf("cannot retrieve entities: (%v)", err)
	}

	log.Debugf("Entity: %x adminEntityList %d entities", signaturePubKey, len(response.Entities))
	return util.SendResponse(response, ctx)
}

func (m *Manager) RequestGas(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var entityID []byte
	var err error
	var response types.MetaResponse

	if m.eth == nil {
		return fmt.Errorf("cannot request for tokens, ethereum client is nil")
	}
	if entityID, _, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}

	entityAddress := ethcommon.BytesToAddress(entityID)

	// check entity exists
	if _, err := m.db.Entity(entityID); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("entity not found")
		}
		return fmt.Errorf("cannot retrieve details of entity %x: (%v)", entityID, err)
	}

	sent, err := m.eth.SendTokens(context.Background(), entityAddress, 0, 0)
	if err != nil {
		return fmt.Errorf("error sending tokens to entity %s : %v", entityAddress.String(), err)
	}

	response.Count = int(sent.Int64())
	return util.SendResponse(response, ctx)
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
