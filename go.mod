module github.com/Fantom-foundation/aida

go 1.19

require (
	github.com/Fantom-foundation/Carmen/go v0.0.0-20221012110807-9e1452a18755
	github.com/Fantom-foundation/go-opera v1.1.1-rc.2
	github.com/Fantom-foundation/substate-cli v0.0.0-20221014124903-8a7400c3b785
	github.com/ethereum/go-ethereum v1.10.8
	github.com/urfave/cli/v2 v2.3.0
)

require (
	github.com/Fantom-foundation/lachesis-base v0.0.0-20220103160934-6b4931c60582 // indirect
	github.com/StackExchange/wmi v0.0.0-20180116203802-5d049714c4a6 // indirect
	github.com/VictoriaMetrics/fastcache v1.6.0 // indirect
	github.com/btcsuite/btcd v0.20.1-beta // indirect
	github.com/cakturk/go-netstat v0.0.0-20200220111822-e5b49efee7a5 // indirect
	github.com/cespare/xxhash/v2 v2.1.1 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.0-20190314233015-f79a8a8ca69d // indirect
	github.com/deckarep/golang-set v1.7.1 // indirect
	github.com/edsrzf/mmap-go v1.0.0 // indirect
	github.com/go-ole/go-ole v1.2.1 // indirect
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/golang/snappy v0.0.3 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/hashicorp/golang-lru v0.5.5-0.20210104140557-80c98217689d // indirect
	github.com/holiman/bloomfilter/v2 v2.0.3 // indirect
	github.com/holiman/uint256 v1.2.0 // indirect
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/mattn/go-sqlite3 v1.11.0 // indirect
	github.com/olekukonko/tablewriter v0.0.5 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/tsdb v0.10.0 // indirect
	github.com/russross/blackfriday/v2 v2.0.1 // indirect
	github.com/shirou/gopsutil v3.21.4-0.20210419000835-c7a38de76ee5+incompatible // indirect
	github.com/shurcooL/sanitized_anchor_name v1.0.0 // indirect
	github.com/syndtr/goleveldb v1.0.1-0.20210305035536-64b5b1c73954 // indirect
	github.com/tklauser/go-sysconf v0.3.5 // indirect
	github.com/tklauser/numcpus v0.2.2 // indirect
	golang.org/x/crypto v0.0.0-20210322153248-0c34fe9e7dc2 // indirect
	golang.org/x/sys v0.0.0-20211019181941-9d821ace8654 // indirect
	gopkg.in/natefinch/npipe.v2 v2.0.0-20160621034901-c1b8fa8bdcce // indirect
)

replace github.com/dvyukov/go-fuzz => github.com/guzenok/go-fuzz v0.0.0-20210103140116-f9104dfb626f

replace github.com/ethereum/go-ethereum => github.com/Fantom-foundation/go-ethereum-substate v1.1.1-0.20221014111822-91701c749a84

replace github.com/Fantom-foundation/go-opera => github.com/Fantom-foundation/go-opera-substate v1.0.1-0.20221010085351-fcd736e326f0

// The Carmen project is integrated as a git-submodule since we need to run extra
// build steps when importing the project. This is handled in the Makefile.
replace github.com/Fantom-foundation/Carmen/go => ./carmen/go
