# Fantom Aida

Aida is a storage tracing tool that collects real-world storage traces from the block-processing on the mainnet and stores them in a compressed file format. 
A storage trace contains a sequence of storage operations with their contract and storage addresses, and storage values. With the sequence of operations the impact of block processing on the StateDB can be simulated in isolation without requiring any other components but the StateDB itself.
Storage operations include EVM operations to read/write the storage of an account, balance operations, snapshot operations to revert modifications among many other operations.
With a storage trace, a replay tool can test and profile a StateDB implementation under real-world condition in complete isolation. With Aida, we can check the functional correctness of a stateDB design and implementation, and we can measure its performance.
