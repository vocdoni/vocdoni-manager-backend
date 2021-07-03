package testcommon

import (
	"encoding/json"
	"math/rand"
	"time"

	randomdata "github.com/Pallinder/go-randomdata"

	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/manager/types"
)

// CreateEntities a given number of random entities
func CreateEntities(size int) ([]*ethereum.SignKeys, []*types.Entity) {
	signers := CreateEthRandomKeysBatch(size)
	mp := make([]*types.Entity, size)
	for i := 0; i < size; i++ {
		// retrieve entity ID
		mp[i] = &types.Entity{
			ID: signers[i].Address().Bytes(),
			EntityInfo: types.EntityInfo{
				Email:                   randomdata.Email(),
				Name:                    randomdata.FirstName(2),
				Type:                    randomdata.Letters(5),
				Size:                    randomdata.Number(1001),
				CensusManagersAddresses: [][]byte{signers[i].Address().Bytes()},
				Origins:                 []types.Origin{types.Token},
				CallbackURL:             "",
				CallbackSecret:          "",
			},
		}
	}
	return signers, mp
}

// CreateMembers a given number of members with its entityID set to entityID
func CreateMembers(entityID []byte, size int) ([]*ethereum.SignKeys, []types.Member, error) {
	signers := CreateEthRandomKeysBatch(size)
	members := make([]types.Member, size)
	// if membersInfo not set generate random data
	for i := 0; i < size; i++ {
		members[i] = types.Member{
			EntityID: entityID,
			PubKey:   signers[i].PublicKey(),
			MemberInfo: types.MemberInfo{
				DateOfBirth:   RandDate(),
				Email:         randomdata.Email(),
				FirstName:     randomdata.FirstName(2),
				LastName:      randomdata.LastName(),
				Phone:         randomdata.PhoneNumber(),
				StreetAddress: randomdata.Address(),
				Consented:     RandBool(),
				// Verified:      RandDate(),
				Origin:       types.Token,
				CustomFields: json.RawMessage([]byte("{}")),
			},
		}
	}
	return signers, members, nil
}

// CreateEthRandomKeysBatch creates a set of eth random signing keys
func CreateEthRandomKeysBatch(n int) []*ethereum.SignKeys {
	s := make([]*ethereum.SignKeys, n)
	for i := 0; i < n; i++ {
		s[i] = ethereum.NewSignKeys()
		if err := s[i].Generate(); err != nil {
			return nil
		}
	}
	return s
}

// RandDate creates a random date
func RandDate() time.Time {
	min := time.Date(1970, 1, 0, 0, 0, 0, 0, time.UTC).Unix()
	max := time.Date(2070, 1, 0, 0, 0, 0, 0, time.UTC).Unix()
	delta := max - min
	sec := rand.Int63n(delta) + min
	return time.Unix(sec, 0)
}

// RandBool creates a random bool
func RandBool() bool {
	return rand.Float32() < 0.5
}
