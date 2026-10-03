// run

// Copyright 2026 The Gallivant Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Test enum types, match statements, and the predeclared Option and Result.

package main

import (
	"errors"
	"fmt"
	"option"
	"result"
	"strconv"
)

type Shape enum {
	Empty
	Circle(radius float64)
	Rect(w, h float64)
}

func (s Shape) Area() float64 {
	match s {
	case Empty:
		return 0
	case Circle(r):
		return 3 * r * r
	case Rect(w, h):
		return w * h
	}
}

type Tree[T any] enum {
	Leaf
	Node(left *Tree[T], val T, right *Tree[T])
}

func (t *Tree[T]) Insert(v T, less func(a, b T) bool) *Tree[T] {
	if t == nil {
		n := Tree[T].Node(nil, v, nil)
		return &n
	}
	match *t {
	case Leaf:
		n := Tree[T].Node(nil, v, nil)
		return &n
	case Node(l, x, r):
		var n Tree[T]
		if less(v, x) {
			n = Tree[T].Node(l.Insert(v, less), x, r)
		} else {
			n = Tree[T].Node(l, x, r.Insert(v, less))
		}
		return &n
	}
}

func (t *Tree[T]) Walk(f func(T)) {
	if t == nil {
		return
	}
	match *t {
	case Leaf:
	case Node(l, x, r):
		l.Walk(f)
		f(x)
		r.Walk(f)
	}
}

func parse(s string) Result[int] {
	n, err := strconv.Atoi(s)
	if err != nil {
		return Err(err)
	}
	return Ok(n)
}

func describe(r Result[int]) string {
	match r {
	case Ok(n):
		return "ok " + strconv.Itoa(n)
	case Err(e):
		return "err " + e.Error()
	}
}

func classify(s Shape) string {
	match s {
	case Empty, Rect(_, _):
		return "flat"
	default:
		return "round"
	}
}

func main() {
	shapes := []Shape{Shape.Empty, Shape.Circle(1), Shape.Rect(2, 3)}
	for _, s := range shapes {
		fmt.Println(s.Area(), classify(s))
	}

	// zero value, comparison, map keys
	var z Shape
	fmt.Println(z == Shape.Empty, Shape.Circle(1) == Shape.Circle(1), Shape.Circle(1) == Shape.Circle(2))
	m := map[Shape]int{}
	m[Shape.Rect(1, 2)]++
	m[Shape.Rect(1, 2)]++
	fmt.Println(m[Shape.Rect(1, 2)], len(m))

	// labeled break out of a match inside a loop
loop:
	for i := 0; ; i++ {
		match shapes[i] {
		case Rect(_, _):
			fmt.Println("rect at", i)
			break loop
		default:
		}
	}

	// Option and Result
	var o Option[int]
	fmt.Println(o == None, Some(3) == Some(3), Some(3) == None)
	o = Some(7)
	fmt.Println(option.Or(o, 0), option.OrZero(Option[int](None)), option.Map(o, strconv.Itoa) == Some("7"))
	fmt.Println(describe(parse("42")), "|", describe(parse("x")))
	fmt.Println(describe(result.Wrap(strconv.Atoi("8"))), result.IsOk(parse("y")))
	var r Result[int] = Err(errors.New("boom"))
	fmt.Println(describe(r), result.Or(r, -1))

	// a generic enum used through a shaped instantiation
	var t *Tree[int]
	for _, v := range []int{5, 3, 8, 1, 4} {
		t = t.Insert(v, func(a, b int) bool { return a < b })
	}
	t.Walk(func(v int) { fmt.Print(v, " ") })
	fmt.Println()

	fmt.Printf("%T %T %T\n", o, r, t)
}
