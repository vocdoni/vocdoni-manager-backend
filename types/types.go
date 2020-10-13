package types

import (
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gitlab.com/vocdoni/go-dvote/util"
)

type CreatedUpdated struct {
	CreatedAt time.Time `json:"createdAt,omitempty" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt,omitempty" db:"updated_at"`
}

type Entity struct {
	CreatedUpdated
	ID           []byte `json:"id" db:"id"`
	IsAuthorized bool   `json:"isAuthorized" db:"is_authorized"`
	EntityInfo
}

type EntityInfo struct {
	Address                 []byte   `json:"address" db:"address"`
	CallbackURL             string   `json:"callbackUrl" db:"callback_url"`
	CallbackSecret          string   `json:"callbackSecret" db:"callback_secret"`
	Email                   string   `json:"email,omitempty" db:"email"`
	Name                    string   `json:"name" db:"name"`
	CensusManagersAddresses [][]byte `json:"censusManagersAddresses,omitempty" db:"census_managers_addresses"`
	Origins                 []Origin `json:"origins" db:"origins"`
}

//go:generate stringer -type=Origin
type Origin int

const (
	Token Origin = iota
	Form
	DB
)

func ToOrigin(origin string) Origin {
	switch origin {
	case "Token":
		return Token
	case "Form":
		return Form
	case "DB":
		return DB
	default:
		return -1
	}
}

type Member struct {
	CreatedUpdated
	ID       uuid.UUID `json:"id" db:"id"`
	EntityID []byte    `json:"entityId" db:"entity_id"`
	PubKey   []byte    `json:"publicKey,omitempty" db:"public_key"`
	MemberInfo
}

type MemberInfo struct {
	DateOfBirth   time.Time       `json:"dateOfBirth,omitempty" db:"date_of_birth"`
	Email         string          `json:"email,omitempty" db:"email"`
	FirstName     string          `json:"firstName,omitempty" db:"first_name"`
	LastName      string          `json:"lastName,omitempty" db:"last_name"`
	Phone         string          `json:"phone,omitempty" db:"phone"`
	StreetAddress string          `json:"streetAddress,omitempty" db:"street_address"`
	Consented     bool            `json:"consented" db:"consented"`
	Verified      time.Time       `json:"verified,omitempty" db:"verified"`
	Origin        Origin          `json:"origin,omitempty" db:"origin"`
	CustomFields  json.RawMessage `json:"customFields,omitempty" db:"custom_fields"`
	Tags          []int32         `json:"tags,omitempty" db:"tags"`
}

// In case COPY FROM is adopted
// func (m *MemberInfo) GetDBFields() []string {
// 	return []string{
// 		"date_of_birth",
// 		"email",
// 		"first_name",
// 		"last_name",
// 		"phone",
// 		"street_address",
// 		"consented",
// 		"verified",
// 		"origin",
// 		"custom_fields",
// 	}
// }

// func (m *MemberInfo) GetActiveDBFields() map[string]interface{} {
// 	ret := make(map[string]interface{})
// 	str := reflect.Indirect(reflect.ValueOf(m))
// 	// var fields []string
// 	for i := 0; i < str.NumField(); i++ {
// 		ret[str.Type().Field(i).Tag.Get("db")] = str.Field(i).Interface()
// 		// fields = append(fields, str.Field(i).Name)
// 	}
// 	return ret
// }

// func (m *MemberInfo) GetRecord() []interface{} {
// 	var list []interface{}
// 	generic := reflect.Indirect(reflect.ValueOf(MemberInfo{}))
// 	totalFields := m.GetDBFields()
// 	activeFields := m.GetActiveDBFields()
// 	// record := reflect.Indirect(reflect.ValueOf(m))
// 	for _, field := range totalFields {
// 		// TODO check ommited
// 		data, ok := activeFields[field]
// 		if ok {
// 			list = append(list, data)
// 		} else {
// 			// generic.FieldByName(field).Type().
// 			list = append(list, generic.FieldByName(field).Interface())
// 		}

// 		// list = append(list, record.FieldByName(field).Elem)
// 	}
// 	return list
// }

// func (m *MemberInfo) Normalize() {
// 	if m.CustomFields == nil {
// 		m.CustomFields = []byte{}
// 	}
// }

type User struct {
	CreatedUpdated
	PubKey         []byte `json:"publicKey" db:"public_key"`
	DigestedPubKey []byte `json:"digestedPublicKey" db:"digested_public_key"`
}

type Census struct {
	EntityID []byte    `json:"entityId" db:"entity_id"`
	ID       HexBytes  `json:"id" db:"id"`
	TargetID uuid.UUID `json:"targetId" db:"target_id"`
	CensusInfo
}

type HexBytes []byte

func (h *HexBytes) UnmarshalJSON(src []byte) error {
	var s string
	if err := json.Unmarshal(src, &s); err != nil {
		return err
	}
	b, err := hex.DecodeString(util.TrimHex(s))
	*h = b
	return err
}

func (h *HexBytes) MarshalJSON() ([]byte, error) {
	return json.Marshal("0x" + hex.EncodeToString(*h))
}

type CensusInfo struct {
	CreatedUpdated
	Name          string   `json:"name,omitempty" db:"name"`
	MerkleRoot    HexBytes `json:"merkleRoot,omitempty" db:"merkle_root"`
	MerkleTreeURI string   `json:"merkleTreeUri,omitempty" db:"merkle_tree_uri"`
	Size          int      `json:"size" db:"size"`
}

type CensusMember struct {
	MemberID uuid.UUID `json:"memberId,omitempty" db:"member_id"`
	CensusID HexBytes  `json:"censusId" db:"census_id"`
}

type Target struct {
	CreatedUpdated
	ID       uuid.UUID       `json:"id" db:"id"`
	EntityID []byte          `json:"entityId" db:"entity_id"`
	Name     string          `json:"name" db:"name"`
	Filters  json.RawMessage `json:"filters" db:"filters"`
}

// EntityMetadata represents an entity metadata
type EntityMetadata struct {
	Version                      string              `json:"version,omitempty"`
	Languages                    []string            `json:"languges,omitempty"`
	Name                         map[string]string   `json:"name,omitempty"`
	Description                  map[string]string   `json:"description,omitempty"`
	VotingProcesses              map[string][]string `json:"votingProcesses,omitempty"`
	NewsFeed                     map[string]string   `json:"newsFeed,omitempty"`
	Media                        map[string]string   `json:"media,omitempty"`
	Actions                      []interface{}       `json:"actions,omitempty"`
	BootEntities                 []interface{}       `json:"bootEntities,omitempty"`
	TrustedEntities              []interface{}       `json:"trustedEntities,omitempty"`
	CensusServiceManagedEntities []interface{}       `json:"censusServiceManagedEntities,omitempty"`
}

// A tag of a given entity for categorizing users
type Tag struct {
	CreatedUpdated
	ID       int32  `json:"id,omitempty" db:"id"`
	EntityID []byte `json:"entityId,omitempty" db:"entity_id"`
	Name     string `json:"name,omitempty" db:"name"`
}
