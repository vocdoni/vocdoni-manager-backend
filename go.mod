module go.vocdoni.io/manager

go 1.16

require (
	firebase.google.com/go/v4 v4.0.0
	github.com/Pallinder/go-randomdata v1.2.0
	github.com/badoux/checkmail v0.0.0-20181210160741-9661bd69e9ad
	github.com/ethereum/go-ethereum v1.10.8
	github.com/frankban/quicktest v1.13.0
	github.com/google/uuid v1.3.0
	github.com/jackc/fake v0.0.0-20150926172116-812a484cc733 // indirect
	github.com/jackc/pgtype v1.3.1-0.20200521144610-9d847241cb8f
	github.com/jackc/pgx v3.6.2+incompatible
	github.com/jmoiron/sqlx v1.2.1-0.20200615141059-0794cb1f47ee
	github.com/knadh/smtppool v0.3.0
	github.com/lib/pq v1.8.0
	github.com/prometheus/client_golang v1.10.0
	github.com/rubenv/sql-migrate v0.0.0-20200616145509-8d140a17f351
	github.com/shirou/gopsutil v3.21.8+incompatible
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.8.1
	go.vocdoni.io/dvote v1.0.4-0.20211111152020-2d4c168f9dc1
	go.vocdoni.io/proto v1.13.3-0.20211027093430-8170bfc7dc03
	google.golang.org/api v0.44.0
	nhooyr.io/websocket v1.8.7
)

// Newer versions of the fuse module removed support for MacOS.
// Unfortunately, its downstream users don't handle this properly,
// so our builds simply break for GOOS=darwin.
// Until either upstream or downstream solve this properly,
// force a downgrade to the commit right before support was dropped.
// It's also possible to use downstream's -tags=nofuse, but that's manual.
// TODO(mvdan): remove once we've untangled module dep loops.
replace bazil.org/fuse => bazil.org/fuse v0.0.0-20200407214033-5883e5a4b512
