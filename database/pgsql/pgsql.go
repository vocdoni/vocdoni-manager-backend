package pgsql

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	migrate "github.com/rubenv/sql-migrate"

	_ "github.com/jackc/pgx/stdlib"
	"gitlab.com/vocdoni/go-dvote/crypto/ethereum"
	"gitlab.com/vocdoni/go-dvote/crypto/snarks"
	"gitlab.com/vocdoni/go-dvote/log"

	"gitlab.com/vocdoni/manager/manager-backend/config"
	"gitlab.com/vocdoni/manager/manager-backend/types"
)

const connectionRetries = 5

type Database struct {
	db *sqlx.DB
	// For using pgx connector
	// pgx    *pgxpool.Pool
	// pgxCtx context.Context
}

// New creates a new postgres SQL database connection
func New(dbc *config.DB) (*Database, error) {
	db, err := sqlx.Open("pgx", fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s client_encoding=%s",
		dbc.Host, dbc.Port, dbc.User, dbc.Password, dbc.Dbname, dbc.Sslmode, "UTF8"))
	if err != nil {
		return nil, fmt.Errorf("error initializing postgres connection handler: %w", err)
	}

	// Try to get a connection, if fails connectionRetries times, return error.
	// This is necessary for ensuting the database connection is alive before going forward.
	for i := 0; i < connectionRetries; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		log.Infof("trying to connect to postgres")
		if _, err = db.Conn(ctx); err == nil {
			break
		}
		log.Warnf("database connection error (%s), retrying...", err)
		time.Sleep(time.Second * 2)
	}
	if err != nil {
		return nil, err
	}
	log.Info("connected to the database")

	// For using pgx connector
	// ctx := context.Background()
	// pgx, err := pgxpool.Connect(ctx, connectionString)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
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
		return fmt.Errorf("cannot initialize postgres transaction: %w", err)
	}
	entity := &types.Entity{
		EntityInfo: *info,
		ID:         entityID,
		CreatedUpdated: types.CreatedUpdated{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
	pgEntity, err := ToPGEntity(entity)
	if err != nil {
		return fmt.Errorf("cannot convert entity data types to postgres types: %w", err)
	}
	// TODO: Calculate EntityID (consult go-dvote)
	insert := `INSERT INTO entities
			(id, is_authorized, address, email, name, callback_url, callback_secret, census_managers_addresses, created_at, updated_at)
			VALUES (:id, :is_authorized, :address, :email, :name, :callback_url, :callback_secret, :pg_census_managers_addresses, :created_at, :updated_at)`
	_, err = tx.NamedExec(insert, pgEntity)
	if err != nil {
		return fmt.Errorf("cannot add insert query in the transaction: %w", err)
	}
	insertOrigins := `INSERT INTO entities_origins (entity_id,origin)
					VALUES ($1, unnest(cast($2 AS Origins[])))`
	_, err = tx.Exec(insertOrigins, entityID, pgEntity.Origins)
	if err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("cannot perform db rollback %v after error %w", rollbackErr, err)
		}
		return fmt.Errorf("cannot add insert query in the transaction: %w", err)
	}
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("cannot commit db queries :%w", err)
	}
	return nil
}

func (d *Database) Entity(entityID []byte) (*types.Entity, error) {
	var pgEntity PGEntity
	selectEntity := `SELECT id, is_authorized, address, email, name, callback_url, callback_secret, census_managers_addresses as "pg_census_managers_addresses"  
						FROM entities WHERE id=$1`
	row := d.db.QueryRowx(selectEntity, entityID)
	err := row.StructScan(&pgEntity)
	if err != nil {
		return nil, err
	}
	entity, err := ToEntity(&pgEntity)
	if err != nil {
		return nil, fmt.Errorf("cannot convert postgres types to entity data types: %w", err)
	}
	origins, err := d.EntityOrigins(entityID)
	if err != nil {
		if err != sql.ErrNoRows {
			return nil, fmt.Errorf("cannot entity origins: %w", err)
		}
		origins = []types.Origin{}

	}
	entity.Origins = origins
	return entity, nil
}

// EntitiesID returns all the entities ID's
func (d *Database) EntitiesID() ([]string, error) {
	var entitiesIDs [][]byte
	entitiesQuery := `SELECT id FROM entities`
	err := d.db.Select(&entitiesIDs, entitiesQuery)
	if err != nil {
		return nil, err
	}
	entities := []string{}
	for _, e := range entitiesIDs {
		entities = append(entities, hex.EncodeToString(e))
	}
	return entities, nil
}

func (d *Database) AuthorizeEntity(entityID []byte) error {
	entity := &types.Entity{ID: entityID, IsAuthorized: true}
	pgentity, err := ToPGEntity(entity)
	if err != nil {
		return fmt.Errorf("cannot convert member data types to postgres types: %w", err)
	}
	update := `UPDATE entities SET
				is_authorized = COALESCE(NULLIF(:is_authorized, false), is_authorized),
				updated_at = now()
				WHERE (id = :id )
				AND  :is_authorized IS DISTINCT FROM is_authorized`
	result, err := d.db.NamedExec(update, pgentity)
	if err != nil {
		return fmt.Errorf("error updating entity: %w", err)
	}
	var rows int64
	if rows, err = result.RowsAffected(); err != nil {
		return fmt.Errorf("cannot get affected rows: %w", err)
	} else if rows == 0 { /* Nothing to update? */
		return fmt.Errorf("already authorized")
	} else if rows != 1 { /* Nothing to update? */
		return fmt.Errorf("could not authorize")
	}
	return nil
}

func (d *Database) UpdateEntity(entityID []byte, info *types.EntityInfo) error {
	entity := &types.Entity{ID: entityID, EntityInfo: *info}
	pgentity, err := ToPGEntity(entity)
	if err != nil {
		return fmt.Errorf("cannot convert member data types to postgres types: %w", err)
	}
	// TODO: Implement Update CensusManagerAddresses (table)
	update := `UPDATE entities SET
				address = COALESCE(NULLIF(:address, decode('','hex')), address),
				name = COALESCE(NULLIF(:name, ''), name),
				callback_url = COALESCE(NULLIF(:callback_url, ''), callback_url),
				callback_secret = COALESCE(NULLIF(:callback_secret, ''), callback_secret),
				email = COALESCE(NULLIF(:email, ''), email),
				updated_at = now()
				WHERE (id = :id )
				AND  (:address IS DISTINCT FROM address OR
				:name IS DISTINCT FROM name OR
				:callback_url IS DISTINCT FROM callback_url OR
				:callback_secret IS DISTINCT FROM callback_secret OR
				:email IS DISTINCT FROM email)`
	result, err := d.db.NamedExec(update, pgentity)
	if err != nil {
		return fmt.Errorf("error updating entity: %w", err)
	}
	var rows int64
	if rows, err = result.RowsAffected(); err != nil {
		return fmt.Errorf("cannot get affected rows: %w", err)
	} else if rows == 0 { /* Nothing to update? */
		return fmt.Errorf("nothing to update")
	} else if rows != 1 { /* Nothing to update? */
		return fmt.Errorf("could not update")
	}
	return nil
}

func (d *Database) EntityOrigins(entityID []byte) ([]types.Origin, error) {
	var stringOrigins []string
	selectOrigins := `SELECT origin FROM entities_origins WHERE entity_id=$1`
	err := d.db.Select(&stringOrigins, selectOrigins, entityID)
	if err != nil {
		return nil, fmt.Errorf("cannot retrieve entity origins: %w", err)
	}
	origins, err := StringToOriginArray(stringOrigins)
	if err != nil {
		return nil, err
	}
	return origins, nil
}

func (d *Database) EntityHas(entityID []byte, memberID *uuid.UUID) bool {
	return true
}

func (d *Database) AddUser(user *types.User) error {
	if user.PubKey == nil {
		return fmt.Errorf("invalid public Key")
	}
	if len(user.DigestedPubKey) == 0 {
		user.DigestedPubKey = snarks.Poseidon.Hash(user.PubKey)
	}
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	insert := `INSERT INTO users
				(public_key, digested_public_key, created_at, updated_at)
				VALUES (:public_key, :digested_public_key, :created_at, :updated_at)`
	result, err := d.db.NamedExec(insert, user)
	if err != nil {
		return fmt.Errorf("cannot add user: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("cannot verify that user was added to the db: %w", err)
	}
	if rows != 1 {
		return fmt.Errorf("cannot add user")
	}
	return nil
}

func (d *Database) User(pubKey []byte) (*types.User, error) {
	var user types.User
	selectQuery := `SELECT
	 				public_key, digested_public_key
					FROM USERS where public_key=$1`
	row := d.db.QueryRowx(selectQuery, pubKey)
	if err := row.StructScan(&user); err != nil {
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
		pgmembers[idx] = PGMember{
			Member: types.Member{
				ID:         tokens[idx],
				EntityID:   entityID,
				MemberInfo: types.MemberInfo{},
				CreatedUpdated: types.CreatedUpdated{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
			},
		}
		// pgmembers[idx].ID = tokens[idx]
		// pgmembers[idx].EntityID = entityID
	}

	tx, err := d.db.Beginx()
	if err != nil {
		return fmt.Errorf("cannot initialize postgres transaction: %w", err)
	}
	insert := `INSERT INTO members
				(id,entity_id, public_key, street_address, first_name, last_name, email, phone, date_of_birth, verified, created_at, updated_at)
				VALUES (:id, :entity_id, :public_key, :street_address, :first_name, :last_name, :email, :phone, :date_of_birth, :verified, :created_at, :updated_at)`
	// result, err = tx.NamedExec(insert, pgmembers)
	if result, err = tx.NamedExec(insert, pgmembers); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("cannot perform db rollback %v after error %w", rollbackErr, err)
		}
		if err != nil {
			return fmt.Errorf("error in bulk import %w", err)
		}
	}
	if rows, err = result.RowsAffected(); err != nil || int(rows) != len(pgmembers) {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("cannot perform db rollback %v after error %w", rollbackErr, err)
		}
		if err != nil {
			return fmt.Errorf("error in bulk import %w", err)
		}
		return fmt.Errorf("should insert %d rows, while inserted %d rows. Rolled back", len(pgmembers), rows)
	}
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("could not commit bulk import %w", err)
	}
	return nil

}

// Store N  new Members associated to the Entity and return  their Tokens
func (d *Database) CreateNMembers(entityID []byte, n int) ([]uuid.UUID, error) {
	var tokens []uuid.UUID
	for i := 0; i < n; i++ {
		tokens = append(tokens, uuid.New())
	}
	return tokens, d.CreateMembersWithTokens(entityID, tokens)
}

// TODO: Implement import members

func (d *Database) AddMember(entityID []byte, pubKey []byte, info *types.MemberInfo) (uuid.UUID, error) {
	var tx *sqlx.Tx
	var err error
	var result *sqlx.Rows
	var id uuid.UUID
	member := &types.Member{EntityID: entityID, PubKey: pubKey, MemberInfo: *info}
	tx, err = d.db.Beginx()
	if err != nil {
		return uuid.Nil, fmt.Errorf("cannot initialize postgres transaction: %w", err)
	}
	_, err = d.User(pubKey)
	if err != nil && err != sql.ErrNoRows {
		return uuid.Nil, fmt.Errorf("error retrieving members corresponding user: %w", err)
	} else if err == sql.ErrNoRows {
		user := &types.User{
			PubKey: pubKey,
			CreatedUpdated: types.CreatedUpdated{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		}
		if len(user.DigestedPubKey) == 0 {
			user.DigestedPubKey = snarks.Poseidon.Hash(user.PubKey)
		}
		insert := `INSERT INTO users
					(public_key, digested_public_key, created_at, updated_at)
					VALUES (:public_key, :digested_public_key, :created_at, :updated_at)`
		var result sql.Result
		if result, err = tx.NamedExec(insert, user); err == nil {
			var rows int64
			if rows, err = result.RowsAffected(); err != nil || rows != 1 {
				return uuid.Nil, fmt.Errorf("error creating user for member: %w", err)
			}
		}
		if err != nil {
			if rollErr := tx.Rollback(); err != nil {
				return uuid.Nil, fmt.Errorf("error rolling back user creation for member: %v after error: %w", rollErr, err)
			}
			return uuid.Nil, fmt.Errorf("error creating user for member: %w", err)
		}
	}
	pgmember, err := ToPGMember(member)
	pgmember.CreatedAt = time.Now()
	pgmember.UpdatedAt = time.Now()
	if err != nil {
		return uuid.Nil, fmt.Errorf("cannot convert member data types to postgres types: %w", err)
	}
	insert := `INSERT INTO members
	 			(entity_id, public_key, street_address, first_name, last_name, email, phone, date_of_birth, verified, custom_fields, created_at, updated_at)
				VALUES (:entity_id, :public_key, :street_address, :first_name, :last_name, :email, :phone, :date_of_birth, :verified, :pg_custom_fields, :created_at, :updated_at)
				RETURNING id`
	// no err is returned if tx violated a db constraint,
	// but we need the result in order to get the created id.
	// LastInsertedID() is not exposed.
	// With Exec(), Scan() is not avaiable and with PrepareStmt()
	// is not possible to use pgmember and a conversion is needed.
	// So if no error is raised and the result has 0 rows it means
	// that something went wrong (no member added).
	if result, err = tx.NamedQuery(insert, pgmember); err != nil {
		if rollErr := tx.Rollback(); err != nil {
			return uuid.Nil, fmt.Errorf("error rolling back member and user creation: %v after error: %w", rollErr, err)
		}
		return uuid.Nil, fmt.Errorf("error adding member to the DB: %w", err)
	}
	if !result.Next() {
		if rollErr := tx.Rollback(); err != nil {
			return uuid.Nil, fmt.Errorf("error rolling back member and user creation: %v after error: %w", rollErr, err)
		}
		return uuid.Nil, fmt.Errorf("no rows affected after adding member, posible violation of db constraints")
	}
	if err = result.Scan(&id); err != nil {
		if rollErr := tx.Rollback(); err != nil {
			return uuid.Nil, fmt.Errorf("error rolling back member and user creation: %v after error: %w", rollErr, err)
		}
		return uuid.Nil, fmt.Errorf("error retrieving new member id: %w", err)
	}
	if err = result.Close(); err != nil {
		if rollErr := tx.Rollback(); err != nil {
			return uuid.Nil, fmt.Errorf("error rolling back member and user creation: %vafter error: %w", rollErr, err)
		}
		return uuid.Nil, fmt.Errorf("error retrieving new member id: %w", err)
	}
	if err = tx.Commit(); err != nil {
		if rollErr := tx.Rollback(); err != nil {
			return uuid.Nil, fmt.Errorf("error rolling back member and user creation: %v after error: %w", rollErr, err)
		}
		return uuid.Nil, fmt.Errorf("error commiting add member transactions to the DB: %w", err)
	}
	return id, err
}

// CreateEthRandomKeysBatch creates a set of eth random signing keys
func createEthRandomKeysBatch(n int) []*ethereum.SignKeys {
	s := make([]*ethereum.SignKeys, n)
	for i := 0; i < n; i++ {
		s[i] = ethereum.NewSignKeys()
		if err := s[i].Generate(); err != nil {
			return nil
		}
	}
	return s
}

func (d *Database) ImportMembersWithPubKey(entityID []byte, info []types.MemberInfo) error {
	var err error
	var result sql.Result
	var rows int64
	if len(info) <= 0 {
		return fmt.Errorf("no member data provided")
	}
	keys := createEthRandomKeysBatch(len(info))
	members := []PGMember{}
	for idx, member := range info {
		pub, _ := keys[idx].HexString()
		pubBytes, err := hex.DecodeString(pub)
		if err != nil {
			return fmt.Errorf("error decoding generated pubKey: %w", err)
		}
		user := &types.User{PubKey: pubBytes}
		err = d.AddUser(user)
		if err != nil {
			return fmt.Errorf("error creating generated user for imported member: %w", err)
		}
		newMember := &types.Member{EntityID: entityID, PubKey: pubBytes, MemberInfo: member}
		pgMember, err := ToPGMember(newMember)
		if err != nil {
			return fmt.Errorf("cannot convert member data types to postgres types: %w", err)
		}
		pgMember.CreatedAt = time.Now()
		pgMember.UpdatedAt = time.Now()
		members = append(members, *pgMember)
	}

	tx, err := d.db.Beginx()
	if err != nil {
		return fmt.Errorf("cannot initialize postgres transaction: %w", err)
	}
	insert := `INSERT INTO members
				(entity_id, public_key, street_address, first_name, last_name, email, phone, date_of_birth, verified, custom_fields, created_at, updated_at)
				VALUES (:entity_id, :public_key, :street_address, :first_name, :last_name, :email, :phone, :date_of_birth, :verified, :pg_custom_fields, :created_at, :updated_at)`
	if result, err = tx.NamedExec(insert, members); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("cannot perform db rollback %v after error %w", rollbackErr, err)
		}
		if err != nil {
			return fmt.Errorf("error in bulk import %w", err)
		}
	}
	if rows, err = result.RowsAffected(); err != nil || int(rows) != len(members) {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("cannot perform db rollback %v after error %w", rollbackErr, err)
		}
		if err != nil {
			return fmt.Errorf("error in bulk import %w", err)
		}
		return fmt.Errorf("should insert %d rows, while inserted %d rows. Rolled back", len(members), rows)
	}
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("could not commit bulk import %w", err)
	}
	return nil
}

func (d *Database) ImportMembers(entityID []byte, info []types.MemberInfo) error {
	// TODO: Check if support for Update a Member is needed
	// TODO: Investigate COPY FROM with pgx
	var err error
	var result sql.Result
	var rows int64
	if len(info) <= 0 {
		return fmt.Errorf("no member data provided")
	}
	members := []PGMember{}
	for _, member := range info {
		newMember := &types.Member{EntityID: entityID, MemberInfo: member}
		pgMember, err := ToPGMember(newMember)
		if err != nil {
			return fmt.Errorf("cannot convert member data types to postgres types: %w", err)
		}
		pgMember.CreatedAt = time.Now()
		pgMember.UpdatedAt = time.Now()
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
		return fmt.Errorf("cannot initialize postgres transaction: %w", err)
	}
	insert := `INSERT INTO members
				(entity_id, public_key, street_address, first_name, last_name, email, phone, date_of_birth, verified, custom_fields, created_at, updated_at)
				VALUES (:entity_id, :public_key, :street_address, :first_name, :last_name, :email, :phone, :date_of_birth, :verified, :pg_custom_fields, :created_at, :updated_at)`
	if result, err = tx.NamedExec(insert, members); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("cannot perform db rollback %v after error %w", rollbackErr, err)
		}
		if err != nil {
			return fmt.Errorf("error in bulk import %w", err)
		}
	}
	if rows, err = result.RowsAffected(); err != nil || int(rows) != len(members) {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("cannot perform db rollback %v after error %w", rollbackErr, err)
		}
		if err != nil {
			return fmt.Errorf("error in bulk import %w", err)
		}
		return fmt.Errorf("should insert %d rows, while inserted %d rows. Rolled back", len(members), rows)
	}
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("could not commit bulk import %w", err)
	}
	return nil
}

// AddMemberBulk imports an array of members to an entity,
// creating the corresponding users.
// ue to PostgreSQL and schema restriction the maximum array
// size acceptable is 5000 members
func (d *Database) AddMemberBulk(entityID []byte, members []types.Member) error {
	// TODO: Check if support for Update a Member is needed
	// TODO: Investigate COPY FROM with pgx
	if len(members) <= 0 {
		return fmt.Errorf("no member data provided")
	}
	tx, err := d.db.Beginx()
	if err != nil {
		return fmt.Errorf("cannot initialize postgres transaction: %w", err)
	}
	users := make([]types.User, len(members))
	pgMembers := []PGMember{}
	for idx, member := range members {
		// User-related
		if len(member.PubKey) == 0 {
			return fmt.Errorf("found empty public keys")
		}
		users[idx] = types.User{
			PubKey:         member.PubKey,
			DigestedPubKey: snarks.Poseidon.Hash(member.PubKey),
			CreatedUpdated: types.CreatedUpdated{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		}
		// Member related
		if len(member.EntityID) > 0 && !bytes.Equal(member.EntityID, entityID) {
			return fmt.Errorf("expected member entityID %x but provided entityID %x", entityID, member.EntityID)
		}
		if len(member.EntityID) == 0 {
			member.EntityID = entityID
		}

		pgMember, err := ToPGMember(&member)
		if err != nil {
			return fmt.Errorf("cannot convert member data types to postgres types: %w", err)
		}
		pgMember.CreatedAt = time.Now()
		pgMember.UpdatedAt = time.Now()
		pgMembers = append(pgMembers, *pgMember)
	}
	insertUsers := `INSERT INTO users
					(public_key, digested_public_key, created_at, updated_at) VALUES (:public_key, :digested_public_key, :created_at, :updated_at)`
	result, err := tx.NamedExec(insertUsers, users)
	if err != nil {
		return fmt.Errorf("error creating users %w", err)
	}
	userRows, err := result.RowsAffected()
	if err != nil || int(userRows) != len(users) {
		// First do rollback for all the error cases
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("cannot perform db rollback %v after error %w", rollbackErr, err)
		}
		// Then handle each error
		if err != nil {
			return fmt.Errorf("error verifying created users: (%v)", err)
		}
		return fmt.Errorf("expected to add %d users but added %d", int(userRows), len(users))

	}

	insert := `INSERT INTO members
				(entity_id, public_key, street_address, first_name, last_name, email, phone, date_of_birth, verified, custom_fields, created_at, updated_at)
				VALUES (:entity_id, :public_key, :street_address, :first_name, :last_name, :email, :phone, :date_of_birth, :verified, :pg_custom_fields, :created_at, :updated_at)`
	if result, err = tx.NamedExec(insert, pgMembers); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("cannot perform db rollback %v after error %w", rollbackErr, err)
		}
		if err != nil {
			return fmt.Errorf("error adding members: (%v)", err)
		}
	}
	memberRows, err := result.RowsAffected()
	if err != nil || int(memberRows) != len(members) || memberRows != userRows {
		// First do rollback for all the error cases
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("cannot perform db rollback %v after verifying created members error %w", rollbackErr, err)
		}
		// Then handle each error
		if err != nil {
			return fmt.Errorf("error verifying created members: (%v)", err)
		}
		if int(memberRows) != len(members) {
			return fmt.Errorf("expected to add %d members but added %d", len(members), memberRows)
		}
		return fmt.Errorf("diference in number of created users(%d) and members (%d)", len(members), memberRows)

	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("could not commit bulk import %w", err)
	}
	log.Debugf("AddMemberBulk: imported %d members and the correspondig users", len(members))
	return nil
}

func (d *Database) UpdateMember(entityID []byte, memberID *uuid.UUID, info *types.MemberInfo) error {
	if memberID == nil {
		return fmt.Errorf("memberID is nil")
	}
	member := &types.Member{ID: *memberID, EntityID: entityID, MemberInfo: *info}
	pgmember, err := ToPGMember(member)
	if err != nil {
		return fmt.Errorf("cannot convert member data types to postgres types: %w", err)
	}
	update := `UPDATE members SET
				street_address = COALESCE(NULLIF(:street_address, ''),  street_address),
				first_name = COALESCE(NULLIF(:first_name, ''), first_name),
				last_name = COALESCE(NULLIF(:last_name, ''), last_name),
				email = COALESCE(NULLIF(:email, ''), email),
				date_of_birth = COALESCE(NULLIF(:date_of_birth, date_of_birth), date_of_birth),
				tags = COALESCE(:pg_tags, CAST(tags as int[])),
				updated_at = now()
				WHERE (id = :id AND entity_id = :entity_id)
				AND  (:street_address IS DISTINCT FROM street_address OR
				:first_name IS DISTINCT FROM first_name OR
				:last_name IS DISTINCT FROM last_name OR
				:email IS DISTINCT FROM email OR
				:date_of_birth IS DISTINCT FROM date_of_birth OR
				:pg_tags  IS DISTINCT FROM tags)`
	var result sql.Result
	if result, err = d.db.NamedExec(update, pgmember); err != nil {
		return fmt.Errorf("error updating member: %w", err)
	}
	var rows int64
	if rows, err = result.RowsAffected(); err != nil {
		return fmt.Errorf("cannot get affected rows: %w", err)
	} else if rows != 1 { /* Nothing to update? */
		return fmt.Errorf("nothing to update: %w", err)
	}
	return nil
}

func (d *Database) AddTag(entityID []byte, tagName string) (int32, error) {
	if tagName == "" {
		log.Debugf("entity %x tried to creat tag with empty name", entityID)
		return 0, fmt.Errorf("invalid tag name")
	}
	tag := types.Tag{
		CreatedUpdated: types.CreatedUpdated{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		EntityID: entityID,
		Name:     tagName,
	}
	insert := `INSERT INTO tags
				(entity_id, name, created_at, updated_at)
				VALUES (:entity_id, :name, :created_at, :updated_at)
				RETURNING id`
	result, err := d.db.NamedQuery(insert, tag)
	if err != nil || !result.Next() {
		return 0, fmt.Errorf("error inserting tag: %w", err)
	}
	var id int32
	err = result.Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("error inserting tag: %w", err)
	}
	return id, nil

}

func (d *Database) ListTags(entityID []byte) ([]types.Tag, error) {
	if len(entityID) == 0 {
		log.Debugf("cannot retrieve tags for empty entityID")
		return nil, fmt.Errorf("invalid entity ID")
	}
	selectQuery := `SELECT id, name 
					FROM tags
					WHERE entity_id=$1`
	var tags []types.Tag
	if err := d.db.Select(&tags, selectQuery, entityID); err != nil {
		return nil, err
	}
	return tags, nil
}

func (d *Database) DeleteTag(entityID []byte, tagID int32) error {
	if len(entityID) == 0 {
		log.Debug("tried to delete tag for empty entityID")
		return fmt.Errorf("invalid entity ID")
	}

	// Check that tag exists
	_, err := d.Tag(entityID, tagID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("removing not existing tag %d for entity %x", tagID, entityID)
		}
		return fmt.Errorf("DeleteTag: error retrieving tag %d for %x : (%v)", tagID, entityID, err)
	}

	// Delete tag from members
	tx, err := d.db.Beginx()
	if err != nil {
		return fmt.Errorf("cannot initialize postgres transaction: %w", err)
	}
	queryData := struct {
		EntityID []byte `db:"entity_id"`
		TagID    int32  `db:"tag_id"`
	}{
		EntityID: entityID,
		TagID:    tagID,
	}
	// WARNING: Here tag is passed directly as to the SQL query since we are sure
	// that a tag with this ID exists
	update := `UPDATE members m SET 
					tags = array_remove(tags, :tag_id)
			    WHERE m.entity_id = :entity_id AND (m.tags && intset(:tag_id))`

	result, err := tx.NamedExec(update, queryData)
	if err != nil {
		return fmt.Errorf("DeleteTag: error adding  tag %d  to members of %x: (%v)", tagID, entityID, err)
	}

	if rows, err := result.RowsAffected(); err != nil {
		return fmt.Errorf("cannot get affected rows: %w", err)
	} else { // Nothing to update?
		log.Debugf("DeleteTag: removed tag from %d members", rows)
	}

	// Delete tags
	deleteQuery := `DELETE FROM tags WHERE id = $1 and entity_id =$2`
	result, err = tx.Exec(deleteQuery, tagID, entityID)
	if err != nil {
		return fmt.Errorf("DeleteTag: error deleting tags for entity %x: %w", entityID, err)
	}

	if _, err := result.RowsAffected(); err != nil {
		return fmt.Errorf("DeleteTag: error deleting tag %d for %x : %w", tagID, entityID, err)
	}
	if err = tx.Commit(); err != nil {
		if rollErr := tx.Rollback(); err != nil {
			return fmt.Errorf("something is very wrong: error rolling back on deleting tag: %v after final commit to DB: %w", rollErr, err)
		}
		return fmt.Errorf("error commiting delete transactions to the DB: %w", err)
	}
	return nil
}

func (d *Database) Tag(entityID []byte, tagID int32) (*types.Tag, error) {
	if len(entityID) == 0 || tagID == 0 {
		log.Debugf("Tag: invalid arguments: tag %d for entity %x", tagID, entityID)
		return nil, fmt.Errorf("invalid arguments")
	}
	selectQuery := `SELECT id, name
					FROM tags
					WHERE entity_id=$1 AND id=$2`
	var tag types.Tag
	if err := d.db.Get(&tag, selectQuery, entityID, tagID); err != nil {
		return nil, err
	}
	return &tag, nil
}

func (d *Database) TagByName(entityID []byte, tagName string) (*types.Tag, error) {
	if len(entityID) == 0 || len(tagName) == 0 {
		log.Debugf("Tag: invalid arguments: tag %s for entity %x", tagName, entityID)
		return nil, fmt.Errorf("invalid arguments")
	}
	selectQuery := `SELECT id, name
					FROM tags
					WHERE entity_id=$1 AND name=$2`
	var tag types.Tag
	if err := d.db.Get(&tag, selectQuery, entityID, tagName); err != nil {
		return nil, err
	}
	return &tag, nil
}

func (d *Database) AddTagToMembers(entityID []byte, members []uuid.UUID, tagID int32) (int, []uuid.UUID, error) {
	// Tags as text[] http://www.databasesoup.com/2015/01/tag-all-things.html
	// Tags as intarray
	var invalidTokens []uuid.UUID
	var updated int

	if len(entityID) == 0 {
		return updated, invalidTokens, fmt.Errorf("invalid arguments")
	}
	if len(members) == 0 {
		return updated, invalidTokens, nil
	}
	_, err := d.Tag(entityID, tagID)
	if err != nil {
		if err == sql.ErrNoRows {
			return updated, invalidTokens, fmt.Errorf("trying to add not existing tag %d for entity %x", tagID, entityID)
		}
		return updated, invalidTokens, fmt.Errorf("error retrieving tag %d for %x : (%v)", tagID, entityID, err)
	}
	type TagData struct {
		MemberID string `db:"member_id"`
		TagID    string `db:"tag_id"`
	}
	idTagsList := make([]*TagData, len(members))
	for i, memberID := range members {
		idTagsList[i] = &TagData{
			MemberID: memberID.String(),
			TagID:    strconv.FormatInt(int64(tagID), 10),
		}
	}

	// WARNING: Here tag is passed directly as to the SQL query since we are sure
	// that a tag with this ID exists
	update := fmt.Sprintf(`UPDATE members m SET 
					tags = array_append(tags, CAST(u.tag_id AS int))
				FROM (VALUES 
					(:member_id, :tag_id)
				)
				AS u(member_id,tag_id)			
		WHERE m.entity_id = decode('%x','hex') AND m.id = uuid(u.member_id) AND NOT (m.tags && intset(%d)) 
		RETURNING m.id`, entityID, tagID)

	result, err := d.db.NamedQuery(update, idTagsList)
	if err != nil {
		return updated, invalidTokens, fmt.Errorf("error adding  tag %d  to members of %x: (%v)", tagID, entityID, err)
	}
	var id uuid.UUID
	invalidTokensMap := make(map[uuid.UUID]bool)
	for _, token := range members {
		invalidTokensMap[token] = true
	}
	for result.Next() {
		if err := result.Scan(&id); err != nil {
			return updated, invalidTokens, fmt.Errorf("error parsing query result: %w", err)
		}
		updated++

		delete(invalidTokensMap, id)
	}
	invalidTokens = make([]uuid.UUID, len(invalidTokensMap))
	i := 0
	for k := range invalidTokensMap {
		invalidTokens[i] = k
		i++
	}
	return updated, invalidTokens, nil
}

func (d *Database) RemoveTagFromMembers(entityID []byte, members []uuid.UUID, tagID int32) (int, []uuid.UUID, error) {
	var invalidTokens []uuid.UUID
	var updated int

	if len(entityID) == 0 {
		return updated, invalidTokens, fmt.Errorf("invalid arguments")
	}
	if len(members) == 0 {
		return updated, invalidTokens, nil
	}
	tag, err := d.Tag(entityID, tagID)
	if err != nil {
		if err == sql.ErrNoRows {
			return updated, invalidTokens, fmt.Errorf("non-existing tag %d for entity %x", tagID, entityID)
		}
		return updated, invalidTokens, fmt.Errorf("RemoveTagFromMembers: error retrieving tag %d for %x : (%v)", tagID, entityID, err)
	}
	type TagData struct {
		MemberID string `db:"member_id"`
		TagID    string `db:"tag_id"`
	}
	idTagsMap := make([]*TagData, len(members))
	for i, memberID := range members {
		idTagsMap[i] = &TagData{
			MemberID: memberID.String(),
			TagID:    strconv.FormatInt(int64(tagID), 10),
		}
	}

	// WARNING: Here tag is passed directly as to the SQL query since we are sure
	// that a tag with this ID exists
	update := fmt.Sprintf(`UPDATE members m SET 
					tags = array_remove(tags, CAST(u.tag_id AS int))
				FROM (VALUES 
					(:member_id, :tag_id)
				)
				AS u(member_id,tag_id)			
				WHERE m.entity_id = decode('%x','hex') AND m.id = uuid(u.member_id) AND (m.tags && intset(%d)) 
				RETURNING m.id`, entityID, tag.ID)

	result, err := d.db.NamedQuery(update, idTagsMap)
	if err != nil {
		return updated, invalidTokens, fmt.Errorf("error removing  tag %d  to members of %x: (%v)", tagID, entityID, err)
	}

	var id uuid.UUID
	invalidTokensMap := make(map[uuid.UUID]bool)
	for _, token := range members {
		invalidTokensMap[token] = true
	}
	for result.Next() {
		if err := result.Scan(&id); err != nil {
			return updated, invalidTokens, fmt.Errorf("error parsing query result: %w", err)
		}
		updated++

		delete(invalidTokensMap, id)
	}
	invalidTokens = make([]uuid.UUID, len(invalidTokensMap))
	i := 0
	for k := range invalidTokensMap {
		invalidTokens[i] = k
		i++
	}
	return updated, invalidTokens, nil
}

// Register member to existing ID and generates corresponding user
func (d *Database) RegisterMember(entityID, pubKey []byte, token *uuid.UUID) error {
	if token == nil {
		return fmt.Errorf("token is nil")
	}
	var tx *sqlx.Tx
	var err error
	member := &types.Member{ID: *token, EntityID: entityID, PubKey: pubKey}
	tx, err = d.db.Beginx()
	if err != nil {
		return fmt.Errorf("cannot initialize postgres transaction: %w", err)
	}
	if len(pubKey) != ethereum.PubKeyLength/2 && len(pubKey) != ethereum.PubKeyLengthUncompressed/2 {
		return fmt.Errorf("invalid public key size")
	}
	_, err = d.User(pubKey)
	if err == sql.ErrNoRows {
		// This is the expected behaviour
		user := &types.User{
			PubKey:         pubKey,
			DigestedPubKey: snarks.Poseidon.Hash(pubKey),
			CreatedUpdated: types.CreatedUpdated{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		}
		insert := `INSERT INTO users
					(public_key, digested_public_key, created_at, updated_at)
					VALUES (:public_key, :digested_public_key, :created_at, :updated_at)`
		result, err := tx.NamedExec(insert, user)
		if err != nil {
			if rollErr := tx.Rollback(); err != nil {
				return fmt.Errorf("error rolling back user creation for member: %v after error: %w", rollErr, err)
			}
			return fmt.Errorf("error creating user for member: %w", err)
		}
		if rows, err := result.RowsAffected(); err != nil || rows != 1 {
			return fmt.Errorf("error creating user for member: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("error retrieving members corresponding user: %w", err)
	}

	pgmember, err := ToPGMember(member)
	if err != nil {
		return fmt.Errorf("cannot convert member data types to postgres types: %w", err)
	}
	update := `UPDATE members SET
				public_key = :public_key,
				updated_at = now(),
				verified = now()
				WHERE (id = :id AND entity_id = :entity_id)`
	var result sql.Result
	if result, err = tx.NamedExec(update, pgmember); err != nil {
		if rollErr := tx.Rollback(); err != nil {
			return fmt.Errorf("error rolling back member and user creation: %v after error: %w", rollErr, err)
		}
		return fmt.Errorf("error adding member to the DB: %w", err)
	}
	if rows, err := result.RowsAffected(); err != nil {
		if rollErr := tx.Rollback(); err != nil {
			return fmt.Errorf("error rolling back member and user creation: %v after not being able to get affected rows: %w", rollErr, err)
		}
		return fmt.Errorf("cannot get affected rows: %w", err)
	} else if rows != 1 { /* Nothing to update? */
		if rollErr := tx.Rollback(); err != nil {
			return fmt.Errorf("error rolling back member and user creation: %v after expecting 1 row update but found %d: %w", rollErr, rows, err)
		}
		return fmt.Errorf("expected 1 row affected after adding member, but found %d, posible violation of db constraints", rows)
	}
	if err = tx.Commit(); err != nil {
		if rollErr := tx.Rollback(); err != nil {
			return fmt.Errorf("error rolling back member and user creation: %v after final commit to DB: %w", rollErr, err)
		}
		return fmt.Errorf("error commiting add member transactions to the DB: %w", err)
	}
	return nil
}

func (d *Database) Member(entityID []byte, memberID *uuid.UUID) (*types.Member, error) {
	if memberID == nil {
		return nil, fmt.Errorf("memberID is nil")
	}
	var pgMember PGMember
	selectQuery := `SELECT
	 				id, entity_id, public_key, street_address, first_name, last_name, email, phone, date_of_birth, verified, custom_fields as "pg_custom_fields", consented, tags as "pg_tags"
					FROM members WHERE id = $1 and entity_id =$2`
	row := d.db.QueryRowx(selectQuery, memberID, entityID)
	if err := row.StructScan(&pgMember); err != nil {
		return nil, err
	}
	member := ToMember(&pgMember)
	return member, nil
}

func (d *Database) MemberByEmail(entityID []byte, email string) (*types.Member, error) {
	if len(email) == 0 || len(entityID) == 0 {
		return nil, fmt.Errorf("invalid arguments")
	}
	var pgMembers []PGMember
	selectQuery := `SELECT
	 				id, entity_id, public_key, street_address, first_name, last_name, email, phone, date_of_birth, verified, custom_fields as "pg_custom_fields", consented, tags as "pg_tags"
					FROM members WHERE entity_id =$1 AND email LIKE $2`
	err := d.db.Select(&pgMembers, selectQuery, entityID, email)
	if err != nil {
		log.Warnf("cannot retrieve member by email: (%v)", err)
		return nil, err
	}
	if len(pgMembers) > 1 {
		log.Warnf("memberByEmail:duplicate email")
		return nil, fmt.Errorf("duplicate email")
	}
	member := ToMember(&pgMembers[0])
	log.Debugf("MEMBER: %w", member)
	return member, nil
}

func (d *Database) Members(entityID []byte, memberIDs []uuid.UUID) ([]types.Member, []uuid.UUID, error) {
	var invalidTokens []uuid.UUID
	var members []types.Member
	if len(entityID) == 0 {
		return members, invalidTokens, fmt.Errorf("invalid arguments")
	}
	if len(memberIDs) == 0 {
		return members, invalidTokens, nil
	}
	// uniqueMembers := util.UniqueUUIDs(members)
	type MemberData struct {
		MemberID string `db:"member_id"`
	}

	membersList := make([]*MemberData, len(memberIDs))
	for i, memberID := range memberIDs {
		membersList[i] = &MemberData{
			MemberID: memberID.String(),
		}
	}

	update := `SELECT id, entity_id, public_key, street_address, first_name, last_name, email, phone, date_of_birth, verified, custom_fields as "pg_custom_fields", consented, tags as "pg_tags"
				FROM members 
				WHERE id IN (
					SELECT CAST(member_id AS uuid) FROM (VALUES 
							(:member_id)
						)
						AS u(member_id)	
					)`

	result, err := d.db.NamedQuery(update, membersList)
	if err != nil {
		return members, invalidTokens, err
	}

	var pgmember PGMember
	invalidTokensMap := make(map[uuid.UUID]bool)
	for _, token := range memberIDs {
		invalidTokensMap[token] = true
	}
	for result.Next() {
		if err := result.StructScan(&pgmember); err != nil {
			return members, invalidTokens, fmt.Errorf("error parsing query result: %w", err)
		}
		members = append(members, *ToMember(&pgmember))

		delete(invalidTokensMap, pgmember.ID)
	}
	invalidTokens = make([]uuid.UUID, len(invalidTokensMap))
	i := 0
	for k := range invalidTokensMap {
		invalidTokens[i] = k
		i++
	}
	return members, invalidTokens, nil
}

func (d *Database) DeleteMember(entityID []byte, memberID *uuid.UUID) error {
	if memberID == nil {
		return fmt.Errorf("memberID is nil")
	}
	var result sql.Result
	var err error
	deleteQuery := `DELETE FROM members WHERE id = $1 and entity_id =$2`
	if result, err = d.db.Exec(deleteQuery, *memberID, entityID); err == nil {
		var rows int64
		if rows, err = result.RowsAffected(); rows != 1 {
			return fmt.Errorf("nothing to delete")
		}
	}
	if err != nil {
		return fmt.Errorf("error deleting member: %w", err)
	}
	return nil
}

func (d *Database) DeleteMembers(entityID []byte, members []uuid.UUID) (int, []uuid.UUID, error) {
	var invalidTokens []uuid.UUID
	var updated int
	if len(entityID) == 0 {
		return updated, invalidTokens, fmt.Errorf("invalid arguments")
	}
	if len(members) == 0 {
		return updated, invalidTokens, nil
	}
	// uniqueMembers := util.UniqueUUIDs(members)
	type MemberData struct {
		MemberID string `db:"member_id"`
	}

	membersList := make([]*MemberData, len(members))
	for i, memberID := range members {
		membersList[i] = &MemberData{
			MemberID: memberID.String(),
		}
	}

	update := fmt.Sprintf(`DELETE FROM members 
					WHERE entity_id =  decode('%x','hex') AND id IN (
						SELECT CAST(member_id AS uuid) FROM (VALUES 
							(:member_id)
						)
						AS u(member_id)	
					)
					RETURNING id`, entityID)

	result, err := d.db.NamedQuery(update, membersList)
	if err != nil {
		return updated, invalidTokens, fmt.Errorf("error removing members of %x: (%v)", entityID, err)
	}

	// if err = result.Scan(&invalidTokens); err != nil {
	// 	log.Errorf("DeleteMembers: cannot parse query result: %w", err)
	// 	return invalidTokens, fmt.Errorf("cannot parse query result: %w", err)
	// }

	var id uuid.UUID
	invalidTokensMap := make(map[uuid.UUID]bool)
	for _, token := range members {
		invalidTokensMap[token] = true
	}
	for result.Next() {
		if err := result.Scan(&id); err != nil {
			return updated, invalidTokens, fmt.Errorf("error parsing query result: %w", err)
		}
		updated++

		delete(invalidTokensMap, id)
	}
	invalidTokens = make([]uuid.UUID, len(invalidTokensMap))
	i := 0
	for k := range invalidTokensMap {
		invalidTokens[i] = k
		i++
	}
	return updated, invalidTokens, nil
}

func (d *Database) MemberPubKey(entityID, pubKey []byte) (*types.Member, error) {
	var pgMember PGMember
	selectQuery := `SELECT
	 				id, entity_id, public_key, street_address, first_name, last_name, email, phone, date_of_birth, verified, custom_fields as "pg_custom_fields"
					FROM members WHERE public_key =$1 AND entity_id =$2`
	row := d.db.QueryRowx(selectQuery, pubKey, entityID)
	if err := row.StructScan(&pgMember); err != nil {
		return nil, err
	}
	member := ToMember(&pgMember)
	return member, nil
}

func (d *Database) MembersTokensEmails(entityID []byte) ([]types.Member, error) {
	selectQuery := `SELECT
	 				id, email
					FROM members WHERE entity_id = $1 AND public_key is null`

	var pgMembers []PGMember
	if err := d.db.Select(&pgMembers, selectQuery, entityID); err != nil {
		return nil, err
	}
	members := make([]types.Member, len(pgMembers))
	for i, member := range pgMembers {
		members[i] = *ToMember(&member)
	}
	return members, nil
}

func (d *Database) CountMembers(entityID []byte) (int, error) {
	if len(entityID) == 0 {
		return 0, fmt.Errorf("invalid entity id")
	}
	selectQuery := `SELECT COUNT(*) FROM members WHERE entity_id=$1`
	var membersCount int
	if err := d.db.Get(&membersCount, selectQuery, entityID); err != nil {
		return 0, err
	}
	return membersCount, nil
}

func (d *Database) ListMembers(entityID []byte, filter *types.ListOptions) ([]types.Member, error) {
	// TODO: Replace limit offset with better strategy, can slow down DB
	// would nee to now last value from previous query
	selectQuery := `SELECT
	 				id, entity_id, public_key, street_address, first_name, last_name, email, phone, date_of_birth, verified, custom_fields as "pg_custom_fields", tags as "pg_tags"
					FROM members WHERE entity_id =$1
					ORDER BY %s %s LIMIT $2 OFFSET $3`
	// Define default values for arguments
	t := reflect.TypeOf(types.MemberInfo{})
	field, found := t.FieldByName(strings.Title("lastName"))
	if !found {
		return nil, fmt.Errorf("lastName field not found in DB. Something is very wrong")
	}
	orderField := field.Tag.Get("db")
	order := "ASC"
	var limit, offset sql.NullInt32
	// default limit should be nil (Postgres BIGINT NULL)
	err := limit.Scan(nil)
	if err != nil {
		return nil, err
	}
	err = offset.Scan(0)
	if err != nil {
		return nil, err
	}
	// offset := 0
	if filter != nil {
		if len(filter.SortBy) > 0 {
			field, found := t.FieldByName(strings.Title(filter.SortBy))
			if found {
				if filter.Order == "descend" {
					order = "DESC"
				}
				orderField = field.Tag.Get("db")
			}
		}
		if filter.Skip > 0 {
			err = offset.Scan(filter.Skip)
			if err != nil {
				return nil, err
			}
		}
		if filter.Count > 0 {
			err = limit.Scan(filter.Count)
			if err != nil {
				return nil, err
			}
		}
	}

	query := fmt.Sprintf(selectQuery, orderField, order)
	var pgMembers []PGMember
	err = d.db.Select(&pgMembers, query, entityID, limit, offset)
	if err != nil {
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
	query := `SELECT u.digested_public_key FROM users u 
			INNER JOIN members m ON m.public_key = u.public_key 
			WHERE m.entity_id = $1`
	if err := d.db.Select(&claims, query, entityID); err != nil {
		return nil, err
	}
	return claims, nil
}

func (d *Database) DumpCensusClaims(entityID []byte, censusID []byte) ([][]byte, error) {
	// Verify that census belongs to this entity
	_, err := d.Census(entityID, censusID)
	if err != nil {
		log.Warnf("expandCensusClaims: cound not retrieve census: (%v)", err)
		return nil, fmt.Errorf("could not retrieve census")
	}
	var claims [][]byte
	query := `SELECT digested_public_key FROM census_members 
			WHERE census_id = $1`
	if err := d.db.Select(&claims, query, censusID); err != nil {
		return nil, err
	}
	return claims, nil
}

func (d *Database) ExpandCensusMembers(entityID, censusID []byte) ([]types.CensusMember, error) {
	// Get target Members with pks
	census, err := d.Census(entityID, censusID)
	if err != nil {
		log.Warnf("expandCensusClaims: cound not retrieve census: (%v)", err)
		return nil, fmt.Errorf("could not retrieve census")
	}
	members, err := d.TargetMembers(entityID, &census.TargetID)
	if err != nil {
		log.Warnf("expandCensusClaims: cound not retrieve target members: (%v)", err)
		return nil, fmt.Errorf("could not retrieve target members")
	}
	ephemeral := false
	// Create census_members struct and fill keys where ncessary
	censusMembers := make([]types.CensusMember, len(members))
	signKeys := ethereum.NewSignKeys()
	for i, member := range members {
		censusMembers[i].CensusID = censusID
		censusMembers[i].MemberID = member.ID
		if len(member.PubKey) == 0 || member.PubKey == nil {
			ephemeral = true
			if err := signKeys.Generate(); err != nil {
				log.Fatalf("expandCensusClaims: cound not generate emphemeral identity: (%v)", err)
				return nil, fmt.Errorf("could not generate emphemeral identity")
			}
			pubKey, privKey := signKeys.HexString()
			pubKey, err = ethereum.DecompressPubKey(pubKey)
			if err != nil {
				return nil, fmt.Errorf("cound not decompress emphemeral identity pubKey: (%v)", err)
			}
			pubKeyBytes, err := hex.DecodeString(pubKey)
			if err != nil {
				return nil, fmt.Errorf("cound not decode to bytes emphemeral identity pubKey: (%v)", err)
			}
			privKeyBytes, err := hex.DecodeString(privKey)
			if err != nil {
				return nil, fmt.Errorf("cound not decode to bytes emphemeral identity pubKey: (%v)", err)
			}
			censusMembers[i].Ephemeral = true
			censusMembers[i].PubKey = pubKeyBytes
			censusMembers[i].PrivKey = privKeyBytes
			censusMembers[i].DigestedPubKey = snarks.Poseidon.Hash(pubKeyBytes)
		} else {
			censusMembers[i].Ephemeral = false
			censusMembers[i].DigestedPubKey = snarks.Poseidon.Hash(member.PubKey)
		}
	}
	tx, err := d.db.Beginx()
	if err != nil {
		return nil, fmt.Errorf("could not initialize postgres transaction: %w", err)
	}
	if ephemeral {
		updateCensus := `UPDATE censuses SET ephemeral = true, size = $1  WHERE id = $2 AND entity_id = $3`
		result, err := tx.Exec(updateCensus, len(censusMembers), censusID, entityID)
		if err != nil {
			return nil, fmt.Errorf("could not update census as ephemeral: %w", err)
		}
		updatedRows, err := result.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf("could not verify updating census as ephemeral: (%v)", err)
		}
		if updatedRows != 1 {
			return nil, fmt.Errorf("could not update census as ephemeral")
		}
	}

	// update census members
	insertMembers := `INSERT INTO census_members (census_id, member_id, ephemeral, public_key, digested_public_key, private_key)
				  VALUES (:census_id, :member_id, :ephemeral, :public_key, :digested_public_key, :private_key)`
	result, err := tx.NamedExec(insertMembers, censusMembers)
	if err != nil {
		if rollErr := tx.Rollback(); err != nil {
			return nil, fmt.Errorf("something is very wrong: error rolling back: %v after error on updating census as ephemeral: %w", rollErr, err)
		}
		return nil, fmt.Errorf("could not add census_members to db: (%v)", err)
	}
	var addedRows int64
	if addedRows, err = result.RowsAffected(); err != nil {
		if rollErr := tx.Rollback(); err != nil {
			return nil, fmt.Errorf("something is very wrong: error rolling back: %v after error on counting affected rows: %w", rollErr, err)
		}
		log.Warnf("expandCensusClaims: could not count affected rows: (%v)", err)
		return nil, fmt.Errorf("could not verify updated rows")
	}
	if addedRows != int64(len(censusMembers)) {
		if rollErr := tx.Rollback(); err != nil {
			return nil, fmt.Errorf("something is very wrong: error rolling back: %v expected to have inserted %d census_members but inserted %d", rollErr, addedRows, len(censusMembers))
		}
		return nil, fmt.Errorf("expected to have inserted %d census_members but inserted %d", addedRows, len(censusMembers))
	}
	if err = tx.Commit(); err != nil {
		if rollErr := tx.Rollback(); err != nil {
			return nil, fmt.Errorf("something is very wrong: error rolling back: %v after final commit to DB: %w", rollErr, err)
		}
		return nil, fmt.Errorf("error commiting transactions to the DB: %w", err)
	}
	return censusMembers, nil
}

func (d *Database) ListEphemeralMemberInfo(entityID, censusID []byte) ([]types.EphemeralMemberInfo, error) {
	// TODO combine this query with the select query
	// TODO Find how to optimize query (searching by member Id that is first on the index?)
	census, err := d.Census(entityID, censusID)
	if err != nil {
		log.Warnf("listEphemeralMemberInfo: cound not retrieve census: (%v)", err)
		return nil, fmt.Errorf("could not retrieve census")
	}
	selectQuery := `SELECT id, first_name, last_name, email, private_key, c.digested_public_key as "digested_public_key"
					FROM  census_members c
					INNER JOIN members m  ON m.id = c.member_id
					WHERE c.census_id = $1 AND c.ephemeral = true`
	var info []types.EphemeralMemberInfo
	if err := d.db.Select(&info, selectQuery, census.ID); err != nil {
		return nil, fmt.Errorf("could not retrieve census members info: (%v)", err)
	}
	return info, nil
}

func (d *Database) EphemeralMemberInfoByEmail(entityID, censusID []byte, email string) (*types.EphemeralMemberInfo, error) {
	// TODO combine this query with the select query
	// TODO Find how to optimize query (searching by member Id that is first on the index?)
	census, err := d.Census(entityID, censusID)
	if err != nil {
		return nil, fmt.Errorf("cound not retrieve census: %w", err)
	}
	member, err := d.MemberByEmail(entityID, email)
	if err != nil {
		return nil, fmt.Errorf("cound not retrieve member by email: %w", err)
	}
	if member.PubKey != nil && len(member.PubKey) > 0 {
		return nil, fmt.Errorf("member not ephmeral: %d %x", len(member.PubKey), member.PubKey)
	}

	selectQuery := `SELECT * FROM census_members
					WHERE census_id = $1 AND ephemeral = true`
	var censusMember types.CensusMember
	// var info types.EphemeralMemberInfo
	if err := d.db.Get(&censusMember, selectQuery, census.ID); err != nil {
		return nil, fmt.Errorf("could not retrieve census members info: %w", err)
	}
	info := types.EphemeralMemberInfo{
		ID:             member.ID,
		FirstName:      member.FirstName,
		LastName:       member.LastName,
		Email:          member.Email,
		PrivKey:        censusMember.PrivKey,
		DigestedPubKey: censusMember.DigestedPubKey,
	}
	return &info, nil
}

func (d *Database) AddTarget(entityID []byte, target *types.Target) (uuid.UUID, error) {
	var err error
	if len(entityID) == 0 {
		return uuid.Nil, fmt.Errorf("adding target for other entity")
	}
	if len(target.EntityID) == 0 {
		target.EntityID = entityID
	}
	if hex.EncodeToString(target.EntityID) != hex.EncodeToString(entityID) {
		return uuid.Nil, fmt.Errorf("trying to add target for another entity")
	}
	target.CreatedAt = time.Now()
	target.UpdatedAt = time.Now()
	insert := `INSERT INTO targets
	 			(entity_id, name, filters, created_at, updated_at)
				VALUES (:entity_id, :name, :filters, :created_at, :updated_at)
				RETURNING id`
	// no err is returned if tx violated a db constraint,
	// but we need the result in order to get the created id.
	// LastInsertedID() is not exposed.
	// With Exec(), Scan() is not avaiable and with PrepareStmt()
	// is not possible to use pgmember and a conversion is needed.
	// So if no error is raised and the result has 0 rows it means
	// that something went wrong (no member added).
	var result *sqlx.Rows
	result, err = d.db.NamedQuery(insert, target)
	if err != nil {
		return uuid.Nil, err
	}
	if !result.Next() {
		return uuid.Nil, fmt.Errorf("result has no rows, posible violation of db constraints")
	}
	var id uuid.UUID
	err = result.Scan(&id)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func (d *Database) Target(entityID []byte, targetID *uuid.UUID) (*types.Target, error) {
	if len(entityID) == 0 || targetID == nil || *targetID == uuid.Nil {
		return nil, fmt.Errorf("error retrieving target")
	}
	selectQuery := `SELECT id, entity_id, name, filters 
					FROM targets
					WHERE entity_id=$1 AND id=$2`
	var target types.Target
	if err := d.db.Get(&target, selectQuery, entityID, targetID); err != nil {
		return nil, err
	}
	return &target, nil
}

func (d *Database) CountTargets(entityID []byte) (int, error) {
	if len(entityID) == 0 {
		return 0, fmt.Errorf("invalid entity id")
	}
	selectQuery := `SELECT COUNT(*) FROM targets WHERE entity_id=$1`
	var targetsCount int
	if err := d.db.Get(&targetsCount, selectQuery, entityID); err != nil {
		return 0, err
	}
	return targetsCount, nil
}

func (d *Database) ListTargets(entityID []byte) ([]types.Target, error) {
	if len(entityID) == 0 {
		return nil, fmt.Errorf("error retrieving target")
	}
	selectQuery := `SELECT id, entity_id, name, filters 
					FROM targets
					WHERE entity_id=$1`
	var targets []types.Target
	if err := d.db.Select(&targets, selectQuery, entityID); err != nil {
		return nil, err
	}
	return targets, nil
}

func (d *Database) TargetMembers(entityID []byte, targetID *uuid.UUID) ([]types.Member, error) {
	// TODO: Implement filters
	if targetID == nil {
		return nil, fmt.Errorf("targetID is nil")
	}
	return d.ListMembers(entityID, &types.ListOptions{})
}

func (d *Database) Census(entityID, censusID []byte) (*types.Census, error) {
	if len(entityID) == 0 || len(censusID) < 1 {
		return nil, fmt.Errorf("error retrieving target")
	}
	var census types.Census
	selectQuery := `SELECT id, entity_id, target_id, name, size, merkle_root, merkle_tree_uri, ephemeral, created_at, updated_at
					FROM censuses
					WHERE entity_id = $1 AND id = $2`
	row := d.db.QueryRowx(selectQuery, entityID, censusID)
	if err := row.StructScan(&census); err != nil {
		return nil, err
	}
	return &census, nil
}

func (d *Database) AddCensus(entityID, censusID []byte, targetID *uuid.UUID, info *types.CensusInfo) error {
	var err error
	var rows int64
	if len(entityID) == 0 || len(censusID) == 0 || targetID == nil || *targetID == uuid.Nil {
		return fmt.Errorf("invalid arguments")
	}

	if info == nil {
		info = &types.CensusInfo{
			MerkleRoot: []byte{},
		}
	}
	if info.MerkleRoot == nil {
		info.MerkleRoot = []byte{}
	}
	// TODO check valid target selecting
	info.CreatedAt = time.Now()
	info.UpdatedAt = time.Now()
	census := types.Census{
		ID:         censusID,
		EntityID:   entityID,
		TargetID:   *targetID,
		CensusInfo: *info,
	}
	insert := `INSERT  
				INTO censuses
	 			(id, entity_id, target_id, name, size, merkle_root, merkle_tree_uri, created_at, updated_at)
				VALUES (:id, :entity_id, :target_id, :name, :size, :merkle_root, :merkle_tree_uri, :created_at, :updated_at)`
	var result sql.Result

	if result, err = d.db.NamedExec(insert, census); err == nil {
		if rows, err = result.RowsAffected(); err == nil && rows != 1 {
			return fmt.Errorf("failed to add census: rows != 1")
		}
	}
	if err != nil {
		return fmt.Errorf("failed to add census: %w", err)
	}
	return nil
}

func (d *Database) AddCensusWithMembers(entityID, censusID []byte, targetID *uuid.UUID, info *types.CensusInfo) (int64, error) {
	var err error
	if len(entityID) == 0 || len(censusID) == 0 || targetID == nil || *targetID == uuid.Nil {
		return 0, fmt.Errorf("invalid arguments")
	}
	// TODO check valid target selecting

	// TODO Enable upon implementing targets (also enalbe manager_test targets)
	// members, err := d.TargetMembers(entityID, targetID)
	// if err != nil {
	// 	return 0, fmt.Errorf("failed to recover target members: %w", err)
	// }
	// if len(members) == 0 {
	// 	return 0, fmt.Errorf("target contains 0 members")
	// }
	// TODO Disable upon implementing targets
	var members []types.Member
	query := `SELECT m.id FROM members m
			INNER JOIN users u ON m.public_key = u.public_key 
			WHERE m.entity_id = $1`
	if err := d.db.Select(&members, query, entityID); err != nil {
		return 0, err
	}
	if len(members) == 0 {
		return 0, fmt.Errorf("target contains 0 members")
	}

	census := types.Census{ID: censusID, EntityID: entityID, TargetID: *targetID, CensusInfo: *info}
	census.CreatedAt = time.Now()
	census.UpdatedAt = time.Now()
	tx, err := d.db.Beginx()
	if err != nil {
		return 0, fmt.Errorf("cannot initialize postgres transaction: %w", err)
	}
	insertCensus := `INSERT  INTO censuses
					(id, entity_id, target_id, name, size, merkle_root, merkle_tree_uri, created_at, updated_at)
					VALUES (:id, :entity_id, :target_id, :name, :size, :merkle_root, :merkle_tree_uri, :created_at, :updated_at)`
	result, err := tx.NamedExec(insertCensus, census)
	if err != nil {
		return 0, fmt.Errorf("cannot add census: %w", err)
	}
	if rows, err := result.RowsAffected(); err != nil || rows != 1 {
		return 0, fmt.Errorf("cannot add census: %w", err)
	}

	censusMembers := make([]types.CensusMember, len(members))
	for idx, member := range members {
		censusMembers[idx].CensusID = censusID
		censusMembers[idx].MemberID = member.ID
	}

	insertMembers := `INSERT INTO census_members (census_id, member_id)
				  VALUES (:census_id, :member_id)`
	result, err = tx.NamedExec(insertMembers, censusMembers)
	if err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return 0, fmt.Errorf("cannot perform db rollback %v after error %w", rollbackErr, err)
		}
		return 0, fmt.Errorf("rolled back due to error inserting census members: %w", err)
	}
	var addedRows int64
	if addedRows, err = result.RowsAffected(); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return 0, fmt.Errorf("cannot perform db rollback %v after error %w", rollbackErr, err)
		}
		return 0, fmt.Errorf("rolled back error retriveing census members added count: %w", err)
	}
	if addedRows != int64(len(censusMembers)) {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return 0, fmt.Errorf("cannot perform db rollback %v after error %w", rollbackErr, err)
		}
		return 0, fmt.Errorf("rolled back because failed to add census members, expected to add %d members but added %d", len(censusMembers), addedRows)
	}
	updateCensus := `UPDATE censuses SET size = $1, updated_at = now() WHERE id = $2`
	result, err = tx.Exec(updateCensus, addedRows, censusID)
	if err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return 0, fmt.Errorf("cannot perform db rollback %v after error %w", rollbackErr, err)
		}
		return 0, fmt.Errorf("rolled back due to error updating census size: %w", err)
	}
	if updated, err := result.RowsAffected(); err != nil || updated != 1 {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return 0, fmt.Errorf("cannot perform db rollback %v after error %w", rollbackErr, err)
		}
		return 0, fmt.Errorf("rolled back due to error updating census size: %w", err)
	}
	if err := tx.Commit(); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return 0, fmt.Errorf("cannot perform db rollback %v after error %w", rollbackErr, err)
		}
		return 0, fmt.Errorf("rolled back because could not commit addCensus and addCensusMembers: %w", err)
	}
	return addedRows, nil
}

func (d *Database) UpdateCensus(entityID, censusID []byte, info *types.CensusInfo) error {
	var err error
	if len(entityID) == 0 || len(censusID) == 0 || info == nil {
		return fmt.Errorf("invalid arguments")
	}
	// TODO check valid target selecting
	if info.MerkleRoot == nil {
		info.MerkleRoot = []byte{}
	}
	info.CreatedAt = time.Now()
	info.UpdatedAt = time.Now()
	census := types.Census{
		ID:         censusID,
		EntityID:   entityID,
		CensusInfo: *info,
	}
	update := `UPDATE censuses SET
				merkle_root = COALESCE(NULLIF(:merkle_root, '' ::::bytea ),  merkle_root),
				merkle_tree_uri = COALESCE(NULLIF(:merkle_tree_uri, ''),  merkle_tree_uri) ,
				updated_at = now()
				WHERE id = :id AND entity_id = :entity_id`
	var result sql.Result
	if result, err = d.db.NamedExec(update, census); err != nil {
		return fmt.Errorf("error updating census: %w", err)
	}
	var rows int64
	if rows, err = result.RowsAffected(); err != nil {
		return fmt.Errorf("cannot get affected rows: %w", err)
	} else if rows != 1 { /* Nothing to update? */
		return fmt.Errorf("nothing to update: %w", err)
	}
	return nil
}

func (d *Database) CountCensus(entityID []byte) (int, error) {
	if len(entityID) == 0 {
		return 0, fmt.Errorf("invalid entity id")
	}
	selectQuery := `SELECT COUNT(*) FROM censuses WHERE entity_id=$1`
	var censusCount int
	if err := d.db.Get(&censusCount, selectQuery, entityID); err != nil {
		return 0, err
	}
	return censusCount, nil
}

func (d *Database) ListCensus(entityID []byte, filter *types.ListOptions) ([]types.Census, error) {
	// check entityID
	if len(entityID) == 0 {
		return nil, fmt.Errorf("error retrieving target")
	}
	// create select query
	selectQuery := `SELECT id, entity_id, target_id, name, merkle_root, merkle_tree_uri, created_at, updated_at
					FROM censuses
					WHERE entity_id=$1
					ORDER BY %s %s LIMIT $2 OFFSET $3`
	// define default values for query args
	t := reflect.TypeOf(types.Census{})
	field, found := t.FieldByName(strings.Title("name"))
	if !found {
		return nil, fmt.Errorf("name field not found in DB. Something is very wrong")
	}
	orderField := field.Tag.Get("db")
	order := "ASC"
	var limit, offset sql.NullInt32
	// default limit should be nil
	if err := limit.Scan(nil); err != nil {
		return nil, err
	}
	if err := offset.Scan(0); err != nil {
		return nil, err
	}
	// offset = 0
	// check filter
	if filter != nil {
		// check sortBy
		if len(filter.SortBy) > 0 {
			if field, found = t.FieldByName(strings.Title(filter.SortBy)); found {
				if filter.Order == "descend" {
					order = "DESC"
				}
				orderField = field.Tag.Get("db")
			}
		}
		// check skip
		if filter.Skip > 0 {
			if err := offset.Scan(filter.Skip); err != nil {
				return nil, err
			}
		}
		// check count
		if filter.Count > 0 {
			if err := limit.Scan(filter.Count); err != nil {
				return nil, err
			}
		}
	}
	query := fmt.Sprintf(selectQuery, orderField, order)
	var censuses []types.Census
	if err := d.db.Select(&censuses, query, entityID, limit, offset); err != nil {
		return nil, err
	}
	return censuses, nil
}

func (d *Database) DeleteCensus(entityID []byte, censusID []byte) error {
	if len(censusID) == 0 || len(entityID) == 0 {
		log.Debug("deleteCensus: invalid arguments")
		return fmt.Errorf("invalid arguments")
	}

	deleteQuery := `DELETE FROM censuses WHERE id = $1 and entity_id =$2`
	result, err := d.db.Exec(deleteQuery, censusID, entityID)
	if err != nil {
		return fmt.Errorf("error deleting census: %w", err)
	}
	if err == nil {
		if rows, err := result.RowsAffected(); rows != 1 {
			return fmt.Errorf("nothing to delete")
		} else if err != nil {
			return fmt.Errorf("error verifying deleted census: %w", err)
		}
	}

	return nil
}

func (d *Database) Ping() error {
	return d.db.Ping()
}

// Migrate performs a concrete migration (up or down)
func (d *Database) Migrate(dir migrate.MigrationDirection) (int, error) {
	n, err := migrate.ExecMax(d.db.DB, "postgres", Migrations, dir, 1)
	if err != nil {
		return 0, fmt.Errorf("failed migration: %w", err)
	}
	return n, nil
}

// Migrate returns the total and applied number of migrations,
// as well a string describing the perform migrations
func (d *Database) MigrateStatus() (int, int, string, error) {
	total, err := Migrations.FindMigrations()
	if err != nil {
		return 0, 0, "", fmt.Errorf("cannot retrieve total migrations status: %w", err)
	}
	record, err := migrate.GetMigrationRecords(d.db.DB, "postgres")
	if err != nil {
		return len(total), 0, "", fmt.Errorf("cannot  retrieve applied migrations status: %w", err)
	}
	recordB, err := json.Marshal(record)
	if err != nil {
		return len(total), len(record), "", fmt.Errorf("failed to parse migration status: %w", err)
	}
	return len(total), len(record), string(recordB), nil
}

// MigrationUpSync performs the missing up migrations in order to reach to highest migration
// available in migrations.go
func (d *Database) MigrationUpSync() (int, error) {
	n, err := migrate.ExecMax(d.db.DB, "postgres", Migrations, migrate.Up, 0)
	if err != nil {
		return 0, fmt.Errorf("cannot  perform missing migrations: %w", err)
	}
	return n, nil
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
