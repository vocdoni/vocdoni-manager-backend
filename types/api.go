package types

import (
	"fmt"
)

type RequestMessage struct {
	MetaRequest `json:"request"`

	ID        string `json:"id"`
	Signature string `json:"signature"`
}

type MetaRequest struct {
	Census      *Census      `json:"census,omitempty"`
	EntityID    string       `json:"entityId,omitempty"`
	Filter      *Filter      `json:"filter,omitempty"`
	ListOptions *ListOptions `json:"listOptions,omitempty"`
	Member      *Member      `json:"member"`
	Method      string       `json:"method"`
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
	Message   string  `json:"message,omitempty"`
	Ok        bool    `json:"ok"`
	PublicKey string  `json:"publicKey"`
	Request   string  `json:"request"`
	Status    *Status `json:"status,omitempty"`
	Timestamp int32   `json:"timestamp"`
}

// SetError sets the MetaResponse's Ok field to false, and Message to a string
// representation of v. Usually, v's type will be error or string.
func (r *MetaResponse) SetError(v interface{}) {
	r.Ok = false
	r.Message = fmt.Sprintf("%s", v)
}

type Member struct {
	DateOfBirth string `json:"dateOfBirth,omitempty"`
	Email       string `json:"email,omitempty"`
	FirstName   string `json:"firstName,omitempty"`
	LastName    string `json:"lastName,omitempty"`
	Phone       string `json:"phone,omitempty"`
}

type Status struct {
	Registered  bool `json:"registered,omitempty"`
	NeedsUpdate bool `json:"needsUpdate,omitempty"`
}

type Census struct {
	Created int32  `json:"created,omitempty"`
	ID      string `json:"id,omitempty"`
	Name    string `json:"name,omitempty"`
	Root    string `json:"root,omitempty"`
	Target  string `json:"target,omitempty"`
	URI     string `json:"uri,omitempty"`
}

type ListOptions struct {
	Count  int    `json:"count,omitempty"`
	Order  string `json:"order,omitempty"`
	Skip   int    `json:"skip,omitempty"`
	SortBy string `json:"sortBy,omitempty"`
}

type Filter struct {
	Member
	Target string `json:"target,omitempty"`
}
