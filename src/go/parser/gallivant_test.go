// Copyright 2026 The Gallivant Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parser

import (
	"go/ast"
	"go/token"
	"strings"
	"testing"
)

// TestGallivantEnumMatch checks that enum types and match statements are
// parsed, and that enum and match remain ordinary identifiers elsewhere.
func TestGallivantEnumMatch(t *testing.T) {
	const src = `package p

type Shape enum {
	Empty
	Circle(radius float64)
	Rect(w, h float64)
	Unit()
}

type Option[T any] enum { None; Some(T) }

type Empty enum {}

type enum struct{}
type E2 enum
type E3 enum.Foo
type E4 enum[int]

func f(s Shape, match int, enum enum, ch chan int) {
	match s {
	case Empty:
	case Circle(r):
		_ = r
	case Rect(w, _):
		_ = w
	default:
	}
	match x := s; x {
	case Empty, Unit():
	}
	match -match {
	}
	match = 3
	match++
	match, _ = 1, 2
	match[0] = 1
	match(1)
	match <- 1
	match.foo()
	match:
	for {
		break match
	}
	_ = match
	_ = enum
}
`
	fset := token.NewFileSet()
	f, err := ParseFile(fset, "gallivant.go", src, ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	var enums []*ast.EnumType
	var matches []*ast.MatchStmt
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.EnumType:
			enums = append(enums, n)
		case *ast.MatchStmt:
			matches = append(matches, n)
		}
		return true
	})

	if len(enums) != 3 {
		t.Fatalf("got %d enum types, want 3", len(enums))
	}
	shape := enums[0]
	if len(shape.Variants) != 4 {
		t.Fatalf("Shape has %d variants, want 4", len(shape.Variants))
	}
	wantNames := []string{"Empty", "Circle", "Rect", "Unit"}
	wantParams := []int{-1, 1, 1, 0} // -1: no parentheses
	for i, v := range shape.Variants {
		if v.Name.Name != wantNames[i] {
			t.Errorf("variant %d: name %s, want %s", i, v.Name.Name, wantNames[i])
		}
		if wantParams[i] < 0 {
			if v.Params != nil {
				t.Errorf("variant %s: unexpected parameter list", v.Name.Name)
			}
		} else if v.Params == nil || len(v.Params.List) != wantParams[i] {
			t.Errorf("variant %s: got params %v, want %d fields", v.Name.Name, v.Params, wantParams[i])
		}
	}
	if n := len(shape.Variants[2].Params.List[0].Names); n != 2 {
		t.Errorf("Rect has %d field names, want 2", n)
	}
	if len(enums[1].Variants) != 2 {
		t.Errorf("Option has %d variants, want 2", len(enums[1].Variants))
	}
	if len(enums[2].Variants) != 0 {
		t.Errorf("Empty has %d variants, want 0", len(enums[2].Variants))
	}
	if end := fset.Position(shape.End()); end.Line != 8 {
		t.Errorf("Shape enum ends on line %d, want 8", end.Line)
	}

	if len(matches) != 3 {
		t.Fatalf("got %d match statements, want 3", len(matches))
	}
	m := matches[0]
	if m.Init != nil {
		t.Errorf("first match has an init statement")
	}
	if id, _ := m.Tag.(*ast.Ident); id == nil || id.Name != "s" {
		t.Errorf("first match tag = %T, want identifier s", m.Tag)
	}
	if len(m.Body.List) != 4 {
		t.Errorf("first match has %d clauses, want 4", len(m.Body.List))
	}
	if cc := m.Body.List[1].(*ast.CaseClause); len(cc.List) != 1 {
		t.Errorf("Circle clause has %d patterns, want 1", len(cc.List))
	} else if call, _ := cc.List[0].(*ast.CallExpr); call == nil || len(call.Args) != 1 {
		t.Errorf("Circle pattern = %T, want call with 1 binding", cc.List[0])
	}
	if matches[1].Init == nil {
		t.Errorf("second match is missing its init statement")
	}
	if u, _ := matches[2].Tag.(*ast.UnaryExpr); u == nil || u.Op != token.SUB {
		t.Errorf("third match tag = %T, want unary expression", matches[2].Tag)
	}

	// The other uses of "match" and "enum" must have been parsed as identifiers.
	var decls []string
	for _, d := range f.Decls {
		if g, _ := d.(*ast.GenDecl); g != nil {
			for _, s := range g.Specs {
				ts := s.(*ast.TypeSpec)
				decls = append(decls, ts.Name.Name)
				switch ts.Name.Name {
				case "enum":
					if _, ok := ts.Type.(*ast.StructType); !ok {
						t.Errorf("type enum: got %T, want struct type", ts.Type)
					}
				case "E2":
					if id, _ := ts.Type.(*ast.Ident); id == nil || id.Name != "enum" {
						t.Errorf("type E2: got %T, want identifier enum", ts.Type)
					}
				case "E3":
					if _, ok := ts.Type.(*ast.SelectorExpr); !ok {
						t.Errorf("type E3: got %T, want selector expression", ts.Type)
					}
				case "E4":
					if _, ok := ts.Type.(*ast.IndexExpr); !ok {
						t.Errorf("type E4: got %T, want index expression", ts.Type)
					}
				}
			}
		}
	}
	if got := strings.Join(decls, " "); got != "Shape Option Empty enum E2 E3 E4" {
		t.Errorf("type declarations: %s", got)
	}

	fn := f.Decls[len(f.Decls)-1].(*ast.FuncDecl)
	var kinds []string
	for _, s := range fn.Body.List {
		switch s := s.(type) {
		case *ast.MatchStmt:
			kinds = append(kinds, "match")
		case *ast.AssignStmt:
			kinds = append(kinds, "assign")
		case *ast.IncDecStmt:
			kinds = append(kinds, "incdec")
		case *ast.ExprStmt:
			kinds = append(kinds, "expr")
		case *ast.SendStmt:
			kinds = append(kinds, "send")
		case *ast.LabeledStmt:
			kinds = append(kinds, "label:"+s.Label.Name)
		default:
			kinds = append(kinds, "other")
		}
	}
	want := "match match match assign incdec assign assign expr send expr label:match assign assign"
	if got := strings.Join(kinds, " "); got != want {
		t.Errorf("statement kinds:\n got %s\nwant %s", got, want)
	}
}

func TestGallivantMatchErrors(t *testing.T) {
	for _, src := range []string{
		"package p; func f(s S) { match s }",
		"package p; func f(s S) { match x := s; { } }",
	} {
		_, err := ParseFile(token.NewFileSet(), "", src, 0)
		if err == nil {
			t.Errorf("%q: expected syntax error", src)
		}
	}
}
