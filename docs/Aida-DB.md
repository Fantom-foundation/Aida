# Aida Database
## Overview
This tool merges various datasets to a single TestDB. TODO: extend overview

<img width="1334" alt="aida-db" src="https://github.com/Fantom-foundation/Aida/assets/40288710/659b26af-83e6-4b20-b34f-445ff5e1c886">

## Classification
TODO

## Requirements
You need a configured Go language environment to build the CLI application.
Please check the [Go documentation](https://go.dev)
for the details of installing the language compiler on your system.

TODO

## Build
To build the `aida-db` application, run `make aida-db`.

The build process downloads all the needed modules and libraries, you don't need to install these manually.
The `aida-db` executable application will be created in `/build` folder.

### Run
To use Aida DB, execute the compiled binary with the command and flags for the desired operation.

```shell
./build/aida-db command [command options] [arguments...]
```

| command  | description                                |
|----------|--------------------------------------------|
| autogen  | Autogen generates aida-db periodically     |
| clone    | Clone can create aida-db copy or subset    |
| generate | Generates aida-db from given events        |
| merge    | Merge source databases into aida-db        |
| stats    | Prints statistics about AidaDb             |

## Stats Command
Prints statistics about AidaDb.
The stats command requires one argument: `<blockNunLast>` - the last block of aida-db.

### Available subcommands

| subcommand | description                                                                                                  |
|------------|--------------------------------------------------------------------------------------------------------------|
| all        | List of all records in AidaDb                                                                                |
| del-acc    | Prints info about given deleted account in AidaDb. Requires arguments `<firstBlockNum>` and `<lastBlockNum>` |


### All subcommand
List of all records in AidaDb.

#### Options
```
all subcommand:
    --aida-db   set substate, updateset and deleted accounts directory
    --account   wanted account
    --log       level of the logging of the app action ("critical", "error", "warning", "notice", "info", "debug"; default: INFO)
```

### DelAcc subcommand
Prints info about given deleted account in AidaDb.

The DelAcc subcommand requires two arguments:
    `<blockNumFirst> <blockNumLast>`

`<blockNumFirst>` and `<blockNumLast>` are the first and
last block of the inclusive range of blocks.

#### Options
```
del-acc subcommand:
    --aida-db   set substate, updateset and deleted accounts directory
    --detailed  prints detailed info with how many records is in each prefix
    --log       level of the logging of the app action ("critical", "error", "warning", "notice", "info", "debug"; default: INFO)
```

## Clone Command
Creates clone of aida-db for desired block range

### Options
```
clone:
    --aida-db     set substate, updateset and deleted accounts directory
    --target-db   path to the target database
    --compact     compact target database
    --validate    enables validation
    --log         level of the logging of the app action ("critical", "error", "warning", "notice", "info", "debug"; default: INFO)
```

## AutoGen Command
Autogen generates aida-db patches and handles second opera for event generation. Generates event file, which is supplied into generate to create aida-db patch.

### Options
```
autogen:
    --aida-db   set substate, updateset and deleted accounts directory
    --chainid   choose chain id
    --db        path to the database
    --compact   compact target database
    --genesis   does not stop the program when results do not match.
    --db-tmp    sets the temporary directory where to place DB data; uses system default if empty
    --cache     cache limit
    --datadir   opera datadir directory
    --output    output path
    --log       level of the logging of the app action ("critical", "error", "warning", "notice", "info", "debug"; default: INFO)
```

## Generate Command
The db generate command requires events as an argument: `<events>`

`<events>` are fed into the opera database (either existing or genesis needs to be specified), processing them generates updated aida-db.

### Options
```
generate:
    --aida-db   set substate, updateset and deleted accounts directory
    --db        path to the database
    --genesis   does not stop the program when results do not match
    --keep-db   if set, statedb is not deleted after run
    --compact   compact target database
    --db-tmp    sets the temporary directory where to place DB data; uses system default if empty
    --chainid   choose chain id
    --cache     cache limit
    --log       level of the logging of the app action ("critical", "error", "warning", "notice", "info", "debug"; default: INFO)
```

## Merge Command
Creates target aida-db by merging source databases from arguments: `<db1> [<db2> <db3> ...]`

### Options
```
merge:
    --aida-db           set substate, updateset and deleted accounts directory
    --delete-source-d   delete source databases while merging into one database
    --compact           compact target database
    --log               level of the logging of the app action ("critical", "error", "warning", "notice", "info", "debug"; default: INFO)
```