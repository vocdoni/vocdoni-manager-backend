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
<p>&nbsp;</p>

## Notifications service

The notifications service is a service that tracks the metadata of the entities stored in an entity manager database, it also tracks each entity
process creation (via EVM compatible chain). Both trackings are used to send push notifications according to a logic with the data tracked.

The notification service needs:

- To be connected to the manager backend.
- To be connected to an external web3 provider.
- To have an IPFS node to conecto to.

**Config file example**

```yaml
datadir: /home/laptop/.dvotenotif
db:
  dbname: vocdoni
  host: 127.0.0.1
  password: vocdoni
  port: 5432
  sslmode: prefer
  user: vocdoni
ethereum:
  bootnodes: '[]'
  chaintype: sokol
  datadir: /home/laptop/.dvotenotif/ethereum
  lightmode: false
  nodeport: 30303
  nowaitsync: true
  processdomain: voting-process.vocdoni.eth
  signingkey: 0x0
  trustedpeers: '[]'
ethereumevents:
  censussync: true
  subscribeonly: true
ipfs:
  configpath: /home/laptop/.dvotenotif/ipfs
  noinit: false
  synckey: ""
  syncpeers: '[]'
logerrorfile: ""
loglevel: debug
logoutput: stdout
metrics:
  enabled: true
  refreshinterval: 10
migrate:
  action: ""
notifications:
  keyfile: /home/laptop/.dvotemanager/key
  service: 1
web3:
  enabled: true
  route: /web3
  rpchost: 127.0.0.1
  rpcport: 9091
  w3external: ws://sokol.poa.network:8546
```

#### Run

```bash
go run cmd/dvotenotif/dvotenotif.go --pushNotificationsKeyFile priv_file --ethSigningKey 0x0 --w3External ws://sokol.poa.network:8546 --ethChain sokol --logLevel debug
```