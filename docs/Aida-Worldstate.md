# Aida Worldstate
## Overview
The purpose of the Aida Worldstate is to build and transfer a minimal world state snapshot
between synced Fantom Opera node database and a target client sub-system.
The Aida Worldstate uses an intermediate database with condensed EVM state to support
alternative client subsystems execution, profiling, and testing. It should not be seen
as a target client database.


## Requirements
You need a configured Go language environment to build the CLI application.
Please check the [Go documentation](https://go.dev)
for the details of installing the language compiler on your system.

TODO

## Build
To build the `aida-worldstate` application, run `make aida-worldstate`.

The build process downloads all the needed modules and libraries, you don't need to install these manually.
The `aida-worldstate` executable application will be created in `/build` folder.

### Run
To use Aida Worldstate, execute the compiled binary with the command and flags for the desired operation.

```shell
./build/aida-worldstate [global options] command [command options] [arguments...]
```

| command      | description                                                        |
|--------------|--------------------------------------------------------------------|
| account, a   | Displays information of an individual account in exported state DB |
| dump, d      | Extracts world state MPT at a given state root into an external DB |
| root, r      | Retrieve root hash of given block number                           |
| evolve, e    | Evolves world state snapshot database into selected target block   |
| compare, cmp | Compare whether states of two databases are identical              |
| clone, c     | Creates a clone of the world state dump database                   |
| info, i      | Retrieves basic info about snapshot database                       |
| version, v   | Provides information about the application version and build       |

### Options
```
GLOBAL:
    --world-state           world state snapshot database path
    --log                   level of the logging of the app action ("critical", "error", "warning", "notice", "info", "debug"; default: INFO)
```

## CmdAccount Command
Provides information and management function for individual accounts in state dump database.

```shell
./build/aida-worldstate --world-state path/to/world_state account subcommand [subcommand options]
```

### Available subcommands

| subcommand   | description                                                                                                           |
|--------------|-----------------------------------------------------------------------------------------------------------------------|
| info, i      | Provides detailed information about the target account                                                                |
| collect, c   | Collects known account addresses from substate database. Requires arguments `<firstBlockNum>` and `<lastBlockNum>`    |
| import, csv  | Imports account addresses or storages for hash mapping from a CSV file. Requires argument `<csvFilePath>` - for stdin |
| unknown, u   | Lists unknown account addresses or storages from the world state database                                             |

### AccountInfo subcommand
Command provides detailed information about the account specified as an argument.
The worldstate-cli account info subcommand requires one argument:
	`<address|hash>`

`<address|hash>` is the target account which information should be provided.

#### Options
```
info subcommand:
    --include-storage   display full storage content
```

### AccountCollect subcommand
Command updates internal map of account hashes for the known accounts in substate database.
The worldstate-cli account collect subcommand requires two arguments:
`<blockNumFirst> <blockNumLast>`

`<blockNumFirst>` and `<blockNumLast>` are the first and
last block of the inclusive range of blocks to be analysed.

#### Options
```
collect subcommand:
    --workers               number of worker threads that execute in parallel (default: 4)
    --substate-db           data directory for substate recorder/replayer
```

### AccountImport subcommand
Command imports account hash to account address mapping from a CSV file.
The worldstate-cli account import subcommand requires one argument:
`<csvFilePath>`

`<csvFilePath>` is the path to the csv file containing account data to be imported

#### Options
```
import subcommand:
    --quiet                 disable progress report (default: false)
```

### Unknown subcommand
Command scans for addresses in the world state database and shows those not available in the address map.

#### Available subcommands
| subcommand  | description                                                      |
|-------------|------------------------------------------------------------------|
| storage     | Lists unknown account storages from the world state database     |
| address     | Lists unknown account addresses from the world state database    |

#### Options
```
storage subcommand:
    --quiet                 disable progress report (default: false)

address subcommand:
    --quiet                 disable progress report (default: false)
```

## Clone Command
Creates a clone of the world state dump database.
```
./build/aida-worldstate --world-state path/to/world_state clone --target-db path/to/target_db
```

### Options
```
clone:
    --target-db    target database path
```

## CompareState Command
Compares given snapshot database against target snapshot database.
```
./build/aida-worldstate --world-state path/to/world_state compare --target-db path/to/target_db
```

### Options
```
compare:
    --target-db    target database path
```

## DumpState Command
Extracts world state MPT trie at given root from input database into state snapshot output database.
```
./build/aida-worldstate --world-state path/to/world_state dump
```

The dump creates a snapshot of all accounts state (including contracts) exporting:
* Balance
* Nonce
* Code (separate storage slot is used to store code data)
* Contract Storage

### Options
```
dump:
    --db                    path to the database
    --db-variant            select a state DB variant
    --source-table          name of the database table to be used
    --root                  state trie root hash to be analysed
    --target-block          target block ID (default: 0)
    --workers               number of worker threads that execute in parallel (default: 4)
    --log                   level of the logging of the app action ("critical", "error", "warning", "notice", "info", "debug"; default: INFO)
```

## EvolveState Command
Evolves world state snapshot database into selected target block.
```
./build/aida-worldstate --world-state path/to/world_state evolve --substate-db path/to/substate_db
```

### Options
```
evolve:
    --target-block          target block ID (default: 0)
    --validate              enables validation
    --workers               number of worker threads that execute in parallel (default: 4)
    --substate-db           data directory for substate recorder/replayer
```

## Root Command
Searches opera database for root hash for supplied block number.
```
./build/aida-worldstate --world-state path/to/world_state root
```

### Options
```
root:
    --db-variant            select a state DB variant
    --source-table          name of the database table to be used
    --root                  state trie root hash to be analysed
    --target-block          target block ID (default: 0)
```

## Info Command
Looks up current block number of database. Retrieves basic info about snapshot database.
```
./build/aida-worldstate --world-state path/to/world_state info
```

## Version Command
Provides information about the application version and build details.
```
./build/aida-worldstate --world-state path/to/world_state version
```
