// Copyright 2026 The Gallivant Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Outside nonil modules every type has a zero value and satisfies zeroable.

package p

func Zero[E zeroable]() E {
	var z E
	return z
}

var (
	_ = Zero[*int]()
	_ = Zero[error]()
	_ = Zero[map[string]int]()
)

func f[E any]() E { return Zero[E]() }

var _ zeroable /* ERROR "cannot use type zeroable outside a type constraint" */
