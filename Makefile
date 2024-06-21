# Copyright 2024 Fantom Foundation
# This file is part of Aida Testing Infrastructure for Sonic.
#
# Aida is free software: you can redistribute it and/or modify
# it under the terms of the GNU Lesser General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# Aida is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
# GNU Lesser General Public License for more details.
#
# You should have received a copy of the GNU Lesser General Public License
# along with Aida. If not, see <http://www.gnu.org/licenses/>.

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

all: aida-rpc aida-sdb aida-vm-adb aida-vm-sdb aida-stochastic-sdb aida-vm aida-profile util-updateset util-db


carmen:
	@cd ./carmen ; \
	make -j

tosca:
	@cd ./tosca ; \
	make -j

aida-rpc: carmen tosca
	GOPROXY=$(GOPROXY) \
	go build -ldflags "-s -w -X 'github.com/Fantom-foundation/Aida/utils.GitCommit=$(BUILD_COMMIT)'" \
	-o $(GO_BIN)/aida-rpc \
	./cmd/aida-rpc

aida-stochastic-sdb: carmen tosca
	GOPROXY=$(GOPROXY) \
	go build -ldflags "-s -w -X 'github.com/Fantom-foundation/Aida/utils.GitCommit=$(BUILD_COMMIT)'" \
	-o $(GO_BIN)/aida-stochastic-sdb \
	./cmd/aida-stochastic-sdb

aida-vm-adb: carmen tosca
	GOPROXY=$(GOPROXY) \
	go build -ldflags "-s -w -X 'github.com/Fantom-foundation/Aida/utils.GitCommit=$(BUILD_COMMIT)'" \
	-o $(GO_BIN)/aida-vm-adb \
	./cmd/aida-vm-adb

aida-vm-sdb: carmen tosca
	GOPROXY=$(GOPROXY) \
	CGO_CFLAGS="-g -O2  -DMDBX_FORCE_ASSERTIONS=1 -Wno-error=strict-prototypes" \
	go build -ldflags "-s -w -X 'github.com/Fantom-foundation/Aida/utils.GitCommit=$(BUILD_COMMIT)'" \
	-o $(GO_BIN)/aida-vm-sdb \
	./cmd/aida-vm-sdb

aida-vm: carmen tosca
	GOPROXY=$(GOPROXY) \
	go build -ldflags "-s -w -X 'github.com/Fantom-foundation/Aida/utils.GitCommit=$(BUILD_COMMIT)'" \
	-o $(GO_BIN)/aida-vm \
	./cmd/aida-vm

aida-sdb: carmen tosca
	GOPROXY=$(GOPROXY) \
	go build -ldflags "-s -w -X 'github.com/Fantom-foundation/Aida/utils.GitCommit=$(BUILD_COMMIT)'" \
	-o $(GO_BIN)/aida-sdb \
	./cmd/aida-sdb

aida-profile: carmen tosca
	GOPROXY=$(GOPROXY) \
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
	CGO_CFLAGS="-g -O2  -DMDBX_FORCE_ASSERTIONS=1 -Wno-error=strict-prototypes" \
	go build -ldflags "-s -w" \
	-o $(GO_BIN)/util-db \
	./cmd/util-db

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
