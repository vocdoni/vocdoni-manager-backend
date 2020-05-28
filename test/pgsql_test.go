package registry

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
	"time"

	"gitlab.com/vocdoni/go-dvote/crypto/ethereum"
	"gitlab.com/vocdoni/go-dvote/util"
	"gitlab.com/vocdoni/vocdoni-manager-backend/config"
	"gitlab.com/vocdoni/vocdoni-manager-backend/test/testcommon"
	"gitlab.com/vocdoni/vocdoni-manager-backend/types"
	// "github.com/google/uuid"
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
	t.Log("0")
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
	}
	t.Log("1")
	err = db.AddEntity(entityID, info)
	if err != nil {
		t.Errorf("Error adding entity to the Postgres DB (pgsql.go:addEntity): %s", err)
	}
	t.Log("Hi")

	entity, err := db.Entity(entityID)
	if err != nil {
		t.Error("Error retrieving entity frome the Postgres DB (pgsql.go:Entity)")
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
