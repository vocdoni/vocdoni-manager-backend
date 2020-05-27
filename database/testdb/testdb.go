package testdb

import (
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"gitlab.com/vocdoni/vocdoni-manager-backend/types"
)

/*
  entity PrivKey: 8023d33644cba3fdd6858ff28cb631818254b8d6baa34a6e98ec3406d4f30b9f
  entity PubKey: 02ed03e6408e34af72a0e062a50cd9e77997c6c0eded5835b7367bb5695e844bf4
  member PrivKey: d37aa0d6865f1b8ea146dc63b4ee797d504a12329686b946851f4af80432a21a
  member PubKey: 020be846bab70b4eff964d74178187832b3c7866f8509de340b6fccc53032834c6
*/

//uuid.Parse("11111111-8888-3333-2222-999999999999")

type Database struct {
}

func New(host string, port int, user, password, dbname string, sslmode string) (*Database, error) {
	return &Database{}, nil
}

func (d *Database) Close() error {
	return nil
}

func (d *Database) Entity(entityID string) (*types.Entity, error) {
	var entity types.Entity
	entity.ID = entityID
	entity.Address = "b662e6ac6e8300f0a03b33c4f8510121ba2d5bde"
	entity.CensusManagersAddresses = []string{"02ed03e6408e34af72a0e062a50cd9e77997c6c0eded5835b7367bb5695e844bf4"}
	entity.Name = "test entity"
	entity.Email = "entity@entity.org"
	return &entity, nil
}

func (d *Database) EntityHas(entityID string, memberID uuid.UUID) bool {
	return true
}

func (d *Database) CreateEntity(entityID string, info *types.EntityInfo) (*types.Entity, error) {
	return d.Entity("b662e6ac6e8300f0a03b33c4f8510121ba2d5bde")
}

func (d *Database) Member(memberID uuid.UUID) (*types.Member, error) {
	var member types.Member
	member.ID = memberID
	member.EntityID = "12345123451234"
	member.Email = "hello@vocdoni.io"
	member.FirstName = "Julian"
	member.LastName = "Assange"
	member.Phone = "+441827738192"
	member.PubKey = "020be846bab70b4eff964d74178187832b3c7866f8509de340b6fccc53032834c6"
	member.DateOfBirth = time.Time{}
	member.StreetAddress = "Yolo St. 550"
	return &member, nil
}

func (d *Database) Census(censusID string) (*types.Census, error) {
	var census types.Census
	census.ID = uuid.New().String()
	return &census, nil
}

func (d *Database) CreateMember(entityID, pubKey string, info *types.MemberInfo) (*types.Member, error) {
	return &types.Member{MemberInfo: *info, ID: uuid.New(), EntityID: entityID, PubKey: pubKey}, nil
}
