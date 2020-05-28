package pgsql

import (
	"github.com/jackc/pgtype"
	"gitlab.com/vocdoni/vocdoni-manager-backend/types"
)

type PGEntity struct {
	types.Entity
	CensusManagersAddresses pgtype.ByteaArray `json:"censusManagersAddresses" db:"pg_census_managers_addresses"`
}

func ToPGEntity(x *types.Entity) (*PGEntity, error) {
	y := &PGEntity{Entity: *x}
	err := y.CensusManagersAddresses.Set(x.CensusManagersAddresses)
	if err != nil {
		return nil, err
	}
	return y, nil
}

func ToEntity(x *PGEntity) (*types.Entity, error) {
	y := x.Entity
	err := x.CensusManagersAddresses.AssignTo(&y.EntityInfo.CensusManagersAddresses)
	if err != nil {
		return nil, err
	}
	return &y, nil
}

type PGMember struct {
	types.Member
	CustomFields pgtype.JSONB `json:"customFields" db:"pg_custom_fields"`
}

func ToPGMember(x *types.Member) (*PGMember, error) {
	var err error
	y := &PGMember{Member: *x}
	if x.MemberInfo.CustomFields == nil {
		err = y.CustomFields.Set([]byte{})
	} else {
		err = y.CustomFields.Set(x.MemberInfo.CustomFields)
	}
	if err != nil {
		return nil, err
	}
	// y.CustomFields = pgtype.JSONB{Bytes: x.MemberInfo.CustomFields, Status: pgtype.Present}
	return y, nil
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
