# Fantom Aida

Aida is a storage tracing tool that collects real-world storage traces from the block-processing on the mainnet and stores them in a compressed file format. 
A storage trace contains a sequence of storage operations with their contract and storage addresses, and storage values. With the sequence of operations the impact of block processing on the StateDB can be simulated in isolation without requiring any other components but the StateDB itself.
Storage operations include EVM operations to read/write the storage of an account, balance operations, snapshot operations to revert modifications among many other operations.
With a storage trace, a replay tool can test and profile a StateDB implementation under real-world condition in complete isolation. With Aida, we can check the functional correctness of a stateDB design and implementation, and we can measure its performance.

## Trace CLI
Trace cli tool provides storage trace recording and replaying functionality.
**Build**
`make trace` generates executable file ./build/trace

`trace` cli has two sub-commands

 - `record` records storage traces of specified block range
 - `replay` replays storage traces of specified block range


### Trace Record
**Run**

`./build/trace record 5000000 5100000`
simulates transaction execution from block 5,000,000 to (and including) block 5,100,000 using [substate](github.com/Fantom-foundation/substate-cli). Storage operations executed during transaction processing are recorded into a compressed file format.

**Options**
 - `--chainid` sets the chain-id (useful if recording from testnet). Default: 250 (mainnet)`
 - `--substatedir` sets directory contain substate database. Default: `./substate.fantom`
 - `--trace-dir` sets trace file output directory. Default: `./`
 - `--trace-debug` print recorded operations. 
 - `--workers` sets the number of worker threads.

### Trace Replay

**Run**

`./build/trace replay 5050000 5100000`
reads the recorded traces and re-execute state operations from block 5,050,000 to 5,100,000. The storage operations are executed sequentially in the order they were recorded. The tool iterates through substates to construct a partial stateDB such that the replayed storage operations can simulate read/write with actual data.

**Options**

 - `--db-impl` select between `geth` and `carmen`. Default: `geth`
 - `--substatedir` sets directory contain substate database. Default: `./substate.fantom`
 - `--trace-dir` sets trace file directory. Default: `./`
 - `--trace-debug` print replayed operations. 
 - `--validate` validate the state after replaying traces.
 - `--workers` sets the number of worker threads.
