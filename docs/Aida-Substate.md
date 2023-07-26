# Aida Substate
## Overview
Aida-substate is a tools for simulate a blockchain virtual machine using substate -- a smallest fragment of world state to execute/validate a transaction. It offers an off-the-chain execution environment which allows execution of transactions in isolation.

![image](https://user-images.githubusercontent.com/6135904/235141883-10ec1587-e2ab-4db9-837c-d10309384855.png)

## Classification:
* System Under Test
  * Virtual Machine
* Functional Tests
  * Detect VM errors
  * Compare transaction receipt against recorded result
* Non-functional Tests
  * Performance
* Data set (offline)
  * Mainnet
  * Testnet

## Requirements
You need a configured Go language environment to build the CLI application.
Please check the [Go documentation](https://go.dev)
for the details of installing the language compiler on your system.

TODO

## Build
To build the `aida-substate` application, run `make aida-substate`.

The build process downloads all the needed modules and libraries, you don't need to install these manually.
The `aida-substate` executable application will be created in `/build` folder.

### Run
To use Aida Substate, execute the compiled binary with the command and flags for the desired operation.

```shell
./build/aida-substate command [command options] [arguments...]
```

| command               | description                                                                     |
|-----------------------|---------------------------------------------------------------------------------|
| replay                | Executes full state transitions and checks output consistency                   |
| gen-deleted-accounts  | Executes full state transitions and records suicided accounts                   |
| storage-size          | Returns the change in storage size by transactions in the specified block range |
| code                  | Write all contracts into a database                                             |
| code-size             | Reports code size and nonce of smart contracts in the specified block range     |
| dump                  | Returns content in substates in json format                                     |
| address-stats         | Computes usage statistics of addresses                                          |
| key-stats             | Computes usage statistics of accessed storage keys                              |
| location-stats        | Computes usage statistics of accessed storage locations                         |

## Replay Command
Executes full state transitions and check output consistency.
```
./build/aida-substate replay --aida-db /path/to/aida_db  --vm-impl <geth, lfvm> <blockNumFirst> <blockNumLast>
```
The substate-cli replay command requires two arguments:
`<blockNumFirst> <blockNumLast>`

`<blockNumFirst>` and `<blockNumLast>` are the first and
last block of the inclusive range of blocks to replay transactions.

### Options
```
replay:
    --chainid                ChainID for replayer (default: 250)
    --profiling-call        enable profiling for EVM call
    --micro-profiling       enable micro-profiling of EVM
    --basic-block-profiling enable profiling of basic block
    --profiling-db-name     set a database name for storing micro-profiling results (default: "./profiling.db")
    --buffer-size           set a buffer size for profiling channel (default: 100000)
    --vm-impl               select between `geth` and `lfvm` (default: "geth")
    --only-successful       only runs transactions that have been successful
    --cpu-profile           records a CPU profile for the replay to be inspected using `pprof`
    --db-impl               select state DB implementation (default: "geth")
    --workers               number of worker threads that execute in parallel (default: 4)
    --skip-transfer-txs     skip executing transactions that only transfer ETH
    --skip-call-txs         skip executing CALL transactions to accounts with contract bytecode
    --skip-create-txs       skip executing CREATE transactions
    --substate-db           data directory for substate recorder/replayer
    --log                   level of the logging of the app action ("critical", "error", "warning", "notice", "info", "debug"; default: INFO)
```

## GenDeletedAccounts Command
Executes full state transitions and record suicided accounts.
```
./build/aida-substate gen-deleted-accounts --substate-db /path/to/substate_db  --deletion-db /path/to/deletion_db <blockNumFirst> <blockNumLast>
```
The substate-cli gen-deleted-accounts command requires two arguments:
`<blockNumFirst> <blockNumLast>`

`<blockNumFirst>` and `<blockNumLast>` are the first and
last block of the inclusive range of blocks to replay transactions.

### Options
```
gen-deleted-accounts:
    --chainid       ChainID for replayer (default: 250)
    --workers       number of worker threads that execute in parallel (default: 4)
    --substate-db   data directory for substate recorder/replayer
    --deletion-db   sets the directory containing deleted accounts database
    --log           level of the logging of the app action ("critical", "error", "warning", "notice", "info", "debug"; default: INFO)
```

## GetStorageUpdateSize Command
Returns changes in storage size by transactions in the specified block range.
```
./build/aida-substate storage-size --substate-db /path/to/substate_db <blockNumFirst> <blockNumLast>
```
The substate-cli storage-size command requires two arguments:
`<blockNumFirst> <blockNumLast>`

`<blockNumFirst>` and `<blockNumLast>` are the first and
last block of the inclusive range of blocks to replay transactions.

Output log format: (block, timestamp, transaction, account, storage update size, storage size in input substate, storage size in output substate)

### Options
```
storage-size:
    --chainid       ChainID for replayer (default: 250)
    --workers       number of worker threads that execute in parallel (default: 4)
    --substate-db   data directory for substate recorder/replayer
    --log           level of the logging of the app action ("critical", "error", "warning", "notice", "info", "debug"; default: INFO)
```

## GetCode Command
Write all contracts into a database.
```
./build/aida-substate code --substate-db /path/to/substate_db --db /path/to/contracts_db <blockNumFirst> <blockNumLast>
```
The substate-cli code command requires two arguments:
`<blockNumFirst> <blockNumLast>`

`<blockNumFirst>` and `<blockNumLast>` are the first and
last block of the inclusive range of blocks to replay transactions.

The contracts of the block range are written into a levelDB database.

### Options
```
code:
    --chainid       ChainID for replayer (default: 250)
    --db            path to contracts database
    --workers       number of worker threads that execute in parallel (default: 4)
    --substate-db   data directory for substate recorder/replayer
```

## GetCodeSize Command
Reports code size and nonce of smart contracts in the specified block range.
```
./build/aida-substate code-size --substate-db /path/to/substate_db <blockNumFirst> <blockNumLast>
```
The substate-cli code-size command requires two arguments:
`<blockNumFirst> <blockNumLast>`

`<blockNumFirst>` and `<blockNumLast>` are the first and
last block of the inclusive range of blocks to replay transactions.

Output log format: (block, timestamp, transaction, account, code size, nonce, transaction type)

### Options
```
code-size:
    --chainid       ChainID for replayer (default: 250)
    --workers       number of worker threads that execute in parallel (default: 4)
    --substate-db   data directory for substate recorder/replayer
```

## SubstateDump Command
Returns content in substates in json format.
```
./build/aida-substate dump --substate-db /path/to/substate_db <blockNumFirst> <blockNumLast>
```
The substate-cli dump command requires two arguments:
`<blockNumFirst> <blockNumLast>`

`<blockNumFirst>` and `<blockNumLast>` are the first and
last block of the inclusive range of blocks to replay transactions.

### Options
```
dump:
    --workers       number of worker threads that execute in parallel (default: 4)
    --substate-db   data directory for substate recorder/replayer
```

## GetAddressStats Command
Computes usage statistics of addresses.
```
./build/aida-substate address-stats --substate-db /path/to/substate_db <blockNumFirst> <blockNumLast>
```
The substate-cli address-stats command requires two arguments:
`<blockNumFirst> <blockNumLast>`

`<blockNumFirst>` and `<blockNumLast>` are the first and
last block of the inclusive range of blocks to be analysed.

Statistics on the usage of addresses are printed to the console.

### Options
```
address-stats:
    --chainid       ChainID for replayer (default: 250)
    --workers       number of worker threads that execute in parallel (default: 4)
    --substate-db   data directory for substate recorder/replayer
```

## GetKeyStats Command
Computes usage statistics of accessed storage keys.
```
./build/aida-substate key-stats --substate-db /path/to/substate_db <blockNumFirst> <blockNumLast>
```
The substate-cli key-stats command requires two arguments:
`<blockNumFirst> <blockNumLast>`

`<blockNumFirst>` and `<blockNumLast>` are the first and
last block of the inclusive range of blocks to be analysed.

Statistics on the usage of accessed storage keys are printed to the console.

### Options
```
key-stats:
    --chainid       ChainID for replayer (default: 250)
    --workers       number of worker threads that execute in parallel (default: 4)
    --substate-db   data directory for substate recorder/replayer
    --log           level of the logging of the app action ("critical", "error", "warning", "notice", "info", "debug"; default: INFO)
```

## GetLocationStats Command
Computes usage statistics of accessed storage locations.
```
./build/aida-substate location-stats --substate-db /path/to/substate_db <blockNumFirst> <blockNumLast>
```
The substate-cli key-stats command requires two arguments:
`<blockNumFirst> <blockNumLast>`

`<blockNumFirst>` and `<blockNumLast>` are the first and
last block of the inclusive range of blocks to be analysed.

Statistics on the usage of accessed storage locations are printed to the console.

### Options
```
location-stats:
    --chainid       ChainID for replayer (default: 250)
    --workers       number of worker threads that execute in parallel (default: 4)
    --substate-db   data directory for substate recorder/replayer
    --log           level of the logging of the app action ("critical", "error", "warning", "notice", "info", "debug"; default: INFO)
```

## Db Command
A set of commands on substate DB

### Available subcommands

| subcommand | description                                                  |
|------------|--------------------------------------------------------------|
| clone      | Clone a given block range from src db to dst db              |
| compact    | Compat LevelDB - discarding deleted and overwritten versions |

### Clone subcommand
The substate-cli db clone command requires four arguments:
    `<srcPath> <dstPath> <blockNumFirst> <blockNumLast>`
`<srcPath>` is the original substate database to read the information.
`<dstPath>` is the target substate database to write the information.
`<blockNumFirst>` and `<blockNumLast>` are the first and
last block of the inclusive range of blocks to clone.

If dstPath doesn't exist, a new substate database is created.
If dstPath exits, substates from src db are merged into dst db. Any overlapping blocks are overwrittenby the new value from src db.

#### Options
```
clone subcommand:
    --workers       number of worker threads that execute in parallel (default: 4)
```

### Compact subcommand
The substate-cli db compact command requires one argument:
	`<dbPath>`

`<dbPath>` is the target LevelDB instance to compact.

#### Options
```
compact subcommand:
    --log           level of the logging of the app action ("critical", "error", "warning", "notice", "info", "debug"; default: INFO)
```
