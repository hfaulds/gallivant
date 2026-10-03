# Enums and `match`

## Enum types

An enum is a named type whose values are exactly one of a fixed set of
*variants*. Each variant has a name and an optional payload, written like a
parameter list.

```go
type Shape enum {
    Empty
    Circle(radius float64)
    Rect(w, h float64)
}
```

Rules:

- An `enum` type may only appear as the right-hand side of a type
  declaration (`type T enum { ... }`). Anonymous enum types are not allowed.
  This keeps every variant nameable as `T.Variant`.
- Variant names must be unique within the enum and start with a letter, like
  any identifier. Exported-ness follows the usual capitalisation rule and
  applies to construction and matching from other packages.
- Payload fields are declared with parameter-list syntax: `Circle(radius
  float64)`, `Rect(w, h float64)`, or unnamed as `Some(T)`. Unnamed payloads
  are positional.
- Enums may be generic: `type Option[T any] enum { None; Some(T) }`.
- The zero value of an enum is its first variant with zero-valued payloads.
  Order your variants so that the first one is the sensible default (this is
  why `Option` lists `None` first).
- An enum is comparable with `==` if every payload type is comparable, and
  can then be used as a map key.
- Methods may be declared on enum types exactly like on any other named type.

### Constructing values

Variants are selected from the type: `Shape.Empty`, `Shape.Circle(2.5)`,
`Option[int].Some(3)`. A payload-less variant is a value; a variant with a
payload is a constructor that must be called. Variant constructors cannot be
used as function values (`f := Shape.Circle` is an error).

### Representation

The compiler lowers an enum to a struct with a tag field followed by one field
per payload of every variant. Payloads are not overlapped, so an enum is as
large as the sum of its payloads plus the tag. This is deliberate: it keeps
the garbage collector's view of the value simple and avoids `unsafe`. If you
need compact unions, use an interface.

`reflect` and `fmt` currently see the lowered struct. Giving enums their own
`reflect.Kind` is future work.

## `match`

`match` is a statement that branches on the variant of an enum value and binds
its payload. It is exhaustive: every variant must be handled or the program
does not compile.

```go
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

Rules:

- The subject must have an enum type (including instances of generic enums
  such as `Option[int]`). Like `switch`, `match` accepts an optional init
  statement: `match x := f(); x { ... }`.
- A case pattern is `Variant` or `Variant(b1, b2, ...)`. Variant names are
  looked up in the subject's enum, not in the surrounding scope, so they are
  written unqualified.
- The number of bindings must equal the number of payload fields. Each binding
  is either `_` or a new identifier declared in the scope of that case with the
  payload field's type.
- A case may list several patterns separated by commas (`case Empty,
  Rect(_, _):`) only if none of them bind identifiers.
- Each variant may appear at most once across all cases.
- `default:` is permitted as a wildcard and switches off the exhaustiveness
  check for that match. Prefer listing every variant.
- `break` leaves the match. `fallthrough` is not permitted.
- A `match` whose cases all end in terminating statements is itself a
  terminating statement, so the `area` function above needs no trailing
  `return`.

### Contextual keywords

`match` is only a keyword at the start of a statement when it is followed by a
token that can start an expression but could not follow an identifier in a
valid statement: another identifier, a literal, `*`, `&`, `-`, `!`, `^`,
`func`, `struct`, `map`, `chan`, or `interface`. In particular
`match(x)`, `match[i] = v`, `match <- v`, `match.Field`, `match = v` and
`match := v` keep their Go meaning. If you need to match on a parenthesised
or receive expression, bind it first: `match v := <-ch; v { ... }`.

`enum` is only a keyword when it appears as the type in a type declaration and
is followed by `{`.

### Lowering

```go
match s {
case Circle(r): body
}
```

becomes, in the compiler's intermediate representation,

```go
tmp := s
switch tmp.tag {
case 1:
    r := tmp.Circle_radius
    body
}
```
