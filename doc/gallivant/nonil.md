# `nonil` modules

`nil` is the source of a large share of Go runtime panics. Gallivant cannot
remove `nil` from the language without breaking every existing program and the
standard library, so it does the next best thing: a module can opt out of
writing `nil` and is pushed towards `Option` and `Result` instead.

## Opting in

```
module example.com/app

go 1.27
nonil
```

`cmd/go` passes `-nonil` to the compiler for every package in a module with the
`nonil` directive. In those packages:

- The predeclared identifier `nil` is an error: `use of nil in nonil module;
  use Option, Result or a zero value instead`.
- Comparing a value with `nil` is therefore impossible; use `match` or
  `Option` instead of `if p != nil`.
- `Err(nil)` is rejected.
- Functions with a result of pointer, map, slice, channel, function or
  interface type are reported by `go vet` (not the compiler) when a result of
  `Option[T]` or `Result[T]` would say what the caller must handle. This check
  is advisory.

Values of nilable types can still *be* nil: a `*T` field's zero value is nil,
and code outside the module can hand you nil. What `nonil` removes is the
habit of producing and comparing against nil in your own code. Dereferencing a
zero-value pointer still panics exactly as in Go.

## Working with the standard library

Many standard-library functions return `(T, error)`. Two tiny helpers bridge
them into `Result`:

```go
func Wrap[T any](v T, err error) Result[T]   // Ok(v) if err == nil, else Err(err)
func (r Result[T]) Unwrap() (T, error)       // the reverse
```

These live in the new standard-library package `result`, so they can be
written in ordinary Go and improved without touching the compiler.

## Status

Designed; the compiler flag and `go.mod` directive are not yet implemented.
