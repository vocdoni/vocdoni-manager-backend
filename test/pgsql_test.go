package registry

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"gitlab.com/vocdoni/vocdoni-manager-backend/config"
	"gitlab.com/vocdoni/vocdoni-manager-backend/test/testcommon"
	"gitlab.com/vocdoni/vocdoni-manager-backend/types"
)

var api testcommon.TestAPI

func TestMain(t *testing.M) {
	api = testcommon.TestAPI{}
	db := &config.DB{Dbname: "vocdonimgr",
		Password: "vocdoni",
		Host:     "127.0.0.1",
		Port:     5432,
		Sslmode:  "disable",
		User:     "vocdoni"}
	api.Start(db, nil)
	time.Sleep(2 * time.Second)
	os.Exit(t.Run())
}

func TestEntity(t *testing.T) {
	var db = api.DB
	entityID := "0x12345123451234"
	info := &types.EntityInfo{
		Address:                 "0x123847192347",
		Email:                   "entity@entity.org",
		Name:                    "test entity",
		CensusManagersAddresses: []string{"0x0", "0x0"},
	}
	err := db.AddEntity(entityID, info)
	if err != nil {
		t.Error("Could not add entity")
	}

	entity, err := db.Entity(entityID)
	if err != nil {
		t.Error("Could not get entity")
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

func TestMember(t *testing.T) {
	t.Error("TODO: Unimplemented")
	// var db = api.DB
	// entityID := "0x12345123451234"
	// info := &types.EntityInfo{
	// 	Address: "0x123847192347",
	// 	Email: "entity@entity.org",
	// 	Name: "test entity",
	// 	CensusManagersAddresses: []string{"0x0", "0x0"},
	// }
	// err := db.AddEntity(entityID, info)
	// if err != nil {
	// 	t.Error("Could not add entity")
	// }

	// entity, err := db.Entity(entityID)
	// if err != nil {
	// 	t.Error("Could not get entity")
	// }
	// member, _ := db.Member(uuid.New())
	// db.AddMember(entity.ID, member)
	// if !resp.Ok {
	// 	t.Error(resp.Message)
	// }
}
