// Copyright 2026 The Gallivant Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package load

import (
	"cmd/go/internal/cfg"
	"cmd/go/internal/modload"
	"cmd/go/internal/str"
)

// checkCaps reports whether p is compiled with -checkcaps, which enforces
// the capability grants on its imports (see doc/gallivant/caps.md). It
// follows the same rules as noNil, with "caps false" in go.mod as the
// opt-out: grants are enforced in your own code by default, never in
// GOROOT or in dependency modules.
func checkCaps(ld *modload.Loader, p *Package) bool {
	if p.Goroot || p.Standard || cfg.GOROOT != "" && str.HasFilePathPrefix(p.Dir, cfg.GOROOT) {
		return false
	}
	if !cfg.ModulesEnabled {
		return true
	}
	if p.Module != nil {
		return p.Module.Caps
	}
	if p.Internal.CmdlineFiles {
		// PackageModuleInfo reports no module for command-line files.
		// Use the main module whose directory contains them, if any.
		if caps, ok := modload.MainModuleCapsForDir(ld, p.Dir); ok {
			return caps
		}
		return true
	}
	return false
}
