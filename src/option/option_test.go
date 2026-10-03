// Copyright 2026 The Gallivant Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package option_test

import (
	"option"
	"strconv"
	"testing"
)

func TestOption(t *testing.T) {
	var none Option[int]
	some := Some(7)

	if option.IsSome(none) || !option.IsNone(none) {
		t.Error("zero Option is not None")
	}
	if !option.IsSome(some) || option.IsNone(some) {
		t.Error("Some(7) is not Some")
	}
	if got := option.Or(none, 3); got != 3 {
		t.Errorf("Or(None, 3) = %d", got)
	}
	if got := option.Or(some, 3); got != 7 {
		t.Errorf("Or(Some(7), 3) = %d", got)
	}
	if got := option.OrZero(none); got != 0 {
		t.Errorf("OrZero(None) = %d", got)
	}
	if got := option.Must(some); got != 7 {
		t.Errorf("Must(Some(7)) = %d", got)
	}
	if got := option.Map(some, strconv.Itoa); got != Some("7") {
		t.Errorf("Map(Some(7), Itoa) = %v", got)
	}
	if got := option.Map(none, strconv.Itoa); got != None {
		t.Errorf("Map(None, Itoa) = %v", got)
	}

	x := 5
	if got := option.FromPtr(&x); got != Some(5) {
		t.Errorf("FromPtr(&5) = %v", got)
	}
	if got := option.FromPtr[int](nil); got != None {
		t.Errorf("FromPtr(nil) = %v", got)
	}
	m := map[string]int{"a": 1}
	a, aok := m["a"]
	if got := option.FromOK(a, aok); got != Some(1) {
		t.Errorf("FromOK(m[a]) = %v", got)
	}
	b, bok := m["b"]
	if got := option.FromOK(b, bok); got != None {
		t.Errorf("FromOK(m[b]) = %v", got)
	}

	defer func() {
		if recover() == nil {
			t.Error("Must(None) did not panic")
		}
	}()
	option.Must(none)
}
