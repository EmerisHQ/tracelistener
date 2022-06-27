# Cosmos SDK multi-version support

**Why**: Cosmos SDK v0.44 is incompatible with v0.42. Go doesn't let you having
two imports of different _minor_ versions so we need to produce two binaries
depending on what SDK version we support.

This approach can be applied to other projects that need to support multiple Cosmos SDK versions but which don't want to spawn and maintain completely different codebases.

The idea here is not producing a single binary, but rather multiple binaries that target/interact with a single SDK version.

Usually minor SDK releases don't break compatibility, so you can pretty much version your binaries only by the major release number.

### How does it work

To make this whole thing work, we had to built a build system on top of a build system.

The central control deck here is `Makefile` which glues everything together, and a copy of _each `go.mod` and `go.sum` needed by each SDK version_.

So for SDK v44, we will find

- `go.mod.v44`
- `go.sum.v44`

under the `mods` directory.

A special mod file called `go.mod.bare` contains the bare minimum definition of the module and some imports to aid with the creation of new SDK-specific mods.

**Rule of thumb**: _never_ commit `go.mod`! If you made some changes to a SDK version's `go.mod` make sure to copy the mod and sum files to the `mods` directory, overwriting what's already there, then commit those.

Implementation files should end with a suffix that specify what version of the SDK they support.

For example, `grpc_cosmos_sdk_v42.go` implements GRPC query methods for the SDK v42.

Then, at the beginning of this file one must include a build directive which denotes what SDK version that file implements:

```go
//go:build sdk_v42

package sdkservice

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// All your good stuff :-)
```

To obtain a list of all the available build tags, run:

```bash
make available-go-tags
```

We're leveraging [Go build tags](https://dave.cheney.net/2013/10/12/how-to-use-conditional-compilation-with-the-go-build-tool) to provide different implementations of functions to the binary, letting us only embedding functionalities that work with a given SDK version.

### How to use it

Building with this build system is simple.

The `sdk_targets.json` file in the root of the `sdk-service` repository holds a JSON list of all the supporte SDK versions.

`make` is automatically aware about them thanks to ✨magic✨.

Firstly, you have to setup the SDK version you want:

```bash
make setup-v42
```

Then, build:

```bash
make build-v42
```

The output will be placed in `build/`.

In a development phase, you usually want to work on a specific SDK changeset, commit it and then switch to another version.

This approach will break some tools so make sure to setup your IDE with the appropriate Go build tag, this way autocompletion and all that good stuff will keep working.

# How to upgrade a dependency

If you run `go mod tidy` you will easily face some strange errors. The `tidy` commands assume every build tag to be enabled (i.e. will consider both sdk v42 and v44 files) and will encounter conflicts.

### Rules:

- never run `go mod tidy`
- only use `go get`, start from the dep you want to upgrade:
  ```bash
  make setup-v42
  go get github.com/mydep@v2.0.0
  ```
- try to build the program (`make build-v42`)
- if there you see errors like this:
  ```bash
  tracelistener/processor/datamarshaler/impl_v44.go:16:2: no required module provides package github.com/cosmos/cosmos-sdk/types/address; to add it:
          go get github.com/cosmos/cosmos-sdk/types/address
  tracelistener/processor/datamarshaler/impl_v44.go:21:2: no required module provides package github.com/cosmos/gaia/v6/app; to add it:
          go get github.com/cosmos/gaia/v6/app
  tracelistener/processor/datamarshaler/impl_v44.go:22:2: no required module provides package github.com/cosmos/ibc-go/v2/modules/apps/transfer/types; to add it:
          go get github.com/cosmos/ibc-go/v2/modules/apps/transfer/types
  tracelistener/processor/datamarshaler/impl_v44.go:23:2: no required module provides package github.com/cosmos/ibc-go/v2/modules/core/02-client/types; to add it:
          go get github.com/cosmos/ibc-go/v2/modules/core/02-client/types
  ```
  run **the first one** (avoid running all of them, they are dependent) suggested `go get`
  ```bash
  go get github.com/cosmos/cosmos-sdk/types/address
  ```
  and try building again, repeating this steps until all errors are gone!
- once you were able to build correctly, store the mods and commit them to the repo:
  ```bash
  make store-mods-v42
  ```
