// Copyright 2026 The Gallivant Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file implements the zero-value rules of nonil modules
// (doc/gallivant/nonil.md): code in a nonil module may not create the zero
// value of a type that has none. Use-before-assignment of variables of such
// types is checked separately, in assigned.go.

package types2

import (
	"cmd/compile/internal/syntax"
	. "internal/types/errors"
	"slices"
)

// hasZero reports whether the zero value of t may be created in a nonil
// module. Pointers, maps, channels, functions, interfaces and unsafe.Pointer
// have no zero value (it would be nil), and neither do structs and arrays
// containing them, nor enums whose first variant has such a payload. A type
// parameter has a zero value if every type in its type set has one; in
// particular, a type parameter constrained by any has none.
//
// A struct type declared in a package that is not compiled with nonil (the
// standard library and dependency modules) has a zero value whatever its
// fields: Go types are designed so that their zero value is ready to use
// (strings.Builder, sync.Mutex, bytes.Buffer), and their packages handle
// the nil fields inside.
func (check *Checker) hasZero(t Type) bool {
	return check.noZeroCause(t, nil) == nil
}

// noZeroCause returns the type, possibly t itself, that prevents t from
// having a zero value, or nil if t has one.
func (check *Checker) noZeroCause(t Type, seen map[*Named]bool) Type {
	switch u := Unalias(t).(type) {
	case nil:
		return nil
	case *Basic:
		if u.kind == UnsafePointer {
			return t
		}
		return nil // includes invalid types, to avoid follow-on errors
	case *Named:
		if seen[u] {
			return nil // invalid recursive type, reported elsewhere
		}
		if seen == nil {
			seen = make(map[*Named]bool)
		}
		if _, ok := u.Underlying().(*Struct); ok && u.obj.pkg != nil && !check.nonilPackage(u.obj.pkg) {
			return nil
		}
		seen[u] = true
		defer delete(seen, u)
		if c := check.noZeroCause(u.Underlying(), seen); c != nil {
			if isNoZeroLeaf(c) && c == u.Underlying() {
				return t // e.g. a named pointer or interface type: report the name
			}
			return c
		}
		return nil
	case *Interface:
		if !u.typeSet().IsMethodSet() {
			return nil // a constraint interface used as a type, reported elsewhere
		}
		return t
	case *Pointer, *Map, *Chan, *Signature:
		return t
	case *Slice:
		return nil // the zero value of a slice behaves like an empty slice
	case *Array:
		if u.len == 0 {
			return nil
		}
		return check.noZeroCause(u.elem, seen)
	case *Struct:
		for _, f := range u.fields {
			if c := check.noZeroCause(f.typ, seen); c != nil {
				return c
			}
		}
		return nil
	case *Enum:
		if len(u.variants) == 0 {
			return nil
		}
		for _, f := range u.variants[0].fields {
			if c := check.noZeroCause(f.typ, seen); c != nil {
				return c
			}
		}
		return nil
	case *TypeParam:
		if u.iface().typeSet().zeroable {
			return nil
		}
		var cause Type
		all(u, func(t, _ Type) bool {
			if t == nil {
				cause = u // no specific types (e.g. any)
				return false
			}
			cause = check.noZeroCause(t, seen)
			return cause == nil
		})
		return cause
	}
	return nil
}

// isNoZeroLeaf reports whether t is itself one of the type kinds without a
// zero value, as opposed to a composite type containing one.
func isNoZeroLeaf(t Type) bool {
	switch u := t.(type) {
	case *Pointer, *Map, *Chan, *Signature, *Interface, *TypeParam:
		return true
	case *Basic:
		return u.kind == UnsafePointer
	}
	return false
}

// implementsZeroable reports whether V satisfies the zeroable part of the
// constraint interface T, if T is or embeds the predeclared zeroable. In a
// nonil package this requires V to have a zero value (see hasZero). In any
// other package, as in Go, every type has a zero value and satisfies it.
func (check *Checker) implementsZeroable(V Type, T *Interface, verb string, cause *string) bool {
	if !T.typeSet().zeroable || check == nil || !check.conf.NoNil {
		return true
	}
	c := check.noZeroCause(V, nil)
	if c == nil {
		return true
	}
	if cause != nil {
		*cause = check.sprintf("%s does not %s zeroable (%s)", V, verb, check.noZeroMsg(V, c, ""))
	}
	return false
}

// checkZero reports an error at at if the code being checked is in a nonil
// module and creates the zero value of T, which has none. what describes the
// construct creating the zero value and hint, if not empty, suggests an
// alternative. The result reports whether no error was reported.
func (check *Checker) checkZero(at poser, T Type, what, hint string) bool {
	if !check.conf.NoNil || check.isCgoGenerated(at.Pos()) {
		return true
	}
	cause := check.noZeroCause(T, nil)
	if cause == nil {
		return true
	}
	check.errorf(at, NoZeroValue, "%s: %s", what, check.noZeroMsg(T, cause, hint))
	return false
}

// noZeroMsg describes why type T, with the given no-zero cause, has no
// zero value, followed by hint if it is not empty.
func (check *Checker) noZeroMsg(T, cause Type, hint string) string {
	msg := check.sprintf("%s has no zero value in a nonil module", T)
	if !Identical(T, cause) {
		msg += check.sprintf(" (it contains %s)", cause)
	}
	if hint != "" {
		msg += "; " + hint
	}
	return msg
}

// noteZeroRead records that the single-value expression e, a map index or
// channel receive of type T, yields the zero value of T when the key is
// missing or the channel is closed. Unless e turns out to be used in a
// comma-ok assignment, as the left-hand side of an assignment, or as an
// expression statement, reportZeroReads reports it.
func (check *Checker) noteZeroRead(e syntax.Expr, T Type) {
	if !check.conf.NoNil || check.hasZero(T) || check.isCgoGenerated(e.Pos()) {
		return
	}
	if check.zeroReads == nil {
		check.zeroReads = make(map[syntax.Expr]Type)
	}
	check.zeroReads[e] = T
}

// zeroReadOk marks e (if it is a noted zero read) as used in a way that does
// not observe the zero value.
func (check *Checker) zeroReadOk(e syntax.Expr) {
	if check.zeroReads != nil {
		delete(check.zeroReads, syntax.Unparen(e))
	}
}

// reportZeroReads reports the zero reads noted by noteZeroRead that remain.
func (check *Checker) reportZeroReads() {
	var list []syntax.Expr
	for e := range check.zeroReads {
		list = append(list, e)
	}
	slices.SortFunc(list, func(a, b syntax.Expr) int {
		return cmpPos(a.Pos(), b.Pos())
	})
	for _, e := range list {
		T := check.zeroReads[e]
		what, hint := "map index", "use v, ok := m[k]"
		if _, ok := e.(*syntax.Operation); ok {
			what, hint = "receive", "use v, ok := <-ch"
		}
		check.errorf(e, NoZeroValue, "%s %s yields a zero value: %s", what, e, check.noZeroMsg(T, check.noZeroCause(T, nil), hint))
	}
	check.zeroReads = nil
}
