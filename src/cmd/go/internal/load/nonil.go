// Copyright 2026 The Gallivant Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package load

import (
	"cmd/go/internal/cfg"
	"cmd/go/internal/modload"
	"cmd/go/internal/str"
)

// noNil reports whether p is compiled with -nonil (see
// doc/gallivant/nonil.md). Nil is rejected by default. Exempt are:
//
//   - anything in GOROOT, including files named on the command line such
//     as those in GOROOT/test;
//   - packages of dependency modules;
//   - packages of a main module whose go.mod says "nonil false", including
//     files named on the command line from that module's directory tree.
//
// Packages outside any module (GOPATH mode, or files named on the command
// line outside a module) have no go.mod to opt out with; they can pass
// -gcflags=-nonil=false, which follows -nonil on the compiler command line.
func noNil(ld *modload.Loader, p *Package) bool {
	if p.Goroot || p.Standard || cfg.GOROOT != "" && str.HasFilePathPrefix(p.Dir, cfg.GOROOT) {
		return false
	}
	if !cfg.ModulesEnabled {
		return true
	}
	if p.Module != nil {
		return p.Module.NoNil
	}
	if p.Internal.CmdlineFiles {
		// PackageModuleInfo reports no module for command-line files.
		// Use the main module whose directory contains them, if any.
		if nonil, ok := modload.MainModuleNoNilForDir(ld, p.Dir); ok {
			return nonil
		}
		return true
	}
	return false
}
