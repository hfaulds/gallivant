// -nonil

// Copyright 2026 The Gallivant Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package p

var p *int = nil /* ERROR "use of nil in nonil module" */

func f(q *int, err error) Result[int] {
	if q == nil /* ERROR "use of nil in nonil module" */ {
		return Err(nil /* ERROR "use of nil in nonil module" */ )
	}
	if err != nil /* ERROR "use of nil in nonil module" */ {
		return Err(err)
	}
	return Ok(*q)
}

func g() Option[*int] {
	var zero int // zero values are fine if the type has one
	_ = zero
	return None
}
