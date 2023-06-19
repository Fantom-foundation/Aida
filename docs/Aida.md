Aida is a testing infrastructure for Fantom's blockchain. The main purpose of Aida is to provide a testing fixture for new block-processing components such as StateDB databases and virtual machines. A StateDB database contains balances, nonce, code, and storage key-value pairs of accounts; a virtual machine can process smart contracts. In general, the block-processing of a blockchain evolves the state from one block/transaction to the next. This state transition is highly complex, and the idea of Aida is to provide testing tools so that it can be selected to test StateDB databases and virtual machines in isolation. Making the StateDB/Virtual Machine the System Under Test (SUT) permits wider coverage and more targeted tests rather than performing integration tests/systems tests with the client, which contains block processing. 
 
<img width="1476" alt="aida-architecture" src="https://github.com/Fantom-foundation/Aida/assets/40288710/0b3a2fb0-ad28-4dd3-ba53-2bedea69f56a">

The testing is performed with various tools and data sets. In the centre of testing is the SubstateDB, which contains 
minimal information about the world-state for executing isolated transactions. A substate is the fragment of the world-state needed for executing a transaction (a subset of storage key/value pairs, only contract information of the involved accounts, etc.). We record in the SubstateDB the substate before and after a transaction. The substate-recorder is a specialized opera-client that records as a side-effect of an event import, the SubstateDB.

We have built a variety of profiling/testing tools based on substate. However, the substate tools are insufficient to check the evolution of the whole state. Hence, we have built tools that can do this.
 
Aida tools have recorders, replayers and generating tools. The recorders have observations about the block processing in the mainnet.
The recorders are extended clients that store as a side-effect of their execution test data into the testDB. The testDB contains 
various key/value stores and files. After observing the execution with the recorder tools, we may need to amend the testDB 
with additional data. For this purpose, we have various generative tools. After the testDB is constructed, the block processing can be isolated via replayers.

Here is the list of tools performing tests and obtaining metrics: 
 - [`aida-substate`](Aida-Substate) A profiling/replay tools using the SubstateDB
 - [`aida-trace`](Aida-Trace) A profiling/replay system for tracing StateDB operations and executing the traces on a StateDB database in isoloation.
 - [`aida-runvm`](Aida-RunVM) A tool that tests the world-state evolution of a virtual and its StateDB database.
 - [`aida-runarchive`](Aida-RunArchive) A tool that tests the world-state evolution for an ArchiveDB.  
 - [`aida-stochastic`](Aida-Stochastic) A tool that uses statistical methods for mimicking real-world workloads for extrapolation and fuzzing.

Here is the list of generator tools producing the TestDB:
 - [`aida-dbmerger`](Aida-DB) A tool for generating the TestDB. 
 - [`aida-worldstate`](Aida-Worldstate) A tool for generating the world-state for the first 4.5M blocks (legacy issues due to old Lachesis client and lack of block-processing of the first 4.5M blocks)
 - [`aida-updateset`](Aida-Updateset) A tool for generating the update sets for priming the world state at any arbitrary height.
