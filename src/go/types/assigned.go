// Copyright 2026 The Gallivant Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file implements the definite-assignment check of nonil modules
// (doc/gallivant/nonil.md): a local variable or named result whose type has
// no zero value (see hasZero) may be declared without a value, but must be
// assigned on every path before it is used.
//
// It mirrors cmd/compile/internal/types2/assigned.go.

package types

import (
	"go/ast"
	"go/token"
	. "internal/types/errors"
)

// A funcBodyInfo is a function body recorded for the unassigned variable
// check, which runs once all bodies (including those of nested function
// literals) have been type-checked.
type funcBodyInfo struct {
	sig  *Signature
	body *ast.BlockStmt
}

// unassignedVars reports uses of variables without a zero value that are
// not definitely assigned, in every recorded function body.
func (check *Checker) unassignedVars() {
	for _, b := range check.bodies {
		check.checkAssigned(b.sig, b.body)
	}
}

// checkAssigned runs the definite-assignment analysis on one function body.
// The analysis is a forward data-flow analysis over the syntax tree. Its
// state is the set of tracked variables that are definitely assigned; at
// control-flow merges states are intersected, and unreachable code has the
// full set. Since a variable declared outside a loop can only become
// assigned inside it, the state at the head of a loop is the state on entry
// and loops need no iteration. Only labels that are the target of a goto
// do, which is handled by repeating the analysis until their states settle.
func (check *Checker) checkAssigned(sig *Signature, body *ast.BlockStmt) {
	a := &assignChecker{check: check, vars: make(map[*Var]int)}

	// Track named results and local variables declared without a value
	// whose type has no zero value. Nested function literals are checked
	// on their own.
	if sig.results != nil {
		for _, v := range sig.results.vars {
			if v.name != "" && v.name != "_" && !check.hasZero(v.typ) {
				a.vars[v] = len(a.vars)
				a.results = append(a.results, v)
			}
		}
	}
	hasGoto := false
	ast.Inspect(body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.FuncLit:
			return false
		case *ast.GenDecl:
			if n.Tok != token.VAR {
				return false
			}
			for _, spec := range n.Specs {
				if spec, _ := spec.(*ast.ValueSpec); spec != nil && spec.Values == nil {
					for _, name := range spec.Names {
						if v := check.varRefs[name]; v != nil && !check.hasZero(v.typ) {
							a.vars[v] = len(a.vars)
						}
					}
				}
			}
		case *ast.BranchStmt:
			if n.Tok == token.GOTO {
				hasGoto = true
			}
		}
		return true
	})
	if len(a.vars) == 0 {
		return
	}

	if hasGoto {
		// Iterate until the states at goto targets no longer change.
		a.labels = make(map[string]assignState)
		for {
			a.again = false
			a.run(body)
			if !a.again {
				break
			}
		}
	}
	a.report = true
	a.run(body)
}

// An assignState is the set of tracked variables that are definitely
// assigned, as a bit set indexed by assignChecker.vars. The set of all
// variables, which is the state of unreachable code, has every bit set.
type assignState []uint64

func (s assignState) has(i int) bool { return s[i/64]&(1<<(i%64)) != 0 }
func (s assignState) set(i int)      { s[i/64] |= 1 << (i % 64) }
func (s assignState) clear(i int)    { s[i/64] &^= 1 << (i % 64) }

func (s assignState) clone() assignState {
	return append(assignState(nil), s...)
}

// meet returns the intersection of s and t; s may be modified.
func (s assignState) meet(t assignState) assignState {
	for i := range s {
		s[i] &= t[i]
	}
	return s
}

func (s assignState) equal(t assignState) bool {
	for i := range s {
		if s[i] != t[i] {
			return false
		}
	}
	return true
}

type assignChecker struct {
	check   *Checker
	vars    map[*Var]int // tracked variables
	results []*Var       // tracked named results
	report  bool         // report errors (on the final pass)

	cur      assignState            // current state
	labels   map[string]assignState // state at goto targets: intersection of the states at the gotos
	seen     map[string]bool        // labels passed in the current pass
	again    bool                   // a goto changed the state of a label already passed
	label    string                 // label of the statement about to be visited, if any
	targets  []*branchTarget        // enclosing break and continue targets, innermost last
	fall     assignState            // state at a fallthrough statement
	reported map[*Var]bool          // variables already reported
}

// A branchTarget is a statement that break (and, for loops, continue)
// statements may refer to.
type branchTarget struct {
	label  string
	isLoop bool
	brk    assignState // intersection of the states at breaks
	cont   assignState // intersection of the states at continues
}

func (a *assignChecker) all() assignState {
	s := make(assignState, (len(a.vars)+63)/64)
	for i := range s {
		s[i] = ^uint64(0)
	}
	return s
}

func (a *assignChecker) run(body *ast.BlockStmt) {
	a.cur = make(assignState, (len(a.vars)+63)/64)
	a.seen = make(map[string]bool)
	a.reported = make(map[*Var]bool)
	a.targets = nil
	a.stmtList(body.List)
}

func (a *assignChecker) stmtList(list []ast.Stmt) {
	for _, s := range list {
		a.stmt(s)
	}
}

func (a *assignChecker) stmt(s ast.Stmt) {
	label := a.label
	a.label = ""

	switch s := s.(type) {
	case nil, *ast.EmptyStmt:

	case *ast.LabeledStmt:
		name := s.Label.Name
		if a.labels != nil {
			if l, ok := a.labels[name]; ok {
				a.cur.meet(l)
			}
		}
		a.seen[name] = true
		a.label = name
		a.stmt(s.Stmt)

	case *ast.BlockStmt:
		a.stmtList(s.List)

	case *ast.ExprStmt:
		a.expr(s.X)
		if call, _ := ast.Unparen(s.X).(*ast.CallExpr); call != nil && a.check.panicCalls[call] {
			a.cur = a.all()
		}

	case *ast.SendStmt:
		a.expr(s.Chan)
		a.expr(s.Value)

	case *ast.DeclStmt:
		d, _ := s.Decl.(*ast.GenDecl)
		if d == nil || d.Tok != token.VAR {
			break
		}
		for _, spec := range d.Specs {
			spec, _ := spec.(*ast.ValueSpec)
			if spec == nil {
				continue
			}
			a.exprs(spec.Values)
			for _, name := range spec.Names {
				if i, ok := a.vars[a.check.varRefs[name]]; ok {
					if spec.Values == nil {
						a.cur.clear(i)
					} else {
						a.cur.set(i)
					}
				}
			}
		}

	case *ast.IncDecStmt:
		a.expr(s.X)

	case *ast.AssignStmt:
		if s.Tok != token.ASSIGN && s.Tok != token.DEFINE {
			// x op= y
			a.exprs(s.Lhs)
			a.exprs(s.Rhs)
			break
		}
		a.exprs(s.Rhs)
		a.assign(s.Lhs)

	case *ast.BranchStmt:
		switch s.Tok {
		case token.BREAK:
			if t := a.target(s.Label, false); t != nil {
				t.brk.meet(a.cur)
			}
		case token.CONTINUE:
			if t := a.target(s.Label, true); t != nil {
				t.cont.meet(a.cur)
			}
		case token.GOTO:
			if a.labels != nil && s.Label != nil {
				name := s.Label.Name
				l, ok := a.labels[name]
				if !ok {
					l = a.all()
				}
				old := l.clone()
				l = l.meet(a.cur)
				a.labels[name] = l
				if a.seen[name] && !l.equal(old) {
					a.again = true
				}
			}
		case token.FALLTHROUGH:
			a.fall = a.cur
		}
		a.cur = a.all()

	case *ast.GoStmt:
		a.expr(s.Call)

	case *ast.DeferStmt:
		a.expr(s.Call)

	case *ast.ReturnStmt:
		a.exprs(s.Results)
		if len(s.Results) == 0 {
			for _, v := range a.results {
				if !a.cur.has(a.vars[v]) {
					a.error(s, v, "result %s is not assigned before return")
				}
			}
		}
		a.cur = a.all()

	case *ast.IfStmt:
		a.stmt(s.Init)
		a.expr(s.Cond)
		entry := a.cur.clone()
		a.stmt(s.Body)
		then := a.cur
		a.cur = entry
		a.stmt(s.Else)
		a.cur.meet(then)

	case *ast.RangeStmt:
		t := &branchTarget{label: label, isLoop: true, brk: a.all(), cont: a.all()}
		a.expr(s.X)
		entry := a.cur.clone()
		var lhs []ast.Expr
		if s.Key != nil {
			lhs = append(lhs, s.Key)
		}
		if s.Value != nil {
			lhs = append(lhs, s.Value)
		}
		a.assign(lhs)
		a.targets = append(a.targets, t)
		a.stmt(s.Body)
		a.targets = a.targets[:len(a.targets)-1]
		a.cur = t.brk.meet(entry) // the loop may end without a break

	case *ast.ForStmt:
		t := &branchTarget{label: label, isLoop: true, brk: a.all(), cont: a.all()}
		a.stmt(s.Init)
		a.expr(s.Cond)
		var entry assignState
		if s.Cond != nil {
			entry = a.cur.clone()
		}
		a.targets = append(a.targets, t)
		a.stmt(s.Body)
		a.targets = a.targets[:len(a.targets)-1]
		a.cur.meet(t.cont)
		a.stmt(s.Post)
		a.cur = t.brk
		if entry != nil {
			a.cur.meet(entry) // the loop may end without a break
		}

	case *ast.SwitchStmt:
		a.stmt(s.Init)
		a.expr(s.Tag)
		a.clauses(label, s.Body, true, true)

	case *ast.TypeSwitchStmt:
		a.stmt(s.Init)
		var guard ast.Expr
		switch g := s.Assign.(type) {
		case *ast.ExprStmt:
			guard = g.X
		case *ast.AssignStmt:
			if len(g.Rhs) == 1 {
				guard = g.Rhs[0]
			}
		}
		if x, _ := ast.Unparen(guard).(*ast.TypeAssertExpr); x != nil {
			a.expr(x.X)
		}
		a.clauses(label, s.Body, false, true)

	case *ast.MatchStmt:
		a.stmt(s.Init)
		a.expr(s.Tag)
		// A match without a default clause is exhaustive.
		a.clauses(label, s.Body, false, false)

	case *ast.SelectStmt:
		t := &branchTarget{label: label, brk: a.all()}
		entry := a.cur
		out := a.all()
		a.targets = append(a.targets, t)
		for _, c := range s.Body.List {
			c, _ := c.(*ast.CommClause)
			if c == nil {
				continue
			}
			a.cur = entry.clone()
			a.stmt(c.Comm)
			a.stmtList(c.Body)
			out.meet(a.cur)
		}
		a.targets = a.targets[:len(a.targets)-1]
		a.cur = out.meet(t.brk)

	default:
		// Other statements are invalid and have been reported by the type
		// checker.
	}
}

// clauses analyzes the clauses of a switch or match statement, whose tag
// has been analyzed. If exprCases is set, the case expressions are
// evaluated. If implicitDefault is set, the statement may complete without
// executing any clause when it has no default clause.
func (a *assignChecker) clauses(label string, body *ast.BlockStmt, exprCases, implicitDefault bool) {
	t := &branchTarget{label: label, brk: a.all()}
	entry := a.cur
	out := a.all()
	hasDefault := false
	var fall assignState // state at a fallthrough into the next clause
	a.targets = append(a.targets, t)
	for _, c := range body.List {
		c, _ := c.(*ast.CaseClause)
		if c == nil {
			continue
		}
		a.cur = entry.clone()
		if c.List == nil {
			hasDefault = true
		} else if exprCases {
			a.exprs(c.List)
		}
		if fall != nil {
			a.cur.meet(fall)
			fall = nil
		}
		a.fall = nil
		a.stmtList(c.Body)
		if a.fall != nil {
			fall = a.fall
			a.fall = nil
		} else {
			out.meet(a.cur)
		}
	}
	a.targets = a.targets[:len(a.targets)-1]
	if !hasDefault && implicitDefault {
		out.meet(entry)
	}
	a.cur = out.meet(t.brk)
}

// target returns the target of a break (or, if cont is set, continue)
// statement with the given label, or nil if there is none (which the type
// checker reports).
func (a *assignChecker) target(label *ast.Ident, cont bool) *branchTarget {
	for i := len(a.targets) - 1; i >= 0; i-- {
		t := a.targets[i]
		if label != nil {
			if t.label == label.Name {
				return t
			}
		} else if t.isLoop || !cont {
			return t
		}
	}
	return nil
}

// assign records the assignment of the left-hand side expressions lhs,
// after evaluating the operands of any that are not plain identifiers.
func (a *assignChecker) assign(lhs []ast.Expr) {
	for _, x := range lhs {
		if _, ok := ast.Unparen(x).(*ast.Ident); !ok {
			a.expr(x)
		}
	}
	for _, x := range lhs {
		if name, ok := ast.Unparen(x).(*ast.Ident); ok {
			if i, ok := a.vars[a.check.varRefs[name]]; ok {
				a.cur.set(i)
			}
		}
	}
}

func (a *assignChecker) exprs(list []ast.Expr) {
	for _, e := range list {
		a.expr(e)
	}
}

// expr checks the uses of tracked variables in e.
func (a *assignChecker) expr(e ast.Expr) {
	if e == nil {
		return
	}
	ast.Inspect(e, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.Ident:
			a.use(n, "%s is used before it is assigned")
		case *ast.FuncLit:
			a.capture(n.Body)
			return false
		}
		return true
	})
}

// capture checks the references to tracked variables in the body of a
// function literal. The function may run at any time after it is created,
// so a variable it reads must be assigned before. Assigning a variable
// without reading it is fine, but does not count as an assignment outside
// the function.
func (a *assignChecker) capture(body *ast.BlockStmt) {
	var inspect func(n ast.Node) bool
	inspect = func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.Ident:
			a.use(n, "%s is captured by a function literal before it is assigned")
		case *ast.AssignStmt:
			if n.Tok == token.ASSIGN || n.Tok == token.DEFINE {
				for _, x := range n.Rhs {
					ast.Inspect(x, inspect)
				}
				for _, x := range n.Lhs {
					if _, ok := ast.Unparen(x).(*ast.Ident); !ok {
						ast.Inspect(x, inspect)
					}
				}
				return false
			}
		}
		return true
	}
	ast.Inspect(body, inspect)
}

// use reports the use of a tracked variable denoted by name that is not
// definitely assigned.
func (a *assignChecker) use(name *ast.Ident, format string) {
	v := a.check.varRefs[name]
	if v == nil {
		return
	}
	if i, ok := a.vars[v]; ok && !a.cur.has(i) {
		a.error(name, v, format)
	}
}

func (a *assignChecker) error(at positioner, v *Var, format string) {
	if !a.report || a.reported[v] {
		return
	}
	a.reported[v] = true
	check := a.check
	check.errorf(at, UnassignedVar, format+" (%s)", v.name, check.noZeroMsg(v.typ, check.noZeroCause(v.typ, nil), ""))
}
