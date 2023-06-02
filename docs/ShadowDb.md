# ShadowDb
## Overview
ShadowDb is a wrapper for any StateDb operations. It runs all operations on two StateDbs simultaneously hence slowing down the command itself.

## Using ShadowDb without existing StateDb
To run for example `runvm` with ShadowDb we need to specify we want to use it with flag `--shadow-db` then we specify implementation with `db-shadow-impl` (carmen, geth...) and variant with `--db-shadow-variant` (go-file, cpp-file...).
Using `--keep-db` will keep both prime and shadow StateDb in structure `path/to/state/db/tmp/prime` and `path/to/state/db/tmp/shadow`.

## Using ShadowDb with existing StateDb
To run for example `apireplay` with ShadowDb we need to respect expected structure. First we specify using ShadowDb with `--shadow-db`. Then we specify path to StateDb and ShadowDb with `--db-src` in which we have to have two StateDbs directories. One named **prime** other named **shadow**. Implementation and Variant are both read from `statedb_info.json`.