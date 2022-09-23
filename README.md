# Fantom Aida

Fantom's off-the-chain EVM and storage testing framework. 

## World State Manager
The purpose of the World State Manager is to build and transfer a minimal world state snapshot
between synced Fantom Opera node database and a target client sub-system.
The World State manager uses an intermediate database with condensed EVM state to support
alternative client subsystems execution, profiling, and testing. It should not be seen
as a target client database.

### Building the World State Manager
You need a configured Go language environment to build the CLI application. Please check the [Go documentation](https://go.dev)
for the details of installing the language compiler on your system.

To build the `worldstate-cli` application, run `make` inside the `/worldstate-cli` subdirectory.

The build process downloads all the needed modules and libraries, you don't need to install these manually.
The `worldstate-cli` executable application will be created in `/worldstate-cli/build` folder.

### Running the World State Manager
To use Aida World State manager, execute the compiled binary with the command and flags for the desired operation.

```shell
worldstate-cli [global options] command [command options] [arguments...]
```

#### Available commands

| command    | description                                                  |
|------------|--------------------------------------------------------------|
| version, v | Provides information about the application version and build |
| help, h    | Shows a list of commands or help for one command             |
