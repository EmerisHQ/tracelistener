# tracelistener

[![codecov](https://codecov.io/gh/allinbits/tracelistener/branch/main/graph/badge.svg?token=7A8OJVUQYJ)](https://codecov.io/gh/allinbits/tracelistener)
[![Build status](https://github.com/allinbits/tracelistener/workflows/Build/badge.svg)](https://github.com/allinbits/tracelistener/commits/main)
[![Tests status](https://github.com/allinbits/tracelistener/workflows/Tests/badge.svg)](https://github.com/allinbits/tracelistener/commits/main)
[![Lint](https://github.com/allinbits/tracelistener/workflows/Lint/badge.svg?token)](https://github.com/allinbits/tracelistener/commits/main)

UNIX named pipes-based real-time state listener for Cosmos SDK blockchains.

See also the [demeris-backend](https://github.com/allinbits/demeris-backend) docs for an overview of the architecture.

## Description

### What it is

Tracelistener is a program that reads the Cosmos SDK `store` in real-time and dumps the result in a relational database, essentially creating a 1:1 copy of the data available in a module's prefix store.

The relational database of choice is CockroachDB — a Postgres protocol-compatible relational database — while the entirety of tracelistener is written in Go.

Tracelistener is a vital component of the Emeris backend, since it provides

- account balances
- staking amounts
- IBC channels, connections, clients informations

without us having to query those information from full-nodes.

By not querying full-nodes, tracelistener reduces nodes load and diminishes the chance of load-related issues — like nodes not receiving/parsing blocks due to high query amount.

Given the tightly-coupled nature of tracelistener to a Cosmos SDK node, they must be executed together on the same machine.

### How it works

The Cosmos SDK has a little-known feature called **store tracing**, which tracks each and every store operation on a file.

Cosmos SDK store defines four kind of store operations:

- `write`
- `delete`
- `read`
- `iterRange`

Tracelistener is only concerned with the first two.

Each store operation is divided by a newline, and the store operation itself is serialized as JSON.

To reduce hard drive load on the hardware node which is running, tracelistener opens a UNIX named pipe (commonly referred to as FIFO) on which the Cosmos SDK node will then write store tracing lines.

A UNIX shell proof of concept can be summarized like this:

```bash
# This example needs to be executed in two separate terminals.

# Terminal 1
mkfifo /tmp/tracelistener.fifo
gaiad start --trace-store /tmp/tracelistener.fifo

# Terminal 2
cat /tmp/tracelistener.fifo
```

In the first terminal we create a named pipe in `/tmp/tracelistener.fifo`, and then start `gaiad` with the `--trace-store`.

`gaiad` will look like it's stuck on the Tendermint initialization phase: it's normal, FIFO's block writes until there's a reader.

In the second Terminal the `cat` command starts printing JSON store tracing lines, and `gaiad` will unblock itself and resume execution.

If `cat` is killed before `gaiad`, the latter will experience a consensus failure: this is normal, and happens because it is not possible for a program to write on a closed pipe.

In a production environment, tracelistener must always be executed before the SDK node, and killed last.

For each JSON line read, tracelistener unmarshals it into a Go struct and proceeds with the parsing routine — we will refer to this object as **trace operation** from now on.

A trace operation is defined as follows:

```go
type TraceOperation struct {
	Operation   string `json:"operation"`
	Key         []byte `json:"key"`
	Value       []byte `json:"value"`
	BlockHeight uint64 `json:"block_height"`
	TxHash      string `json:"tx_hash"`
}
```

In tracelistener, a *processor* is an entity that is capable of handling SDK store rows.

Right now there's only one processor, called *gaia**.***

Each processor contains **modules**, which are entities capable of

- understanding what's inside a trace operation
- unmarshal the protobuf bytes contained in `Value`
- return a database object and `INSERT` statement to be executed

To understand where to route each trace operation, processors look at the prefix bytes on each operation `Key`.

Each module is responsible of validating a trace operation against a well-defined set of rules, because `Key` prefixes could be shared among different Cosmos SDK modules — for example, the `0x02` prefix is used by the IBC channels module as well as the `supply` one, so the IBC channels module must be sure to not write `supply` database rows in its table.

Once a trace operation has been processed, it is then sent over for database execution.

Database schema is automatically migrated each time tracelistener is executed, but this behavior will change in the future.

## Dependencies

 - CockroachDB
 - your Cosmos SDK-based blockchain node

## Configuration

tracelistener can be configured either through a configuration file or through environment variables.

The configuration file must be named `tracelistener.toml` and must live in either:
 - `/etc/tracelister/tracelistener.toml`
 - `$HOME/.tracelistener/tracelistener.toml`
 - `./tracelistener.toml`

Every configuration entry can be accessed through an environment variable with the same name all uppercase, 
prefixed with the `TRACELISTENER_` string.

While the configuration file field names are case-insensitive, environment variables are case-sensitive.

|Configuration value|Default value|Required|Meaning|
| --- | --- | --- | --- |
|`FIFOPath`|`.tracelistener.fifo`|no|UNIX named pipe path where tracelistener will read data from|
|`DatabaseConnectionURL`| |yes|Database connection URL used to connect to CockroachDB|
|`LogPath`|`./tracelistener.log`|no|Path where tracelistener will write its log file|
|`Type`| |yes|Type of data processor used by tracelistener to process data it reads from `FIFOPath`|
|`Debug`|`false`|no|Enable debug logs, disable file logging|

### Type-specific configuration

**Gaia** configuration

This configuration is used when `Type` is `gaia`.

|Configuration value|Default value|Required|Meaning|
| --- | --- | --- | --- |
|`ProcessorsEnabled`|`bank`|no|List of module processors to be enabled, and which will process data coming from `FIFOPath`|

The list of processors for **gaia** is the following:

 - `bank`
 - `ibc`
 - `liquidityPool`
 - `liquiditySwaps`

## Running tracelistener

0. run a CockroachDB instance somewhere

1. write a configuration file or set the environment appropriately.

2. build it:
    ```shell
    go build -v --ldflags="-s -w"  github.com/allinbits/tracelistener/cmd/tracelistener
    ```
3. run it:
    ```shell
   ./tracelistener
    ```
4. run your chain with the `--trace-store` parameter
    ```shell
   # in this instance, tracelistener FIFO path is set to /home/tl/tracelistener.fifo
   gaiad start --trace-store=/home/tl/tracelistener.fifo 
   ```

## Docker container

This repository contains a Docker image which can be used to run a tracelistener container.

Build it with:

```shell
 docker build -t tracelistener:latest --build-arg GIT_TOKEN={YOUR-TOKEN} .
 ```

## Dependencies & Licenses

The list of non-{Cosmos, AiB, Tendermint} dependencies and their licenses are:

|Module   	                  |License          |
|---	                      |---  	        |
|containerd/fifo   	          |Apache 2.0   	|
|go.uber.org/zap   	          |MIT           	|
|gorilla/websocket   	      |BSD-2   	        |
|cockroachdb/cockroach-go     |Apache 2.0   	|
|stretchr/testify   	      |MIT   	        |
|gogo/protobuf   	          |Only on redistr. |
|go-playground/validator   	  |MIT   	        |
|nxadm/tail   	              |MIT   	        |
|iamolegga/enviper   	      |MIT   	        |
|spf13/viper   	              |MIT   	        |
|jackc/pgx   	              |MIT   	        |
|jmoiron/sqlx   	          |MIT   	        |
|gin-gonic/gin   	          |MIT   	        |
|natefinch/lumberjack  	      |MIT   	        |
|lib/pq   	                  |Unrestricted   	|
|ethereum/go-ethereum   	  |GNU LGPL         |