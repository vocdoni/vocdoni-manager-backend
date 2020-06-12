package types

import (
	"fmt"

	"github.com/google/uuid"
)

type RequestMessage struct {
	MetaRequest `json:"request"`

	ID        string `json:"id"`
	Signature string `json:"signature"`
}

type MetaRequest struct {
	Amount      int          `json:"amount,omitempty"`
	Census      *Census      `json:"census,omitempty"`
	EntityID    string       `json:"entityId,omitempty"`
	Filter      *Target      `json:"filter,omitempty"`
	ListOptions *ListOptions `json:"listOptions,omitempty"`
	Member      *Member      `json:"member"`
	MembersInfo []MemberInfo `json:"membersInfo"`
	Method      string       `json:"method"`
	PubKey      string       `json:"publicKey,omitempty"`
	Signature   string       `json:"signature,omitempty"`
	Scope       string       `json:"scope,omitempty"`
	Status      *Status      `json:"status,omitempty"`
	Timestamp   int32        `json:"timestamp"`
	Token       string       `json:"token,omitempty"`
}

// ResponseMessage wraps an api response
type ResponseMessage struct {
	MetaResponse `json:"response"`

	ID        string `json:"id"`
	Signature string `json:"signature"`
}

// MetaResponse contains all of the possible request fields.
// Fields must be in alphabetical order
// Those fields with valid zero-values (such as bool) must be pointers
type MetaResponse struct {
	Members       []Member     `json:members,omitempty`
	Message       string       `json:"message,omitempty"`
	Ok            bool         `json:"ok"`
	PublicKey     string       `json:"publicKey,omitempty"`
	Request       string       `json:"request"`
	Status        *Status      `json:"status,omitempty"`
	Timestamp     int32        `json:"timestamp"`
	Tokens        []uuid.UUID  `json:"tokens"`
	MembersTokens []TokenEmail `json:"membersTokens"`
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
	Registered  bool `json:"registered,omitempty"`
	NeedsUpdate bool `json:"needsUpdate,omitempty"`
}

type ListOptions struct {
	Count  int    `json:"count,omitempty"`
	Order  string `json:"order,omitempty"`
	Skip   int    `json:"skip,omitempty"`
	SortBy string `json:"sortBy,omitempty"`
}
