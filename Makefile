# --------------------------------------------------------------------------
# Makefile for the Fantom Aida World State Manager CLI
#
# v1.0 (2022/09/22) - Initial version
#
# (c) Fantom Foundation, 2022
# --------------------------------------------------------------------------

# what are we building
PROJECT := $(shell basename "$(PWD)")
GO_BIN := $(CURDIR)/build

# compile time variables will be injected into the app
APP_VERSION := 1.0
BUILD_DATE := $(shell date "+%a, %d %b %Y %T")
BUILD_COMPILER := $(shell go version)
BUILD_COMMIT := $(shell git show --format="%H" --no-patch)
BUILD_COMMIT_TIME := $(shell git show --format="%cD" --no-patch)
GOPROXY ?= "https://proxy.golang.org,direct"

.PHONY: all clean help test carmen tosca

all: aida-rpc aida-sdb aida-vm-adb aida-vm-sdb aida-stochastic-sdb aida-vm aida-profile util-worldstate util-updateset util-db


carmen:
	@cd ./carmen ; \
	make -j

tosca:
	@cd ./tosca ; \
	make -j

aida-rpc: carmen tosca
	GOPROXY=$(GOPROXY) \
	GOPRIVATE=github.com/Fantom-foundation/Carmen \
	go build -ldflags "-s -w -X 'github.com/Fantom-foundation/Aida/utils.GitCommit=$(BUILD_COMMIT)'" \
	-o $(GO_BIN)/aida-rpc \
	./cmd/aida-rpc

aida-stochastic-sdb: carmen tosca
	GOPROXY=$(GOPROXY) \
	GOPRIVATE=github.com/Fantom-foundation/Carmen \
	go build -ldflags "-s -w -X 'github.com/Fantom-foundation/Aida/utils.GitCommit=$(BUILD_COMMIT)'" \
	-o $(GO_BIN)/aida-stochastic-sdb \
	./cmd/aida-stochastic-sdb

aida-vm-adb: carmen tosca
	GOPROXY=$(GOPROXY) \
	GOPRIVATE=github.com/Fantom-foundation/Carmen \
	go build -ldflags "-s -w -X 'github.com/Fantom-foundation/Aida/utils.GitCommit=$(BUILD_COMMIT)'" \
	-o $(GO_BIN)/aida-vm-adb \
	./cmd/aida-vm-adb

aida-vm-sdb: carmen tosca
	GOPROXY=$(GOPROXY) \
	GOPRIVATE=github.com/Fantom-foundation/Carmen \
	CGO_CFLAGS="-g -O2  -DMDBX_FORCE_ASSERTIONS=1 -Wno-error=strict-prototypes" \
	go build -ldflags "-s -w -X 'github.com/Fantom-foundation/Aida/utils.GitCommit=$(BUILD_COMMIT)'" \
	-o $(GO_BIN)/aida-vm-sdb \
	./cmd/aida-vm-sdb

aida-vm: carmen tosca
	GOPROXY=$(GOPROXY) \
	GOPRIVATE=github.com/Fantom-foundation/Carmen \
	go build -ldflags "-s -w -X 'github.com/Fantom-foundation/Aida/utils.GitCommit=$(BUILD_COMMIT)'" \
	-o $(GO_BIN)/aida-vm \
	./cmd/aida-vm

aida-sdb: carmen tosca
	GOPROXY=$(GOPROXY) \
	GOPRIVATE=github.com/Fantom-foundation/Carmen \
	go build -ldflags "-s -w -X 'github.com/Fantom-foundation/Aida/utils.GitCommit=$(BUILD_COMMIT)'" \
	-o $(GO_BIN)/aida-sdb \
	./cmd/aida-sdb

aida-profile: carmen tosca
	GOPROXY=$(GOPROXY) \
	GOPRIVATE=github.com/Fantom-foundation/Carmen \
	CGO_CFLAGS="-g -O2  -DMDBX_FORCE_ASSERTIONS=1 -Wno-error=strict-prototypes" \
	go build -ldflags "-s -w -X 'github.com/Fantom-foundation/Aida/utils.GitCommit=$(BUILD_COMMIT)'" \
	-o $(GO_BIN)/aida-profile \
	./cmd/aida-profile

util-updateset: carmen tosca
	GOPROXY=$(GOPROXY) \
	go build -ldflags "-s -w" \
	-o $(GO_BIN)/util-updateset \
	./cmd/util-updateset

util-db: carmen tosca
	GOPROXY=$(GOPROXY) \
	go build -ldflags "-s -w" \
	-o $(GO_BIN)/util-db \
	./cmd/util-db

util-worldstate: carmen tosca
	@go build \
		-ldflags="-X 'github.com/Fantom-foundation/Aida/cmd/util-worldstate/version.Version=$(APP_VERSION)' -X 'github.com/Fantom-foundation/Aida/cmd/util-worldstate/version.Time=$(BUILD_DATE)' -X 'github.com/Fantom-foundation/Aida/cmd/util-worldstate/version.Compiler=$(BUILD_COMPILER)' -X 'github.com/Fantom-foundation/Aida/cmd/util-worldstate/version.Commit=$(BUILD_COMMIT)' -X 'github.com/Fantom-foundation/Aida/cmd/util-worldstate/version.CommitTime=$(BUILD_COMMIT_TIME)'" \
		-o $(GO_BIN)/util-worldstate \
		-v \
		./cmd/util-worldstate

test: carmen tosca
	@go test ./...

clean:
	cd ./carmen ; \
	make clean ; \
	cd ../tosca ; \
	make clean ; \
	cd .. ; \
	rm -fr ./build/*

help: Makefile
	@echo "Choose a make command in "$(PROJECT)":"
	@echo
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
	@echo
