package types

import (
	"time"

	"github.com/google/uuid"
)

type Entity struct {
	ID string `json:"id" db:"id"`
	EntityInfo
}

type EntityInfo struct {
	Address         string   `json:"address" db:"address"`
	Name            string   `json:"name" db:"name"`
	ManagersPubKeys []string `json:"managersPublicKeys,omitempty" db:"managersPublicKeys"`
}

type Member struct {
	ID       uuid.UUID `json:"id" db:"id"`
	EntityID string    `json:"entityId" db:"entityId"`
	PubKey   string    `json:"publicKey" db:"publicKey"`
	MemberInfo
}

type MemberInfo struct {
	DateOfBirth string `json:"dateOfBirth,omitempty" db:"dateOfBirth"`
	Email       string `json:"email,omitempty" db:"email"`
	FirstName   string `json:"firstName,omitempty" db:"firstName"`
	LastName    string `json:"lastName,omitempty" db:"lastName"`
	Phone       string `json:"phone,omitempty" db:"phone"`
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
