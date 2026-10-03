# Import capabilities

Gallivant builds the design of [`gocaps`](https://github.com/hfaulds/caps) into
the compiler. Third-party dependencies that reach for dangerous parts of the
standard library (filesystem, network, process execution, environment,
`unsafe`) must be explicitly granted those capabilities at the import site, or
the package fails to compile.

```go
import (
    "github.com/example/logger" with [file.write]
)
```

```
main.go:5:2: import "github.com/example/logger" uses capabilities [file.write, net] but is only granted [file.write]
	[net] github.com/example/logger/transport.go:12: net.Dial
```

## Syntax

A grant is part of the import spec:

```
ImportSpec = [ "." | PackageName ] ImportPath [ CapGrant ] .
CapGrant   = "with" "[" [ Capability { "," Capability } [ "," ] ] "]" .
Capability = identifier { "." identifier } .
```

```go
import "example.com/a" with [net]

import (
    log "github.com/example/logger" with [file.write, net]
    "example.com/b" with [
        exec,
        env,
    ]
    "example.com/c" with [] // explicitly granted nothing
)
```

`with` is a contextual keyword: in Go only `;` can follow an import path, so
an identifier there is unambiguous, and code that uses `with` as an
identifier (including as an import name) keeps compiling. A grant on a
standard-library import or on a package in the same module is allowed and
ignored. If the same path is imported more than once in a package, the
grants are combined.

`gofmt` formats grants and sorts imports with single-line grants as usual.
An import whose grant spans several lines is not moved, and the imports on
either side of it are sorted separately.

## Capabilities

| Capability | Triggered by |
| --- | --- |
| `file.read` | `os.Open`, `os.ReadFile`, `os.Stat`, `filepath.Walk`, … |
| `file.write` | `os.Create`, `os.WriteFile`, `os.Remove`, `os.Mkdir*`, `fmt.Print*`, `log.*`, … |
| `net` | `net.Dial*`, `net.Listen*`, `net/http.Get`, `(*http.Client).Do`, `crypto/tls.Dial`, … |
| `exec` | `os/exec.Command`, `syscall.Exec`, `os.StartProcess`, … |
| `env` | `os.Getenv`, `os.Setenv`, `os.Environ`, `os.UserHomeDir`, … |
| `unsafe` | importing `unsafe`, `reflect`, `plugin` or `C`; assembly files |

The full table lives in `src/cmd/compile/internal/caps/capmap.go`.

## How it works

For each package P the compiler computes

```
direct(P)     = ⋃ Required(s) for every standard-library symbol s referenced in P
transitive(P) = ⋃ effective(Q) for every non-standard-library import Q of P
effective(P)  = direct(P) ∪ transitive(P)
```

and records `effective(P)`, P's module path and the origin of each capability
in P's export data. When compiling P, each import spec of a package from a
*different module* is compared against its `with [...]` grant. Missing
capabilities are compile errors.

The trust boundary is the module edge, and only your own code is checked:

- Only packages of main modules (the module you are building, or every
  module of a workspace) are checked. A dependency's imports of other
  modules are not: its capabilities, and those of everything it imports,
  are charged to the grant on your import of it. Ordinary Go modules
  therefore work as dependencies without knowing about grants.
- Standard library imports are never checked.
- Imports of packages in the same module as P are never checked.
- Your own code's direct use of the standard library is unconstrained.

`cmd/go` passes the module path of the package being compiled to the compiler
with the new `-modpath` flag, and `-checkcaps` for packages of main modules.
Packages compiled without a module path (GOPATH mode, ad-hoc files) are
treated as their own module, so every third-party import is checked.

## Differences from gocaps

- Enforcement happens in the compiler, so there is nothing to install and no
  way to forget to run it.
- gocaps resolves interface-dispatched calls (an `io.Writer` backed by an
  `*os.File`) with a call-graph analysis. The compiler only sees static
  selectors, so capabilities are charged where the concrete standard-library
  symbol is named. In practice that is the package that opened the file or
  dialled the connection, which is what the import graph attributes it to.
- Chains of origin are stored in export data and printed as related
  information in the diagnostic.
