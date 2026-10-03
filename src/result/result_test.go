// Copyright 2026 The Gallivant Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package result_test

import (
	"errors"
	"result"
	"strconv"
	"testing"
)

func TestResult(t *testing.T) {
	boom := errors.New("boom")
	ok := result.Wrap(strconv.Atoi("42"))
	bad := result.Wrap(strconv.Atoi("x"))

	if !result.IsOk(ok) || result.IsOk(bad) {
		t.Errorf("IsOk: %v %v", ok, bad)
	}
	if v, err := result.Unwrap(ok); v != 42 || err != nil {
		t.Errorf("Unwrap(ok) = %d, %v", v, err)
	}
	if v, err := result.Unwrap(bad); v != 0 || err == nil {
		t.Errorf("Unwrap(bad) = %d, %v", v, err)
	}
	if got := result.Must(ok); got != 42 {
		t.Errorf("Must(ok) = %d", got)
	}
	if got := result.Or(bad, -1); got != -1 {
		t.Errorf("Or(bad, -1) = %d", got)
	}
	if got := result.Map(ok, strconv.Itoa); got != Ok("42") {
		t.Errorf("Map(ok) = %v", got)
	}
	var e Result[int] = Err(boom)
	if got := result.Map(e, strconv.Itoa); got != Err(boom) {
		t.Errorf("Map(Err) = %v", got)
	}
	if got := result.Then(ok, func(n int) Result[string] { return Ok(strconv.Itoa(n * 2)) }); got != Ok("84") {
		t.Errorf("Then(ok) = %v", got)
	}
	if got := result.ToOption(ok); got != Some(42) {
		t.Errorf("ToOption(ok) = %v", got)
	}
	if got := result.ToOption(bad); got != None {
		t.Errorf("ToOption(bad) = %v", got)
	}

	defer func() {
		if r := recover(); r != boom {
			t.Errorf("Must(Err) panicked with %v", r)
		}
	}()
	result.Must(e)
}
