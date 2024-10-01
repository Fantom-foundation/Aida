# Aida

Aida is a block-processing testing infrastructure for EVM-compatible chains.

## Building the source

Building `aida` requires a Go (version 1.21 or later), plus submodules require either Docker or C compiler with bazel 5.2.3.
Once the dependencies are installed, run

```shell
git submodule update --init --recursive
```
to clone the submodules such as `Carmen` and `Tosca`.\
Then run
```shell
make
```
which builds all Aida tools. The aida tools can be found in ```build/...```.

### Documentation

The tools documentation can be found on the [Wiki](https://github.com/Fantom-foundation/Aida/wiki) page.

## Testing 

Aida-db, a test database, is required for testing. You can obtain a new aida-db update an existing aida-db using the following command:
```
./build/util-db update --aida-db output/path --chain-id 250 --db-tmp path/to/tmp/direcotry
```

# Empty changes
