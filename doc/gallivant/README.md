# Gallivant

Gallivant is a fork of the Go programming language (currently tracking
`go1.27.1`). The thesis: most Go is not written to be performant, so the
language can afford to trade a little machine sympathy for ergonomics and
security.

Gallivant adds four things to Go. Everything else, including the toolchain,
`go build`, `go test`, modules and the standard library, is unchanged.

| Feature | Status | Design |
| --- | --- | --- |
| `enum` types (tagged unions / sum types) | compiler done; `gofmt`/`go vet` in progress | [enums.md](enums.md) |
| `match` statement with exhaustiveness checking | compiler done; `gofmt`/`go vet` in progress | [enums.md](enums.md) |
| Predeclared `Option[T]` and `Result[T]` enums, packages `option` and `result` | done | [option-result.md](option-result.md) |
| Import capabilities (`//caps:` directives) | done | [caps.md](caps.md) |
| `nonil` modules | done | [nonil.md](nonil.md) |

## Building

Gallivant builds exactly like Go. You need a bootstrap Go toolchain of
`go1.24.6` or later:

```sh
cd src
GOROOT_BOOTSTRAP=$HOME/sdk/go1.24.6 ./make.bash
export PATH=$PWD/../bin:$PATH
go version
```

### Iterating on the toolchain

`go install cmd/compile` (or `cmd/link`, `cmd/go`) rebuilds a single tool
quickly. The go command keys its build cache on the compiler's *version
string*, which does not change between your edits, so follow a tool rebuild
with `go clean -cache` or stale objects will be linked and you will chase
ghosts.

## Compatibility

Gallivant is a strict superset of Go 1.27. Every valid Go program is a valid
Gallivant program with the same meaning. The new keywords `enum` and `match`
are *contextual*: they only act as keywords in positions where an identifier
could never have appeared in Go, so existing code that uses `match` or `enum`
as identifiers keeps compiling.

The new predeclared identifiers `Option`, `Result`, `Some`, `None`, `Ok` and
`Err` live in the universe scope, exactly like `any`, `min` and `max`. They can
be shadowed by package-level or local declarations, so existing code that
defines its own `Result` type keeps working.

## Syncing with upstream

The `upstream` remote points at `https://github.com/golang/go.git`. To pick up
a new Go release:

```sh
git fetch upstream --tags
git merge go1.XX.Y
```

Gallivant touches the following areas of the tree, so expect conflicts there:

- `src/cmd/compile/internal/syntax` – scanner, parser and AST (`enum`, `match`)
- `src/cmd/compile/internal/types2` and `src/go/types` – type checking
- `src/cmd/compile/internal/noder` – unified IR export and lowering
- `src/cmd/compile/internal/importer`, `src/go/internal/gcimporter` – export data
- `src/internal/pkgbits` – export data codes
- `src/go/ast`, `src/go/parser`, `src/go/printer` – tooling (`gofmt`, `go vet`)
- `src/cmd/go/internal/work` – passes module information to the compiler
- `src/cmd/compile/internal/caps` – capability analysis
