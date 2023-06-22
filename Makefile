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

.PHONY: all clean help test tosca

all: aida-api-replay aida-worldstate aida-updateset aida-db aida-trace aida-runarchive aida-runvm aida-stochastic aida-substate

aida-api-replay:
	@cd carmen/go/lib ; \
	./build_libcarmen.sh ; \
	cd ../../.. ; \
	GOPROXY=$(GOPROXY) \
	GOPRIVATE=github.com/Fantom-foundation/Carmen \
	go build -ldflags "-s -w -X 'github.com/Fantom-foundation/Aida/utils.GitCommit=$(BUILD_COMMIT)'" \
	-o $(GO_BIN)/aida-api-replay \
	./cmd/api-replay-cli

carmen/go/lib/libcarmen.so:
	@cd carmen/go/lib ; \
	./build_libcarmen.sh ;

tosca:
	@cd ./tosca ; \
	make

aida-worldstate:
	@go build \
		-ldflags="-X 'github.com/Fantom-foundation/Aida/cmd/worldstate-cli/version.Version=$(APP_VERSION)' -X 'github.com/Fantom-foundation/Aida/cmd/worldstate-cli/version.Time=$(BUILD_DATE)' -X 'github.com/Fantom-foundation/Aida/cmd/worldstate-cli/version.Compiler=$(BUILD_COMPILER)' -X 'github.com/Fantom-foundation/Aida/cmd/worldstate-cli/version.Commit=$(BUILD_COMMIT)' -X 'github.com/Fantom-foundation/Aida/cmd/worldstate-cli/version.CommitTime=$(BUILD_COMMIT_TIME)'" \
		-o $(GO_BIN)/aida-worldstate \
		-v \
		./cmd/worldstate-cli

aida-stochastic: carmen/go/lib/libcarmen.so tosca
	GOPROXY=$(GOPROXY) \
	GOPRIVATE=github.com/Fantom-foundation/Carmen,github.com/Fantom-foundation/go-opera-fvm \
	go build -ldflags "-s -w -X 'github.com/Fantom-foundation/Aida/utils.GitCommit=$(BUILD_COMMIT)'" \
	-o $(GO_BIN)/aida-stochastic \
	./cmd/stochastic-cli

aida-trace: carmen/go/lib/libcarmen.so tosca
	GOPROXY=$(GOPROXY) \
	GOPRIVATE=github.com/Fantom-foundation/Carmen \
	go build -ldflags "-s -w -X 'github.com/Fantom-foundation/Aida/utils.GitCommit=$(BUILD_COMMIT)'" \
	-o $(GO_BIN)/aida-trace \
	./cmd/trace-cli

aida-runarchive: carmen/go/lib/libcarmen.so tosca
	GOPROXY=$(GOPROXY) \
	GOPRIVATE=github.com/Fantom-foundation/Carmen \
	go build -ldflags "-s -w -X 'github.com/Fantom-foundation/Aida/utils.GitCommit=$(BUILD_COMMIT)'" \
	-o $(GO_BIN)/aida-runarchive \
	./cmd/runarchive-cli

aida-runvm: carmen/go/lib/libcarmen.so tosca
	GOPROXY=$(GOPROXY) \
	GOPRIVATE=github.com/Fantom-foundation/Carmen \
	CGO_CFLAGS="-g -O2  -DMDBX_FORCE_ASSERTIONS=1 -Wno-error=strict-prototypes" \
	go build -ldflags "-s -w -X 'github.com/Fantom-foundation/Aida/utils.GitCommit=$(BUILD_COMMIT)'" \
	-o $(GO_BIN)/aida-runvm \
	./cmd/runvm-cli

aida-substate: carmen/go/lib/libcarmen.so tosca
	GOPROXY=$(GOPROXY) \
	GOPRIVATE=github.com/Fantom-foundation/Carmen,github.com/Fantom-foundation/go-opera-fvm \
	go build -ldflags "-s -w -X 'github.com/Fantom-foundation/Aida/utils.GitCommit=$(BUILD_COMMIT)'" \
	-o $(GO_BIN)/aida-substate \
	./cmd/substate-cli

aida-updateset: carmen/go/lib/libcarmen.so tosca
	GOPROXY=$(GOPROXY) \
	go build -ldflags "-s -w" \
	-o $(GO_BIN)/aida-updateset \
	./cmd/updateset-cli

aida-db: carmen/go/lib/libcarmen.so tosca
	GOPROXY=$(GOPROXY) \
	go build -ldflags "-s -w" \
	-o $(GO_BIN)/aida-db \
	./cmd/db-cli

test:
	@go test ./...

clean:
	cd carmen/go ; \
	rm -f lib/libcarmen.so ; \
	cd ../cpp ; \
	bazel clean ; \
	cd ../../tosca ; \
	make clean ; \
	cd .. ; \
	rm -fr ./build/*

help: Makefile
	@echo "Choose a make command in "$(PROJECT)":"
	@echo
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
	@echo
