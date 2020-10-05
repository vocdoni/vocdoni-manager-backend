package pgsql

import (
	"github.com/jackc/pgtype"
	"gitlab.com/vocdoni/manager/manager-backend/types"
)

type PGEntity struct {
	types.Entity
	CensusManagersAddresses pgtype.ByteaArray `json:"censusManagersAddresses" db:"pg_census_managers_addresses"`
	Origins                 pgtype.EnumArray  `db:"origins"`
}

func ToPGEntity(x *types.Entity) (*PGEntity, error) {
	y := &PGEntity{Entity: *x}
	err := y.CensusManagersAddresses.Set(x.CensusManagersAddresses)
	if err != nil {
		return nil, err
	}
	vsm := make([]string, len(x.Origins))
	for i, v := range x.Origins {
		vsm[i] = v.String()
	}
	pgOrigins, err := ToPGOriginArray(x.Origins)
	if err != nil {
		return nil, err
	}
	y.Origins = *pgOrigins
	return y, nil
}

func ToEntity(x *PGEntity) (*types.Entity, error) {
	y := x.Entity
	err := x.CensusManagersAddresses.AssignTo(&y.EntityInfo.CensusManagersAddresses)
	if err != nil {
		return nil, err
	}
	if x.Origins.Status != pgtype.Null {
		origins, err := ToOriginArray(x.Origins)
		if err != nil {
			return nil, err
		}
		y.EntityInfo.Origins = origins
	}

	// err = x.Origins.AssignTo(&y.EntityInfo.Origins)
	if err != nil {
		return nil, err
	}
	return &y, nil
}

func ToPGOriginArray(x []types.Origin) (*pgtype.EnumArray, error) {
	var origin pgtype.EnumArray
	copy := make([]string, len(x))
	for i, v := range x {
		copy[i] = v.String()
	}
	err := origin.Set(copy)
	if err != nil {
		return nil, err
	}
	pgOrigin := pgtype.EnumArray(origin)
	return &pgOrigin, nil
}

func ToOriginArray(p pgtype.EnumArray) ([]types.Origin, error) {
	var origin []string
	p.AssignTo(&origin)
	copy := make([]types.Origin, len(origin))
	for i, v := range origin {
		copy[i] = types.ToOrigin(v)
	}
	return copy, nil
}

func StringToOriginArray(s []string) ([]types.Origin, error) {
	copy := make([]types.Origin, len(s))
	for i, v := range s {
		copy[i] = types.ToOrigin(v)
	}
	return copy, nil
}

type PGMember struct {
	types.Member
	CustomFields pgtype.JSONB     `json:"customFields" db:"pg_custom_fields"`
	Tags         pgtype.Int4Array `json:"tags" db:"pg_tags"`
}

func ToPGMember(x *types.Member) (*PGMember, error) {
	var err error
	y := &PGMember{Member: *x}
	err = y.CustomFields.Set(x.MemberInfo.CustomFields)
	// if x.MemberInfo.CustomFields == nil {
	// 	err = y.CustomFields.Set(json.RawMessage{})
	// } else {
	// 	err = y.CustomFields.Set(x.MemberInfo.CustomFields)
	// }
	if err != nil {
		return nil, err
	}
	err = y.Tags.Set(x.MemberInfo.Tags)
	if err != nil {
		return nil, err
	}
	// y.CustomFields = pgtype.JSONB{Bytes: x.MemberInfo.CustomFields, Status: pgtype.Present}
	return y, nil
}

func ToMember(x *PGMember) *types.Member {
	y := x.Member
	// y.MemberInfo.CustomFields = x.CustomFields.Bytes
	x.CustomFields.AssignTo(y.MemberInfo.CustomFields)
	x.Tags.AssignTo(&y.MemberInfo.Tags)
	return &y
}

//go:generate stringer -type=OrderBySQLi
type OrderBySQLi int

const (
	DateOfBirth OrderBySQLi = iota
	Email
	FirstName
	LastName
	Phone
	StreetAddress
	Consented
	Verified
	Origin
	CustomFields
	Name
	MerkleRoot
	MerkleTreeURI
	Size
	CreatedAt
	UpdatedAt
)

func ToOrderBySQLi(orderBy string) OrderBySQLi {
	switch orderBy {
	case "dateOfBirth":
		return DateOfBirth
	case "email":
		return Email
	case "firstName":
		return FirstName
	case "lastName":
		return LastName
	case "phone":
		return Phone
	case "streetAddress":
		return StreetAddress
	case "consented":
		return Consented
	case "verified":
		return Verified
	case "origin":
		return Origin
	case "customFields":
		return CustomFields
	case "name":
		return Name
	case "merkleRoot":
		return MerkleRoot
	case "merkleTreeURI":
		return MerkleTreeURI
	case "size":
		return Size
	case "createdAt":
		return CreatedAt
	case "updatedAt":
		return UpdatedAt
	default:
		return -1
	}
}
