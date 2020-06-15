package testdb

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"gitlab.com/vocdoni/go-dvote/crypto/snarks"
	"gitlab.com/vocdoni/go-dvote/util"
	"gitlab.com/vocdoni/vocdoni-manager-backend/types"
)

/*
  entity PrivKey: 8023d33644cba3fdd6858ff28cb631818254b8d6baa34a6e98ec3406d4f30b9f
  entity PubKey: 02ed03e6408e34af72a0e062a50cd9e77997c6c0eded5835b7367bb5695e844bf4
  member PrivKey: d37aa0d6865f1b8ea146dc63b4ee797d504a12329686b946851f4af80432a21a
  member PubKey: 020be846bab70b4eff964d74178187832b3c7866f8509de340b6fccc53032834c6
*/

// uuid.Parse("11111111-8888-3333-2222-999999999999")

type Database struct {
}

func New() (*Database, error) {
	return &Database{}, nil
}

func (d *Database) Close() error {
	return nil
}

func (d *Database) Entity(entityID []byte) (*types.Entity, error) {
	var entity types.Entity
	entity.ID = entityID
	eid, err := hex.DecodeString(util.TrimHex("b662e6ac6e8300f0a03b33c4f8510121ba2d5bde"))
	if err != nil {
		return nil, fmt.Errorf("error decoding entity address: %s", err)
	}
	entity.Address = eid
	managerAddresses, err := hex.DecodeString("02ed03e6408e34af72a0e062a50cd9e77997c6c0eded5835b7367bb5695e844bf4")
	if err != nil {
		return nil, fmt.Errorf("error decoding manager address: %s", err)
	}
	entity.CensusManagersAddresses = [][]byte{managerAddresses}
	entity.Name = "test entity"
	entity.Email = "entity@entity.org"
	return &entity, nil
}

func (d *Database) EntityHas(entityID []byte, memberID uuid.UUID) bool {
	return true
}

func (d *Database) EntityOrigins(entityID []byte) ([]types.Origin, error) {
	return nil, nil
}

func (d *Database) AddEntity(entityID []byte, info *types.EntityInfo) error {
	return nil
}

func (d *Database) Member(memberID uuid.UUID) (*types.Member, error) {
	var member types.Member
	member.ID = memberID
	eid, err := hex.DecodeString(util.TrimHex("b662e6ac6e8300f0a03b33c4f8510121ba2d5bde"))
	if err != nil {
		return nil, fmt.Errorf("error decoding entity address: %s", err)
	}
	member.EntityID = eid
	member.Email = "hello@vocdoni.io"
	member.FirstName = "Julian"
	member.LastName = "Assange"
	member.Phone = "+441827738192"
	member.PubKey = []byte("020be846bab70b4eff964d74178187832b3c7866f8509de340b6fccc53032834c6")
	member.DateOfBirth = time.Time{}
	member.StreetAddress = "Yolo St. 550"
	return &member, nil
}

func (d *Database) MemberPubKey(pubKey, entityID []byte) (*types.Member, error) {
	var member types.Member
	member.ID = uuid.New()
	member.EntityID = entityID
	member.Email = "hello@vocdoni.io"
	member.FirstName = "Julian"
	member.LastName = "Assange"
	member.Phone = "+441827738192"
	member.PubKey = []byte("020be846bab70b4eff964d74178187832b3c7866f8509de340b6fccc53032834c6")
	member.DateOfBirth = time.Time{}
	member.StreetAddress = "Yolo St. 550"
	return &member, nil
}

func (d *Database) ListMembers(entityID []byte, filter *types.ListOptions) ([]types.Member, error) {
	return nil, nil
}

func (d *Database) Census(censusID []byte) (*types.Census, error) {
	var census types.Census
	census.ID = []byte("0x0")
	return &census, nil
}

func (d *Database) AddMember(entityID, pubKey []byte, info *types.MemberInfo) error {
	// return &types.Member{MemberInfo: *info, ID: uuid.New(), EntityID: entityID, PubKey: pubKey}, nil
	return nil
}

func (d *Database) AddMemberBulk(entityID []byte, info []types.MemberInfo) error {
	return nil
}

func (d *Database) UpdateMember(memberID uuid.UUID, pubKey []byte, info *types.MemberInfo) error {
	return nil
}

func (d *Database) CreateMembersWithTokens(entityID []byte, tokens []uuid.UUID) error {
	return nil
}

func (d *Database) MembersTokensEmails(entityID []byte) ([]types.Member, error) {
	return nil, nil
}

func (d *Database) AddUser(user *types.User) error {
	return nil
}

func (d *Database) User(pubKey []byte) (*types.User, error) {
	return &types.User{
		PubKey:         pubKey,
		DigestedPubKey: snarks.Poseidon.Hash(pubKey),
	}, nil
}
