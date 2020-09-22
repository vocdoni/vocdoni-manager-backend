package testmanager

import (
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"gitlab.com/vocdoni/go-dvote/util"
	"gitlab.com/vocdoni/manager/manager-backend/config"
	"gitlab.com/vocdoni/manager/manager-backend/test/testcommon"
	"gitlab.com/vocdoni/manager/manager-backend/types"
)

var api testcommon.TestAPI

func TestMain(m *testing.M) {
	rand.Seed(time.Now().UnixNano())
	api = testcommon.TestAPI{Port: 12000 + rand.Intn(1000)}
	db := &config.DB{
		Dbname:   "vocdonimgr",
		Password: "vocdoni",
		Host:     "127.0.0.1",
		Port:     5432,
		Sslmode:  "disable",
		User:     "vocdoni",
	}
	if err := api.Start(db, "/api"); err != nil {
		log.Printf("SKIPPING: could not start the API: %v", err)
		return
	}
	if err := api.DB.Ping(); err != nil {
		log.Printf("SKIPPING: could not connect to DB: %v", err)
		return
	}
	os.Exit(m.Run())
}

func TestSignUp(t *testing.T) {
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// create entity
	signers, entities := testcommon.CreateEntities(2)
	// create and make simple request
	var req types.MetaRequest
	req.Method = "signUp"
	resp := wsc.Request(req, signers[0])
	if !resp.Ok {
		t.Fatalf("entity singup without data failed: %+v", req)
	}
	// cannot add twice
	resp2 := wsc.Request(req, signers[0])
	if resp2.Ok {
		t.Fatal("entity must be unique, cannot add twice")
	}

	if targets, err := api.DB.ListTargets(entities[0].ID); err != nil || len(targets) != 1 {
		t.Fatal("entities \"all\" automatically created target could not be retrieved")
	}

	// verify that information gets stored correctly
	req.Method = "signUp"
	req.Entity = &types.EntityInfo{}
	req.Entity.Name = entities[1].Name
	req.Entity.Email = entities[1].Email
	resp = wsc.Request(req, signers[1])
	if !resp.Ok {
		t.Fatalf("entity singUp with data failed: %+v", req)
	}

	entity, err := api.DB.Entity(entities[1].ID)
	if err != nil {
		t.Fatal("error retrieving entity after signUp")
	}
	if entity.Name != entities[1].Name || entity.Email != entities[1].Email {
		t.Fatalf("entity signUp data were not stored correctly")
	}
}

func TestUpdateEntity(t *testing.T) {
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// Create and add entity
	signers, entities := testcommon.CreateEntities(1)
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &types.EntityInfo{Address: entities[0].Address, CensusManagersAddresses: entities[0].CensusManagersAddresses}); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}
	// update without data should fail
	var req types.MetaRequest
	req.Method = "updateEntity"
	resp := wsc.Request(req, signers[0])
	if resp.Ok {
		t.Fatalf("entity update without data succeeded: %+v", req)
	}

	// update with correct data should succeed
	req.Entity = &types.EntityInfo{
		Name:  entities[0].Name,
		Email: entities[0].Email,
	}
	resp = wsc.Request(req, signers[0])
	if !resp.Ok {
		t.Fatalf("entity update with data failed: %+v", req)
	}

	entity, err := api.DB.Entity(entities[0].ID)
	if err != nil {
		t.Fatal("error retrieving entity after signUp")
	}
	if entity.Name != entities[0].Name || entity.Email != entities[0].Email {
		t.Fatalf("entity data were not updated correctly")
	}

	// should not update data that are not allowed to be updated
	testURL := "test"
	req.Entity = &types.EntityInfo{
		Name:        "New",
		Email:       "New",
		CallbackURL: testURL,
	}
	resp = wsc.Request(req, signers[0])
	if !resp.Ok {
		t.Fatalf("entity update with data failed: %+v", req)
	}

	entity, err = api.DB.Entity(entities[0].ID)
	if err != nil {
		t.Fatal("error retrieving entity after signUp")
	}
	if entity.CallbackURL == testURL {
		t.Fatalf("entity data were updated while they should not")
	}

}

func TestListMembers(t *testing.T) {
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// create entity
	entitySigners, entities := testcommon.CreateEntities(3)
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}
	// add members
	// create members
	_, members, err := testcommon.CreateMembers(entities[0].ID, 3)
	if err != nil {
		t.Fatalf("cannot create members: %s", err)
	}
	memInfo := make([]types.Member, len(members))
	for idx, mem := range members {
		memInfo[idx] = *mem
	}
	// add members
	if err := api.DB.AddMemberBulk(entities[0].ID, memInfo); err != nil {
		t.Fatalf("cannot add members into database: %s", err)
	}
	// create and make request
	var req types.MetaRequest
	req.Method = "listMembers"
	req.ListOptions = &types.ListOptions{
		Count:  10,
		Order:  "ascend",
		Skip:   2,
		SortBy: "lastName",
	}
	// create and make request
	resp := wsc.Request(req, entitySigners[0])
	if !resp.Ok {
		t.Fatalf("request failed: %+v", req)
	}
	if len(resp.Members) != 1 {
		t.Fatalf("expected %d members, but got %d", 1, len(resp.Members))
	}

	// check members are returned ordered
	// add entity
	if err := api.DB.AddEntity(entities[1].ID, &entities[1].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}
	// add members
	// create members
	_, members, err = testcommon.CreateMembers(entities[1].ID, 10)
	if err != nil {
		t.Fatalf("cannot create members: %s", err)
	}
	memInfo = make([]types.Member, len(members))
	for idx, mem := range members {
		memInfo[idx] = *mem
	}
	// add members
	if err := api.DB.AddMemberBulk(entities[1].ID, memInfo); err != nil {
		t.Fatalf("cannot add members into database: %s", err)
	}

	req.Method = "listMembers"
	req.ListOptions = &types.ListOptions{
		Count:  0,
		Order:  "ascend",
		Skip:   0,
		SortBy: "firstName",
	}
	// create and make request
	resp = wsc.Request(req, entitySigners[1])
	if !resp.Ok {
		t.Fatalf("request failed: %+v", req)
	}
	if len(resp.Members) != 10 {
		t.Fatalf("expected %d members, but got %d", 1, len(resp.Members))
	}

	// check sqli guard (protection against sqli)
	req.Method = "listMembers"
	req.ListOptions = &types.ListOptions{
		Count:  0,
		Order:  "ascend",
		Skip:   0,
		SortBy: "*",
	}
	// create and make request
	resp = wsc.Request(req, entitySigners[1])
	if resp.Ok {
		t.Fatalf("request failed: %+v", req)
	}

	req.Method = "listMembers"
	req.ListOptions = &types.ListOptions{
		Count:  0,
		Order:  "ascend",
		Skip:   0,
		SortBy: " ",
	}
	// create and make request
	resp = wsc.Request(req, entitySigners[1])
	if resp.Ok {
		t.Fatalf("request failed: %+v", req)
	}

	req.Method = "listMembers"
	req.ListOptions = &types.ListOptions{
		Count:  0,
		Order:  "ascend",
		Skip:   0,
		SortBy: "(case/**/when/**/1=1/**/then/**/email/**/else/**/phone/**/end);",
	}
	// create and make request
	resp = wsc.Request(req, entitySigners[1])
	if resp.Ok {
		t.Fatalf("request failed: %+v", req)
	}

	t.Logf("members: %+v", resp.Members)
}

func TestGetMember(t *testing.T) {
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// create entity
	entitySigners, entities := testcommon.CreateEntities(1)
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}

	// add members
	// create members
	_, members, err := testcommon.CreateMembers(entities[0].ID, 1)
	if err != nil {
		t.Fatalf("cannot create members: %s", err)
	}

	// add member
	var memberID uuid.UUID
	if memberID, err = api.DB.AddMember(entities[0].ID, members[0].PubKey, &members[0].MemberInfo); err != nil {
		t.Fatalf("cannot add member into database: %s", err)
	}

	// create and make request
	var req types.MetaRequest
	req.Method = "getMember"
	req.MemberID = &memberID
	resp := wsc.Request(req, entitySigners[0])
	t.Log(resp)
	if !resp.Ok || hex.EncodeToString(resp.Member.PubKey) != hex.EncodeToString(members[0].PubKey) {
		t.Fatalf("request failed: %+v", req)
	}

}

func TestUpdateMember(t *testing.T) {
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// create entity
	entitySigners, entities := testcommon.CreateEntities(1)
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}

	// add members
	// create members
	_, members, err := testcommon.CreateMembers(entities[0].ID, 1)
	if err != nil {
		t.Fatalf("cannot create members: %s", err)
	}

	// add member
	if members[0].ID, err = api.DB.AddMember(entities[0].ID, members[0].PubKey, &members[0].MemberInfo); err != nil {
		t.Fatalf("cannot add member into database: %s", err)
	}

	// newMember := members[0]
	memInfo := members[0].MemberInfo
	newMember := &types.Member{}
	newMember.ID = members[0].ID
	newMember.EntityID = members[0].EntityID
	newMember.MemberInfo = memInfo
	newMember.Email = "upd"
	newMember.FirstName = "upd"
	newMember.LastName = "upd"
	newMember.StreetAddress = ""

	// create and make request
	var req types.MetaRequest
	req.Method = "updateMember"
	req.Member = newMember
	resp := wsc.Request(req, entitySigners[0])
	t.Log(resp)
	if !resp.Ok {
		t.Fatalf("request failed: %+v", req)
	}

	var member *types.Member
	if member, err = api.DB.Member(entities[0].ID, &members[0].ID); err != nil {
		t.Fatalf("cannot retrieve udpated member from database: %s", err)
	}
	if member.Email != "upd" || member.FirstName != "upd" || member.LastName != "upd" {
		t.Fatalf("cannot update member fields")
	}
	if member.StreetAddress != members[0].StreetAddress {
		t.Fatalf("updating non corresponding fields")
	}

}

func TestDeleteMember(t *testing.T) {
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// create entity
	entitySigners, entities := testcommon.CreateEntities(1)
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}

	// add members
	// create members
	_, members, err := testcommon.CreateMembers(entities[0].ID, 1)
	if err != nil {
		t.Fatalf("cannot create members: %s", err)
	}

	// add member
	if members[0].ID, err = api.DB.AddMember(entities[0].ID, members[0].PubKey, &members[0].MemberInfo); err != nil {
		t.Fatalf("cannot add member into database: %s", err)
	}

	// create and make request
	var req types.MetaRequest
	req.Method = "deleteMember"
	req.MemberID = &members[0].ID
	resp := wsc.Request(req, entitySigners[0])
	t.Log(resp)
	if !resp.Ok {
		t.Fatalf("request failed: %+v", req)
	}

	if _, err := api.DB.Member(entities[0].ID, &members[0].ID); err != sql.ErrNoRows {
		t.Fatalf("could retrieve deleted member from database: %s", err)
	}

}

func TestGenerateTokens(t *testing.T) {
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// create entity
	entitySigners, entities := testcommon.CreateEntities(1)
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into DB: %s", err)
	}
	// create and make request
	var req types.MetaRequest
	randAmount := rand.Intn(100)
	req.Amount = randAmount
	req.Method = "generateTokens"
	resp := wsc.Request(req, entitySigners[0])
	if !resp.Ok {
		t.Fatalf("request failed: %+v", req)
	}

	if len(resp.Tokens) != randAmount {
		t.Fatalf("expected %d tokens, but got %d", randAmount, len(resp.Tokens))
	}
	// another entity cannot request
}

func TestExportTokens(t *testing.T) {
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// create entity
	entitySigners, entities := testcommon.CreateEntities(1)
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}

	// create members
	_, members, err := testcommon.CreateMembers(entities[0].ID, 3)
	if err != nil {
		t.Fatalf("unable to create testing members: %s", err)
	}
	memInfo := make([]types.Member, len(members))
	for idx, mem := range members {
		memInfo[idx] = *mem
	}
	// add members
	if err := api.DB.AddMemberBulk(entities[0].ID, memInfo); err != nil {
		t.Error(err)
	}

	// Test that members with public keys are not
	var req types.MetaRequest
	req.Method = "exportTokens"
	resp := wsc.Request(req, entitySigners[0])
	if !resp.Ok {
		t.Fatalf("request failed: %+v", req)
	}
	if len(resp.MembersTokens) != 0 {
		t.Fatalf("expected 0 tokens, but got %d", len(resp.MembersTokens))
	}

	// Check that members without public keys are exported
	var importMembers []types.MemberInfo
	for i := 0; i < 10; i++ {
		info := types.MemberInfo{FirstName: fmt.Sprintf("Name%d", i), LastName: fmt.Sprintf("LastName%d", i)}
		importMembers = append(importMembers, info)
	}
	err = api.DB.ImportMembers(entities[0].ID, importMembers)
	if err != nil {
		t.Fatalf("cannot add members to Postgres DB (pgsql.go:importMembers): %s", err)
	}

	resp = wsc.Request(req, entitySigners[0])
	if !resp.Ok {
		t.Fatalf("request failed: %+v", req)
	}
	if len(resp.MembersTokens) != len(importMembers) {
		t.Fatalf("expected %d tokens, but got %d", len(importMembers), len(resp.MembersTokens))
	}

}

func TestGetTarget(t *testing.T) {
	var targetID uuid.UUID
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// create entity
	entitySigners, entities := testcommon.CreateEntities(1)
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}

	inTarget := &types.Target{EntityID: entities[0].ID, Name: "all", Filters: json.RawMessage([]byte("{}"))}

	// test adding target
	if targetID, err = api.DB.AddTarget(entities[0].ID, inTarget); err != nil {
		t.Fatalf("cannot add created target into database: %s", err)
	}
	// create and make request
	var req types.MetaRequest
	req.Method = "getTarget"
	req.TargetID = &targetID
	resp := wsc.Request(req, entitySigners[0])
	t.Log(resp)
	if !resp.Ok || resp.Target.Name != "all" {
		t.Fatalf("request failed: %+v", req)
	}

}

func TestListTargets(t *testing.T) {
	var targetID uuid.UUID
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// create entity
	entitySigners, entities := testcommon.CreateEntities(1)
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}
	inTarget := &types.Target{EntityID: entities[0].ID, Name: "all", Filters: json.RawMessage([]byte("{}"))}

	// test adding target
	if targetID, err = api.DB.AddTarget(entities[0].ID, inTarget); err != nil {
		t.Fatalf("cannot add created target into database: %s", err)
	}
	// create and make request
	var req types.MetaRequest
	req.Method = "listTargets"
	resp := wsc.Request(req, entitySigners[0])
	t.Log(resp)
	if !resp.Ok || len(resp.Targets) != 1 || resp.Targets[0].Name != "all" || resp.Targets[0].ID != targetID {
		t.Fatalf("request failed: %+v", req)
	}

}

func TestDumpTarget(t *testing.T) {
	var targetID uuid.UUID
	n := 3 // number of members
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// create entity
	entitySigners, entities := testcommon.CreateEntities(1)
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}
	t.Log(hex.EncodeToString(entities[0].ID))
	// create members
	_, members, err := testcommon.CreateMembers(entities[0].ID, n)
	if err != nil {
		t.Fatalf("cannot create members: %s", err)
	}
	memInfo := make([]types.Member, len(members))
	for idx, mem := range members {
		memInfo[idx] = *mem
	}
	// add members
	if err := api.DB.AddMemberBulk(entities[0].ID, memInfo); err != nil {
		t.Fatalf("cannot add members into database: %s", err)
	}

	inTarget := &types.Target{EntityID: entities[0].ID, Name: "all", Filters: json.RawMessage([]byte("{}"))}

	// test adding target
	if targetID, err = api.DB.AddTarget(entities[0].ID, inTarget); err != nil {
		t.Fatalf("cannot add created target into database: %s", err)
	}
	// create and make request
	var req types.MetaRequest
	req.Method = "dumpTarget"
	req.TargetID = &targetID
	resp := wsc.Request(req, entitySigners[0])
	t.Log(resp)
	if !resp.Ok || len(resp.Claims) != n {
		t.Fatalf("request failed: %+v", req)
	}
	// another entity cannot request
}

func TestImportMembers(t *testing.T) {
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// create entity
	entitySigners, entities := testcommon.CreateEntities(1)
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}
	// create members
	_, members, err := testcommon.CreateMembers(entities[0].ID, 3)
	if err != nil {
		t.Fatalf("cannot create members: %s", err)
	}
	memInfo := make([]types.MemberInfo, len(members))
	for idx, mem := range members {
		memInfo[idx] = mem.MemberInfo
	}
	// add members
	// if err := api.DB.AddMemberBulk(entities[0].ID, memInfo); err != nil {
	// 	t.Fatalf("cannot add members into database: %s", err)
	// }
	// create and make request
	var req types.MetaRequest
	req.MembersInfo = make([]types.MemberInfo, len(members))
	req.Method = "importMembers"
	for idx, mem := range members {
		req.MembersInfo[idx] = mem.MemberInfo
	}
	resp := wsc.Request(req, entitySigners[0])
	if !resp.Ok {
		t.Fatalf("request failed: %+v", req)
	}
	// another entity cannot request
}

func TestAddCensus(t *testing.T) {
	var req types.MetaRequest
	var censusInfo *types.CensusInfo
	var root, idBytes []byte
	var err error
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// create entity
	entitySigners, entities := testcommon.CreateEntities(1)
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}

	//Add target
	target := &types.Target{EntityID: entities[0].ID, Name: "all", Filters: json.RawMessage([]byte("{}"))}

	var targetID uuid.UUID
	targetID, err = api.DB.AddTarget(entities[0].ID, target)
	if err != nil {
		t.Fatalf("cannot add created target into database: %s", err)
	}

	// add members
	// create members
	_, members, err := testcommon.CreateMembers(entities[0].ID, 3)
	if err != nil {
		t.Fatalf("cannot create members: %s", err)
	}
	memInfo := make([]types.Member, len(members))
	for idx, mem := range members {
		memInfo[idx] = *mem
	}
	// add members
	if err := api.DB.AddMemberBulk(entities[0].ID, memInfo); err != nil {
		t.Fatalf("cannot add members into database: %s", err)
	}

	// Genreate ID and root
	id := util.RandomHex(len(entities[0].ID))
	if idBytes, err = hex.DecodeString(util.TrimHex(id)); err != nil {
		t.Fatalf("cannot decode randpom id: %s", err)
	}

	if root, err = hex.DecodeString(util.RandomHex(len(entities[0].ID))); err != nil {
		t.Fatalf("cannot generate root: %s", err)
	}
	name := fmt.Sprintf("census%s", strconv.Itoa(rand.Int()))
	// create census info
	censusInfo = &types.CensusInfo{
		Name:          name,
		MerkleRoot:    root,
		MerkleTreeURI: fmt.Sprintf("ipfs://%s", util.TrimHex(id)),
	}

	//Test simple add census

	req.Method = "addCensus"
	req.CensusID = id
	req.Census = censusInfo
	req.TargetID = &targetID

	resp := wsc.Request(req, entitySigners[0])
	if !resp.Ok {
		t.Fatalf("unable to create a random census: %s", resp.Message)
	}

	//Verify that census exists
	census, err := api.DB.Census(entities[0].ID, idBytes)
	if err != nil {
		t.Fatalf("unable to recover created census: %s", err)
	}
	if census.Name != name || census.Size != 3 {
		t.Fatal("census stored incorrectly")
	}

	//Test that empty censusID fails
	req.CensusID = ""
	resp = wsc.Request(req, entitySigners[0])
	if resp.Ok {
		t.Fatalf("able to create a census without censusId: %s", resp.Message)
	}

	// Test that members without keys are not counted
	if _, err = api.DB.CreateNMembers(entities[0].ID, 10); err != nil {
		t.Fatalf("cannot generate random members (%v)", err)
	}

	// Genreate ID and root
	id = util.RandomHex(len(entities[0].ID))
	if root, err = hex.DecodeString(util.RandomHex(len(entities[0].ID))); err != nil {
		t.Fatalf("cannot generate root: %s", err)
	}
	name = fmt.Sprintf("census%s", strconv.Itoa(rand.Int()))
	// create census info
	censusInfo = &types.CensusInfo{
		Name:          name,
		MerkleRoot:    root,
		MerkleTreeURI: fmt.Sprintf("ipfs://%s", util.TrimHex(id)),
	}

	req.Method = "addCensus"
	req.CensusID = id
	req.Census = censusInfo
	req.TargetID = &targetID

	resp = wsc.Request(req, entitySigners[0])
	if !resp.Ok {
		t.Fatalf("unable to create a random census: %s", resp.Message)
	}

	if census.Size != 3 {
		t.Fatal("census size  incorrect")
	}

}

func TestGetCensus(t *testing.T) {
	var req types.MetaRequest
	var censusInfo *types.CensusInfo
	var root, idBytes []byte
	var err error
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// create entity
	entitySigners, entities := testcommon.CreateEntities(1)
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}

	//Add target
	target := &types.Target{EntityID: entities[0].ID, Name: "all", Filters: json.RawMessage([]byte("{}"))}

	var targetID uuid.UUID
	targetID, err = api.DB.AddTarget(entities[0].ID, target)
	if err != nil {
		t.Fatalf("cannot add created target into database: %s", err)
	}

	// Genreate ID and root
	id := util.RandomHex(len(entities[0].ID))
	if idBytes, err = hex.DecodeString(util.TrimHex(id)); err != nil {
		t.Fatalf("cannot decode randpom id: %s", err)
	}
	if root, err = hex.DecodeString(util.RandomHex(len(entities[0].ID))); err != nil {
		t.Fatalf("cannot generate root: %s", err)
	}
	name := fmt.Sprintf("census%s", strconv.Itoa(rand.Int()))
	// create census info
	censusInfo = &types.CensusInfo{
		Name:          name,
		MerkleRoot:    root,
		MerkleTreeURI: fmt.Sprintf("ipfs://%s", util.TrimHex(id)),
	}
	err = api.DB.AddCensus(entities[0].ID, idBytes, &targetID, censusInfo)
	if err != nil {
		t.Fatal("error adding census to the db")
	}

	//Test simple get census
	req.Method = "getCensus"
	req.CensusID = id

	resp := wsc.Request(req, entitySigners[0])
	if !resp.Ok {
		t.Fatalf("unable to retrieve a random census: %s", resp.Message)
	}
	if resp.Target.ID != targetID {
		t.Fatalf("target from retrieved census does not match original target: %s", resp.Message)
	}

	//Test that empty censusID fails
	req.CensusID = ""
	resp = wsc.Request(req, entitySigners[0])
	if resp.Ok {
		t.Fatalf("able to retrieve a census without censusId: %s", resp.Message)
	}

	//Test that random censusID fails
	req.CensusID = util.RandomHex(len(entities[0].ID))
	resp = wsc.Request(req, entitySigners[0])
	if resp.Ok {
		t.Fatalf("able to retrieve a census without censusId: %s", resp.Message)
	}
}

func TestListCensus(t *testing.T) {
	var req types.MetaRequest
	var censusInfo *types.CensusInfo
	var root, idBytes []byte
	var err error
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}

	// create entity
	entitySigners, entities := testcommon.CreateEntities(1)
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}

	//Test to  get 0 censuses
	req.Method = "listCensus"
	req.ListOptions = new(types.ListOptions)

	resp := wsc.Request(req, entitySigners[0])
	if !resp.Ok || len(resp.Censuses) != 0 {
		t.Fatalf("error in requesting censuses when no censuses exist: %s", resp.Message)
	}

	//Add target
	target := &types.Target{EntityID: entities[0].ID, Name: "all", Filters: json.RawMessage([]byte("{}"))}

	var targetID uuid.UUID
	targetID, err = api.DB.AddTarget(entities[0].ID, target)
	if err != nil {
		t.Fatalf("cannot add created target into database: %s", err)
	}

	// Genreate ID and root
	id := util.RandomHex(len(entities[0].ID))
	if idBytes, err = hex.DecodeString(util.TrimHex(id)); err != nil {
		t.Fatalf("cannot decode randpom id: %s", err)
	}
	if root, err = hex.DecodeString(util.RandomHex(len(entities[0].ID))); err != nil {
		t.Fatalf("cannot generate root: %s", err)
	}
	name := fmt.Sprintf("census%s", strconv.Itoa(rand.Int()))
	// create census info
	censusInfo = &types.CensusInfo{
		Name:          name,
		MerkleRoot:    root,
		MerkleTreeURI: fmt.Sprintf("ipfs://%s", util.TrimHex(id)),
	}

	err = api.DB.AddCensus(entities[0].ID, idBytes, &targetID, censusInfo)
	if err != nil {
		t.Fatal("error adding census to the db")
	}

	//Test to  get 1 censuses
	resp = wsc.Request(req, entitySigners[0])
	if !resp.Ok || len(resp.Censuses) != 1 {
		t.Fatalf("error in requesting censuses when 1 census exists: %s", resp.Message)
	}

	//Add second census (needs second target)
	target = &types.Target{EntityID: entities[0].ID, Name: "all1", Filters: json.RawMessage([]byte("{}"))}

	targetID, err = api.DB.AddTarget(entities[0].ID, target)
	if err != nil {
		t.Fatalf("cannot add second target into database: %s", err)
	}
	id = util.RandomHex(len(entities[0].ID))
	idBytes, err = hex.DecodeString(util.TrimHex(id))
	if err != nil {
		t.Fatalf("cannot decode random id: %s", err)
	}
	censusInfo.Name = fmt.Sprintf("census%s", strconv.Itoa(rand.Int()))

	err = api.DB.AddCensus(entities[0].ID, idBytes, &targetID, censusInfo)
	if err != nil {
		t.Fatal("unable to create second census (pgsql.go:AddCensus)")
	}

	//Test simple get census
	resp = wsc.Request(req, entitySigners[0])
	if !resp.Ok || len(resp.Censuses) != 2 {
		t.Fatalf("unable to list censuses when 2 censuses exist: %s", resp.Message)
	}

	// check order
	req.ListOptions = &types.ListOptions{
		Count:  0,
		Order:  "descend",
		Skip:   0,
		SortBy: "name",
	}

	resp = wsc.Request(req, entitySigners[0])
	if !resp.Ok || len(resp.Censuses) != 2 {
		t.Fatalf("unable to list censuses when 2 censuses exist: %s", resp.Message)
	}

	req.ListOptions = &types.ListOptions{
		Count:  0,
		Order:  "descend",
		Skip:   1,
		SortBy: "name",
	}

	resp = wsc.Request(req, entitySigners[0])
	if !resp.Ok || len(resp.Censuses) != 1 {
		t.Fatalf("unable to list censuses when 1 census should be returned: %s", resp.Message)
	}
}

func TestDeleteCensus(t *testing.T) {
	var req types.MetaRequest
	var censusInfo *types.CensusInfo
	var root, idBytes []byte
	var err error
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// create entity
	entitySigners, entities := testcommon.CreateEntities(2)
	if err != nil {
		t.Fatalf("cannot create entities: %s", err)
	}
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}

	//Add target
	target := &types.Target{EntityID: entities[0].ID, Name: "all", Filters: json.RawMessage([]byte("{}"))}

	var targetID uuid.UUID
	targetID, err = api.DB.AddTarget(entities[0].ID, target)
	if err != nil {
		t.Fatalf("cannot add created target into database: %s", err)
	}

	// Genreate ID and root
	id := util.RandomHex(len(entities[0].ID))
	if idBytes, err = hex.DecodeString(util.TrimHex(id)); err != nil {
		t.Fatalf("cannot decode randpom id: %s", err)
	}
	if root, err = hex.DecodeString(util.RandomHex(len(entities[0].ID))); err != nil {
		t.Fatalf("cannot generate root: %s", err)
	}
	name := fmt.Sprintf("census%s", strconv.Itoa(rand.Int()))
	// create census info
	censusInfo = &types.CensusInfo{
		Name:          name,
		MerkleRoot:    root,
		MerkleTreeURI: fmt.Sprintf("ipfs://%s", util.TrimHex(id)),
	}

	err = api.DB.AddCensus(entities[0].ID, idBytes, &targetID, censusInfo)
	if err != nil {
		t.Fatal("error adding census to the db")
	}

	//Test that census cannot be deleted from another entity
	req.Method = "deleteCensus"
	req.CensusID = id

	resp := wsc.Request(req, entitySigners[1])
	if resp.Ok {
		t.Fatalf("able to delete census of another entity: %s", resp.Message)
	}

	//Test simple delete census
	req.Method = "deleteCensus"
	req.CensusID = id

	resp = wsc.Request(req, entitySigners[0])
	if !resp.Ok {
		t.Fatalf("unable to delete census: %s", resp.Message)
	}

	count, err := api.DB.CountCensus(entities[0].ID)
	if err != nil {
		t.Fatalf("error counting censues: %s", err)
	}
	if count != 0 {
		t.Fatalf("expected to count 0 censuses but got %d", count)
	}

	//Test that empty censusID fails
	req.CensusID = ""
	resp = wsc.Request(req, entitySigners[0])
	if resp.Ok {
		t.Fatalf("able to delete a census without censusId: %s", resp.Message)
	}

	//Test that random censusID fails
	req.CensusID = util.RandomHex(len(entities[0].ID))
	resp = wsc.Request(req, entitySigners[0])
	if resp.Ok {
		t.Fatalf("able to delete a census with a random censusId: %s", resp.Message)
	}

	//TODO test that if census had members in census_members they were removed
}

func TestSendValidationLink(t *testing.T) {
	// connect to endpoint
	wsc, err := testcommon.NewAPIConnection(fmt.Sprintf("ws://127.0.0.1:%d/api/manager", api.Port), t)
	// check connected successfully
	if err != nil {
		t.Fatalf("unable to connect with endpoint :%s", err)
	}
	// create entity
	entitySigners, entities := testcommon.CreateEntities(2)
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}
	// create members
	_, members, err := testcommon.CreateMembers(entities[0].ID, 2)
	if err != nil {
		t.Fatalf("cannot create members: %s", err)
	}

	// add unverified and verified member
	memberIDUnverified, err := api.DB.AddMember(entities[0].ID, members[0].PubKey, &members[0].MemberInfo)
	if err != nil {
		t.Fatalf("cannot add member into database: %s", err)
	}
	members[1].Verified = time.Now()
	memberIDVerified, err := api.DB.AddMember(entities[0].ID, members[1].PubKey, &members[1].MemberInfo)
	if err != nil {
		t.Fatalf("cannot add member into database: %s", err)
	}

	// Valid request for unverified member should succeed
	var req types.MetaRequest
	req.Method = "sendValidationLink"
	req.MemberID = &memberIDUnverified
	resp := wsc.Request(req, entitySigners[0])
	t.Log(resp)
	if !resp.Ok {
		t.Fatalf("failed to send validation link to unverified member: \n%v\n%v", req, resp)
	}

	// Verified member request should fail
	req.MemberID = &memberIDUnverified
	resp = wsc.Request(req, entitySigners[1])
	t.Log(resp)
	if resp.Ok {
		t.Fatalf("did not fail to send validation link to member of non-existing entity-member combination : \n%v\n%v", req, resp)
	}

	// Verified member request should fail
	req.MemberID = &memberIDVerified
	resp = wsc.Request(req, entitySigners[0])
	t.Log(resp)
	if resp.Ok {
		t.Fatalf("did not fail to send validation link to verified member : \n%v\n%v", req, resp)
	}

	// Verified member request should fail
	*req.MemberID = uuid.New()
	resp = wsc.Request(req, entitySigners[0])
	t.Log(resp)
	if resp.Ok {
		t.Fatalf("did not fail to send validation link to non-existing member : \n%v\n%v", req, resp)
	}

}
