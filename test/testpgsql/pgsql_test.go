package testpgsql

import (
	"bytes"
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

	qt "github.com/frankban/quicktest"
	"github.com/google/uuid"
	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/dvote/util"
	"go.vocdoni.io/manager/config"
	"go.vocdoni.io/manager/test/testcommon"
	"go.vocdoni.io/manager/types"
)

var api testcommon.TestAPI

func TestMain(m *testing.M) {
	api = testcommon.TestAPI{Port: 12000 + rand.Intn(1000)}
	db := &config.DB{
		Dbname:   "vocdonimgr",
		Password: "vocdoni",
		Host:     "127.0.0.1",
		Port:     5432,
		Sslmode:  "disable",
		User:     "vocdoni",
	}
	if err := api.Start(db, ""); err != nil {
		log.Printf("SKIPPING: could not start the API: %v", err)
		return
	}
	os.Exit(m.Run())
	if err := api.DB.Ping(); err != nil {
		log.Printf("SKIPPING: could not connect to DB: %v", err)
		return
	}
	os.Exit(m.Run())
}

func TestEntity(t *testing.T) {
	entitySigner := ethereum.NewSignKeys()
	entitySigner.Generate()

	eid := entitySigner.Address().Bytes()
	entityID := ethereum.HashRaw(eid)
	entityIDStr := hex.EncodeToString(entityID)
	t.Log(entityIDStr)

	info := &types.EntityInfo{
		// Email:                   "entity@entity.org",
		Name:                    "test entity",
		CensusManagersAddresses: [][]byte{{1, 2, 3}},
		Origins:                 []types.Origin{types.Token},
	}

	startEntities, err := api.DB.EntitiesID()
	if err != nil {
		t.Fatalf("cannot get entities from the Postgres DB (pgsql.go:Entities): %s", err)
	}

	if err = api.DB.AddEntity(entityID, info); err != nil {
		t.Fatalf("cannot add entity to the Postgres DB (pgsql.go:addEntity): %s", err)
	}

	entity, err := api.DB.Entity(entityID)
	if err != nil {
		t.Fatalf("cannot fetch entity from the Postgres DB (pgsql.go:Entity): %s", err)
	}
	marshalledEntityInfo, err := json.Marshal(entity.EntityInfo)
	if err != nil {
		t.Fatalf("cannot marshal retrieved Entity info: %s", err)
	}
	marshalledInfo, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("cannot marshal provided Entity info: %s", err)
	}
	if string(marshalledEntityInfo) != string(marshalledInfo) {
		t.Fatalf("expected %s info, but got %s", string(marshalledInfo), string(marshalledEntityInfo))
	}

	updateInfo := &types.EntityInfo{
		CensusManagersAddresses: [][]byte{{1, 2, 3}},         //same
		Origins:                 []types.Origin{types.Token}, //same
		Name:                    "test1 entity",
		CallbackURL:             "http://127.0.0.1/extapi",
		CallbackSecret:          "asdafgewgrf",
	}
	count, err := api.DB.UpdateEntity(entityID, updateInfo)
	if err != nil {
		t.Fatalf("cannot update entity Entity info: (%s)", err)
	}
	if count != 1 {
		t.Fatalf("expected to update one row but updated %d", count)
	}
	updatedEntity, err := api.DB.Entity(entityID)
	if err != nil {
		t.Fatalf("cannot fetch entity from the Postgres DB (pgsql.go:Entity): %s", err)
	}
	marshalledUpdatedEntityInfo, err := json.Marshal(updatedEntity.EntityInfo)
	if err != nil {
		t.Fatalf("cannot marshal retrieved Entity info: %s", err)
	}
	marshalleUpdatedInfo, err := json.Marshal(updateInfo)
	if err != nil {
		t.Fatalf("cannot marshal provided Entity info: %s", err)
	}
	if string(marshalleUpdatedInfo) != string(marshalledUpdatedEntityInfo) {
		t.Fatalf("expected \n%s info, but got \n%s", string(marshalleUpdatedInfo), string(marshalledUpdatedEntityInfo))
	}

	if updatedEntity.IsAuthorized {
		t.Fatalf("error entity authorized not explicitly: (%v)", err)
	}

	if err := api.DB.AuthorizeEntity(entityID); err != nil {
		t.Fatalf("error authorizing entity: (%v)", err)
	}
	updatedEntity, err = api.DB.Entity(entityID)
	if err != nil {
		t.Fatalf("cannot fetch entity from the Postgres DB (pgsql.go:Entity): %s", err)
	}
	if !updatedEntity.IsAuthorized {
		t.Fatalf("error entity not authorized even though authorizedEntity was succesful: (%v)", err)
	}

	if err := api.DB.AuthorizeEntity(entityID); err == nil {
		t.Fatalf("managed to reauthorize entity: (%v)", err)
	} else if err.Error() != "already authorized" {
		t.Fatalf("error while trying to reauthorize entity: (%v)", err)
	}

	finalEntities, err := api.DB.EntitiesID()
	if err != nil {
		t.Fatalf("cannot get entities from the Postgres DB (pgsql.go:Entities): %s", err)
	}
	if len(finalEntities)-len(startEntities) != +1 {
		t.Fatalf("expected to have created 1 new entity but created %d", len(finalEntities)-len(startEntities))
	}

	entitySigner.Generate()
	if err := api.DB.DeleteEntity(ethereum.HashRaw(entitySigner.PublicKey())); err == nil {
		t.Fatalf("could delete a random entity entity: %s", err)
	}

	if err := api.DB.DeleteEntity(entityID); err != nil {
		t.Fatalf("could not delete entity: %s", err)
	}
}

func TestUser(t *testing.T) {
	var err error
	userSigner := ethereum.NewSignKeys()
	userSigner.Generate()
	user := &types.User{PubKey: userSigner.Public.X.Bytes()}
	err = api.DB.AddUser(user)
	if err != nil {
		t.Fatalf("cannot add user to the Postgres DB (pgsql.go:addUser): %s", err)
	}
	_, err = api.DB.User(userSigner.Public.X.Bytes())
	if err != nil {
		t.Fatalf("cannot fetch user from the Postgres DB (pgsql.go:User): %s", err)
	}
}

func TestMember(t *testing.T) {
	var id uuid.UUID
	var initialCount, count int
	c := qt.New(t)
	db := api.DB
	// Create or retrieve existing entity
	// create entity
	_, entities := testcommon.CreateEntities(2)
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}

	// Create pubkey and Add membmer to the db
	memberSigner := ethereum.NewSignKeys()
	memberSigner.Generate()
	pubBytes := memberSigner.PublicKey()
	memberInfo := &types.MemberInfo{}
	memberInfo.FirstName = "Lak"
	memberInfo.LastName = "Lik"
	memberInfo.DateOfBirth.Round(time.Microsecond).UTC()
	memberInfo.Verified.Round(time.Microsecond)
	user := &types.User{PubKey: pubBytes}
	err := api.DB.AddUser(user)
	if err != nil {
		t.Fatalf("cannot add user to the Postgres DB (pgsql.go:addUser) %s", err)
	}
	if initialCount, err = api.DB.CountMembers(entities[0].ID); err != nil {
		t.Fatalf("cannot count members correctly: %+v", err)
	}

	id, err = api.DB.AddMember(entities[0].ID, pubBytes, memberInfo)
	if err != nil {
		t.Fatalf("cannot add member to the Postgres DB (pgsql.go:addMember): %s", err)
	}

	if count, err = api.DB.CountMembers(entities[0].ID); err != nil || count != initialCount+1 {
		t.Fatalf("expected %d counted: %d\ncannot count members correctly: %+v", initialCount+1, count, err)
	}

	// cannot add twice
	id2, err := api.DB.AddMember(entities[0].ID, pubBytes, memberInfo)
	if id2 != uuid.Nil {
		t.Fatalf("cannot add member twice to the Postgres DB (pgsql.go:addMember): %s", err)
	}

	// Query by ID
	member, err := db.Member(entities[0].ID, &id)
	if err != nil {
		t.Errorf("Error retrieving member from the Postgres DB (pgsql.go:Member): %s", err)
	}

	// Query by Public Key
	memberPubKey, err := db.MemberPubKey(entities[0].ID, pubBytes)
	if err != nil {
		t.Fatalf("cannot fetch member from the Postgres DB (pgsql.go:MemberPubKey): %s", err)
	}
	if memberPubKey.ID != member.ID {
		t.Fatalf("error retrieving member using MemberPubKey")
	}
	// Check first timestamps that need different handling
	// and then assing one to another so that the rest of test doesnt fail
	if !memberInfo.Verified.Equal(member.Verified) {
		t.Fatalf("expected %s verified on member info, but got %s", memberInfo.Verified, member.Verified)
	}
	if !memberInfo.DateOfBirth.Equal(member.DateOfBirth) {
		t.Fatalf("expected %s dateofbirth on member info, but got %s", memberInfo.DateOfBirth, member.DateOfBirth)
	}
	memberInfo.DateOfBirth = member.DateOfBirth
	memberInfo.Verified = member.Verified

	// Check fetchd data match send data
	marshalledMemberInfo, err := json.Marshal(member.MemberInfo)
	if err != nil {
		t.Fatalf("cannot marshal member info from query: %s", err)
	}
	marshalledInfo, err := json.Marshal(memberInfo)
	if err != nil {
		t.Fatalf("cannot marshal member info: %s", err)
	}

	if string(marshalledMemberInfo) != string(marshalledInfo) {
		t.Fatalf("expected %s marshaled member info, but got %s", marshalledMemberInfo, marshalledInfo)
	}

	// Query by UUID
	member, err = db.Member(entities[0].ID, &member.ID)
	if err != nil {
		t.Fatalf("cannot fetch user from the Postgres DB (pgsql.go:Member): %s", err)
	}
	if !bytes.Equal(member.PubKey, pubBytes) {
		t.Fatalf("expected %s member pubkey, but got %s", member.PubKey, pubBytes)
	}

	// Test SetMemberInfo
	newInfo := &types.MemberInfo{Email: "updated@mail.com", FirstName: ""}
	count, err = api.DB.UpdateMember(entities[0].ID, &member.ID, newInfo)
	if err != nil {
		t.Fatalf("cannot update user info to the Postgres DB (pgsql.go:updateMember): %s", err)
	}
	if count != 1 {
		t.Fatalf("expected to update one row but updated %d", count)
	}
	newMember, err := db.Member(entities[0].ID, &member.ID)
	if err != nil {
		t.Fatalf("cannot fetch user from the Postgres DB (pgsql.go:Member): %s", err)
	}
	if newMember.Email != "updated@mail.com" {
		t.Fatal("updateMember failed to update member Email in the Postgres DB (pgsql.go:Member)")
	}
	if newMember.FirstName != "Lak" {
		t.Fatal("updateMember with an empty string ovewrites the Name field while it shouldn't(pgsql.go:updateMember)")
	}

	// Test Bulk Info
	var bulkMembersInfo []types.MemberInfo
	for i := 0; i < 10; i++ {
		info := types.MemberInfo{FirstName: fmt.Sprintf("Name%d", i), LastName: fmt.Sprintf("LastName%d", i)}
		bulkMembersInfo = append(bulkMembersInfo, info)
	}
	err = api.DB.ImportMembers(entities[0].ID, bulkMembersInfo)
	if err != nil {
		t.Fatalf("cannot add members to Postgres DB (pgsql.go:AddMemberBulk): %s", err)
	}

	if count, err = api.DB.CountMembers(entities[0].ID); err != nil || count != initialCount+11 {
		t.Fatalf("expected %d counted: %d\ncannot count members correctly: %+v", initialCount+11, count, err)
	}

	// Test Selecting all members
	allMembers, err := api.DB.ListMembers(entities[0].ID, &types.ListOptions{})
	if err != nil {
		t.Fatalf("cannot select all members from Postgres DB (pgsql.go:ListMembers): %s", err)
	}

	// Query multiple members by []UUID
	members, invalidIDs, err := api.DB.Members(entities[0].ID, []uuid.UUID{allMembers[0].ID, allMembers[1].ID})
	if err != nil {
		t.Fatal("error retrieving members by uuid (pgsql.go:Members)")
	}
	if len(members) != 2 || len(invalidIDs) != 0 {
		t.Fatal("received unexpected results retrieving members by uuid (pgsql.go:Members)")
	}
	for _, member := range members {
		if member.ID != allMembers[0].ID && member.ID != allMembers[1].ID {
			t.Fatalf("retrieved member with ID %s while expected (%s || %s) (pgsql.go:Members)", member.ID.String(), id.String(), id2.String())
		}
	}

	tempUUID := uuid.New()
	members, invalidIDs, err = api.DB.Members(entities[0].ID, []uuid.UUID{allMembers[0].ID, allMembers[1].ID, tempUUID})
	if err != nil {
		t.Fatal("error retrieving members by uuid (pgsql.go:Members)")
	}
	if len(members) != 2 || len(invalidIDs) != 1 || invalidIDs[0] != tempUUID {
		t.Fatal("received unexpected results retrieving members by uuid (pgsql.go:Members)")
	}

	// Test Selecting filtered members
	limit := 5
	filter := &types.ListOptions{
		Skip:   2,
		Count:  limit,
		SortBy: "lastName",
		Order:  "descend",
	}
	members, err = api.DB.ListMembers(entities[0].ID, filter)
	if err != nil {
		t.Fatalf("cannot select all members from Postgres DB (pgsql.go:ListMembers): %s", err)
	}
	if len(members) > limit {
		t.Fatalf("expected limit to be less than members length, %d <= %d", len(members), limit)
	}

	// Test Selecting all members and retrieving just their uuids and emails
	tokenMembers, err := api.DB.MembersTokensEmails(entities[0].ID)
	if err != nil {
		t.Fatal("cannot fetch tokens and emails from the Prostgres DB (pgsql.go:MembersTokensEmails)")
	}
	if len(tokenMembers) != len(bulkMembersInfo) {
		t.Fatalf("Expected retrieving tokens for %d members but instead retrieved %d (pgsql.go:MembersTokensEmails)", len(allMembers), len(tokenMembers))
	}

	// Test deleting member
	// 1. Can delete existing member
	if err := api.DB.DeleteMember(entities[0].ID, &member.ID); err != nil {
		t.Fatalf("error deleting member %+v", err)
	}
	if _, err = db.Member(entities[0].ID, &member.ID); err != sql.ErrNoRows {
		t.Fatalf("error retrieving deleting member %+v", err)
	}

	// 2. Get error deleting inexisting member
	if err := api.DB.DeleteMember(entities[0].ID, &uuid.UUID{}); err == nil {
		t.Fatalf("managed to delete random member %+v", err)
	}

	// Test Register Flow
	if err := api.DB.AddEntity(entities[1].ID, &entities[1].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}
	_, registerMembers, err := testcommon.CreateMembers(entities[1].ID, 1)
	if err != nil {
		t.Fatalf("cannot create members: %s", err)
	}
	n := 5
	tokens, err := api.DB.CreateNMembers(entities[1].ID, n)
	if err != nil {
		t.Fatalf("unable to create members using CreateNMembers:  (%+v)", err)
	}
	// 0. Checking that the user did not exist already
	_, err = api.DB.User(registerMembers[0].PubKey)
	if err != sql.ErrNoRows {
		t.Fatalf("unable to retrieve registered member:  (%+v)", err)
	}
	// 1. Registering member
	if err := api.DB.RegisterMember(entities[1].ID, registerMembers[0].PubKey, &tokens[0]); err != nil {
		t.Fatalf("unable to register member using existing token:  (%+v)", err)
	}
	// 2. Checking that the member was created with right values
	registeredMember, err := api.DB.Member(entities[1].ID, &tokens[0])
	if err != nil {
		t.Fatalf("unable to retrieve registered member:  (%+v)", err)
	}
	if hex.EncodeToString(registeredMember.PubKey) != hex.EncodeToString(registerMembers[0].PubKey) {
		t.Fatalf("unable to set registered members pubkey. Expected: %q got  %s", hex.EncodeToString(registerMembers[0].PubKey), hex.EncodeToString(registeredMember.PubKey))
	}
	// 3. Checking that the corresponding User was also created
	registeredUser, err := api.DB.User(registerMembers[0].PubKey)
	if err != nil {
		t.Fatalf("unable to retrieve registered user:  (%+v)", err)
	}
	if hex.EncodeToString(registeredUser.PubKey) != hex.EncodeToString(registerMembers[0].PubKey) {
		t.Fatalf("unable to set registered users pubke %s", hex.EncodeToString(member.PubKey))
	}
	entityID := hex.EncodeToString(entities[1].ID)
	t.Logf("entityID %s", entityID)
	randToken := tokens[1].String()
	t.Logf("token: %s", randToken)
	// 4. Not allowing register of random token
	u := uuid.New()
	if err := api.DB.RegisterMember(entities[1].ID, registerMembers[0].PubKey, &u); err == nil {
		t.Fatalf("able to register member using random token:  (%+v)", err)
	}
	// 5. Not allowing using existing token with different entity ID
	if err := api.DB.RegisterMember(entities[0].ID, registerMembers[0].PubKey, &tokens[1]); err == nil {
		t.Fatalf("able to register member using existing token but to non-correspondig entity:  (%+v)", err)
	}

	members, err = api.DB.ListMembers(entities[0].ID, nil)
	if err != nil {
		t.Fatalf("cannot select all members from Postgres DB (pgsql.go:ListMembers): %s", err)
	}
	// 5. Test delete  duplicate members
	updatedCount, _, err := api.DB.DeleteMembers(entities[0].ID, []uuid.UUID{members[0].ID, members[0].ID})
	if err != nil {
		t.Fatalf("cannot delete  members from Postgres DB (pgsql.go:DeleteMembers): %s", err)
	}
	if updatedCount != 1 {
		t.Fatalf("expected to delete %d but deleted %d members", 1, updatedCount)
	}
	members = members[1:]
	memberIDs := make([]uuid.UUID, len(members))
	for i, member := range members {
		memberIDs[i] = member.ID
	}

	// Test delete members
	updatedCount, _, err = api.DB.DeleteMembers(entities[0].ID, memberIDs)
	if err != nil {
		t.Fatalf("cannot delete  members from Postgres DB (pgsql.go:DeleteMembers): %s", err)
	}
	if updatedCount != len(memberIDs) {
		t.Fatalf("expected to delete %d but deleted %d members", len(memberIDs), updatedCount)
	}
	if n, err = api.DB.CountMembers(entities[0].ID); err != nil {
		t.Fatalf("cannot count  members from Postgres DB (pgsql.go:DeleteMembers): %s", err)
	}
	if n != 0 {
		t.Fatalf("expected to find 0 members but found %d (pgsql.go:DeleteMembers)", n)
	}
	// Test delete non existing
	tempUUID = uuid.New()
	updatedCount, invalidIDs, err = api.DB.DeleteMembers(entities[0].ID, []uuid.UUID{tempUUID})
	if err != nil {
		t.Fatalf("error deleting random member from Postgres DB (pgsql.go:DeleteMembers): %s", err)
	}
	if len(invalidIDs) != 1 || invalidIDs[0] != tempUUID || updatedCount != 0 {
		t.Fatal("recieved not expected invalid token (pgsql.go:DeleteMembers)")
	}

	// test AddMemberBulk
	bulkKeys := make([][]byte, 10)
	bulkMembers := make([]types.Member, 10)
	bulkSigner := ethereum.NewSignKeys()
	for i := range bulkMembers {
		if err = bulkSigner.Generate(); err != nil {
			t.Fatalf("error generating ethereum keys: (%v)", err)
		}
		bulkMembers[i].PubKey = bulkSigner.PublicKey()
		bulkKeys[i] = bulkSigner.PublicKey()
	}

	if err := api.DB.AddMemberBulk(entities[0].ID, bulkMembers); err != nil {
		t.Fatalf("error adding membbers (pgsql.go:AddMemberBulk): (%v)", err)
	}
	// verify added users and members
	for _, key := range bulkKeys {
		user, err = api.DB.User(key)
		if err != nil || user == nil {
			t.Fatalf("could not retrieve user added using AddMemberBulk: (%v)", err)
		}
		member, err = api.DB.MemberPubKey(entities[0].ID, key)
		if err != nil || member == nil {
			t.Fatalf("could not retrieve member added using AddMemberBulk: (%v)", err)
		}
	}

	// verify that duplicated members cannot be added
	if err := api.DB.AddMemberBulk(entities[0].ID, bulkMembers); err == nil {
		t.Fatalf("managed to add duplicate membbers (pgsql.go:AddMemberBulk)")
	}

	// test DeleteMembersByKeys
	updated, invalidKeys, err := api.DB.DeleteMembersByKeys(entities[0].ID, bulkKeys[:9])
	c.Assert(err, qt.IsNil)
	c.Assert(bulkKeys[:9], qt.HasLen, updated)
	if len(invalidKeys) > 0 {
		t.Fatal("unexpected invalid keys")
	}

	// test DeleteMembersByKeys that should return only invalid keys, since everything was deleted
	updated, invalidKeys, err = api.DB.DeleteMembersByKeys(entities[0].ID, bulkKeys[:9])
	c.Assert(err, qt.IsNil)
	c.Assert(updated, qt.Equals, 0)
	c.Assert(invalidKeys, qt.HasLen, len(bulkKeys[:9]))

	// test Duplicate count
	updated, invalidKeys, err = api.DB.DeleteMembersByKeys(entities[0].ID, [][]byte{bulkKeys[9:][0], bulkKeys[9:][0]})
	c.Assert(err, qt.IsNil)
	c.Assert(invalidKeys, qt.HasLen, 0)
	c.Assert(updated, qt.Equals, 1)

	// cleaning up
	for _, entity := range entities {
		if err := api.DB.DeleteEntity(entity.ID); err != nil {
			t.Errorf("error deleting test entity: %w", err)
		}

	}

}

func TestTarget(t *testing.T) {
	var inTarget, outTarget *types.Target
	var targets []types.Target
	var targetID uuid.UUID
	var err error
	var count int

	// create entity
	_, entities := testcommon.CreateEntities(1)
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}

	// Check able to list 0 targets
	if targets, err := api.DB.ListTargets(entities[0].ID); err != nil || len(targets) != 0 {
		t.Fatalf("errors retrieving all targets: %s", err)
	}

	//Verify that 0 targets are counted
	if count, err = api.DB.CountTargets(entities[0].ID); err != nil || count != 0 {
		t.Fatalf("expected %d counted: %d\ncannot count targets correctly: %+v", 0, count, err)
	}

	inTarget = &types.Target{EntityID: entities[0].ID, Name: "all", Filters: json.RawMessage([]byte("{}"))}

	// test adding target
	if targetID, err = api.DB.AddTarget(entities[0].ID, inTarget); err != nil {
		t.Fatalf("cannot add created target into database: %s", err)
	}

	// retrieve added target
	if outTarget, err = api.DB.Target(entities[0].ID, &targetID); err != nil || outTarget.Name != inTarget.Name {
		t.Fatalf("error retrieving created target from database: %s", err)
	}

	// Check not allowing duplicate targets
	if targetID, err = api.DB.AddTarget(entities[0].ID, inTarget); err == nil {
		t.Fatalf("able to add duplicate target into database: %s", err)
	}

	// Check able to list all (1 for now) targets
	if targets, err = api.DB.ListTargets(entities[0].ID); err != nil || len(targets) != 1 {
		t.Fatalf("errors retrieving all targets: %s", err)
	}

	//Verify that 0 targets are counted
	if count, err := api.DB.CountTargets(entities[0].ID); err != nil || count != 1 {
		t.Errorf("counted %d", count)
		t.Fatalf("cannot count targets correctly: %+v", err)
	}
}

func TestCensus(t *testing.T) {
	var root, idBytes []byte
	var err error
	c := qt.New(t)
	// create entity
	_, entities := testcommon.CreateEntities(1)
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}

	// add target
	target := &types.Target{EntityID: entities[0].ID, Name: "all", Filters: json.RawMessage([]byte("{}"))}

	var targetID uuid.UUID
	targetID, err = api.DB.AddTarget(entities[0].ID, target)
	if err != nil {
		t.Fatalf("cannot add target into database: %s", err)
	}

	// Genrate ID and root
	id := util.RandomHex(len(entities[0].ID))
	if idBytes, err = hex.DecodeString(util.TrimHex(id)); err != nil {
		t.Fatalf("cannot decode random id: %s", err)
	}
	root = util.RandomBytes(32)
	name := fmt.Sprintf("census%s", strconv.Itoa(rand.Int()))
	// create census info
	censusInfo := &types.CensusInfo{
		Name:          name,
		MerkleRoot:    root,
		MerkleTreeURI: fmt.Sprintf("ipfs://%s", util.TrimHex(id)),
	}

	//Verify that 0 census are counted
	if count, err := api.DB.CountCensus(entities[0].ID); err != nil || count != 0 {
		t.Errorf("counted %d", count)
		t.Fatalf("cannot count censuses correctly: %+v", err)
	}

	if err := api.DB.AddCensus(entities[0].ID, idBytes, &targetID, censusInfo); err != nil {
		t.Fatalf("cannot add census into database: %s", err)
	}

	//Verify that census exists
	if census, err := api.DB.Census(entities[0].ID, idBytes); err != nil || census.Name != name {
		t.Fatalf("unable to recover created census: %s", err)
	}

	//Verify that one census is counted
	if count, err := api.DB.CountCensus(entities[0].ID); err != nil || count != 1 {
		t.Errorf("counted %d", count)
		t.Fatalf("cannot count censuses correctly: %+v", err)
	}

	//Verify that cannot add duplicate census
	if err := api.DB.AddCensus(entities[0].ID, idBytes, &targetID, censusInfo); err == nil {
		t.Fatal("able to create duplicate census into database")
	}

	//Verify that cannot add  census for inexisting target
	id = util.RandomHex(len(entities[0].ID))
	if idBytes, err = hex.DecodeString(util.TrimHex(id)); err != nil {
		t.Fatalf("cannot decode random id: %s", err)
	}
	if err := api.DB.AddCensus(entities[0].ID, idBytes, &uuid.UUID{}, censusInfo); err == nil {
		t.Fatal("able to create census for inexisting target (pgsql.go:AddCensus)")
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

	//Verify that one census is counted
	if count, err := api.DB.CountCensus(entities[0].ID); err != nil || count != 2 {
		t.Errorf("counted %d", count)
		t.Fatalf("cannot count censuses correctly: %+v", err)
	}

	var censuses []types.Census
	censuses, err = api.DB.ListCensus(entities[0].ID, &types.ListOptions{})
	if err != nil || len(censuses) != 2 {
		t.Fatal("unable to list censuses correctly (pgsql.go:Censuses)")
	}

	// Verify that an existing census can be deleted
	if err = api.DB.DeleteCensus(entities[0].ID, censuses[0].ID); err != nil {
		t.Fatalf("cannot delete census correctly (pgsql.go:DeleteCensus): %v", err)
	}

	census, err := api.DB.Census(entities[0].ID, censuses[0].ID)
	if err != nil && err != sql.ErrNoRows {
		t.Fatalf("error in checking wether census was deleted correctly (pgsql.go:DeleteCensus): %v", err)
	}
	if err == nil || census != nil {
		t.Fatal(" census was not deleted correctly (pgsql.go:DeleteCensus)")
	}

	//Verify that one census is counted
	if count, err := api.DB.CountCensus(entities[0].ID); err != nil || count != 1 {
		t.Errorf("counted %d", count)
		t.Fatalf("cannot count censuses correctly: %+v", err)
	}

	// Verify that a random non-existing census cannot be deleted
	randomCensusID := util.RandomHex(len(censuses[0].ID))
	randomCensusIDBytes, _ := hex.DecodeString(randomCensusID)
	if err = api.DB.DeleteCensus(entities[0].ID, randomCensusIDBytes); err == nil {
		t.Fatalf("managed to delete a census with a random generated census ID (pgsql.go:DeleteCensus): %v", err)
	}

	//Verify that one census is counted
	if count, err := api.DB.CountCensus(entities[0].ID); err != nil || count != 1 {
		t.Errorf("counted %d", count)
		t.Fatalf("cannot count censuses correctly: %+v", err)
	}

	// Test Ephemeral censuses
	// create members
	_, members, _ := testcommon.CreateMembers(entities[0].ID, 2)
	memberIDs, err := api.DB.CreateNMembers(entities[0].ID, 4)
	if err != nil {
		t.Fatalf("cannot generate random members (%v)", err)
	}
	err = api.DB.RegisterMember(entities[0].ID, members[0].PubKey, &memberIDs[0])
	if err != nil {
		t.Fatalf("cannot register member: (%v)", err)
	}
	err = api.DB.RegisterMember(entities[0].ID, members[1].PubKey, &memberIDs[1])
	if err != nil {
		t.Fatalf("cannot register member: (%v)", err)
	}
	id = util.RandomHex(len(entities[0].ID))
	idBytes, err = hex.DecodeString(util.TrimHex(id))
	if err != nil {
		t.Fatalf("cannot decode random id: %s", err)
	}
	err = api.DB.AddCensus(entities[0].ID, idBytes, &targetID, &types.CensusInfo{Name: id, Ephemeral: true})
	c.Assert(err, qt.IsNil, qt.Commentf("cannot add census"))

	// Test that members without keys are not counted
	// Verify that the corresponding members are also deleted
	censusMembers, err := api.DB.ExpandCensusMembers(entities[0].ID, idBytes)
	c.Assert(err, qt.IsNil, qt.Commentf("cannot expand census claims"))
	c.Assert(censusMembers, qt.HasLen, len(memberIDs), qt.Commentf("expected to extract %d census members but extracted %d", len(memberIDs), len(censusMembers)))

	census, err = api.DB.Census(entities[0].ID, idBytes)
	if err != nil {
		t.Fatalf("cannot retrieve census: (%v)", err)
	}
	if !census.Ephemeral {
		t.Fatal("expected ephemeral census but got non-ephemeral one")
	}
	censusClaims, err := api.DB.DumpCensusClaims(entities[0].ID, idBytes)
	if err != nil {
		t.Fatalf("cannot dump census claims: (%v)", err)
	}
	if len(censusClaims) != len(censusMembers) {
		t.Fatalf("expected to dump %d claims but dumped %d", len(censusMembers), len(censusClaims))
	}
	countEphemeral := 0
	for i, claim := range censusClaims {
		if len(claim) == 0 || fmt.Sprintf("%x", claim) != fmt.Sprintf("%x", censusMembers[i].DigestedPubKey) {
			t.Fatalf("expected digested pubKey %x but found %x", censusMembers[i].DigestedPubKey, claim)
		}
		if censusMembers[i].Ephemeral {
			countEphemeral++
		}
	}
	if countEphemeral != 2 {
		t.Fatalf("expected to find 2 ephemeral members but found %d", countEphemeral)
	}

	ephemeralMembers, err := api.DB.ListEphemeralMemberInfo(entities[0].ID, idBytes)
	if err != nil {
		t.Fatalf("cannot list ephemeral members info: (%v)", err)
	}
	if len(ephemeralMembers) != 2 {
		t.Fatalf("expected to find 2 ephemeral members but found %d", len(ephemeralMembers))
	}
	email := "test@mail.com"
	count, err := api.DB.UpdateMember(entities[0].ID, &memberIDs[2], &types.MemberInfo{Email: email})
	if err != nil {
		t.Fatalf("cannot update member: (%v)", err)
	}
	if count != 1 {
		t.Fatalf("expected to update one row but updated %d", count)
	}
	ephemeralMember, err := api.DB.EphemeralMemberInfoByEmail(entities[0].ID, idBytes, email)
	if err != nil {
		t.Fatalf("cannot retrieve ephemeral member info by email: (%v)", err)
	}
	if ephemeralMember.ID != memberIDs[2] {
		t.Fatalf("retrieved wrong ephemeral member info by email: (%v)", err)
	}

	merkleRoot := util.RandomBytes(32)
	merkleTreeUri := "ipfs://..."
	info := &types.CensusInfo{
		MerkleRoot:    merkleRoot,
		MerkleTreeURI: merkleTreeUri,
	}
	count, err = api.DB.UpdateCensus(entities[0].ID, idBytes, info)
	if err != nil {
		t.Fatalf("cannot update census: (%v)", err)
	}
	if count != 1 {
		t.Fatalf("expected to update one row but updated %d", count)
	}
	census, err = api.DB.Census(entities[0].ID, idBytes)
	if err != nil {
		t.Fatalf("cannot retrieve census: (%v)", err)
	}
	if fmt.Sprintf("%x", census.MerkleRoot) != fmt.Sprintf("%x", merkleRoot) || census.MerkleTreeURI != merkleTreeUri {
		t.Fatalf("could not update census: (%v)", err)
	}
	// merkleTreeUri = "ipfs://new"
	info = &types.CensusInfo{}
	count, err = api.DB.UpdateCensus(entities[0].ID, idBytes, info)
	if err != nil {
		t.Fatalf("cannot update census: (%v)", err)
	}
	if count != 1 {
		t.Fatalf("expected to update one row but updated %d", count)
	}
	census, err = api.DB.Census(entities[0].ID, idBytes)
	if err != nil {
		t.Fatalf("cannot retrieve census: (%v)", err)
	}
	if fmt.Sprintf("%x", census.MerkleRoot) != fmt.Sprintf("%x", merkleRoot) || census.MerkleTreeURI != merkleTreeUri {
		t.Fatalf("erroneously updated censusInfo: (%v)", err)
	}

	err = api.DB.DeleteEntity(entities[0].ID)
	if err != nil {
		t.Errorf("error cleaning up %v", err)
	}
}

func TestTags(t *testing.T) {
	// create entity
	_, entities := testcommon.CreateEntities(1)
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}

	// create members
	// Test that members without keys are not counted
	memberIDs, err := api.DB.CreateNMembers(entities[0].ID, 2)
	if err != nil {
		t.Fatalf("cannot generate random members (%v)", err)
	}

	tags, err := api.DB.ListTags(entities[0].ID)
	if err != nil {
		t.Fatalf("error retrieving entity members:  (%v)", err)
	}
	if len(tags) != 0 {
		t.Fatalf("found %d tags while waiting 0", len(tags))
	}

	// Add tag
	tagID, err := api.DB.AddTag(entities[0].ID, "TestTag")
	if err != nil {
		t.Fatalf("error creating tag:  (%v)", err)
	}
	tag, err := api.DB.Tag(entities[0].ID, tagID)
	if err != nil {
		t.Fatalf("error retrieving newly created tag:  (%v)", err)
	}

	tagByName, err := api.DB.TagByName(entities[0].ID, "TestTag")
	if err != nil {
		t.Fatalf("error retrieving newly created tag:  (%v)", err)
	}
	if tagByName.ID != tag.ID {
		t.Fatal("getting Tag and TagByName returns a different result")
	}
	// non existing tag should return sql.ErrNoRows
	_, err = api.DB.TagByName(entities[0].ID, "NonExistingTag")
	if err != sql.ErrNoRows {
		t.Fatalf("unexpected response for retrieving non-existing tag:  (%v)", err)
	}

	// list tags
	tags, err = api.DB.ListTags(entities[0].ID)
	if err != nil {
		t.Fatalf("error retrieving entity members:  (%v)", err)
	}
	if len(tags) != 1 {
		t.Fatalf("found %d tags while waiting 0", len(tags))
	}
	if tags[0].ID != tagID {
		t.Fatalf("listTags returns different tags than expected")
	}

	// Add tag to members
	added, invalidIDs, err := api.DB.AddTagToMembers(entities[0].ID, memberIDs, tag.ID)
	if err != nil || added != len(memberIDs) || len(invalidIDs) != 0 {
		t.Fatalf("unable to add member tags:  (%v)", err)
	}
	// verify tags were registered correctly
	taggedMembers, err := api.DB.ListMembers(entities[0].ID, nil)
	if err != nil {
		t.Fatalf("error retrieving entity members:  (%v)", err)
	}
	for _, member := range taggedMembers {
		// since it is the only tag of the elements it is enough to check array size
		if len(member.Tags) != 1 || member.Tags[0] != tag.ID {
			t.Fatalf("Did not update correctly member tags")
		}
	}

	// test that duplicate tag names for the same entity are not allowed
	_, err = api.DB.AddTag(entities[0].ID, "TestTag")
	if err == nil {
		t.Fatal("able to create tag with duplicate name")
	}

	// verify that the same tag cannot be added twice
	// Add tag to members
	updated, invalidIDs, err := api.DB.AddTagToMembers(entities[0].ID, memberIDs, tag.ID)
	if err != nil {
		t.Fatalf("error adding tag to members:  (%v)", err)
	}
	if updated != 0 || len(invalidIDs) != len(memberIDs) {
		t.Fatal("able to add the same member tag twice")
	}

	// verify that wrong IDs are returned as invalidIDs
	tempUUID := uuid.New()
	updated, invalidIDs, err = api.DB.AddTagToMembers(entities[0].ID, []uuid.UUID{tempUUID}, tag.ID)
	if err != nil {
		t.Fatalf("error adding tag to members:  (%v)", err)
	}
	if updated != 0 || len(invalidIDs) != 1 || invalidIDs[0] != tempUUID {
		t.Fatal("able to add tag to random member")
	}

	// verify that wrong IDs are returned as invalidIDs from RemoveTagFromMembers
	deleted, invalidIDs, err := api.DB.RemoveTagFromMembers(entities[0].ID, []uuid.UUID{tempUUID}, tag.ID)
	if err != nil {
		t.Fatalf("error removing tag from members:  (%v)", err)
	}
	if deleted != 0 || len(invalidIDs) != 1 || invalidIDs[0] != tempUUID {
		t.Fatal("able to delete tag from random member")
	}

	// delete added tag (with duplicates)
	memberIDs = append(memberIDs, memberIDs[0])
	deleted, invalidIDs, err = api.DB.RemoveTagFromMembers(entities[0].ID, memberIDs, tag.ID)
	if err != nil {
		t.Fatalf("error removing tag from members:  (%v)", err)
	}
	if deleted != len(memberIDs)-1 || len(invalidIDs) != 0 {
		t.Fatal("unexpected result removing tag from members")
	}
	// verify tags were removed correctly
	taggedMembers, err = api.DB.ListMembers(entities[0].ID, nil)
	if err != nil {
		t.Fatalf("error retrieving entity members:  (%v)", err)
	}
	for _, member := range taggedMembers {
		// since it is the only tag of the elements it is enough to check array size
		if len(member.Tags) != 0 {
			t.Fatalf("Did not delete correctly member tags")
		}
	}

	// add again tag (ignoring duplicates) and delete the tag (removing it automatically from the members that have it)
	added, invalidIDs, err = api.DB.AddTagToMembers(entities[0].ID, memberIDs, tag.ID)
	if err != nil {
		t.Fatalf("unable to add member tags:  (%v)", err)
	}
	if added != len(memberIDs)-1 || len(invalidIDs) != 0 {
		t.Fatalf("unexpexted result adding member tags:  (%v)", err)
	}
	if err = api.DB.DeleteTag(entities[0].ID, tag.ID); err != nil {
		t.Fatalf("unable to delete tag that exists for members:  (%v)", err)
	}
	taggedMembers, err = api.DB.ListMembers(entities[0].ID, nil)
	if err != nil {
		t.Fatalf("error retrieving entity members:  (%v)", err)
	}
	for _, member := range taggedMembers {
		// since it is the only tag of the elements it is enough to check array size
		if len(member.Tags) != 0 {
			t.Fatalf("Did not cascade delete correctly member tags")
		}
	}
	if _, err := api.DB.Tag(entities[0].ID, tag.ID); err == nil || err != sql.ErrNoRows {
		t.Fatalf("tag was not deleted correctly %v", err)
	}

	// add again tag and delete the tag without members using it
	tagID, err = api.DB.AddTag(entities[0].ID, "TestTag")
	if err != nil {
		t.Fatalf("error creating tag:  (%v)", err)
	}
	if err = api.DB.DeleteTag(entities[0].ID, tagID); err != nil {
		t.Fatalf("unable to delete tag that exists for members:  (%v)", err)
	}
	if _, err := api.DB.Tag(entities[0].ID, tag.ID); err == nil || err != sql.ErrNoRows {
		t.Fatalf("tag was not deleted correctly %v", err)
	}
}
