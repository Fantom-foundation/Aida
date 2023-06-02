# Aida RunVM
## Overview
A tool for simulating a block processing with an experimental StateDB and/or virtual machine. TODO: extend overview

![run-vm](https://github.com/Fantom-foundation/Aida/assets/40288710/93ca0824-f2cf-4743-ab88-9f72c38f42e8)

## Classification
* System Under Test
    * StateDB
    * Virtual Machine
* Configurable Functional Tests
    * End state: compare StateDB to the generated world-state in last block
    * Transaction: check per transaction the modified portion of world-state (ie. substate)
    * VM: check receipt of transactions
* Non-Functional Tests
    * Memory-Consumption
    * Disk-Space
    * Runtime of operations
* Data set (offline)
    * Mainnet substate
    * Testnet substate

## Requirements
You need a configured Go language environment to build the CLI application.
Please check the [Go documentation](https://go.dev)
for the details of installing the language compiler on your system.

TODO

## Build
To build the `aida-runvm` application, run `make aida-runvm`.

The build process downloads all the needed modules and libraries, you don't need to install these manually.
The `aida-runvm` executable application will be created in `/build` folder.

## Run
```
./build/aida-runvm --aida-db path/to/aida-db --db-impl <geth/carmen/memory/flat> --vm-impl <geth, lfvm> <blockNumFirst> <blockNumLast>
```
This command performs block processing of the specified block range (inclusive). The initial StateDB is primed using substate from `--aida-db`. During block processing, a transaction calls a virtual machine which issues a series of StateDB operations to a selected storage system.

### Options
```
GLOBAL:
    --aida-db value            set substate, updateset and deleted accounts directory
    --deletion-db value        sets the directory containing deleted accounts database
    --update-db value          set update-set database directory
    --substate-db value        data directory for substate recorder/replayer
    --carmen-schema value      select the DB schema used by Carmen's current state DB (default: 0)
    --db-impl value            select state DB implementation (default: "geth")
    --db-variant value         select a state DB variant
    --db-src value             sets the directory contains source state DB data
    --db-tmp value             sets the temporary directory where to place state DB data; uses system default if empty
    --db-logging               enable logging of all DB operations (default: false)
    --archive                  set node type to archival mode. If set, the node keep all the EVM state history; otherwise the state history will be pruned. (default: false)
    --archive-variant value    set the archive implementation variant for the selected DB implementation, ignored if not running in archive mode
    --shadow-db value          use this flag when using an existing ShadowDb
    --db-shadow-impl value     select state DB implementation to shadow the prime DB implementation
    --db-shadow-variant value  select a state DB variant to shadow the prime DB implementation
    --vm-impl value            select VM implementation (default: "geth")
    --memory-breakdown         enables printing of memory usage breakdown (default: false)
    --memory-profile value     enables memory allocation profiling
    --profile                  enables profiling (default: false)
    --cpu-profile value        enables CPU profiling
    --random-seed value        set random seed (default: -1)
    --prime-threshold value    set number of accounts written to stateDB before applying pending state updates (default: 0)
    --prime-random             randomize order of accounts in StateDB priming (default: false)
    --skip-priming             if set, DB priming should be skipped; most useful with the 'memory' DB implementation (default: false)
    --update-buffer-size       buffer size for holding update set in MiB (default: 1<<64 - 1)
    --chainid value            ChainID for replayer (default: 250)
    --continue-on-failure      continue execute after validation failure detected (default: false)
    --quiet         disable progress report (default: false)
    --sync-period value        defines the number of blocks per sync-period (default: 300)
    --keep-db                  if set, statedb is not deleted after run (default: false)
    --max-transactions value   limit the maximum number of processed transactions, default: unlimited (default: -1)
    --validate-tx              enables transaction state validation (default: false)
    --validate-ws              enables end-state validation (default: false)
    --validate                 enables validation (default: false)
    --workers value            number of worker threads that execute in parallel (default: 4)
    --erigonbatchsize value    batch size for the execution stage (default: "512M")
    --log                      level of the logging of the app action ("critical", "error", "warning", "notice", "info", "debug"; default: INFO)
```