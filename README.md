# Fantom Aida
Aida is a storage tracing tool that collects real-world storage traces 
from the block-processing on the main net and stores them in a compressed file format.

A storage trace contains a sequence of storage operations with their contract 
and storage addresses, and storage values. With the sequence of operations 
the impact of block processing on the StateDB can be simulated in isolation without requiring 
any other components but the StateDB itself.
Storage operations include EVM operations to read/write the storage of an account, 
balance operations, snapshot operations to revert modifications among many other operations.
With a storage trace, a replay tool can test and profile a StateDB implementation 
under real-world condition in complete isolation. 

With Aida, we can check the functional correctness 
of a stateDB design and implementation, and we can measure its performance.
Aida consists of several tools including the world-state manager creating 
initial world-states from the mainnet for a block, and trace that records
and replays storage traces.

## World State Manager
The purpose of the World State Manager is to build and transfer a minimal world state snapshot
between synced Fantom Opera node database and a target client sub-system.
The World State manager uses an intermediate database with condensed EVM state to support
alternative client subsystems execution, profiling, and testing. It should not be seen
as a target client database.

### Building the World State Manager
You need a configured Go language environment to build the CLI application. 
Please check the [Go documentation](https://go.dev)
for the details of installing the language compiler on your system.

To build the `gen-world-state` application, run `make gen-world-state`.

The build process downloads all the needed modules and libraries, you don't need to install these manually.
The `gen-world-state` executable application will be created in `/build` folder.

### Running the World State Manager
To use Aida World State manager, execute the compiled binary with the command and flags for the desired operation.

```shell
worldstate-cli [global options] command [command options] [arguments...]
```

#### Available commands

| command    | description                                                        |
|------------|--------------------------------------------------------------------|
| account, a | Displays information of an individual account in exported state DB |
| dump, d    | Extracts world state MPT at a given state root into an external DB |
| version, v | Provides information about the application version and build       |
| help, h    | Shows a list of commands or help for one command                   |


## Trace CLI
Trace cli tool provides storage trace recording and replaying functionality.
**Build**
`make trace` generates executable file ./build/trace

`trace` cli has two sub-commands

 - `record` records storage traces of specified block range
 - `replay` replays storage traces of specified block range


### Trace Record
**Run**

`./build/aida-trace record 5000000 5100000`
simulates transaction execution from block 5,000,000 to (and including) block 5,100,000 using [substate](github.com/Fantom-foundation/substate-cli). Storage operations executed during transaction processing are recorded into a compressed file format.

**Options**
 - `--chainid` sets the chain-id (useful if recording from testnet). Default: 250 (mainnet)`
 - `--cpuprofile` records a CPU profile for the replay to be inspected using `pprof`
 - `--disable-progress` disable progress report. Default: `false`
 - `--substatedir` sets directory containing substate database. Default: `./substate.fantom`
 - `--trace-dir` sets trace file output directory. Default: `./`
 - `--trace-debug` print recorded operations. 
 - `--workers` sets the number of worker threads.

### Trace Replay

**Run**

`./build/aida-trace replay --worldstatedir path/to/world-state 5050000 5100000`
reads the recorded traces and re-executes state operations from block 5,050,000 to 5,100,000. The tool initializes stateDB with accounts in the world state from option `--worldstatedir`. The storage operations are executed and update the stateDB sequentially in the order they were recorded.

**Options**

 - `--cpuprofile` records a CPU profile for the replay to be inspected using `pprof`
 - `--db-impl` select between `memory`, `geth`, `carmen` and `flat`. Default: `geth`
 - `--db-variant` select between implementation specific sub-variants, e.g. `go-ldb` or `cpp-file`
 - `--db-logging` enables the logging of each StateDB operation
 - `--db-shadow-impl` if set, a shadow DB implementation of the given type is running in parallel to the primary StateDB
 - `--db-shadow-variant` the variant to be used for the shadow StateDB instance, if enabled
 - `--disable-progress` disable progress report. Default: `false`
 - `--prime-random` randomize order of accounts in StateDB priming.
 - `--prime-seed` set seed for randomizing priming.
 - `--prime-threshold` set number of accounts written to stateDB before applying pending state updates.
 - `--profile` records and displays summary information on operation performance
 - `--substatedir` sets directory containing substate database. Default: `./substate.fantom`
 - `--tracedir` sets trace file directory. Default: `./`
 - `--trace-debug` print replayed operations.
 - `--updatedir` sets directory containing update-set database.
 - `--validate` validate the state after replaying traces.
 - `--workers` sets the number of worker threads.

### Trace Replay Substate

**Run**

`./build/aida-trace replay-substate 5050000 5100000`
reads the recorded traces and re-executes state operations from block 5,050,000 to 5,100,000. The storage operations are executed sequentially in the order they were recorded. The tool iterates through substates to construct a partial stateDB such that the replayed storage operations can simulate read/write with actual data.

**Options**

 - `--cpuprofile` records a CPU profile for the replay to be inspected using `pprof`
 - `--db-impl` select between `memory`, `geth`, `carmen` and `flat`. Default: `geth`
 - `--db-variant` select between implementation specific sub-variants, e.g. `go-ldb` or `cpp-file`
 - `--db-tmp-dir` the directory to use for the temporary StateDB, uses system default if empty
 - `--db-logging` enables the logging of each StateDB operation
 - `--db-shadow-impl` if set, a shadow DB implementation of the given type is running in parallel to the primary StateDB
 - `--db-shadow-variant` the variant to be used for the shadow StateDB instance, if enabled
 - `--disable-progress` disable progress report. Default: `false`
 - `--prime-random` randomize order of accounts in StateDB priming.
 - `--prime-seed` set seed for randomizing priming.
 - `--prime-threshold` set number of accounts written to stateDB before applying pending state updates.
 - `--profile` records and displays summary information on operation performance
 - `--skip-priming` if set, priming will be skipped and no data will be preloaded in the stateDB
 - `--substatedir` sets directory containing substate database. Default: `./substate.fantom`
 - `--tracedir` sets trace file directory. Default: `./`
 - `--trace-debug` print replayed operations.
 - `--validate` validate the state after replaying traces.
 - `--workers` sets the number of worker threads.

### Run VM

**Run**

`./build/aida-runvm --updatedir path/to/updatedb --db-impl [geth/carmen/memory/flat] 4564026 5000000`
executes transactions from block 4,564,026 to 5,000,000. The tool initializes stateDB with accounts in the world state from option `--worldstatedir`. Each transaction calls VM which issues a series of StateDB operations to a selected storage system.

**Options**

 - `--archive` enable the recording of archive data
 - `--archive-variant` selects between different archive implementation variants. Options are empty (=implementation default), `ldb`, `sql`
 - `--chainid` sets the chain-id (useful if recording from testnet). Default: 250 (mainnet)`
 - `--cpuprofile` records a CPU profile for the replay to be inspected using `pprof`
 - `--db-impl` select between `memory`, `geth`, `carmen` and `flat`. Default: `geth`
 - `--db-variant` select between implementation specific sub-variants, e.g. `go-ldb` or `cpp-file`
 - `--db-tmp-dir` the directory to use for the temporary StateDB, uses system default if empty
 - `--db-logging` enables the logging of each StateDB operation
 - `--db-shadow-impl` if set, a shadow DB implementation of the given type is running in parallel to the primary StateDB
 - `--db-shadow-variant` the variant to be used for the shadow StateDB instance, if enabled
 - `--disable-progress` disable progress report. Default: `false`
 - `--max-transactions` limit the maximum number of processed transactions, default: unlimited (`-1`)
 - `--memory-breakdown` display detailed memory usage breakdown after priming and after the EVM runs
 - `--prime-random` randomize order of accounts in StateDB priming.
 - `--prime-seed` set seed for randomizing priming.
 - `--prime-threshold` set number of accounts written to stateDB before applying pending state updates.
 - `--profile` records and displays summary information on operation performance
 - `--skip-priming` if set, priming will be skipped and no data will be preloaded in the stateDB
 - `--substatedir` sets directory containing substate database. Default: `./substate.fantom`
 - `--tracedir` sets trace file directory. Default: `./`
 - `--trace-debug` print replayed operations.
 - `--updatedir` sets directory containing update-set database.
 - `--validate` validate the state after replaying traces.
 - `--workers` sets the number of worker threads.
 - `--vm-impl` select between `geth` and `lfvm`. Default: `geth`


### Run Archive

**Run**

`./build/aida-runarchive --substatedir path/to/substatedb --db-src-dir path/to/statedb/with/archive 4564026 5000000`
executes transactions from block 4,564,026 to 5,000,000 using the historic data in the provided archive. Each transaction
loads the historic state of its block and executes the transaction on it in read-only mode.

**Options**

- `--archive-variant` selects between different archive implementation variants. Options are empty (=implementation default), `ldb`, `sql`
 - `--chainid` sets the chain-id (useful if recording from testnet). Default: 250 (mainnet)`
 - `--cpuprofile` records a CPU profile for the replay to be inspected using `pprof`
 - `--db-impl` select between `memory`, `geth`, `carmen` and `flat`. Default: `geth`
 - `--db-variant` select between implementation specific sub-variants, e.g. `go-ldb` or `cpp-file`
 - `--disable-progress` disable progress report. Default: `false`
 - `--substatedir` sets directory containing substate database. Default: `./substate.fantom`
 - `--validate-tx` validate the effects of each transaction.
 - `--workers` sets the number of worker threads.
 - `--vm-impl` select between `geth` and `lfvm`. Default: `geth`

### Generate an update-set database

**Run**

`./build/aida-trace gen-update-set --worldstatedir path/to/world-state --updatedir path/to/updatedb 4564026 41000000 1000000`
generates piecewise update-sets (merges of output substates) at every 1000000 blocks starting from block 4564026 to block 41000000 and stores them in updateDB. SubstateAlloc of block 4564025 from the world state is reocrded as the first update-set if --worldstatedir is provided. The subsequence update-sets happen at block 5000000 and every 1000000 blocks afterwards. 

**Options**

 - `--substatedir` sets directory containing substate database. Default: `./substate.fantom`
 - `--updatedir` sets directory containing update-set database.
 - `--validate` validate the state after replaying traces.
 - `--worldstatedir` sets directory containing world state database.
 - `--workers` sets the number of worker threads.
