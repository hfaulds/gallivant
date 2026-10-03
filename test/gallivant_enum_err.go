// errorcheck

// Copyright 2026 The Gallivant Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Verify that enum and match errors are reported.

package p

type Shape enum {
	Empty
	Circle(radius float64)
	Rect(w, h float64)
}

func f(s Shape, x int) {
	match s { // ERROR "not exhaustive: missing Rect"
	case Empty:
	case Circle(_):
	}

	match s {
	case Empty:
	case Circle(r): // ERROR "declared and not used: r"
	case Rect(w, _): // ERROR "declared and not used: w"
	}

	match s {
	case Empty:
	case Empty: // ERROR "duplicate case Empty"
	case Circle: // ERROR "write Circle"
	case Rect(w): // ERROR "2 payload field\(s\) but pattern has 1"
		_ = w
	}

	match x { // ERROR "int is not an enum type"
	}

	_ = Shape.Circle // ERROR "variant constructor Shape.Circle must be called"
	_ = Shape.Circle("x") // ERROR "cannot use"
	_ = Shape.Missing // ERROR "undefined"
	var _ Shape = Shape{} // ERROR "invalid composite literal type Shape"

	o := None // ERROR "use of None in assignment"
	_ = o
}
