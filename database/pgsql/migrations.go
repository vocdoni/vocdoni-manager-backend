package pgsql

import (
	"fmt"

	migrate "github.com/rubenv/sql-migrate"
	"go.vocdoni.io/dvote/log"
	"go.vocdoni.io/manager/database"
)

// Migrations available
var Migrations = migrate.MemoryMigrationSource{
	Migrations: []*migrate.Migration{
		{
			Id:   "1",
			Up:   []string{migration1up},
			Down: []string{migration1down},
		},
		{
			Id:   "2",
			Up:   []string{migration2up},
			Down: []string{migration2down},
		},
		{
			Id:   "3",
			Up:   []string{migration3up},
			Down: []string{migration3down},
		},
		{
			Id:   "4",
			Up:   []string{migration4up},
			Down: []string{migration4down},
		},
	},
}

const migration1up = `
-- NOTES
-- 1. pgcrpyto is assumed to be enabled in public needing superuser access
--    CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;
-- 2. All columns are defined as NOT NULL to ease communication with Golang

CREATE EXTENSION IF NOT EXISTS pgcrypto SCHEMA public;

-- SQL in section 'Up' is executed when this migration is applied
--------------------------- TABLES DEFINITION
-------------------------------- -------------------------------- -------------------------------- 

--------------------------- ENTITTIES
-- An Entity/Organization

CREATE TABLE entities (
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    id bytea NOT NULL ,
    address bytea NOT NULL,
    email text NOT NULL,
    name text NOT NULL,
    callback_url text NOT NULL,
    callback_secret text NOT NULL,
    census_managers_addresses bytea[] NOT NULL
);

ALTER TABLE ONLY entities
    ADD CONSTRAINT entities_pkey PRIMARY KEY (id);

ALTER TABLE ONLY entities
    ADD CONSTRAINT entities_address_unique UNIQUE (address);

--------------------------- ENTITTIES_ORIGINS
-- The different types of origins and how the entitties support them

CREATE TYPE origins AS ENUM (
    'Token',
    'Form',
    'DB'
);

CREATE TABLE entities_origins (
    origin origins NOT NULL DEFAULT 'Form' ,
    entity_id bytea NOT NULL
);

ALTER TABLE entities_origins
    ADD CONSTRAINT entities_origins_entity_id_fkey FOREIGN KEY (entity_id) REFERENCES entities(id) ON DELETE CASCADE;

ALTER TABLE entities_origins
    ADD CONSTRAINT entities_origins_pkey  PRIMARY KEY (origin, entity_id);

--------------------------- USERS
-- A user, i.e. someone who is defined uniquely by his public key

CREATE TABLE users (
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    public_key bytea NOT NULL, 
    digested_public_key bytea NOT NULL
    --TODO: CONSTRAINT sizes of address
);

ALTER TABLE ONLY users
    ADD CONSTRAINT users_pkey PRIMARY KEY (public_key);

ALTER TABLE ONLY users
    ADD CONSTRAINT users_digested_public_key_unique UNIQUE (digested_public_key);

--------------------------- MEMBERS
-- A member is a user (N to 1) that is registered to an entity 
-- members N - 1 entities
-- members N - 1 users

CREATE TABLE members (
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    id uuid DEFAULT public.gen_random_uuid() NOT NULL,
    entity_id  bytea NOT NULL,
    public_key  bytea, 
    street_address  text NOT NULL,
    first_name  text NOT NULL,
    last_name  text NOT NULL,
    email  text NOT NULL,
    phone  text NOT NULL,
    date_of_birth timestamp with time zone NOT NULL,
    origin origins NOT NULL DEFAULT 'Token',
    consented boolean DEFAULT false NOT NULL,
    verified timestamp with time zone NOT NULL,
    custom_fields jsonb NOT NULL DEFAULT '{}'::jsonb
    --TODO: add JSONB with optional fields
    --TODO: CONSTRAINT sizes of address
);

ALTER TABLE ONLY members
    ADD CONSTRAINT members_pkey PRIMARY KEY (id);

ALTER TABLE ONLY members
    ADD CONSTRAINT members_entity_id_fkey FOREIGN KEY (entity_id) REFERENCES entities(id) ON DELETE CASCADE;

ALTER TABLE ONLY members
    ADD CONSTRAINT members_public_key_fkey FOREIGN KEY (public_key) REFERENCES users(public_key); -- ON DELETE CASCADE? 

ALTER TABLE ONLY members
    ADD CONSTRAINT members_entity_id_public_key_unique UNIQUE (entity_id, public_key);

ALTER TABLE ONLY members
    ADD CONSTRAINT members_entity_id_origins_fkey FOREIGN KEY (origin, entity_id) REFERENCES entities_origins;

--------------------------- PUSHTOKEN
-- A user push token for notifications
-- push_tokens N - 1 User

CREATE TABLE push_tokens (
    user_id bytea NOT NULL,  
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    token text NOT NULL
);

ALTER TABLE push_tokens
  ADD CONSTRAINT push_tokens_pkey PRIMARY KEY (token);

ALTER TABLE ONLY push_tokens
    ADD CONSTRAINT push_tokens_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(public_key) ON DELETE CASCADE;

--------------------------- TARGETS
-- A target is a set of filters maintained by an entity
-- targets N - 1 entities

CREATE TABLE targets (
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    id uuid DEFAULT public.gen_random_uuid() NOT NULL,
    entity_id bytea NOT NULL,
    name text NOT NULL,
    filters jsonb NOT NULL DEFAULT '{}'::jsonb
);

ALTER TABLE ONLY targets
    ADD CONSTRAINT targets_pkey PRIMARY KEY (id);

ALTER TABLE ONLY targets
    ADD CONSTRAINT targets_entity_id_fkey FOREIGN KEY (entity_id) REFERENCES entities(id) ON DELETE CASCADE;

ALTER TABLE ONLY targets
    ADD CONSTRAINT targets_entity_id_name_unique UNIQUE (entity_id, name);

--------------------------- CENSUS
-- A census is target that is exported, represinting the members that fullfil the filters of the target. 
-- census N - 1 entities
-- census N - 1 targets ???

CREATE TABLE censuses (
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    target_id uuid NOT NULL,
    id bytea NOT NULL,
    name text NOT NULL,
    entity_id bytea NOT NULL,
    merkle_root bytea NOT NULL, 
    merkle_tree_uri text NOT NULL,
    size integer NOT NULL
);

ALTER TABLE ONLY censuses
    ADD CONSTRAINT censuses_pkey PRIMARY KEY (id);

ALTER TABLE ONLY censuses
    ADD CONSTRAINT censuses_target_id_fkey FOREIGN KEY (target_id) REFERENCES targets(id);

ALTER TABLE ONLY censuses
    ADD CONSTRAINT censuses_entity_id_fkey FOREIGN KEY (entity_id) REFERENCES entities(id) ON DELETE CASCADE;

---------------------------- CENSUS MEMBERS
-- Census members is the relationship holds the members that exist in a census
-- census_members N - 1 members
-- census_members N - 1 census

CREATE TABLE census_members (
  member_id uuid NOT NULL,
  census_id bytea NOT NULL
);

ALTER TABLE ONLY census_members
    ADD CONSTRAINT census_members_pkey PRIMARY KEY (member_id, census_id);

ALTER TABLE ONLY census_members
    ADD CONSTRAINT census_members_member_id_fkey FOREIGN KEY (member_id) REFERENCES members(id) ON DELETE CASCADE;

ALTER TABLE ONLY census_members
    ADD CONSTRAINT census_members_census_id_fkey FOREIGN KEY (census_id) REFERENCES censuses(id) ON DELETE CASCADE;
`

const migration1down = `
DROP TABLE census_members;
DROP TABLE censuses;
DROP TABLE targets;
DROP TABLE push_tokens;
DROP TABLE members;
DROP TABLE users;
DROP TABLE entities_origins;
DROP TYPE origins;
DROP TABLE entities;
DROP EXTENSION IF EXISTS pgcrypto;
`

const migration2up = `
ALTER TABLE entities ADD COLUMN is_authorized boolean DEFAULT false NOT NULL;
`

const migration2down = `
ALTER TABLE entities DROP COLUMN is_authorized;
`

const migration3up = `
CREATE EXTENSION IF NOT EXISTS intarray SCHEMA public;
CREATE TABLE tags  (
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    id  SERIAL NOT NULL,
    name TEXT NOT NULL,
    entity_id bytea NOT NULL
);
ALTER TABLE ONLY tags
    ADD CONSTRAINT tags_pkey PRIMARY KEY (id);
ALTER TABLE ONLY tags
    ADD CONSTRAINT tags_name_entity_id_unique UNIQUE (name,entity_id);
ALTER TABLE ONLY tags
    ADD CONSTRAINT tags_entity_id_fkey FOREIGN KEY (entity_id) REFERENCES entities(id) ON DELETE CASCADE;
ALTER TABLE ONLY members ADD COLUMN tags int[] NOT NULL DEFAULT '{}';
`

const migration3down = `
DROP TABLE tags;
ALTER TABLE ONLY members DROP COLUMN tags;
DROP EXTENSION IF EXISTS intarray;
`

const migration4up = `
ALTER TABLE ONLY censuses
    ADD COLUMN ephemeral boolean DEFAULT false NOT NULL;
ALTER TABLE ONLY census_members
    ADD COLUMN ephemeral boolean DEFAULT false NOT NULL,
    ADD COLUMN public_key bytea,
    ADD COLUMN digested_public_key bytea,
    ADD COLUMN private_key bytea;
`

// ALTER TABLE ONLY members
//     ADD CONSTRAINT members_entity_id_email_unique UNIQUE (entity_id, email);

const migration4down = `
ALTER TABLE ONLY censuses
    DROP COLUMN ephemeral;
ALTER TABLE ONLY census_members
    DROP COLUMN ephemeral,
    DROP COLUMN public_key,
    DROP COLUMN digested_public_key,
    DROP COLUMN private_key;
`

// ALTER TABLE ONLY members
// DROP CONSTRAINT members_entity_id_email_unique;

func Migrator(action string, db database.Database) error {
	switch action {
	case "upSync":
		log.Infof("checking if DB is up to date")
		mTotal, mApplied, _, err := db.MigrateStatus()
		if err != nil {
			return fmt.Errorf("could not retrieve migrations status: (%v)", err)
		}
		if mTotal > mApplied {
			log.Infof("applying missing %d migrations to DB", mTotal-mApplied)
			n, err := db.MigrationUpSync()
			if err != nil {
				return fmt.Errorf("could not apply necessary migrations (%v)", err)
			}
			if n != mTotal-mApplied {
				return fmt.Errorf("could not apply all necessary migrations (%v)", err)
			}
		} else if mTotal < mApplied {
			return fmt.Errorf("someting goes terribly wrong with the DB migrations")
		}
	case "up", "down":
		log.Info("applying migration")
		op := migrate.Up
		if action == "down" {
			op = migrate.Down
		}
		n, err := db.Migrate(op)
		if err != nil {
			return fmt.Errorf("error applying migration: (%v)", err)
		}
		if n != 1 {
			return fmt.Errorf("reported applied migrations !=1")
		}
		log.Infof("%q migration complete", action)
	case "status":
		break
	default:
		return fmt.Errorf("unknown migrate command")
	}

	total, actual, record, err := db.MigrateStatus()
	if err != nil {
		return fmt.Errorf("could not retrieve migrations status: (%v)", err)
	}
	log.Infof("Total Migrations: %d\nApplied migrations: %d (%s)", total, actual, record)
	return nil
}
