# Aida Stochastic
## Overview
This is a stochastic tool for producing random workloads using Markovian Processes. TODO: extend overview

![stochastic diagram](https://github.com/Fantom-foundation/Aida/assets/40288710/1b980c8e-84a0-4b0e-95a0-f7a9f63cab26)


## Classification
* System Under Test
    * StateDB
* Configurable Functional Tests
    * Check results of StateDB operations with a reference implementation
* Non-Functional Tests
    * Memory-Consumption
    * Disk-Space
    * Runtime of operations
* Data set
    * Synthetic

## Requirements
You need a configured Go language environment to build the CLI application.
Please check the [Go documentation](https://go.dev)
for the details of installing the language compiler on your system.

TODO

## Build
To build the `aida-stochastic` application, run `make aida-stochastic`.

The build process downloads all the needed modules and libraries, you don't need to install these manually.
The `aida-stochastic` executable application will be created in `/build` folder.

### Run
To use Aida Stochastic, execute the compiled binary with the command and flags for the desired operation.

```shell
./build/aida-stochastic command [command options] [arguments...]
```

| command    | description                                                                        |
|------------|------------------------------------------------------------------------------------|
| estimate   | Estimates parameters of access distributions and produces a simulation file        |
| generate   | Generate uniform events file                                                       |
| record     | Record StateDB events while processing blocks                                      |
| replay     | Simulates StateDB operations using a random generator with realistic distributions |
| visualize  | Produces a graphical view of the estimated parameters for various distributions    |

## Generate Command
```
./build/aida-stochastic generate
./build/aida-stochastic estimate events.json
./build/aida-stochastic replay <block#> simulation.json
```

The stochastic produces an events.json file with uniform parameters

### Options
```
replay:
    --block-length          defines the number of transactions per block (default: 10)
    --sync-period           defines the number of blocks per sync-period (default: 300)
    --transaction-length    determines indirectly the length of a transaction (default: 10)
    --num-contracts         number of contracts to create (default: 1_000)
    --num-keys              number of keys to generate (default: 1_000)
    --num-values            number of values to generate (default: 1_000)
    --snapshot-depth        depth of snapshot history (default: 100)
```

## Estimate Command
```
./build/aida-stochastic record <blockNumFirst> <blockNumLast>
./build/aida-stochastic estimate events.json
./build/aida-stochastic replay <simulationLength> <simulation.json>
```

The stochastic estimator command requires one argument: `<events.json>`

`<events.json>` is the event file produced by the stochastic recorder.

## Record Command
Recorder collects events while running the block processor. Produces event statistics for estimator (as events.json).

```
./build/aida-stochastic record <blockNumFirst> <blockNumLast>
```

The stochastic record command requires two arguments: `<blockNumFirst> <blockNumLast>`

`<blockNumFirst>` and `<blockNumLast>` are the first and
last block for recording events.

### Options
```
record:
    --aida-db            set substate, updateset and deleted accounts directory
    --cpu-profile        enables CPU profiling
    --quiet              disable progress report (default: false)
    --sync-period        defines the number of blocks per sync-period (default: 300)
    --chainid            ChainID for replayer (default: 250)
    --output             output path
    --workers            number of worker threads that execute in parallel (default: 4)
    --substate-db        data directory for substate recorder/replayer
```

## Replay Command
Replay performs random walk / does sampling for executing StateDB operations

```
./build/aida-stochastic record <blockNumFirst> <blockNumLast>
./build/aida-stochastic estimate events.json
./build/aida-stochastic replay <simulationLength> <simulation.json>
```

The stochastic replay command requires two argument: `<simulationLength> <simulation.json>`

`<simulationLength>` determines the number of blocks
`<simulation.json>` contains the simulation parameters produced by the stochastic estimator.

### Options
```
replay:
    --aida-db               set substate, updateset and deleted accounts directory
    --carmen-schema         select the DB schema used by Carmen's current state DB (default: 0)
    --continue-on-failure   continue execute after validation failure detected (default: false)
    --cpu-profile           enables CPU profiling
    --debug-from            sets the first block to print trace debug (default: 0)
    --quiet                 disable progress report (default: false)
    --memory-breakdown      enables printing of memory usage breakdown (default: false)
    --random-seed           set random seed (default: -1)
    --db-impl               select state DB implementation (default: "geth")
    --db-variant            select a state DB variant
    --db-tmp                sets the temporary directory where to place state DB data; uses system default if empty
    --db-logging            enable logging of all DB operations (default: false)
    --trace                 enable tracing
    --trace-debug           enable debug output for tracing
    --trace-file            set storage trace's output directory
    --shadow-db             use this flag when using an existing ShadowDb
    --db-shadow-impl        select state DB implementation to shadow the prime DB implementation
    --db-shadow-variant     select a state DB variant to shadow the prime DB implementation
    --balance-range         sets the balance range of the stochastic simulation
    --nonce-range           sets nonce range for stochastic simulation
    --log                   level of the logging of the app action ("critical", "error", "warning", "notice", "info", "debug"; default: INFO)
```

## Visualize Command
Visualize collected events and estimation parameters. Uses web-browser for visualization.

On http://localhost:PORT displays:
* Counting/queuing statistics
* Stationary distributions
* Markov chains
* Etc.

```
./build/aida-stochastic record <blockNumFirst> <blockNumLast>
./build/aida-stochastic visualize events.json
```

The stochastic visualize command requires one argument: `<events.json>`

`<events.json>` is the event file produced by the stochastic recorder.`

### Options
```
visualize:
    --port  enable visualization on `PORT` (default: 8080)
```
