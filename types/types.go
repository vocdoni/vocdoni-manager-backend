package types

import (
	"time"

	"github.com/google/uuid"
)

type CreatedUpdated struct {
	CreatedAt time.Time `json:"-" db:"created_at"`
	UpdatedAt time.Time `json:"-" db:"updated_at"`
}

type Entity struct {
	CreatedUpdated
	ID         string `json:"id" db:"id"`
	EntityInfo `entityinfo`
}

type EntityInfo struct {
	Address                 string   `json:"address" db:"address"`
	Email                   string   `json:"email,omitempty" db:"email"`
	Name                    string   `json:"name" db:"name"`
	CensusManagersAddresses []string `json:"censusManagersAddresses,omitempty" db:"census_managers_addresses"`
}

type Member struct {
	CreatedUpdated
	ID         uuid.UUID `json:"id" db:"id"`
	EntityID   string    `json:"entityId" db:"entity_id"`
	PubKey     string    `json:"publicKey" db:"public_key"`
	MemberInfo `memberinfo`
}

type MemberInfo struct {
	DateOfBirth   time.Time `json:"dateOfBirth,omitempty" db:"date_of_birth"`
	Email         string    `json:"email,omitempty" db:"email"`
	FirstName     string    `json:"firstName,omitempty" db:"first_name"`
	LastName      string    `json:"lastName,omitempty" db:"last_name"`
	Phone         string    `json:"phone,omitempty" db:"phone"`
	StreetAddress string    `json:"streetAddress,omitempty" db:"street_address"`
	Consented     bool      `json:"consented,omitempty" db:"consented"`
	Verified      time.Time `json:"verified,omitempty" db:"verified"`
	CustomFields  []byte    `json:"customFields" db:"custom_fields"`
}

type User struct {
	PubKey         string `json:"publicKey" db:"publicKey"`
	DigestedPubKey string `json:"digestedPublicKey" db:"digestedPublicKey"`
}

type Census struct {
	EntityID string `json:"entityId" db:"entityId"`
	ID       string `json:"id" db:"id"`
	TargetID string `json:"targetId" db:"targetId"`
	CensusInfo
}

type CensusInfo struct {
	Created time.Time `json:"created,omitempty"`
	Name    string    `json:"name,omitempty"`
	Root    string    `json:"root,omitempty"`
	URI     string    `json:"uri,omitempty"`
}

type Target struct {
	ID       string            `json:"id" db:"id"`
	EntityID string            `json:"entityId" db:"entityId"`
	Name     string            `json:"name" db:"name"`
	Filters  map[string]string `json:"filters" db:"filters"`
}
