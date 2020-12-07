module gitlab.com/vocdoni/manager/manager-backend

go 1.14

require (
	firebase.google.com/go/v4 v4.0.0
	github.com/Pallinder/go-randomdata v1.2.0
	github.com/badoux/checkmail v0.0.0-20181210160741-9661bd69e9ad
	github.com/dgraph-io/badger/v2 v2.2007.2 // indirect
	github.com/dgraph-io/ristretto v0.0.3 // indirect
	github.com/ethereum/go-ethereum v1.9.20
	github.com/go-chi/chi v4.1.2+incompatible // indirect
	github.com/gobuffalo/packr/v2 v2.7.1
	github.com/golang/snappy v0.0.2 // indirect
	github.com/google/go-cmp v0.5.2 // indirect
	github.com/google/uuid v1.1.1
	github.com/jackc/fake v0.0.0-20150926172116-812a484cc733 // indirect
	github.com/jackc/pgtype v1.3.1-0.20200521144610-9d847241cb8f
	github.com/jackc/pgx v3.6.2+incompatible
	github.com/jmoiron/sqlx v1.2.1-0.20200615141059-0794cb1f47ee
	github.com/klauspost/compress v1.11.1 // indirect
	github.com/knadh/smtppool v0.2.1
	github.com/lib/pq v1.8.0
	github.com/prometheus/client_golang v1.7.1
	github.com/rubenv/sql-migrate v0.0.0-20200616145509-8d140a17f351
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.7.0
	gitlab.com/vocdoni/go-dvote v0.6.3-0.20201110140317-7b855a3d21cc
	golang.org/x/crypto v0.0.0-20201002170205-7f63de1d35b0 // indirect
	golang.org/x/mod v0.3.0 // indirect
	golang.org/x/net v0.0.0-20201002202402-0a1ea396d57c // indirect
	golang.org/x/sys v0.0.0-20201005065044-765f4ea38db3 // indirect
	golang.org/x/text v0.3.3 // indirect
	golang.org/x/tools v0.0.0-20200617042924-7f3f4b10a808 // indirect
	google.golang.org/api v0.17.0
	nhooyr.io/websocket v1.8.6
)

replace github.com/knadh/smtppool => github.com/emmdim/smtppool v0.2.2-0.20201207174605-7a30c40886de
