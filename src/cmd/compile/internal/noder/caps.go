// Copyright 2026 The Gallivant Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package noder

import (
	"fmt"
	pathpkg "path"
	"path/filepath"
	"strconv"
	"strings"

	"cmd/compile/internal/base"
	"cmd/compile/internal/caps"
	"cmd/compile/internal/syntax"
	"cmd/compile/internal/types2"
)

// checkCaps implements Gallivant's import capability check for the
// package being compiled (see doc/gallivant/caps.md).
//
// It computes the package's direct capabilities from the standard-library
// symbols it names, unions in the effective capabilities recorded in the
// export data of every non-standard-library import, stores the result in
// caps.Local so that it is written to this package's export data, and
// finally, if -checkcaps is set, reports every import of a package from
// another module whose "with [...]" grant does not cover that package's effective capabilities.
func checkCaps(m posMap, noders []*noder, pkg *types2.Package, info *types2.Info) {
	if base.Flag.Std {
		// Standard-library packages are never checked and never contribute
		// transitively (direct uses of the standard library are charged
		// where the symbol is named), so they carry no fact.
		return
	}

	pkgPath := base.Ctxt.Pkgpath
	direct := caps.NewSet()
	chain := map[string]string{}

	// origin renders pos as "pkgpath/file.go:LINE" for chain entries.
	origin := func(pos syntax.Pos) string {
		return fmt.Sprintf("%s:%d", pathpkg.Join(pkgPath, filepath.Base(pos.RelFilename())), pos.RelLine())
	}
	charge := func(set caps.Set, c caps.Cap, desc func() string) {
		set.Add(c)
		if _, ok := chain[string(c)]; !ok {
			chain[string(c)] = desc()
		}
	}

	// Assembly can do anything.
	if base.Flag.SymABIs != "" || base.Flag.AsmHdr != "" {
		charge(direct, caps.CapUnsafe, func() string { return pkgPath + ": assembly files" })
	}

	// importSite records the first import spec of each import path and the
	// capabilities granted to it across all files of the package.
	type importSite struct {
		pos      syntax.Pos
		resolved string // path after -importcfg importmap, as used for export data
		granted  caps.Set
	}
	sites := map[string]*importSite{}
	var order []string // import paths in first-seen order, for deterministic diagnostics

	for _, p := range noders {
		for _, decl := range p.file.DeclList {
			imp, ok := decl.(*syntax.ImportDecl)
			if !ok || imp.Path == nil || imp.Path.Bad {
				continue
			}
			path, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				continue
			}

			for _, c := range caps.RequiredOnImport(path) {
				charge(direct, c, func() string { return fmt.Sprintf("%s: import %q", origin(imp.Path.Pos()), path) })
			}

			site := sites[path]
			if site == nil {
				resolved := path
				if mapped, ok := base.Flag.Cfg.ImportMap[path]; ok {
					resolved = mapped
				}
				site = &importSite{pos: imp.Pos(), resolved: resolved, granted: caps.NewSet()}
				sites[path] = site
				order = append(order, path)
			}

			for _, x := range imp.Caps {
				name := syntax.String(x)
				if c, ok := caps.Parse(name); ok {
					site.granted.Add(c)
				} else {
					base.ErrorfAt(m.makeXPos(x.Pos()), 0, "unknown capability %s (want one of %s)", name, caps.FormatList(caps.All()))
				}
			}
		}

		// Direct references to standard-library symbols.
		syntax.Inspect(p.file, func(n syntax.Node) bool {
			sel, ok := n.(*syntax.SelectorExpr)
			if !ok {
				return true
			}
			selPkg, name, ok := resolveSelector(info, sel)
			if !ok || !caps.IsStdlib(selPkg) {
				return true
			}
			for _, c := range caps.Required(selPkg, name) {
				charge(direct, c, func() string { return fmt.Sprintf("%s: %s.%s", origin(sel.Pos()), selPkg, name) })
			}
			return true
		})
	}

	// Transitive capabilities from non-standard-library imports.
	transitive := caps.NewSet()
	for _, path := range order {
		site := sites[path]
		if caps.IsStdlib(site.resolved) {
			continue
		}
		fact := caps.Lookup(site.resolved)
		if fact == nil {
			continue
		}
		for _, c := range fact.Caps {
			charge(transitive, c, func() string {
				entry := fmt.Sprintf("%s: import %q", origin(site.pos), path)
				if dep := fact.Chain[string(c)]; dep != "" {
					entry += caps.ChainSep + dep
				}
				return entry
			})
		}
	}

	effective := direct.Union(transitive)
	caps.Local = &caps.Fact{
		Caps:       effective.Sorted(),
		ModulePath: base.Flag.ModPath,
		Chain:      chain,
	}

	// Check grants at every module-crossing import. Only packages of main
	// modules are checked; a dependency's capabilities are charged to
	// the main module's grant on whichever import reaches it.
	if !base.Flag.CheckCaps {
		base.ExitIfErrors()
		return
	}
	for _, path := range order {
		site := sites[path]
		if caps.IsStdlib(site.resolved) {
			continue
		}
		fact := caps.Lookup(site.resolved)
		if fact == nil {
			continue
		}
		if fact.ModulePath != "" && fact.ModulePath == base.Flag.ModPath {
			continue // same module: trusted
		}
		required := caps.NewSet(fact.Caps...)
		missing := required.Diff(site.granted)
		if len(missing) == 0 {
			continue
		}
		var sb strings.Builder
		fmt.Fprintf(&sb, "import %q uses capabilities %s but is only granted %s",
			path, caps.FormatList(required.Sorted()), caps.FormatList(site.granted.Sorted()))
		for _, c := range missing.Sorted() {
			src := fact.Chain[string(c)]
			if src == "" {
				continue
			}
			for _, step := range strings.Split(src, caps.ChainSep) {
				fmt.Fprintf(&sb, "\n\t[%s] %s", c, step)
			}
		}
		base.ErrorfAt(m.makeXPos(site.pos), 0, "%s", sb.String())
	}
	base.ExitIfErrors()
}

// resolveSelector resolves a selector expression to (pkgPath, symbolName).
// symbolName is "Func" for package-level objects or "Type.Method" for
// method values and method expressions on a named type. It returns
// ok=false for selectors that do not refer to an object of an imported
// package (e.g. field access on a local struct).
func resolveSelector(info *types2.Info, sel *syntax.SelectorExpr) (string, string, bool) {
	if selection, ok := info.Selections[sel]; ok {
		if selection.Kind() == types2.MethodVal || selection.Kind() == types2.MethodExpr {
			if named := namedType(selection.Recv()); named != nil {
				if obj := named.Obj(); obj != nil && obj.Pkg() != nil {
					return obj.Pkg().Path(), obj.Name() + "." + selection.Obj().Name(), true
				}
			}
		}
	}
	if obj := info.Uses[sel.Sel]; obj != nil {
		if pkg := obj.Pkg(); pkg != nil {
			return pkg.Path(), obj.Name(), true
		}
	}
	return "", "", false
}

// namedType strips a pointer and aliases and returns the underlying
// *types2.Named, or nil if t isn't (a pointer to) a named type.
func namedType(t types2.Type) *types2.Named {
	if t == nil {
		return nil
	}
	t = types2.Unalias(t)
	if ptr, ok := t.(*types2.Pointer); ok {
		t = types2.Unalias(ptr.Elem())
	}
	if n, ok := t.(*types2.Named); ok {
		return n
	}
	return nil
}
