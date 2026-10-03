// -nonil

// Copyright 2026 The Gallivant Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// The zeroable constraint in nonil modules (doc/gallivant/nonil.md).

package p

import "strings"

type T struct{ x int }

type S struct{ p *T }

func Zero[E zeroable]() E {
	var z E
	return z
}

func Make[E zeroable](n int) []E { return make([]E, n) }

func New[E zeroable]() *E { return new(E) }

func Get[K comparable, V zeroable](m map[K]V, k K) V { return m[k] }

func Any[E any]() E {
	var z E
	return z /* ERROR "z is used before it is assigned (E has no zero value in a nonil module)" */
}

var (
	_ = Zero[int]()
	_ = Zero[string]()
	_ = Zero[T]()
	_ = Zero[Option[*T]]()
	_ = Zero[strings.Builder]() // declared outside the package
	_ = Zero[* /* ERROR "*T does not satisfy zeroable (*T has no zero value in a nonil module)" */ T]()
	_ = Zero[S /* ERROR "S does not satisfy zeroable (S has no zero value in a nonil module (it contains *T))" */]()
	_ = Zero[error /* ERROR "error does not satisfy zeroable" */]()
	_ = Make[int](3)
	_ = Get(map[string]int{}, "a")
	_ = Get[string, * /* ERROR "*T does not satisfy zeroable" */ T](map[string]*T{}, "a")
)

func nested[A zeroable, B any, C ~int | ~string, D ~*int]() {
	_ = Zero[A]()
	_ = Zero[B /* ERROR "B does not satisfy zeroable" */]()
	_ = Zero[C]()
	_ = Zero[D /* ERROR "D does not satisfy zeroable" */]()
}

// zeroable combines with other constraints.
type Number interface {
	zeroable
	~int | ~float64
}

type Keyed interface {
	comparable
	zeroable
}

func Sum[N Number](xs []N) N {
	var s N
	for _, x := range xs {
		s += x
	}
	return s
}

func Index[K Keyed](xs []K, x K) int {
	var z K
	for i, y := range xs {
		if y == x && y != z {
			return i
		}
	}
	return -1
}

var _ = Index[* /* ERROR "*T does not satisfy zeroable" */ T]([]*T{}, &T{})

// zeroable is a constraint only.
var _ zeroable /* ERROR "cannot use type zeroable outside a type constraint: interface is (or embeds) zeroable" */

type U interface {
	int | zeroable /* ERROR "cannot use zeroable in union" */
}
