// Copyright 2026 The Gallivant Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package p

import "unsafe"

type Shape enum {
	Empty
	Circle(radius float64)
	Rect(w, h float64)
	Unit()
	Empty /* ERROR "Empty redeclared" */
	_ /* ERROR "non-blank" */
}

type List enum {
	Nil
	Cons(head int, tail *List)
}

type Bad /* ERROR "invalid recursive type" */ enum {
	Loop(x Bad)
}

func (s Shape) Area() float64 {
	match s {
	case Empty:
		return 0
	case Circle(r):
		return 3 * r * r
	case Rect(w, h):
		return w * h
	case Unit():
		return 1
	}
}

func construct() {
	var s Shape = Shape.Empty
	s = Shape.Circle(1)
	s = Shape.Rect(1, 2)
	s = Shape.Unit()
	s = Shape.Circle() /* ERROR "not enough arguments" */
	s = Shape.Rect(1, 2, 3 /* ERROR "too many arguments" */ )
	s = Shape.Circle("x" /* ERROR "cannot use" */ )
	s = Shape /* ERROR "Shape is not a function" */ .Empty()
	s = Shape.Missing /* ERROR "undefined" */
	f := Shape /* ERROR "variant constructor Shape.Circle must be called" */ .Circle
	_ = f
	_ = s
	_ = Shape /* ERROR "variant constructor" */ .Unit
	var n List
	var l List = List.Cons(1, &n)
	_ = l
	_ = s == Shape.Empty
	_ = unsafe.Sizeof(s)
}

func matching(s Shape, x int) {
	match s {
	case Empty:
	case Circle(r):
		_ = r
	case Rect(w, _):
		_ = w
	case Unit():
	}

	match /* ERROR "not exhaustive: missing Rect, Unit" */ s {
	case Empty, Circle(_):
	}

	match s {
	case Empty:
	default:
	}

	match s {
	case Empty, Circle(r /* ERROR "cannot bind r in a case with multiple patterns" */ ):
		_ = r
	default:
	}

	match s {
	case Empty:
	case Empty /* ERROR "duplicate case Empty" */ :
	case Circle /* ERROR "write Circle" */ :
	case Rect /* ERROR "2 payload field(s) but pattern has 1" */ (w):
		_ = w
	case Unit /* ERROR "0 payload field(s) but pattern has 1" */ (_):
	case Nope /* ERROR "Shape has no variant Nope" */ :
	case 1 /* ERROR "invalid match pattern" */ :
	}

	match x /* ERROR "int is not an enum type" */ {
	case 1:
	}

	match s {
	case Circle(_):
		break
	case Empty:
		fallthrough /* ERROR "fallthrough statement out of place" */
	default:
	}

L:
	match s {
	case Empty:
		break L
	default:
	}
}

type Option2[T any] enum {
	None2
	Some2(T)
}

func generic() {
	var o Option2[int] = Option2[int].Some2(3)
	o = Option2[int].None2
	match o {
	case None2:
	case Some2(v):
		var _ int = v
	}
	_ = Option2[string].Some2(1 /* ERROR "cannot use 1" */ )
}

func predeclared(err error) Option[int] {
	var a Option[int] = None
	var b Option[int] = Some(1)
	c := Some(2)
	var d Option[float64] = Some(3)
	_ = Some(4) == d /* ERROR "mismatched types Option[int] and Option[float64]" */
	var e Result[int] = Ok(1)
	var f Result[int] = Err(err)
	g := Ok("x")
	var h Result[string] = g
	_ = []any{a, b, c, d, e, f, g, h}
	x := None /* ERROR "use of None in assignment (needs an Option type)" */
	_ = x
	y := Err /* ERROR "use of Err(err) in assignment (needs a Result type)" */ (err)
	_ = y
	var z any = None /* ERROR "use of None in variable declaration" */
	_ = z
	_ = a == None
	_ = f == Err(err)
	_ = None /* ERROR "operator == not defined" */ == None
	match b {
	case None:
	case Some(v):
		_ = v + 1
	}
	match f {
	case Ok(v):
		_ = v
	case Err(e):
		_ = e.Error()
	}
	return None
}

func wrap(v int, err error) Result[int] {
	if err != nil {
		return Err(err)
	}
	return Ok(v)
}
