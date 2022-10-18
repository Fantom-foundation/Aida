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

.PHONY: all clean help test

all: gen-world-state trace

gen-world-state:
	@go build \
		-ldflags="-X 'github.com/Fantom-foundation/Aida/cmd/gen-world-state/version.Version=$(APP_VERSION)' -X 'github.com/Fantom-foundation/Aida/cmd/gen-world-state/version.Time=$(BUILD_DATE)' -X 'github.com/Fantom-foundation/Aida/cmd/gen-world-state/version.Compiler=$(BUILD_COMPILER)' -X 'github.com/Fantom-foundation/Aida/cmd/gen-world-state/version.Commit=$(BUILD_COMMIT)' -X 'github.com/Fantom-foundation/Aida/cmd/gen-world-state/version.CommitTime=$(BUILD_COMMIT_TIME)'" \
		-o $(GO_BIN)/gen-world-state \
		-v \
		./cmd/gen-world-state

trace:
	cd carmen/go ; \
	go generate ./... ; \
	cd ../.. ; \
	GOPROXY=$(GOPROXY) \
	GOPRIVATE=github.com/Fantom-foundation/Carmen \
	@go build -ldflags "-s -w" \
       	-o $(GO_BIN)/trace \
	./cmd/

test:
	@go test ./...

clean:
	cd carmen/go ; \
	rm -f lib/libstate.so ; \
	cd ../cpp ; \
	bazel clean ; \
	cd ../.. ; \
	rm -fr ./build/*

help: Makefile
	@echo "Choose a make command in "$(PROJECT)":"
	@echo
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
	@echo