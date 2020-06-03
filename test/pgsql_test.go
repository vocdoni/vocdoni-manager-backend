package registry

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
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

func TestMain(t *testing.M) {
	api = testcommon.TestAPI{}
	db := &config.DB{
		Dbname:   "vocdonimgr",
		Password: "vocdoni",
		Host:     "127.0.0.1",
		Port:     5432,
		Sslmode:  "disable",
		User:     "vocdoni",
	}
	api.Start(db, nil)
	os.Exit(t.Run())
}

func TestEntity(t *testing.T) {
	db := api.DB
	entitySigner := new(ethereum.SignKeys)
	entitySigner.Generate()
	entityAddress := entitySigner.EthAddrString()
	eid, err := hex.DecodeString(util.TrimHex(entityAddress))
	if err != nil {
		t.Errorf("error decoding entity address: %s", err)
	}

	entityID := ethereum.HashRaw(eid)
	info := &types.EntityInfo{
		Address: eid,
		// Email:                   "entity@entity.org",
		Name:                    "test entity",
		CensusManagersAddresses: [][]byte{{1, 2, 3}},
		Origins:                 []types.Origin{types.Token.Origin()},
	}

	err = db.AddEntity(entityID, info)
	if err != nil {
		t.Errorf("Error adding entity to the Postgres DB (pgsql.go:addEntity): %s", err)
	}

	entity, err := db.Entity(entityID)
	if err != nil {
		t.Error("Error retrieving entity from the Postgres DB (pgsql.go:Entity)")
	}
	marshalledEntityInfo, err := json.Marshal(entity.EntityInfo)
	marshalledInfo, err := json.Marshal(info)
	if err != nil {
		t.Error("Error marshaling Entity info")
	}
	if string(marshalledEntityInfo) != string(marshalledInfo) {
		t.Error("Entity info not stored correctly in the DB")
	}
}

func TestUser(t *testing.T) {
	db := api.DB
	userSigner := new(ethereum.SignKeys)
	userSigner.Generate()
	user := &types.User{PubKey: userSigner.Public.X.Bytes()}
	err := db.AddUser(user)
	if err != nil {
		t.Errorf("Error adding user to the Postgres DB (pgsql.go:addUser): %s", err)
	}
	_, err = db.User(userSigner.Public.X.Bytes())
	if err != nil {
		t.Errorf("Error retrieving user from the Postgres DB (pgsql.go:User): %s", err)
	}
}

func TestMember(t *testing.T) {
	db := api.DB
	// Create Entity
	entity, err := createEntity(db)
	if err != nil {
		t.Errorf("Error creating entity the Postgres DB (pgsql.go:addEntity): %s", err)
	}

	// Create pubkey and Add membmer to the db
	memberSigner := new(ethereum.SignKeys)
	memberSigner.Generate()
	memberInfo := &types.MemberInfo{}
	memberInfo.DateOfBirth.Round(time.Microsecond).UTC()
	memberInfo.Verified.Round(time.Microsecond)
	err = db.AddMember(entity.ID, memberSigner.Public.X.Bytes(), memberInfo)
	if err != nil {
		t.Errorf("Error adding user to the Postgres DB (pgsql.go:addMember): %s", err)
	}

	// Query by Public Key
	member, err := db.MemberPubKey(memberSigner.Public.X.Bytes())
	if err != nil {
		t.Errorf("Error retrieving user from the Postgres DB (pgsql.go:MemberPubKey): %s", err)
	}
	// Check first timestamps that need different handling
	// and then assing one to another so that the rest of test doesnt fail
	if !memberInfo.Verified.Equal(member.Verified) {
		t.Error("Timestamps Error (verified)")
	}
	if !memberInfo.DateOfBirth.Equal(member.DateOfBirth) {
		t.Error("Timestamps Error (DateOfBirth)")
	}
	memberInfo.DateOfBirth = member.DateOfBirth
	memberInfo.Verified = member.Verified

	// Check retrieved data match send data
	marshalledMemberInfo, err := json.Marshal(member.MemberInfo)
	if err != nil {
		t.Error("Error marshaling member info from query")
	}
	marshalledInfo, err := json.Marshal(memberInfo)
	if err != nil {
		t.Error("Error marshaling member info")
	}

	if string(marshalledMemberInfo) != string(marshalledInfo) {
		t.Error("Member info not stored correctly in the DB")
	}

	// Query by UUID
	member, err = db.Member(member.ID)
	if err != nil {
		t.Errorf("Error retrieving user from the Postgres DB (pgsql.go:Member): %s", err)
	}
	if !bytes.Equal(member.PubKey, memberSigner.Public.X.Bytes()) {
		t.Error("Member public not stored correctly in the Postgres DB (pgsql.go:Member):")
	}

	// Test SetMemberInfo
	newInfo := &types.MemberInfo{Consented: true}
	err = db.SetMemberInfo(member.ID, newInfo)
	if err != nil {
		t.Errorf("Error updating user info to the Postgres DB (pgsql.go:setMemberInfo): %s", err)
	}
	newMember, err := db.Member(member.ID)
	if err != nil {
		t.Errorf("Error retrieving user from the Postgres DB (pgsql.go:Member): %s", err)
	}
	if newMember.Consented != true {
		t.Error("setMemberInfo failed to update member Consent in the Postgres DB (pgsql.go:Member)")
	}
}

func createEntity(db database.Database) (*types.Entity, error) {
	entitySigner := new(ethereum.SignKeys)
	entitySigner.Generate()
	entityAddress := entitySigner.EthAddrString()
	eid, err := hex.DecodeString(util.TrimHex(entityAddress))
	if err != nil {
		return nil, err
	}
	entityID := ethereum.HashRaw(eid)
	info := &types.EntityInfo{
		Address: eid,
		// Email:                   "entity@entity.org",
		Name:                    "test entity",
		CensusManagersAddresses: [][]byte{{1, 2, 3}},
		Origins:                 []types.Origin{types.Token.Origin()},
	}
	entity := &types.Entity{ID: ethereum.HashRaw(eid), EntityInfo: *info}
	err = db.AddEntity(entityID, info)
	if err != nil {
		return nil, err
	}
	return entity, nil
}
