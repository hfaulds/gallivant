// Copyright 2026 The Gallivant Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package syntax_test

import (
	"strings"
	"testing"

	. "cmd/compile/internal/syntax"
)

func TestGallivantParse(t *testing.T) {
	const src = `package p

type Shape enum {
	Empty
	Circle(radius float64)
	Rect(w, h float64)
	Unit()
}

type Option[T any] enum { None; Some(T) }

type enum struct{}
type E2 enum
type E3 enum.Foo

func f(s Shape, match int, enum enum, ch chan int) {
	match s {
	case Empty:
	case Circle(r):
		_ = r
	case Rect(w, _):
		_ = w
	default:
	}
	match x := s; x {
	case Empty, Unit:
	}
	match -match {
	}
	match = 3
	match++
	match, _ = 1, 2
	match[0] = 1
	match(1)
	match <- 1
	match.foo()
	match:
	for {
		break match
	}
	_ = match
	_ = enum
}
`
	f, err := Parse(nil, strings.NewReader(src), func(err error) { t.Error(err) }, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	Fprint(&b, f, LineForm)
	out := b.String()
	for _, want := range []string{"type Shape enum{Empty; Circle(radius float64); Rect(w, h float64); Unit()}", "Circle(radius float64)", "match s {", "match x := s; x {", "match -match {", "case Rect(w, _):"} {
		if !strings.Contains(out, want) {
			t.Errorf("printed output missing %q:\n%s", want, out)
		}
	}
}
