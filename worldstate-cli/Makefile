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

all: worldstate-cli

worldstate-cli:
	@go build \
		-ldflags="-X 'github.com/Fantom-foundation/Aida-Testing/worldstate-cli/cmd/build.Version=$(APP_VERSION)' -X 'github.com/Fantom-foundation/Aida-Testing/worldstate-cli/cmd/build.Time=$(BUILD_DATE)' -X 'github.com/Fantom-foundation/Aida-Testing/worldstate-cli/cmd/build.Compiler=$(BUILD_COMPILER)' -X 'github.com/Fantom-foundation/Aida-Testing/worldstate-cli/cmd/build.Commit=$(BUILD_COMMIT)' -X 'github.com/Fantom-foundation/Aida-Testing/worldstate-cli/cmd/build.CommitTime=$(BUILD_COMMIT_TIME)'" \
		-o $(GO_BIN)/worldstate-cli \
		./cmd

test:
	@go test ./...

help: Makefile
	@echo "Choose a make command in "$(PROJECT)":"
	@echo
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
	@echo

.PHONY: all
