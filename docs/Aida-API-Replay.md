# Aida API-Replay
## Overview
**API-Replay** is a command for testing RPC interface inside StateDB. It tests the correctness of the historic data in StateDB, communication between StateDB and VM and the RPC interface itself.

It replays **RPC requests** into the **StateDB** and compares the result with response in record. Any unmatched results are logged and if not specifically turned off with `--continue-on-failure` flag, it will shut down the replay since any inconsistency in data needs to be investigated immediately.

Substate is necessary for extracting timestamp of block in order to start EVM correctly.

ShadowDb can be used with API-Replay - see [ShadowDb documentation](https://github.com/Fantom-foundation/Aida/wiki/ShadowDb) for more details

As off right now, these are the supported methods for both `eth` and `ftm` namespaces:
1. getBalance
2. getTransactionCount
3. call
4. getCode
5. getStorageAt

![API-Replay](https://user-images.githubusercontent.com/84449820/234000908-d1108a9f-0b61-448f-8fb8-9feb4cd13a83.png)

## Classification
TODO

## Requirements
You need a configured Go language environment to build the CLI application.
Please check the [Go documentation](https://go.dev)
for the details of installing the language compiler on your system.

TODO

## Build
To build the `aida-api-replay` application, run `make aida-api-replay`.

The build process downloads all the needed modules and libraries, you don't need to install these manually.
The `aida-api-replay` executable application will be created in `/build` folder.

## Run
```
./build/aida-apireplay --api-recording path/to/api-recording --db-src path/to/statedb/with/archive --substate-db path/to/substate <blockNumFirst> <blockNumLast>
```
executes recorded requests into StateDB with block range between **blockNumFirst-blockNumLast** and compares its results with recorded responses. \
**Requests need to be in block range of given StateDB otherwise they will not be executed.**

### Options
```
GLOBAL:
    --aida-db               set substate, updateset and deleted accounts directory
    --api-recording         path to file with recordings
    --shadow-db             enable shadowDb
    --chainid               choose chain id
    --continue-on-failure   does not stop the program when results do not match.
    --db-src                path to StateDB with archive
    --db-variant            select between different StateDB implementation variants
    --db-logging            add detailed logging of db
    --log                   level of the logging of the app action ("critical", "error", "warning", "notice", "info", "debug"; default: INFO)
    --substate-db           path to Substate
    --vm-impl               select VM implementation
    --workers               number of worker threads that execute in parallel (default: 4)
    --trace                 enable tracing
    --trace-debug           enable debug output for tracing
    --trace-file            set storage trace's output directory
```

