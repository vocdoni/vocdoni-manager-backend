package types

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	dvotetypes "go.vocdoni.io/dvote/types"
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
	CallbackURL             string   `json:"callbackUrl" db:"callback_url"`
	CallbackSecret          string   `json:"callbackSecret" db:"callback_secret"`
	Email                   string   `json:"email,omitempty" db:"email"`
	Name                    string   `json:"name" db:"name"`
	Type                    string   `json:"type" db:"type"`
	Size                    int      `json:"size" db:"size"`
	CensusManagersAddresses [][]byte `json:"censusManagersAddresses,omitempty" db:"census_managers_addresses"`
	Origins                 []Origin `json:"origins" db:"origins"`
}

//go:generate stringer -type=Origin
type Origin int

const (
	Token Origin = iota
	Form
	DB
	API
)

func ToOrigin(origin string) Origin {
	switch origin {
	case "Token":
		return Token
	case "Form":
		return Form
	case "DB":
		return DB
	case "API":
		return API
	default:
		return Token
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

type HexBytes = dvotetypes.HexBytes
type CensusInfo struct {
	CreatedUpdated
	Name          string `json:"name,omitempty" db:"name"`
	MerkleRoot    []byte `json:"merkleRoot,omitempty" db:"merkle_root"`
	MerkleTreeURI string `json:"merkleTreeUri,omitempty" db:"merkle_tree_uri"`
	Size          int    `json:"size" db:"size"`
	Ephemeral     bool   `json:"ephemeral" db:"ephemeral"`
	ProcessID     []byte `json:"processId,omitempty" db:"process_id"`
}

type CensusMember struct {
	MemberID       uuid.UUID `json:"memberId,omitempty" db:"member_id"`
	CensusID       []byte    `json:"censusId,omitempty" db:"census_id"`
	Ephemeral      bool      `json:"ephemeral" db:"ephemeral"`
	PrivKey        []byte    `json:"privateKey,omitempty" db:"private_key"`
	PubKey         []byte    `json:"publicKey,omitempty" db:"public_key"`
	DigestedPubKey []byte    `json:"digestedPublicKey,omitempty" db:"digested_public_key"`
}

type EphemeralMemberInfo struct {
	ID             uuid.UUID `json:"id,omitempty" db:"id"`
	FirstName      string    `json:"firstName,omitempty" db:"first_name"`
	LastName       string    `json:"lastName,omitempty" db:"last_name"`
	Email          string    `json:"email,omitempty" db:"email"`
	PrivKey        []byte    `json:"privateKey,omitempty" db:"private_key"`
	DigestedPubKey []byte    `json:"digestedPublicKey,omitempty" db:"digested_public_key"`
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

// NewsFeed represents the news feed content of each entry in the NewsFeed entity metadata list
type NewsFeed struct {
	Items []NewsFeedItem `json:"items,omitempty"`

	Version     string `json:"version,omitempty"`
	Title       string `json:"title,omitempty"`
	HomePageURL string `json:"home_page_url,omitempty"`
	Description string `json:"description,omitempty"`
	FeedURL     string `json:"feed_url,omitempty"`
	Icon        string `json:"icon,omitempty"`
	Favicon     string `json:"favicon,omitempty"`
	Expired     bool   `json:"expired,omitempty"`
}

// NewsFeedItem represents each Item in the NewsFeed Items
type NewsFeedItem struct {
	Tags          []interface{}      `json:"tags,omitempty"`
	Author        NewsFeedItemAuthor `json:"author,omitempty"`
	ID            string             `json:"id,omitempty"`
	Title         string             `json:"title,omitempty"`
	Summary       string             `json:"summary,omitempty"`
	ContentText   string             `json:"content_text,omitempty"`
	ContentHTML   string             `json:"content_html,omitempty"`
	URL           string             `json:"url,omitempty"`
	Image         string             `json:"image,omitempty"`
	DatePublished string             `json:"date_published,omitempty"`
	DateModified  string             `json:"date_modified,omitempty"`
}

type NewsFeedItemAuthor struct {
	Name string `json:"name,omitempty"`
	URL  string `json:"url,omitempty"`
}

// A tag of a given entity for categorizing users
type Tag struct {
	CreatedUpdated
	ID       int32  `json:"id,omitempty" db:"id"`
	EntityID []byte `json:"entityId,omitempty" db:"entity_id"`
	Name     string `json:"name,omitempty" db:"name"`
}
