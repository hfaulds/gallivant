// -nonil

// Copyright 2026 The Gallivant Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Zero values in nonil modules (doc/gallivant/nonil.md).

package p

import (
	"strings"
	"sync"
)

type T struct{ x int }

type S struct {
	n int
	p *T
}

type Z struct {
	n int
	s []int
	a [2]string
}

type Named *T

type Shape enum {
	Empty
	Circle(r float64)
}

type Ref enum {
	Some(*T)
	Nothing
}

// Types with a zero value.
var (
	_ int
	_ string
	_ []*T
	_ Z
	_ [0]*T
	_ Shape
	_ Option[*T]
	_ Result[int]
	_ strings.Builder // declared outside the package: its zero value is ready to use
	_ sync.Map
)

// Types without one.
var (
	p1/* ERROR "variable p1 declared without a value: *T has no zero value in a nonil module; give it a value or use an Option" */ *T
	_/* ERROR "has no zero value" */ map[int]int
	_/* ERROR "has no zero value" */ chan int
	_/* ERROR "has no zero value" */ func()
	_/* ERROR "has no zero value" */ error
	_/* ERROR "has no zero value" */ any
	_/* ERROR "S has no zero value in a nonil module (it contains *T)" */ S
	_/* ERROR "has no zero value" */ [2]*T
	_/* ERROR "Named has no zero value in a nonil module;" */ Named
	_/* ERROR "has no zero value" */ Ref
	_/* ERROR "has no zero value" */ Result[*T]
)

var p2 = &T{}

func literals(t *T) {
	_ = Z{}
	_ = S{p: t}
	_ = S{n: 1} /* ERROR "S literal omits field p: *T has no zero value" */
	_ = S{}     /* ERROR "S{} literal: S has no zero value" */
	_ = []S{{n: 1} /* ERROR "omits field p" */}
	_ = [2]*T{t, t}
	_ = [2]*T{t}      /* ERROR "array literal sets 1 of 2 elements" */
	_ = [...]*T{1: t} /* ERROR "array literal sets 1 of 2 elements" */
	_ = []*T{t, t}
	_ = []*T{2: t} /* ERROR "slice literal sets 1 of 3 elements" */
	_ = []int{5: 1}
	_ = map[int]*T{1: t}
	_ = strings.Builder{}
}

func builtins[E any, I ~int](n int, s []*T, e []E) {
	_ = new(int)
	_ = new(T)
	_ = new(* /* ERROR "new(*T): *T has no zero value in a nonil module; use &v or new(v) with a value v" */ T)
	_ = new(S /* ERROR "has no zero value" */)
	_ = new(E /* ERROR "E has no zero value" */)
	_ = new(I)
	_ = make([]int, n)
	_ = make([]*T, 0, n)
	_ = make([]*T, n /* ERROR "make([]*T, n): *T has no zero value in a nonil module; make it with length 0 and append" */)
	_ = make([]E, 1 /* ERROR "has no zero value" */)
	_ = make(map[int]*T)
	_ = make(chan *T)
	clear(s /* ERROR "clear(s): *T has no zero value" */)
	clear(make([]int, n))
	clear(map[int]*T{})
	_ = e
}

func reads(m map[string]*T, mi map[string]int, ch chan *T) {
	_ = mi["a"]
	_ = m /* ERROR "map index m[\"a\"] yields a zero value: *T has no zero value in a nonil module; use v, ok := m[k]" */ ["a"]
	if t, ok := m["a"]; ok {
		_ = t
	}
	m["b"] = &T{}
	_ = <- /* ERROR "receive <-ch yields a zero value" */ ch
	<-ch
	if t, ok := <-ch; ok {
		_ = t
	}
	select {
	case <-ch:
	case t := <- /* ERROR "receive <-ch yields a zero value" */ ch:
		_ = t
	case t, ok := <-ch:
		_, _ = t, ok
	}
	for t := range ch {
		_ = t
	}
	for _, t := range m {
		_ = t
	}
}
