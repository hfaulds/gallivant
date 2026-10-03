// Copyright 2026 The Gallivant Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package result provides helper functions for the predeclared Result type
// and bridges it to the (T, error) convention used by the standard library.
//
// Result is a predeclared enum:
//
//	type Result[T any] enum {
//		Ok(T)
//		Err(error)
//	}
package result

// Wrap converts a (value, error) pair into a Result: Err(err) if err is
// non-nil, and Ok(v) otherwise.
//
//	n := result.Wrap(strconv.Atoi(s))
func Wrap[T any](v T, err error) Result[T] {
	if err != nil {
		return Err(err)
	}
	return Ok(v)
}

// Unwrap converts a Result back into a (value, error) pair for use with
// APIs that expect the standard convention. The value is the zero value of
// T when r is an Err.
func Unwrap[T any](r Result[T]) (T, error) {
	match r {
	case Ok(v):
		return v, nil
	case Err(err):
		var zero T
		return zero, err
	}
}

// IsOk reports whether r is an Ok.
func IsOk[T any](r Result[T]) bool {
	match r {
	case Ok(_):
		return true
	case Err(_):
		return false
	}
}

// Must returns the value held by r and panics with the error if r is an Err.
func Must[T any](r Result[T]) T {
	match r {
	case Ok(v):
		return v
	case Err(err):
		panic(err)
	}
}

// Or returns the value held by r, or def if r is an Err.
func Or[T any](r Result[T], def T) T {
	match r {
	case Ok(v):
		return v
	case Err(_):
		return def
	}
}

// Map returns Ok(f(v)) if r is Ok(v), and r's error otherwise.
func Map[T, U any](r Result[T], f func(T) U) Result[U] {
	match r {
	case Ok(v):
		return Ok(f(v))
	case Err(err):
		return Err(err)
	}
}

// Then returns f(v) if r is Ok(v), and r's error otherwise. It chains
// fallible operations.
func Then[T, U any](r Result[T], f func(T) Result[U]) Result[U] {
	match r {
	case Ok(v):
		return f(v)
	case Err(err):
		return Err(err)
	}
}

// ToOption returns Some(v) if r is Ok(v), and None otherwise, discarding
// the error.
func ToOption[T any](r Result[T]) Option[T] {
	match r {
	case Ok(v):
		return Some(v)
	case Err(_):
		return None
	}
}
