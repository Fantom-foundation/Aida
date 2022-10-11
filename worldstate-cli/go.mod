module github.com/Fantom-foundation/Aida-Testing/worldstate-cli

go 1.19

require github.com/op/go-logging v0.0.0-20160315200505-970db520ece7

require (
	github.com/DataDog/zstd v1.4.5 // indirect
	github.com/StackExchange/wmi v0.0.0-20180116203802-5d049714c4a6 // indirect
	github.com/VictoriaMetrics/fastcache v1.6.0 // indirect
	github.com/btcsuite/btcd v0.20.1-beta // indirect
	github.com/cespare/xxhash/v2 v2.1.1 // indirect
	github.com/cockroachdb/errors v1.8.1 // indirect
	github.com/cockroachdb/logtags v0.0.0-20190617123548-eb05cc24525f // indirect
	github.com/cockroachdb/pebble v0.0.0-20220524133354-f30672e7240b // indirect
	github.com/cockroachdb/redact v1.0.8 // indirect
	github.com/cockroachdb/sentry-go v0.6.1-cockroachdb.2 // indirect
	github.com/go-ole/go-ole v1.2.1 // indirect
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/gogo/protobuf v1.3.1 // indirect
	github.com/golang/snappy v0.0.3 // indirect
	github.com/hashicorp/golang-lru v0.5.5-0.20210104140557-80c98217689d // indirect
	github.com/holiman/bloomfilter/v2 v2.0.3 // indirect
	github.com/klauspost/compress v1.11.7 // indirect
	github.com/kr/pretty v0.1.0 // indirect
	github.com/kr/text v0.1.0 // indirect
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/olekukonko/tablewriter v0.0.5 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/tsdb v0.7.1 // indirect
	github.com/shirou/gopsutil v3.21.4-0.20210419000835-c7a38de76ee5+incompatible // indirect
	github.com/tklauser/go-sysconf v0.3.5 // indirect
	github.com/tklauser/numcpus v0.2.2 // indirect
	golang.org/x/crypto v0.0.0-20210322153248-0c34fe9e7dc2 // indirect
	golang.org/x/exp v0.0.0-20200513190911-00229845015e // indirect
	golang.org/x/sys v0.0.0-20210909193231-528a39cd75f3 // indirect
	golang.org/x/text v0.3.7 // indirect
	gopkg.in/urfave/cli.v1 v1.20.0 // indirect
)

require (
	github.com/Fantom-foundation/lachesis-base v0.0.0-20220904103856-4bb2a8448a22
	github.com/cpuguy83/go-md2man/v2 v2.0.2 // indirect
	github.com/ethereum/go-ethereum v1.9.22
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/syndtr/goleveldb v1.0.1-0.20210305035536-64b5b1c73954
	github.com/urfave/cli/v2 v2.3.0
)

//replace github.com/ethereum/go-ethereum => github.com/Fantom-foundation/go-ethereum v1.10.8-ftm-rc7
replace github.com/ethereum/go-ethereum => github.com/Fantom-foundation/go-ethereum-substate v1.1.0
