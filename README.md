# Aida

Aida is a block-processing testing infrastructure for EVM-compatible chains.

## Building the source

Building `aida` requires both a Go (version 1.21 or later), a C compiler, and bazel 5.2.3. Once the dependencies are installed, run

```shell
make
```
which builds all Aida tools. The aida tools can be found in ```build/...```.

## Testing 

Aida-db, a test database, is required for testing. You can obtain a new aida-db update an existing aida-db using the following command:
```
./build/util-db update --aida-db output/path --chain-id 250 --db-tmp path/to/tmp/direcotry
```

### Documentation

The documentation can be found on the wiki page:
https://github.com/Fantom-foundation/Aida/wiki
