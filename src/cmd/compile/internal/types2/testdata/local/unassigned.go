// -nonil

// Copyright 2026 The Gallivant Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Definite assignment of variables without a zero value in nonil modules
// (doc/gallivant/nonil.md).

package p

import "errors"

type T struct{ x int }

func get() *T     { return &T{} }
func cond() bool  { return true }
func use(...any)  {}
func fail() error { return errors.New("fail") }

func zeroable() {
	var n int
	var s []*T
	var o Option[*T]
	use(n, s, o)
}

func simple() {
	var p *T
	use(p /* ERROR "p is used before it is assigned (*T has no zero value in a nonil module)" */)
	var q *T
	q = get()
	use(q)
	var r, s *T = get(), get()
	use(r, s)
}

func branches(c bool) {
	var p *T
	if c {
		p = get()
	} else {
		p = &T{}
	}
	use(p)

	var q *T
	if c {
		q = get()
	}
	use(q /* ERROR "q is used before it is assigned" */)

	var r *T
	if c {
		r = get()
	} else {
		panic("no r")
	}
	use(r)

	var t *T
	if !c {
		return
	}
	t = get()
	use(t)
}

func switches(x int, sh Shape) {
	var p *T
	switch x {
	case 1:
		p = get()
	case 2:
		p = get()
	default:
		p = get()
	}
	use(p)

	var q *T
	switch x {
	case 1:
		q = get()
	}
	use(q /* ERROR "q is used before it is assigned" */)

	var r *T
	switch x {
	case 1:
		r = get()
		fallthrough
	case 2:
		use(r /* ERROR "r is used before it is assigned" */)
	default:
		r = get()
	}

	var t *T
	switch x {
	case 1:
		t = get()
		fallthrough
	case 2:
		t = get()
	default:
		t = get()
	}
	use(t)

	var u *T
	match sh {
	case Empty:
		u = get()
	case Circle(_):
		u = get()
	}
	use(u)

	var v *T
	match sh {
	case Empty:
		v = get()
	case Circle(_):
	}
	use(v /* ERROR "v is used before it is assigned" */)

	var w *T
	var i any = x
	switch i.(type) {
	case int:
		w = get()
	}
	use(w /* ERROR "w is used before it is assigned" */)
}

type Shape enum {
	Empty
	Circle(r float64)
}

func loops(xs []int) {
	var p *T
	for range xs {
		p = get()
	}
	use(p /* ERROR "p is used before it is assigned" */)

	var q *T
	for {
		q = get()
		break
	}
	use(q)

	var r *T
	for {
		if cond() {
			break
		}
		r = get()
	}
	use(r /* ERROR "r is used before it is assigned" */)

	var s *T
	for i := 0; i < 3; i++ {
		s = get()
	}
	use(s /* ERROR "s is used before it is assigned" */)

	var t *T
	for {
		t = get()
		if cond() {
			break
		}
	}
	use(t)

	var u *T
outer:
	for {
		for {
			if cond() {
				break outer
			}
			u = get()
		}
	}
	use(u /* ERROR "u is used before it is assigned" */)

	var v *T
	for i := 0; i < 3; i = (*v /* ERROR "v is used before it is assigned" */).x {
		if cond() {
			continue
		}
		v = get()
	}

	var w *T
	for _, w = range []*T{get()} {
	}
	use(w /* ERROR "w is used before it is assigned" */)

	for {
		var x *T
		use(x /* ERROR "x is used before it is assigned" */)
		x = get()
	}
}

func selects(c chan int) {
	var p *T
	select {
	case <-c:
		p = get()
	default:
		p = get()
	}
	use(p)

	var q *T
	select {
	case <-c:
		q = get()
	default:
	}
	use(q /* ERROR "q is used before it is assigned" */)
}

func gotos() {
	var p *T
	goto L
L:
	use(p /* ERROR "p is used before it is assigned" */)

	var q *T
	q = get()
	goto M
M:
	use(q)

	var r *T
	goto N
O:
	use(r /* ERROR "r is used before it is assigned" */)
	return
N:
	goto O
}

func closures() {
	var p *T
	f := func() { use(p /* ERROR "p is captured by a function literal before it is assigned" */) }
	p = get()
	f()

	var q *T
	q = get()
	g := func() { use(q) }
	g()

	var r *T
	h := func() { r = get() } // writing is fine...
	h()
	use(r /* ERROR "r is used before it is assigned" */) // ...but does not assign r here
}

func fields() {
	var p *T
	(*p /* ERROR "p is used before it is assigned" */).x = 1
	_ = p
}

func results(c bool) (p *T, n int, err error) {
	err = fail()
	if c {
		return // ERROR "result p is not assigned before return"
	}
	p = get()
	return
}

func results4(c bool) (p *T, err error) {
	p = get()
	if c {
		return // ERROR "result err is not assigned before return"
	}
	return p, fail()
}

func results2() (p *T, err error) {
	return get(), fail()
}

func results3() (p *T) {
	defer func() { use(p /* ERROR "p is captured by a function literal before it is assigned" */) }()
	p = get()
	return
}

func redeclare() {
	var p *T
	p, ok := get(), true
	use(p, ok)
}
