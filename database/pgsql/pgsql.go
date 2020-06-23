package pgsql

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	_ "github.com/jackc/pgx/stdlib"
	"gitlab.com/vocdoni/go-dvote/crypto/snarks"

	log "gitlab.com/vocdoni/go-dvote/log"
	"gitlab.com/vocdoni/vocdoni-manager-backend/config"
	"gitlab.com/vocdoni/vocdoni-manager-backend/types"
)

type Database struct {
	db *sqlx.DB
	// For using pgx connector
	// pgx    *pgxpool.Pool
	// pgxCtx context.Context
}

func New(dbc *config.DB) (*Database, error) {
	db, err := sqlx.Open("pgx", fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s client_encoding=%s",
		dbc.Host, dbc.Port, dbc.User, dbc.Password, dbc.Dbname, dbc.Sslmode, "UTF8"))
	if err != nil {
		return nil, err
	}
	// For using pgx connector
	// ctx := context.Background()
	// pgx, err := pgxpool.Connect(ctx, connectionString)
	if err != nil {
		log.Debug(fmt.Errorf("Unable to connect to database: %v\n", err))
		return nil, fmt.Errorf("Unable to connect to database: %v\n", err)
	}
	// TODO: Set MaxOpenConnections, MaxLifetime (MaxIdle?)
	// MaxOpen should be the number of expected clients? (Different apis?)
	// db.SetMaxOpenConns(2)

	// return &Database{db: db, pgx: pgx, pgxCtx: ctx}, err
	return &Database{db: db}, err
}

func (d *Database) Close() error {
	defer d.db.Close()
	// defer d.pgx.Close()
	return nil
	// return d.db.Close()
}

func (d *Database) AddEntity(entityID []byte, info *types.EntityInfo) error {
	var err error
	if info.Address == nil {
		return fmt.Errorf("entity address not found")
	}
	if info.CensusManagersAddresses == nil {
		return fmt.Errorf("census manager addresses not found")
	}
	tx, err := d.db.Beginx()
	if err != nil {
		return err
	}
	entity := &types.Entity{EntityInfo: *info, ID: entityID}
	pgEntity, err := ToPGEntity(entity)
	if err != nil {
		return err
	}
	// TODO: Calculate EntityID (consult go-dvote)
	insert := `INSERT INTO entities
					(id, address, email, name, census_managers_addresses)
					VALUES (:id, :address, :email, :name, :pg_census_managers_addresses)`
	_, err = tx.NamedExec(insert, pgEntity)
	if err != nil {
		return err
	}
	insertOrigins := `INSERT INTO entities_origins (entity_id,origin)
						VALUES ($1, unnest(cast($2 AS Origins[])))`
	_, err = tx.Exec(insertOrigins, entityID, pgEntity.Origins)
	if err != nil {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {

			return rollbackErr
		}
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (d *Database) Entity(entityID []byte) (*types.Entity, error) {
	var pgEntity PGEntity
	selectEntity := `SELECT id, address, email, name, census_managers_addresses as "pg_census_managers_addresses"  FROM entities WHERE id=$1`
	row := d.db.QueryRowx(selectEntity, entityID)
	err := row.StructScan(&pgEntity)
	if err != nil {
		return nil, err
	}
	entity, err := ToEntity(&pgEntity)
	origins, err := d.EntityOrigins(entityID)
	if err != nil {
		return nil, err
	}
	entity.Origins = origins
	return entity, nil
}

func (d *Database) EntityOrigins(entityID []byte) ([]types.Origin, error) {
	var stringOrigins []string
	selectOrigins := `SELECT origin FROM entities_origins WHERE entity_id=$1`
	err := d.db.Select(&stringOrigins, selectOrigins, entityID)
	if err != nil {
		return nil, err
	}
	origins, err := StringToOriginArray(stringOrigins)
	if err != nil {
		return nil, err
	}
	return origins, nil
}

func (d *Database) EntityHas(entityID []byte, memberID uuid.UUID) bool {
	return true
}

func (d *Database) AddUser(user *types.User) error {
	if user.PubKey == nil {
		return fmt.Errorf("Invalid public Key")
	}
	if len(user.DigestedPubKey) == 0 {
		user.DigestedPubKey = snarks.Poseidon.Hash(user.PubKey)
	}
	insert := `INSERT INTO users
	 				(public_key, digested_public_key)
					 VALUES (:public_key, :digested_public_key)`
	_, err := d.db.NamedExec(insert, user)
	if err != nil {
		return err
	}
	return nil
}

func (d *Database) User(pubKey []byte) (*types.User, error) {
	var user types.User
	selectQuery := `SELECT
	 			public_key, digested_public_key
				FROM USERS where public_key=$1`
	row := d.db.QueryRowx(selectQuery, pubKey)
	err := row.StructScan(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (d *Database) CreateMembersWithTokens(entityID []byte, tokens []uuid.UUID) error {
	var err error
	var result sql.Result
	var rows int64
	pgmembers := make([]PGMember, len(tokens))
	for idx := range pgmembers {
		if tokens[idx] == uuid.Nil {
			return fmt.Errorf("error parsing the uuids")
		}
		pgmembers[idx] = PGMember{Member: types.Member{ID: tokens[idx], EntityID: entityID, MemberInfo: types.MemberInfo{}}}
		// pgmembers[idx].ID = tokens[idx]
		// pgmembers[idx].EntityID = entityID
	}

	tx, err := d.db.Beginx()
	if err != nil {
		return err
	}
	insert := `INSERT INTO members
					(id, entity_id)
					VALUES (:id, :entity_id)`
	insert = `INSERT INTO members
	(id,entity_id, public_key, street_address, first_name, last_name, email, phone, date_of_birth, verified)
	VALUES (:id, :entity_id, :public_key, :street_address, :first_name, :last_name, :email, :phone, :date_of_birth, :verified)`
	// result, err = tx.NamedExec(insert, pgmembers)
	if result, err = tx.NamedExec(insert, pgmembers); err != nil {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			return rollbackErr
		}
		if err != nil {
			return fmt.Errorf("Error in bulk import %w", err)
		}
	}
	if rows, err = result.RowsAffected(); err != nil || int(rows) != len(pgmembers) {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			return rollbackErr
		}
		if err != nil {
			return fmt.Errorf("Error in bulk import %w", err)
		}
		return fmt.Errorf("Should insert %d rows, while inserted %d rows. Rolled back.", len(pgmembers), rows)
	}
	// rows, errRows := result.RowsAffected()
	// if err != nil || int(rows) != len(pgmembers) {

	// 	rollbackErr := tx.Rollback()
	// 	if rollbackErr != nil {
	// 		return rollbackErr
	// 	}
	// 	if err != nil {
	// 		return fmt.Errorf("Error in bulk import %w", err)
	// 	}
	// 	if errRows != nil {
	// 		return fmt.Errorf("Error in bulk import %w", errRows)
	// 	}
	// 	return fmt.Errorf("Should insert %d rows, while inserted %d rows. Rolled back.", len(pgmembers), rows)
	// }
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("Could not commit bulk import %w", err)
	}
	return nil

}

// TODO: Implement import members

func (d *Database) AddMember(entityID []byte, pubKey []byte, info *types.MemberInfo) (uuid.UUID, error) {
	var err error
	var result *sqlx.Rows
	var id uuid.UUID
	member := &types.Member{EntityID: entityID, PubKey: pubKey, MemberInfo: *info}
	_, err = d.User(pubKey)
	if err != nil {
		return uuid.Nil, err
	}
	pgmember, err := ToPGMember(member)
	if err != nil {
		return uuid.Nil, err
	}
	insert := `INSERT INTO members
	 				(entity_id, public_key, street_address, first_name, last_name, email, phone, date_of_birth, verified, custom_fields)
					 VALUES (:entity_id, :public_key, :street_address, :first_name, :last_name, :email, :phone, :date_of_birth, :verified, :pg_custom_fields)
				RETURNING id`
	// no err is returned if tx violated a db constraint,
	// but we need the result in order to get the created id.
	// LastInsertedID() is not exposed.
	// With Exec(), Scan() is not avaiable and with PrepareStmt()
	// is not possible to use pgmember and a conversion is needed.
	// So if no error is raised and the result has 0 rows it means
	// that something went wrong (no member added).
	if result, err = d.db.NamedQuery(insert, pgmember); err != nil {
		return uuid.Nil, err
	}
	if !result.Next() {
		return uuid.Nil, fmt.Errorf("result has no rows, posible violation of db constraints")
	}
	if err := result.Scan(&id); err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func (d *Database) ImportMembers(entityID []byte, info []types.MemberInfo) error {
	// TODO: Check if support for Update a Member is needed
	// TODO: Investigate COPY FROM with pgx
	var err error
	var result sql.Result
	var rows int64
	if len(info) <= 0 {
		return fmt.Errorf("No member data provided")
	}
	members := []PGMember{}
	for _, member := range info {
		newMember := &types.Member{EntityID: entityID, MemberInfo: member}
		pgMember, err := ToPGMember(newMember)
		if err != nil {
			return err
		}
		members = append(members, *pgMember)
	}
	// Effort to use COPY FROM with pgx
	// fields := []string{"public_key", "entity_id"}
	// fields = append(fields, info[0].GetDBFields()...)
	// // str := reflect.Indirect(reflect.ValueOf(types.MemberInfo{}))
	// // var fields []string
	// // for i := 0; i < str.Type().NumField(); i++ {
	// // 	fields = append(fields, str.Type().Field(i).Name)
	// // }
	// members := [][]interface{}{}
	// var eid interface{} = entityID
	// var pubKey []byte = make([]byte, 0)
	// var pK interface{} = pubKey
	// for _, member := range info {
	// 	// members= append(members, member.GetRecord())
	// 	// var ret  []interface{}
	// 	// for jdx, _ := range member {

	// 	// }
	// 	member.Origin = types.DB.Origin()
	// 	// entry := []interface{}
	// 	entry := []interface{}{pK, eid}
	// 	entry = append(entry, member.GetRecord()...)
	// 	members = append(members, entry)
	// }
	// count, err := d.pgx.CopyFrom(d.pgxCtx, pgx.Identifier{"members"}, fields, pgx.CopyFromRows(members))
	// if count != int64(len(info)) {
	// 	return fmt.Errorf("Bulk insert members error. Needed to insert %d members but insterted %d members", len(info), count)
	// }
	tx, err := d.db.Beginx()
	if err != nil {
		return err
	}
	insert := `INSERT INTO members
					(entity_id, public_key, street_address, first_name, last_name, email, phone, date_of_birth, verified, custom_fields)
					VALUES (:entity_id, :public_key, :street_address, :first_name, :last_name, :email, :phone, :date_of_birth, :verified, :pg_custom_fields)`
	if result, err = tx.NamedExec(insert, members); err != nil {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			return rollbackErr
		}
		if err != nil {
			return fmt.Errorf("Error in bulk import %w", err)
		}
	}
	if rows, err = result.RowsAffected(); err != nil || int(rows) != len(members) {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			return rollbackErr
		}
		if err != nil {
			return fmt.Errorf("Error in bulk import %w", err)
		}
		return fmt.Errorf("Should insert %d rows, while inserted %d rows. Rolled back.", len(members), rows)
	}
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("Could not commit bulk import %w", err)
	}
	return nil
}

func (d *Database) AddMemberBulk(entityID []byte, members []types.Member) error {
	// TODO: Check if support for Update a Member is needed
	// TODO: Investigate COPY FROM with pgx
	var err error
	var result sql.Result
	var rows int64
	if len(members) <= 0 {
		return fmt.Errorf("No member data provided")
	}
	tx, err := d.db.Beginx()
	if err != nil {
		return err
	}
	users := make([]types.User, len(members))
	pgMembers := []PGMember{}
	for idx, member := range members {
		// User-related
		if len(member.PubKey) == 0 {
			return fmt.Errorf("found empty public keys")
		}
		users[idx] = types.User{PubKey: member.PubKey, DigestedPubKey: snarks.Poseidon.Hash(member.PubKey)}
		// Member related
		if hex.EncodeToString(member.EntityID) != hex.EncodeToString(entityID) {
			return fmt.Errorf("Trying to import members for other entity")
		}
		pgMember, err := ToPGMember(&member)
		if err != nil {
			return err
		}
		pgMembers = append(pgMembers, *pgMember)
	}
	insertUsers := `INSERT INTO users
				(public_key, digested_public_key) VALUES (:public_key, :digested_public_key)`
	if result, err = tx.NamedExec(insertUsers, users); err != nil {
		return fmt.Errorf("Error creating users %w", err)
	}

	insert := `INSERT INTO members
					(entity_id, public_key, street_address, first_name, last_name, email, phone, date_of_birth, verified, custom_fields)
					VALUES (:entity_id, :public_key, :street_address, :first_name, :last_name, :email, :phone, :date_of_birth, :verified, :pg_custom_fields)`
	if result, err = tx.NamedExec(insert, pgMembers); err != nil {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			return rollbackErr
		}
		if err != nil {
			return fmt.Errorf("Error in bulk import %w", err)
		}
	}
	if rows, err = result.RowsAffected(); err != nil || int(rows) != len(members) {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			return rollbackErr
		}
		if err != nil {
			return fmt.Errorf("Error in bulk import %w", err)
		}
		return fmt.Errorf("Should insert %d rows, while inserted %d rows. Rolled back.", len(members), rows)
	}
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("Could not commit bulk import %w", err)
	}
	return nil
}

func (d *Database) UpdateMember(memberID uuid.UUID, pubKey []byte, info *types.MemberInfo) error {
	member := &types.Member{ID: memberID, PubKey: pubKey, MemberInfo: *info}
	pgmember, err := ToPGMember(member)
	if err != nil {
		return err
	}
	var update string
	if pubKey != nil {
		update = `UPDATE members
	 				SET (public_key, street_address, first_name, last_name, email, phone, date_of_birth, verified, consented)
					= (:public_key, :street_address, :first_name, :last_name, :email, :phone, :date_of_birth, :verified, :consented)
					WHERE id = :id`
	} else {
		update = `UPDATE members
	 				SET (street_address, first_name, last_name, email, phone, date_of_birth, verified, consented)
					= (:street_address, :first_name, :last_name, :email, :phone, :date_of_birth, :verified, :consented)
					WHERE id = :id`
	}
	_, err = d.db.NamedExec(update, pgmember)
	if err != nil {
		return err
	}
	return nil
}

func (d *Database) Member(entityID []byte, memberID uuid.UUID) (*types.Member, error) {
	var pgMember PGMember
	selectQuery := `SELECT
	 				id, entity_id, public_key, street_address, first_name, last_name, email, phone, date_of_birth, verified, custom_fields as "pg_custom_fields", consented
					FROM members WHERE id = $1 and entity_id =$2`
	row := d.db.QueryRowx(selectQuery, memberID, entityID)
	err := row.StructScan(&pgMember)
	member := ToMember(&pgMember)
	if err != nil {
		log.Debug(err)
		return nil, err
	}
	return member, nil
}

func (d *Database) MemberPubKey(entityID, pubKey []byte) (*types.Member, error) {
	var pgMember PGMember
	selectQuery := `SELECT
	 				id, entity_id, public_key, street_address, first_name, last_name, email, phone, date_of_birth, verified, custom_fields as "pg_custom_fields"
					FROM members WHERE public_key =$1 AND entity_id =$2`
	row := d.db.QueryRowx(selectQuery, pubKey, entityID)
	err := row.StructScan(&pgMember)
	member := ToMember(&pgMember)
	if err != nil {
		log.Debug(err)
		return nil, err
	}
	return member, nil
}

func (d *Database) MembersTokensEmails(entityID []byte) ([]types.Member, error) {
	selectQuery := `SELECT
	 				id, email
					FROM members WHERE entity_id =$1`

	var pgMembers []PGMember
	err := d.db.Select(&pgMembers, selectQuery, entityID)
	if err != nil {
		log.Debug(err)
		return nil, err
	}
	members := make([]types.Member, len(pgMembers))
	for i, member := range pgMembers {
		members[i] = *ToMember(&member)
	}
	return members, nil
}

func (d *Database) ListMembers(entityID []byte, filter *types.ListOptions) ([]types.Member, error) {
	var order, offset, limit string
	orderQuery := ""
	offsetQuery := ""
	// TODO: Replace limit offset with better strategy, can slow down DB
	// would nee to now last value from previous query
	selectQuery := `SELECT
	 				id, entity_id, public_key, street_address, first_name, last_name, email, phone, date_of_birth, verified, custom_fields as "pg_custom_fields"
					FROM members WHERE entity_id =$1`
	query := selectQuery
	t := reflect.TypeOf(types.MemberInfo{})
	if filter != nil {
		if len(filter.SortBy) > 0 {
			field, found := t.FieldByName(strings.Title(filter.SortBy))
			if found {
				if filter.Order == "asc" || filter.Order == "desc" {
					order = filter.Order
				}
				orderQuery = fmt.Sprintf("ORDER BY %s %s", field.Tag.Get("db"), order)
			}
		}
		if filter.Skip > 0 {
			offset = strconv.Itoa(filter.Skip)
		} else {
			offset = "0"
		}
		if filter.Count > 0 {
			limit = strconv.Itoa(filter.Count)
		} else {
			limit = "NULL"
		}
		offsetQuery = fmt.Sprintf("LIMIT %s OFFSET %s", limit, offset)
		query = fmt.Sprintf("%s %s %s", selectQuery, orderQuery, offsetQuery)
	}
	var pgMembers []PGMember
	// query = fmt.Sprintf("%s %s %s", selectQuery, orderQuery, offsetQuery)
	err := d.db.Select(&pgMembers, query, entityID)
	if err != nil {
		log.Debug(err)
		return nil, err
	}
	members := make([]types.Member, len(pgMembers))
	for i, member := range pgMembers {
		members[i] = *ToMember(&member)
	}
	return members, nil
}

func (d *Database) DumpClaims(entityID []byte) ([][]byte, error) {
	var claims [][]byte
	var query string
	var err error
	query = `SELECT u.digested_public_key FROM users u 
			INNER JOIN members m ON m.public_key = u.public_key 
			WHERE m.entity_id = $1`
	if err = d.db.Select(&claims, query, entityID); err != nil {
		log.Debug(err)
		return nil, err
	}
	return claims, nil
}

func (d *Database) Census(censusID []byte) (*types.Census, error) {
	var census types.Census
	census.ID = []byte("0x0")
	return &census, nil
}

func (d *Database) Ping() error {
	return d.db.Ping()
}

// func (p *types.MembersCustomFields) Value() (driver.Value, error) {
// 	j, err := json.Marshal(p)
// 	return j, err
// }

// type StringArray []string

// func (p []string) Value() (driver.Value, error) {
// 	j, err := json.Marshal(p)
// 	return j, err
// }
