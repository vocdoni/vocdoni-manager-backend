package testpgsql

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"gitlab.com/vocdoni/go-dvote/crypto/ethereum"
	"gitlab.com/vocdoni/go-dvote/util"
	"gitlab.com/vocdoni/vocdoni-manager-backend/config"
	"gitlab.com/vocdoni/vocdoni-manager-backend/database"
	"gitlab.com/vocdoni/vocdoni-manager-backend/test/testcommon"
	"gitlab.com/vocdoni/vocdoni-manager-backend/types"
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
	api.Start(db, "")
	os.Exit(m.Run())
}

func TestEntity(t *testing.T) {
	entitySigner := new(ethereum.SignKeys)
	entitySigner.Generate()
	entityAddress := entitySigner.EthAddrString()
	eid, err := hex.DecodeString(util.TrimHex(entityAddress))
	if err != nil {
		t.Fatalf("cannot decode entity address: %s", err)
	}

	entityID := ethereum.HashRaw(eid)
	info := &types.EntityInfo{
		Address: eid,
		// Email:                   "entity@entity.org",
		Name:                    "test entity",
		CensusManagersAddresses: [][]byte{{1, 2, 3}},
		Origins:                 []types.Origin{types.Token},
	}

	err = api.DB.AddEntity(entityID, info)
	if err != nil {
		t.Fatalf("cannot add entity to the Postgres DB (pgsql.go:addEntity): %s", err)
	}

	entity, err := api.DB.Entity(entityID)
	if err != nil {
		t.Fatalf("cannot fetch entity from the Postgres DB (pgsql.go:Entity): %s", err)
	}
	marshalledEntityInfo, err := json.Marshal(entity.EntityInfo)
	marshalledInfo, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("cannot marshal Entity info: %s", err)
	}
	if string(marshalledEntityInfo) != string(marshalledInfo) {
		t.Fatalf("expected %s info, but got %s", string(marshalledEntityInfo), string(marshalledInfo))
	}
}

func TestUser(t *testing.T) {
	var err error
	userSigner := new(ethereum.SignKeys)
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
	// Create or fetch existing entity
	entityAddress := "30ed83726db2f7d28a58ecf0071b7dcd08f7b1e2"
	entity, err := loadOrGenEntity(entityAddress, api.DB)
	if err != nil {
		t.Fatalf("cannot create or fetch entity address: %s", err)
	}

	// Create pubkey and Add membmer to the db
	memberSigner := new(ethereum.SignKeys)
	memberSigner.Generate()
	memberInfo := &types.MemberInfo{}
	memberInfo.DateOfBirth.Round(time.Microsecond).UTC()
	memberInfo.Verified.Round(time.Microsecond)
	user := &types.User{PubKey: memberSigner.Public.X.Bytes()}
	err = api.DB.AddUser(user)
	if err != nil {
		t.Fatalf("cannot add user to the Postgres DB (pgsql.go:addUser) %s", err)
	}
	err = api.DB.AddMember(entity.ID, memberSigner.Public.X.Bytes(), memberInfo)
	if err != nil {
		t.Fatalf("cannot add member to the Postgres DB (pgsql.go:addMember): %s", err)
	}

	// Query by Public Key
	member, err := api.DB.MemberPubKey(memberSigner.Public.X.Bytes(), entity.ID)
	if err != nil {
		t.Fatalf("cannot fetch member from the Postgres DB (pgsql.go:MemberPubKey): %s", err)
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
	member, err = api.DB.Member(member.ID)
	if err != nil {
		t.Fatalf("cannot fetch user from the Postgres DB (pgsql.go:Member): %s", err)
	}
	if !bytes.Equal(member.PubKey, memberSigner.Public.X.Bytes()) {
		t.Fatalf("expected %s member pubkey, but got %s", member.PubKey, memberSigner.Public.X.Bytes())
	}

	// Test SetMemberInfo
	newInfo := &types.MemberInfo{Consented: true}
	err = api.DB.UpdateMember(member.ID, nil, newInfo)
	if err != nil {
		t.Fatalf("cannot update user info to the Postgres DB (pgsql.go:setMemberInfo): %s", err)
	}
	newMember, err := api.DB.Member(member.ID)
	if err != nil {
		t.Fatalf("cannot fetch user from the Postgres DB (pgsql.go:Member): %s", err)
	}
	if newMember.Consented != true {
		t.Fatal("setMemberInfo failed to update member Consent in the Postgres DB (pgsql.go:Member)")
	}

	// Test Bulk Info
	var bulkMembers []types.MemberInfo
	for i := 0; i < 10; i++ {
		info := types.MemberInfo{FirstName: fmt.Sprintf("Name%d", i), LastName: fmt.Sprintf("LastName%d", i)}
		bulkMembers = append(bulkMembers, info)
	}
	err = api.DB.AddMemberBulk(entity.ID, bulkMembers)
	if err != nil {
		t.Fatalf("cannot add members to Postgres DB (pgsql.go:AddMemberBulk): %s", err)
	}

	// Test Selecting all members
	allMembers, err := api.DB.ListMembers(entity.ID, &types.ListOptions{})
	if err != nil {
		t.Fatalf("cannot select all members from Postgres DB (pgsql.go:MembersFiltered): %s", err)
	}

	// Test Selecting filtered members
	limit := 5
	filter := &types.ListOptions{
		Skip:   2,
		Count:  limit,
		SortBy: "lastName",
		Order:  "desc",
	}
	members, err := api.DB.ListMembers(entity.ID, filter)
	if len(members) > limit {
		t.Fatalf("expected limit to be less than members length, %d <= %d", len(members), limit)
	}

	// Test Selecting all members and retrieving just their uuids and emails
	tokenMembers, err := api.DB.MembersTokensEmails(entity.ID)
	if len(tokenMembers) != len(allMembers) {
		t.Fatal("cannot fetch tokens and emails from the Prostgres DB (pgsql.go:MembersTokensEmails")
	}
}

func loadOrGenEntity(address string, db database.Database) (*types.Entity, error) {
	eid, err := hex.DecodeString(util.TrimHex(address))
	if err != nil {
		return nil, err
	}
	entityID := ethereum.HashRaw(eid)
	entity, err := api.DB.Entity(entityID)
	if err != nil {
		info := &types.EntityInfo{
			Address: eid,
			// Email:                   "entity@entity.org",
			Name:                    "test entity",
			CensusManagersAddresses: [][]byte{{1, 2, 3}},
			Origins:                 []types.Origin{types.Token},
		}
		entity = &types.Entity{ID: ethereum.HashRaw(eid), EntityInfo: *info}
		err = api.DB.AddEntity(entityID, info)
		if err != nil {
			return nil, err
		}
	}
	return entity, nil
}
