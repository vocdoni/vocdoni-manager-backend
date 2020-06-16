package testcommon

import (
	"encoding/hex"
	"encoding/json"
	"math/rand"
	"time"

	randomdata "github.com/Pallinder/go-randomdata"

	"gitlab.com/vocdoni/go-dvote/crypto/ethereum"
	// dvote "gitlab.com/vocdoni/go-dvote/util"
	"gitlab.com/vocdoni/vocdoni-manager-backend/types"
	"gitlab.com/vocdoni/vocdoni-manager-backend/util"
)

// CreateEntities a given number of random entities
func CreateEntities(size int) ([]*ethereum.SignKeys, []*types.Entity, error) {
	var entityID, entityAddress []byte
	var err error
	signers := CreateEthRandomKeysBatch(size)
	// ethereum.HashRaw(signers[0].EthAddrString())
	mp := make([]*types.Entity, size)
	for i := 0; i < size; i++ {
		// retrieve entity ID
		if entityAddress, err = util.SignerEntityAddress(*signers[i]); err != nil {
			return nil, nil, err
		}
		if entityID, err = util.SignerEntityID(*signers[i]); err != nil {
			return nil, nil, err
		}
		mp[i] = &types.Entity{
			ID: entityID,
			EntityInfo: types.EntityInfo{
				Address:                 entityAddress,
				Email:                   randomdata.Email(),
				Name:                    randomdata.FirstName(2),
				CensusManagersAddresses: [][]byte{{1, 2, 3}},
				Origins:                 []types.Origin{types.Token},
			},
		}
	}
	return signers, mp, nil
}

// CreateMembers a given number of members with its entityID set to entityID
func CreateMembers(entityID []byte, size int) ([]*ethereum.SignKeys, []*types.Member, error) {
	signers := CreateEthRandomKeysBatch(size)
	members := make([]*types.Member, size)
	// if membersInfo not set generate random data
	for i := 0; i < size; i++ {
		pub, _ := signers[i].HexString()
		pubBytes, err := hex.DecodeString(pub)
		if err != nil {
			return nil, nil, err
		}
		members[i] = &types.Member{
			EntityID: entityID,
			PubKey:   pubBytes,
			MemberInfo: types.MemberInfo{
				DateOfBirth:   RandDate(),
				Email:         randomdata.Email(),
				FirstName:     randomdata.FirstName(2),
				LastName:      randomdata.LastName(),
				Phone:         randomdata.PhoneNumber(),
				StreetAddress: randomdata.Address(),
				Consented:     RandBool(),
				Verified:      RandDate(),
				Origin:        types.Token,
				CustomFields:  json.RawMessage([]byte{110, 117, 108, 108}),
			},
		}
	}
	return signers, members, nil
}

// CreateEthRandomKeysBatch creates a set of eth random signing keys
func CreateEthRandomKeysBatch(n int) []*ethereum.SignKeys {
	s := make([]*ethereum.SignKeys, n)
	for i := 0; i < n; i++ {
		s[i] = new(ethereum.SignKeys)
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
