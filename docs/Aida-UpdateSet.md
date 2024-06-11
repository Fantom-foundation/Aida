# Aida UpdateSet
## Overview
Generate UpdateSet from substate. UpdateSet is a merge of substate range. Used as precomputed sets of merged substates for quicker advancing world state. TODO: extend overview

<img width="643" alt="updateset" src="https://github.com/Fantom-foundation/Aida/assets/40288710/efda76e8-0d6e-4b34-9fac-9431caae3f6f">

## Classification
TODO

## Requirements
You need a configured Go language environment to build the CLI application.
Please check the [Go documentation](https://go.dev)
for the details of installing the language compiler on your system.

TODO

## Build
To build the `aida-updateset` application, run `make aida-updateset`.

The build process downloads all the needed modules and libraries, you don't need to install these manually.
The `aida-updateset` executable application will be created in `/build` folder.

### Run
To use Aida UpdateSet, execute the compiled binary with the command and flags for the desired operation.

```shell
./build/aida-updateset command [command options] [arguments...]
```

| command      | description                                             |
|--------------|---------------------------------------------------------|
| generate     | Generate update-set from substate                       |
| stats        | Print number of accounts and storage keys in update-set |

## GenUpdateSet Command
Generate the UpdateSet.

```
./build/aida-updateset generate --world-state path/to/world-state --update-db path/to/update-db `<blockNumLast>` `<interval>`
```
generates piecewise update-sets (merges of output substates) at every `<interval>` blocks starting from block 4564026 to block `<blockNumLast>` and stores them in updateDB. WorldState of block 4564025 from the world state is reocrded as the first update-set if --world-state is provided. The subsequence update-sets happen every `<interval>` blocks afterwards.

### Options
```
GenUpdateSet:
    --chainid   choose chain id
    --deletion-db           sets the directory containing deleted accounts database
    --update-db             set update-set database directory
    --update-buffer-size    buffer size for holding update set in MiB (default: 1<<64 - 1)
    --validate              enables validation (default: false)
    --world-state           world state snapshot database path
    --workers               number of worker threads that execute in parallel (default: 4)
    --substate-db           data directory for substate recorder/replayer
    --log                   level of the logging of the app action ("critical", "error", "warning", "notice", "info", "debug"; default: INFO)
```

## UpdateSetStats Command
Print number of accounts and storage keys in update-set.

```
./build/aida-updateset stats --aida-db /path/to/aida-db --update-db path/to/update-db `<blockNumLast>`
```
The stats command requires one arguments: `<blockNumLast>` - the last block of update-set.

### Options
```
UpdateSetStats:
    --update-db             set update-set database directory
    --aida-db               set substate, updateset and deleted accounts directory
```


