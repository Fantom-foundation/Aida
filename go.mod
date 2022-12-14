module github.com/Fantom-foundation/Aida

go 1.19

require (
	github.com/Fantom-foundation/Carmen/go v0.0.0-00010101000000-000000000000
	github.com/Fantom-foundation/go-opera v1.1.1-rc.2
	github.com/Fantom-foundation/lachesis-base v0.0.0-20220904103856-4bb2a8448a22
	github.com/Fantom-foundation/substate-cli v0.0.0-20221109110938-3d145e50b9b9
	github.com/dsnet/compress v0.0.1
	github.com/ethereum/go-ethereum v1.10.25
	github.com/fatih/color v1.13.0
	github.com/olekukonko/tablewriter v0.0.5
	github.com/op/go-logging v0.0.0-20160315200505-970db520ece7
	github.com/syndtr/goleveldb v1.0.1-0.20210305035536-64b5b1c73954
	github.com/urfave/cli/v2 v2.19.2
	golang.org/x/exp v0.0.0-20221004215720-b9f4876ce741
	golang.org/x/text v0.3.7
)

require (
	github.com/DataDog/zstd v1.5.2 // indirect
	github.com/VictoriaMetrics/fastcache v1.12.0 // indirect
	github.com/btcsuite/btcd v0.20.1-beta // indirect
	github.com/cakturk/go-netstat v0.0.0-20200220111822-e5b49efee7a5 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/cockroachdb/errors v1.9.0 // indirect
	github.com/cockroachdb/logtags v0.0.0-20211118104740-dabe8e521a4f // indirect
	github.com/cockroachdb/pebble v0.0.0-20220524133354-f30672e7240b // indirect
	github.com/cockroachdb/redact v1.1.3 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.2 // indirect
	github.com/deckarep/golang-set v1.7.1 // indirect
	github.com/edsrzf/mmap-go v1.0.0 // indirect
	github.com/getsentry/sentry-go v0.14.0 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/hashicorp/golang-lru v0.5.5-0.20210104140557-80c98217689d // indirect
	github.com/holiman/bloomfilter/v2 v2.0.3 // indirect
	github.com/holiman/uint256 v1.2.0 // indirect
	github.com/klauspost/compress v1.15.11 // indirect
	github.com/kr/pretty v0.3.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.16 // indirect
	github.com/mattn/go-runewidth v0.0.14 // indirect
	github.com/mattn/go-sqlite3 v1.11.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/tsdb v0.10.0 // indirect
	github.com/rivo/uniseg v0.4.2 // indirect
	github.com/rogpeppe/go-internal v1.9.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/shirou/gopsutil v3.21.11+incompatible // indirect
	github.com/tklauser/go-sysconf v0.3.10 // indirect
	github.com/tklauser/numcpus v0.5.0 // indirect
	github.com/xrash/smetrics v0.0.0-20201216005158-039620a65673 // indirect
	github.com/yusufpapurcu/wmi v1.2.2 // indirect
	golang.org/x/crypto v0.0.0-20221005025214-4161e89ecf1b // indirect
	golang.org/x/sys v0.0.0-20220928140112-f11e5e49a4ec // indirect
	gopkg.in/natefinch/npipe.v2 v2.0.0-20160621034901-c1b8fa8bdcce // indirect
)

replace github.com/dvyukov/go-fuzz => github.com/guzenok/go-fuzz v0.0.0-20210103140116-f9104dfb626f

replace github.com/ethereum/go-ethereum => github.com/Fantom-foundation/go-ethereum-substate v1.1.1-0.20221214043804-b08bae8523ea

replace github.com/Fantom-foundation/go-opera => github.com/Fantom-foundation/go-opera-substate v1.0.1-0.20221124030310-9d06e1f98a18

// The Carmen project is integrated as a git-submodule since we need to run extra
// build steps when importing the project. This is handled in the Makefile.
replace github.com/Fantom-foundation/Carmen/go => ./carmen/go
