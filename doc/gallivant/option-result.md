# `Option` and `Result`

Gallivant predeclares two generic enums in the universe scope:

```go
type Option[T any] enum {
    None
    Some(T)
}

type Result[T any] enum {
    Ok(T)
    Err(error)
}
```

`Result` wraps the existing `error` interface rather than introducing a new
error type, so every existing `error` value and every `errors` helper keeps
working. `Err(err)` with a nil `err` is permitted but pointless; `nonil`
modules reject it (see [nonil.md](nonil.md)).

## Constructors

Besides the ordinary `Option[int].Some(3)` spelling, four predeclared
identifiers make construction short:

| Expression | Type | Notes |
| --- | --- | --- |
| `Some(x)` | `Option[T]` | `T` is taken from the assignment target if there is one, otherwise it is the default type of `x`. |
| `None` | untyped | Behaves like `nil`: it takes its type from context and is an error without one. |
| `Ok(x)` | `Result[T]` | As `Some`. |
| `Err(e)` | untyped | Behaves like `nil`: assignable to any `Result[T]`, comparable with any `Result[T]`. |

```go
func find(xs []int, want int) Option[int] {
    for _, x := range xs {
        if x == want {
            return Some(x)
        }
    }
    return None
}

func parse(s string) Result[int] {
    n, err := strconv.Atoi(s)
    if err != nil {
        return Err(err)
    }
    return Ok(n)
}

match parse("42") {
case Ok(n):
    fmt.Println(n)
case Err(err):
    fmt.Println("bad input:", err)
}
```

`Some`, `Ok` and `Err` are built-in functions in the same sense as `len` and
`append`: they must be called and cannot be used as values. `None` is a
predeclared untyped value, like `nil`.

All six identifiers can be shadowed, so a package that already declares its
own `Result` type continues to compile unchanged.

## Zero values

`var o Option[int]` is `None`. `var r Result[int]` is `Ok(0)`, because the
first variant is the zero value and `Err(nil)` would be worse.
