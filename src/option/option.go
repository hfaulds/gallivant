// Copyright 2026 The Gallivant Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package option provides helper functions for the predeclared Option type.
//
// Option is a predeclared enum:
//
//	type Option[T any] enum {
//		None
//		Some(T)
//	}
//
// Most code should use match to inspect an Option. The functions here cover
// the common cases where a match would be noise.
package option

// IsSome reports whether o holds a value.
func IsSome[T any](o Option[T]) bool {
	match o {
	case Some(_):
		return true
	case None:
		return false
	}
}

// IsNone reports whether o is None.
func IsNone[T any](o Option[T]) bool {
	return !IsSome(o)
}

// Or returns the value held by o, or def if o is None.
func Or[T any](o Option[T], def T) T {
	match o {
	case Some(v):
		return v
	case None:
		return def
	}
}

// OrZero returns the value held by o, or the zero value of T if o is None.
func OrZero[T any](o Option[T]) T {
	var zero T
	return Or(o, zero)
}

// Must returns the value held by o and panics if o is None.
func Must[T any](o Option[T]) T {
	match o {
	case Some(v):
		return v
	case None:
		panic("option: Must called on None")
	}
}

// Map returns Some(f(v)) if o is Some(v), and None otherwise.
func Map[T, U any](o Option[T], f func(T) U) Option[U] {
	match o {
	case Some(v):
		return Some(f(v))
	case None:
		return None
	}
}

// FromPtr returns Some(*p) if p is non-nil, and None otherwise. It is the
// bridge from APIs that use nil pointers to signal absence.
func FromPtr[T any](p *T) Option[T] {
	if p == nil {
		return None
	}
	return Some(*p)
}

// FromOK returns Some(v) if ok is true, and None otherwise. It converts the
// results of a comma-ok expression such as a map index or type assertion:
//
//	v, ok := m[key]
//	o := option.FromOK(v, ok)
func FromOK[T any](v T, ok bool) Option[T] {
	if !ok {
		return None
	}
	return Some(v)
}
