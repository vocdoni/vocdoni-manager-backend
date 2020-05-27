package pgsql

import (
	"log"

	"github.com/jackc/pgtype"
	"gitlab.com/vocdoni/vocdoni-manager-backend/types"
)

type PGEntity struct {
	types.Entity
	CensusManagersAddresses pgtype.TextArray `json:"censusManagersAddresses" db:"pg_census_managers_addresses"`
}

func ToPGEntity(x *types.Entity) *PGEntity {
	y := &PGEntity{Entity: *x}
	y.CensusManagersAddresses.Set(x.CensusManagersAddresses)
	return y
}

func ToEntity(x *PGEntity) *types.Entity {
	y := x.Entity
	err := x.CensusManagersAddresses.AssignTo(&y.CensusManagersAddresses)
	if err != nil {
		log.Fatal(err)
	}
	return &y
}

type PGMember struct {
	types.Member
	CustomFields pgtype.JSONB `json:"customFields" db:"pg_custom_fields"`
}

func ToPGMember(x *types.Member) *PGMember {
	y := &PGMember{Member: *x}
	if x.MemberInfo.CustomFields == nil {
		y.CustomFields.Set([]byte{})
	} else {
		y.CustomFields.Set(x.MemberInfo.CustomFields)
	}
	// y.CustomFields = pgtype.JSONB{Bytes: x.MemberInfo.CustomFields, Status: pgtype.Present}
	return y
}

// func (m *MemberInfo) Normalize() {
// 	if m.CustomFields == nil {
// 		m.CustomFields = []byte{}
// 	}
// }

func ToMember(x *PGMember) *types.Member {
	y := x.Member
	y.MemberInfo.CustomFields = x.CustomFields.Bytes
	return &y
}
