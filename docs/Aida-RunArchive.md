# Aida RunArchive
## Overview
A tool for simulating opera block processing on real world historical data (mainnet/testnet). TODO: extend overview

## Classification
* System Under Test
    * ArchiveDB
    * Virtual machine
* Functional Tests
    * Substate: check per transaction the modified portion of world-state (ie. substate)
* Data set (offline)
    * Mainnet
    * Testnet

## Requirements
You need a configured Go language environment to build the CLI application.
Please check the [Go documentation](https://go.dev)
for the details of installing the language compiler on your system.

TODO

## Build
To build the `aida-runarchive` application, run `make aida-runarchive`.

The build process downloads all the needed modules and libraries, you don't need to install these manually.
The `aida-runarchive` executable application will be created in `/build` folder.

## Run
```
./build/aida-runarchive --substate-db path/to/substatedb --db-src path/to/statedb/with/archive <blockNumFirst> <blockNumLast>
```
executes transactions from block `<blockNumFirst>` to `<blockNumLast>` using the historic data in the provided archive. Each transaction loads the historic state of its block and executes the transaction on it in read-only mode.

![runarchive diagram](https://github.com/Fantom-foundation/Aida/assets/40288710/05032eb5-8d15-4251-ac80-7735fb20c0b2)


### Options
```
GLOBAL:
    --cpu-profile   records a CPU profile for the replay to be inspected using `pprof`
    --chainid       sets the chain-id (useful if recording from testnet) (default: 250 (mainnet))
    --aida-db       set substate, updateset and deleted accounts directory
    --db-src        sets the directory contains source state DB data
    --validate-tx   validate the effects of each transaction
    --shadow-db use this flag when using an existing ShadowDb
    --vm-impl       select between `geth` and `lfvm` (default: "geth")
    --workers       number of worker threads that execute in parallel (default: 4)
    --substate-db   sets directory containing substate database (default: `./substate.fantom`)
    --log           level of the logging of the app action ("critical", "error", "warning", "notice", "info", "debug"; default: INFO)
```