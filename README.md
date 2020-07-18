# Vocdoni Manager Backend

Vocdoni manager backend

### Init docker and connect

#### Build

```bash
$ docker-compose build
```

#### Run

Start postgres, dvotemanager and a testing frontend:

```bash
$ docker-compose up
```

Start only postgres and the fronted:

```bash
$ docker-compose up -d postgres
$ docker-compose up -d webmanager
```

#### Stop and delete database

```bash
$ docker-compose down -v
```

#### Connect dvotemanager

```bash
$ go run cmd/dvotemanager/dvotemanager.go --dbSslmode="disable" --dbUser="vocdoni" --dbPassword="vocdoni" --dbName="vocdonimgr"
```

#### Integration test

```bash
$ cd test && go test ./...
```
