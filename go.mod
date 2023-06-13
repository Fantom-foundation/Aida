module github.com/Fantom-foundation/rc-testing

go 1.19

require (
	github.com/Fantom-foundation/Substate v0.0.0-20230224090651-4c8c024214f4
	github.com/Fantom-foundation/go-opera v1.1.1-rc.2
	github.com/Fantom-foundation/go-opera-base v0.0.0-00010101000000-000000000000
	github.com/Fantom-foundation/lachesis-base v0.0.0-20221208123620-82a6d15f995c
	github.com/ethereum/go-ethereum v1.10.25
	github.com/google/martian v2.1.0+incompatible
	github.com/op/go-logging v0.0.0-20160315200505-970db520ece7
	github.com/syndtr/goleveldb v1.0.1-0.20220614013038-64ee5596c38a
	github.com/urfave/cli/v2 v2.24.4
)

require (
	github.com/VictoriaMetrics/fastcache v1.12.0 // indirect
	github.com/btcsuite/btcd v0.22.0-beta // indirect
	github.com/cakturk/go-netstat v0.0.0-20200220111822-e5b49efee7a5 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.2 // indirect
	github.com/deckarep/golang-set v1.8.0 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/hashicorp/golang-lru v0.5.5-0.20210104140557-80c98217689d // indirect
	github.com/holiman/bloomfilter/v2 v2.0.3 // indirect
	github.com/holiman/uint256 v1.2.1 // indirect
	github.com/mattn/go-runewidth v0.0.14 // indirect
	github.com/mattn/go-sqlite3 v1.11.0 // indirect
	github.com/olekukonko/tablewriter v0.0.5 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/tsdb v0.10.0 // indirect
	github.com/rivo/uniseg v0.4.2 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/shirou/gopsutil v3.21.11+incompatible // indirect
	github.com/tklauser/go-sysconf v0.3.11 // indirect
	github.com/tklauser/numcpus v0.6.0 // indirect
	github.com/xrash/smetrics v0.0.0-20201216005158-039620a65673 // indirect
	github.com/yusufpapurcu/wmi v1.2.2 // indirect
	golang.org/x/crypto v0.1.0 // indirect
	golang.org/x/net v0.10.0 // indirect
	golang.org/x/sys v0.8.0 // indirect
	gopkg.in/natefinch/npipe.v2 v2.0.0-20160621034901-c1b8fa8bdcce // indirect
)

//replace github.com/ethereum/go-ethereum => github.com/Fantom-foundation/go-ethereum v1.10.8-ftm-rc11
replace github.com/ethereum/go-ethereum => github.com/cyberbono3/go-ethereum-substate v1.1.1-0.20230310040339-50aac04770b2

replace github.com/Fantom-foundation/go-opera-base => ./go-opera

replace github.com/Fantom-foundation/Substate => github.com/Fantom-foundation/Substate v0.0.0-20230512151613-f039b23bf3f7
