# Aida Trace
## Overview
Aida-Trace is a storage tracing tool that collects real-world storage traces
from block processing on the main net and stores them in a compressed file format.

A storage trace contains a sequence of storage operations with their contract
and storage addresses and storage values. With the sequence of operations,
the impact of block processing on the StateDB can be simulated in isolation without requiring
any other components but the StateDB itself.
Storage operations include EVM operations to read/write the storage of an account,
balance operations, and snapshot operations to revert modifications, among many other operations.
With a storage trace, a replay tool can test and profile a StateDB implementation
under real-world conditions in complete isolation.

With Aida, we can check the functional correctness
of a StateDB design and implementation, and we can measure its performance.
Aida consists of several tools, including the world-state manager creating
initial world-states from the mainnet for a block, and trace that records
and replays storage traces.

## Classification
* System Under Test
    * StateDB
* Configurable Functional Tests
    * World-state: compare with generated world-state in last block
    * Substate: check per transaction the modified portion of world-state (ie. substate)
    * Check results of StateDB operations
* Non-Functional Tests
    * Memory-Consumption
    * Disk-Space
    * Runtime of operations
* Data set (offline)
    * Mainnet
    * Testnet
    * Stochastic traces

## Requirements
You need a configured Go language environment to build the CLI application.
Please check the [Go documentation](https://go.dev)
for the details of installing the language compiler on your system.

TODO

## Build
To build the `aida-trace` application, run `make aida-trace`.

The build process downloads all the needed modules and libraries, you don't need to install these manually.
The `aida-trace` executable application will be created in `/build` folder.

### Run
To use Aida Trace, execute the compiled binary with the command and flags for the desired operation.

```shell
./build/aida-trace command [command options] [arguments...]
```

| command           | description                                                       |
|-------------------|-------------------------------------------------------------------|
| record            | Captures and records StateDB operations while processing blocks   |
| replay            | Executes storage trace                                            |
| replay-substate   | Executes storage trace using substates                            |
| compare-log       | Compares storage debug log between record and replay              |

## TraceRecord Command
Captures and records StateDB operations while processing blocks

```
./build/aida-trace record --aida-db /path/to/aida_db --tracedir /path/to/output <blockNumFirst> <blockNumLast>
```

<img width="1529" alt="record" src="https://github.com/Fantom-foundation/Aida/assets/40288710/f8d01a54-ff1f-4c4a-a4ad-92b4daaa8a0f">

simulates transaction execution from block `<blockNumFirst>` to (and including) block `<blockNumLast>` using [substate](https://github.com/Fantom-foundation/Substate). Storage operations executed during transaction processing are recorded into a compressed file format.

### Options
```
record:
    --update-buffer-size    buffer size for holding update set in MiB (default: 1<<64 - 1)
    --cpu-profile           records a CPU profile for the replay to be inspected using `pprof`
    --sync-period           defines the number of blocks per sync-period (default: 300)
    --quiet                 disable progress report (default: false)
    --chainid               ChainID for replayer (default: 250)
    --trace-file            set storage trace's output directory
    --trace-debug           enable debug output for tracing
    --debug-from            sets the first block to print trace debug (default: 0)
    --aida-db               set substate, updateset and deleted accounts directory
    --workers               number of worker threads that execute in parallel (default: 4)
    --substate-db           data directory for substate recorder/replayer
    --log                   level of the logging of the app action ("critical", "error", "warning", "notice", "info", "debug"; default: INFO)
```

## TraceReplay Command
Executes storage trace

```
./build/aida-trace replay --aida-db /path/to/aida-db --trace-file /path/to/trace_file <blockNumFirst> <blockNumLast>
```

<img width="1680" alt="replay" src="https://github.com/Fantom-foundation/Aida/assets/40288710/bd922eb5-c8cb-4beb-8506-4cc0263a0882">

reads the recorded traces and re-executes state operations from block `<blockNumFirst>` to `<blockNumLast>`. The tool initializes stateDB with accounts in the world state from option `--aida-db`. The storage operations are executed and update the stateDB sequentially in the order they were recorded.

### Options
```
replay:
    --carmen-schema         select the DB schema used by Carmen's current state DB (default: 0)
    --chainid               ChainID for replayer (default: 250)
    --cpu-profile           records a CPU profile for the replay to be inspected using `pprof`
    --deletion-db           sets the directory containing deleted accounts database
    --quiet                 disable progress report (default: false)
    --sync-period           defines the number of blocks per sync-period (default: 300)
    --keep-db               if set, statedb is not deleted after run
    --memory-breakdown      enables printing of memory usage breakdown (default: false)
    --memory-profile        enables memory allocation profiling
    --random-seed           set random seed (default: -1)
    --prime-threshold       set number of accounts written to stateDB before applying pending state updates (default: 0)
    --profile               enables profiling (default: false)
    --prime-random          randomize order of accounts in StateDB priming (default: false)
    --skip-priming          if set, DB priming should be skipped; most useful with the 'memory' DB implementation (default: false)
    --db-impl               select state DB implementation (default: "geth")
    --db-variant            select a state DB variant
    --db-src                sets the directory contains source state DB data
    --db-tmp                sets the temporary directory where to place state DB data; uses system default if empty
    --db-logging            enable logging of all DB operations (default: false)
    --db-shadow-impl        select state DB implementation to shadow the prime DB implementation
    --db-shadow-variant     select a state DB variant to shadow the prime DB implementation
    --trace-file            set storage trace's output directory
    --trace-debug           enable debug output for tracing
    --debug-from            sets the first block to print trace debug (default: 0)
    --update-db             set update-set database directory
    --validate              enables validation (default: false)
    --validate-ws           enables end-state validation (default: false)
    --aida-db               set substate, updateset and deleted accounts directory
    --workers               number of worker threads that execute in parallel (default: 4)
    --substate-db           data directory for substate recorder/replayer
    --log                   level of the logging of the app action ("critical", "error", "warning", "notice", "info", "debug"; default: INFO)
```

## TraceReplaySubstate Command
Executes storage trace using substates

```
./build/aida-trace replay-substate --aida-db /path/to/aida-db --trace-file /path/to/trace_file <blockNumFirst> <blockNumLast>
```
reads the recorded traces and re-executes state operations from block `<blockNumFirst>` to `<blockNumLast>`. The storage operations are executed sequentially in the order they were recorded. The tool iterates through substates to construct a partial stateDB such that the replayed storage operations can simulate read/write with actual data.

### Options
```
replay-substate:
    --chainid               ChainID for replayer (default: 250)
    --cpu-profile           records a CPU profile for the replay to be inspected using `pprof`
    --quiet                 disable progress report (default: false)
    --prime-random          randomize order of accounts in StateDB priming (default: false)
    --random-seed           set random seed (default: -1)
    --prime-threshold       set number of accounts written to stateDB before applying pending state updates (default: 0)
    --profile               enables profiling (default: false)
    --db-impl               select state DB implementation (default: "geth")
    --db-variant            select a state DB variant
    --db-logging            enable logging of all DB operations (default: false)
    --db-shadow-impl        select state DB implementation to shadow the prime DB implementation
    --db-shadow-variant     select a state DB variant to shadow the prime DB implementation
    --sync-period           defines the number of blocks per sync-period (default: 300)
    --trace-file            set storage trace's output directory
    --trace-debug           enable debug output for tracing
    --debug-from            sets the first block to print trace debug (default: 0)
    --validate              enables validation (default: false)
    --validate-ws           enables end-state validation (default: false)
    --aida-db               set substate, updateset and deleted accounts directory
    --workers               number of worker threads that execute in parallel (default: 4)
    --substate-db           data directory for substate recorder/replayer
    --log                   level of the logging of the app action ("critical", "error", "warning", "notice", "info", "debug"; default: INFO)
```

## TraceCompareLog Command
Compares storage debug log between record and replay

```
./build/aida-trace compare-log --aida-db /path/to/aida-db --trace-file /path/to/trace_file <blockNumFirst> <blockNumLast>
```
The trace compare-log command requires two arguments:
`<blockNumFirst> <blockNumLast>`

`<blockNumFirst>` and `<blockNumLast>` are the first and
last block of the inclusive range of blocks to replay storage traces.`,

### Options
```
compare-log:
    --chainid               ChainID for replayer (default: 250)
    --quiet                 disable progress report (default: false)
    --db-impl               select state DB implementation (default: "geth")
    --sync-period           defines the number of blocks per sync-period (default: 300)
    --trace-file            set storage trace's output directory
    --trace-debug           enable debug output for tracing
    --aida-db               set substate, updateset and deleted accounts directory
    --workers               number of worker threads that execute in parallel (default: 4)
    --substate-db           data directory for substate recorder/replayer
    --log                   level of the logging of the app action ("critical", "error", "warning", "notice", "info", "debug"; default: INFO)
```