# Gallivant

Gallivant is a fork of the Go programming language, currently tracking
`go1.27.1`. It is built on the thesis that most Go is not written to be
performant, so the language can trade a little machine sympathy for
ergonomics and security. It is a strict superset of Go: every Go program
compiles unchanged and means the same thing.

## Features

**Enum types and exhaustive `match`** ([design](doc/gallivant/enums.md)).
A `match` that misses a variant does not compile.

```go
type Shape enum {
	Empty
	Circle(radius float64)
	Rect(w, h float64)
}

func area(s Shape) float64 {
	match s {
	case Empty:
		return 0
	case Circle(r):
		return math.Pi * r * r
	case Rect(w, h):
		return w * h
	}
}
```

**Predeclared `Option[T]` and `Result[T]`** ([design](doc/gallivant/option-result.md)).
`Some`, `None`, `Ok` and `Err` construct them. The `option` and `result`
standard-library packages bridge to existing Go APIs.

```go
func parse(s string) Result[int] {
	return result.Wrap(strconv.Atoi(s)) // Ok(n) or Err(err)
}

v, ok := m[k]
o := option.FromOK(v, ok) // Option[V]: Some(v) if ok, else None
```

**Import capabilities** ([design](doc/gallivant/caps.md)).
A dependency from another module that uses the filesystem, network, process
execution, environment or `unsafe` must be granted that capability at the
import site. The compiler enforces it.

```go
import (
	"github.com/example/logger" with [file.write]
)
```

**`nonil` modules** ([design](doc/gallivant/nonil.md)).
A `nonil` line in go.mod makes `nil` a compile error in that module's code.

```
module example.com/app

go 1.27
nonil
```

## Building

You need a bootstrap Go toolchain of `go1.24.6` or later. Any Go >= 1.24.6
in your PATH works, or install one:

```sh
go install golang.org/dl/go1.24.6@latest && go1.24.6 download
```

Then build Gallivant:

```sh
cd src
GOROOT_BOOTSTRAP=$(go1.24.6 env GOROOT) ./make.bash
export PATH=$PWD/../bin:$PATH
go version
```

Set `GOTOOLCHAIN=local` so the go command never downloads a different
toolchain.

`go install cmd/compile` (or `cmd/link`, `cmd/go`) rebuilds a single tool.
The build cache is keyed on the version string, which does not change, so run
`go clean -cache` afterwards or stale objects will be reused.

## Using it in a project

Write a go.mod with `go 1.27`, and optionally `nonil`. Then build and test
with Gallivant's `go` exactly as you would with Go:

```sh
go build ./...
go test ./...
```

`gofmt`, `go vet` and other go/types-based tools understand the new syntax.
Editors and linters that bundle their own copy of `golang.org/x/tools` need
the same small updates this fork made to its vendored copy.

## Status

Done: enum types and `match` (compiler, `gofmt`, `go vet`), `Option` and
`Result` with the `option` and `result` packages, import capabilities, and
`nonil` modules.

Not done: enums have no `reflect.Kind` of their own, so `fmt` prints the
lowered struct unless you define a `String` method; a `go vet` check nudging
nilable results towards `Option`/`Result`.

See [doc/gallivant/README.md](doc/gallivant/README.md) for the full table,
compatibility notes and the list of touched source areas.

## Syncing with upstream

The `upstream` remote points at https://github.com/golang/go.git. To pick up
a new Go release:

```sh
git fetch upstream --tags
git merge go1.XX.Y
```

## License

Unless otherwise noted, the source files are distributed under the BSD-style
license found in the LICENSE file.
