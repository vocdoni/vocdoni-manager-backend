#!/usr/bin/env bash
# Creates a database, user, and extensions to use Devicehub
# $DB is the database to create
# $USER is the user to create and give full permissions on the database
# This script asks for the password of such user
set -e
DB='vocdonimgr'

psql --username "$POSTGRES_USER" --dbname "postgres" --command "CREATE DATABASE $DB ENCODING = 'UTF8' LC_COLLATE = 'en_US.utf8' LC_CTYPE = 'en_US.utf8';"
# psql -d $DB -c "CREATE USER $POSTGRES_USER WITH PASSWORD '$POSTGRES_PASSWORD';" # Create user Devicehub uses to access db
psql --username "$POSTGRES_USER" --dbname "$DB" -c "GRANT ALL PRIVILEGES ON DATABASE $DB TO $POSTGRES_USER;" # Give access to the db
# psql --username "$POSTGRES_USER" --dbname "$DB"  -c "CREATE EXTENSION pgcrypto SCHEMA public;" # Enable pgcrypto
