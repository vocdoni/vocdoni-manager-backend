package types

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

type RequestMessage struct {
	MetaRequest json.RawMessage `json:"request"`

	ID        string   `json:"id"`
	Signature HexBytes `json:"signature"`
}

type MetaRequest struct {
	Amount   int         `json:"amount,omitempty"`
	AuthHash string      `json:"authHash,omitempty"`
	Census   *CensusInfo `json:"census,omitempty"`
	CensusID string      `json:"censusId,omitempty"`
	//TODO Keys HexBytes when API supports protobuf or similar
	Keys          []string     `json:"keys,omitempty"` // claim Keys
	Email         string       `json:"email,omitempty"`
	EntityID      string       `json:"entityId,omitempty"`
	Entity        *EntityInfo  `json:"entity,omitempty"`
	Filter        *Target      `json:"filter,omitempty"`
	ListOptions   *ListOptions `json:"listOptions,omitempty"`
	MemberID      *uuid.UUID   `json:"memberId,omitempty"`
	MemberIDs     []uuid.UUID  `json:"memberIds,omitempty"`
	Member        *Member      `json:"member,omitempty"`
	MemberInfo    *MemberInfo  `json:"memberInfo,omitempty"`
	MembersInfo   []MemberInfo `json:"membersInfo,omitempty"`
	Method        string       `json:"method"`
	InvalidClaims [][]byte     `json:"invalidClaims"`
	PubKey        string       `json:"publicKey,omitempty"`
	ProcessID     string       `json:"processId,omitempty"`
	Signature     string       `json:"signature,omitempty"`
	Scope         string       `json:"scope,omitempty"`
	Status        *Status      `json:"status,omitempty"`
	TagID         int32        `json:"tagId,omitempty"`
	TagName       string       `json:"tagName,omitempty"`
	TargetID      *uuid.UUID   `json:"targetId,omitempty"`
	Timestamp     int32        `json:"timestamp"`
	Token         string       `json:"token,omitempty"`
	Topic         string       `json:"topic,omitempty"`
}

// ResponseMessage wraps an api response
type ResponseMessage struct {
	MetaResponse json.RawMessage `json:"response"`

	ID        string   `json:"id"`
	Signature HexBytes `json:"signature"`
}

// MetaResponse contains all of the possible request fields.
// Fields must be in alphabetical order
// Those fields with valid zero-values (such as bool) must be pointers
type MetaResponse struct {
	Census        *Census      `json:"census,omitempty"`
	Censuses      []Census     `json:"censuses,omitempty"`
	Claims        [][]byte     `json:"claims,omitempty"`
	Count         int          `json:"count,omitempty"`
	Entity        *Entity      `json:"entity,omitempty"`
	InvalidIDs    []uuid.UUID  `json:"invalidIds,omitempty"`
	Member        *Member      `json:"member,omitempty"`
	Members       []Member     `json:"members,omitempty"`
	MembersTokens []TokenEmail `json:"membersTokens,omitempty"`
	Message       string       `json:"message,omitempty"`
	Ok            bool         `json:"ok"`
	PublicKey     string       `json:"publicKey,omitempty"`
	Request       string       `json:"request"`
	Status        *Status      `json:"status,omitempty"`
	Tag           *Tag         `json:"tag,omitempty"`
	Tags          []Tag        `json:"tags,omitempty"`
	Target        *Target      `json:"target,omitempty"`
	Targets       []Target     `json:"targets,omitempty"`
	Timestamp     int32        `json:"timestamp"`
	Token         string       `json:"token,omitempty"`
	Tokens        []uuid.UUID  `json:"tokens,omitempty"`
	TokenStatus   string       `json:"tokenStatus,omitempty"`
}

// SetError sets the MetaResponse's Ok field to false, and Message to a string
// representation of v. Usually, v's type will be error or string.
func (r *MetaResponse) SetError(v interface{}) {
	r.Ok = false
	r.Message = fmt.Sprintf("%s", v)
}

type TokenEmail struct {
	Token uuid.UUID `json:"tokens"`
	Email string    `json:"emails"`
}

type Status struct {
	Registered  bool `json:"registered"`
	NeedsUpdate bool `json:"needsUpdate"`
}

type ListOptions struct {
	Count  int    `json:"count,omitempty"`
	Order  string `json:"order,omitempty"`
	Skip   int    `json:"skip,omitempty"`
	SortBy string `json:"sortBy,omitempty"`
}
