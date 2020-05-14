package types

import "fmt"

type RequestMessage struct {
	MetaRequest `json:"request"`

	ID        string `json:"id"`
	Signature string `json:"signature"`
}

type MetaRequest struct {
	EntityId  string `json:"entityId,omitempty"`
	Method    string `json:"method"`
	Signature string `json:"signature,omitempty"`
	Timestamp int32  `json:"timestamp"`
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
	Message   string `json:"message,omitempty"`
	Ok        bool   `json:"ok"`
	Request   string `json:"request"`
	Timestamp int32  `json:"timestamp"`
}

// SetError sets the MetaResponse's Ok field to false, and Message to a string
// representation of v. Usually, v's type will be error or string.
func (r *MetaResponse) SetError(v interface{}) {
	r.Ok = false
	r.Message = fmt.Sprintf("%s", v)
}
