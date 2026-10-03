// Copyright 2026 The Gallivant Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package caps

import "sort"

// Fact is the per-package capability record carried in export data.
//
// Caps is the package's effective capability set: the union of the
// capabilities it uses directly (via standard-library symbol references,
// trust-escape imports and assembly) and the effective capabilities of
// every non-standard-library package it imports. It is stored sorted
// (see Set.Sorted) so export data is deterministic.
//
// ModulePath is the path of the module the package belongs to, as passed
// to the compiler with -modpath. It is empty for packages compiled without
// module information (GOPATH mode, ad-hoc files); importers treat such
// packages as belonging to a different module and always check them.
//
// Chain maps each cap name to a human-readable description of where it
// originates, built bottom-up through the import graph. Direct uses record
// the standard-library symbol or import ("pkg/file.go:LINE: os.Create");
// transitive uses prepend the local import site, with steps separated by
// ChainSep.
type Fact struct {
	Caps       []Cap
	ModulePath string
	Chain      map[string]string // cap name → origin description
}

// ChainSep separates the steps of a Chain entry.
const ChainSep = " → "

// String renders the fact's capability set, e.g. "[file:write, net]".
func (f *Fact) String() string {
	if f == nil {
		return "[]"
	}
	return FormatList(f.Caps)
}

// Encoder is the subset of *pkgbits.Encoder used to write a Fact.
type Encoder interface {
	Bool(bool) bool
	Len(int)
	String(string)
}

// Decoder is the subset of *pkgbits.Decoder used to read a Fact.
type Decoder interface {
	Bool() bool
	Len() int
	String() string
}

// Write appends f to w. A nil f is written as "no fact", which Read
// returns as nil. The layout is:
//
//	Bool(hasFact)
//	String(ModulePath)
//	Len(len(Caps)) Caps...
//	Len(len(Chain)) (String(cap) String(origin))...
//
// Every reader of the public root must consume exactly this layout.
func (f *Fact) Write(w Encoder) {
	if !w.Bool(f != nil) {
		return
	}
	w.String(f.ModulePath)
	w.Len(len(f.Caps))
	for _, c := range f.Caps {
		w.String(string(c))
	}
	keys := make([]string, 0, len(f.Chain))
	for k := range f.Chain {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	w.Len(len(keys))
	for _, k := range keys {
		w.String(k)
		w.String(f.Chain[k])
	}
}

// Read reads a Fact written by Write. It returns nil if no fact was
// written.
func Read(r Decoder) *Fact {
	if !r.Bool() {
		return nil
	}
	f := &Fact{ModulePath: r.String()}
	n := r.Len()
	f.Caps = make([]Cap, n)
	for i := range f.Caps {
		f.Caps[i] = Cap(r.String())
	}
	m := r.Len()
	if m > 0 {
		f.Chain = make(map[string]string, m)
	}
	for i := 0; i < m; i++ {
		k := r.String()
		f.Chain[k] = r.String()
	}
	return f
}

// imported holds the facts of every package imported (directly or
// indirectly, since export data is read for all of them) by the package
// being compiled, keyed by resolved import path.
var imported = map[string]*Fact{}

// Record remembers the fact read from pkgPath's export data.
func Record(pkgPath string, f *Fact) {
	if f != nil {
		imported[pkgPath] = f
	}
}

// Lookup returns the recorded fact for pkgPath, or nil.
func Lookup(pkgPath string) *Fact {
	return imported[pkgPath]
}

// Local is the fact computed for the package being compiled. It is nil
// until the noder has analyzed the package and for standard-library
// packages.
var Local *Fact
