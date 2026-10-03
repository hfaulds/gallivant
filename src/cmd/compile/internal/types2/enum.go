// Copyright 2026 The Gallivant Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types2

import (
	"cmd/compile/internal/syntax"
	. "internal/types/errors"
	"strings"
)

// ----------------------------------------------------------------------------
// API

// An Enum represents an enum type: a tagged union whose values are exactly
// one of a fixed set of variants, each carrying an optional payload.
//
// Enum types only ever appear as the underlying type of a defined type:
//
//	type Shape enum {
//		Empty
//		Circle(radius float64)
//		Rect(w, h float64)
//	}
type Enum struct {
	variants []*Variant
	obj      *TypeName // the declaring type name; may be nil for synthetic enums
}

// NewEnum returns a new enum type with the given variants. The variants'
// indices must match their position in the slice; their types are set to
// typ, which should be the defined type whose underlying type the enum is.
func NewEnum(variants []*Variant, typ Type) *Enum {
	e := &Enum{variants: variants}
	for i, v := range variants {
		if v.index != i {
			panic("variant index mismatch")
		}
		if typ != nil {
			v.typ = typ
		}
	}
	e.markComplete()
	return e
}

// NumVariants returns the number of variants of the enum.
func (e *Enum) NumVariants() int { return len(e.variants) }

// Variant returns the i'th variant for 0 <= i < NumVariants().
func (e *Enum) Variant(i int) *Variant { return e.variants[i] }

// VariantByName returns the variant with the given name, or nil.
func (e *Enum) VariantByName(name string) *Variant {
	for _, v := range e.variants {
		if v.name == name {
			return v
		}
	}
	return nil
}

func (e *Enum) Underlying() Type { return e }
func (e *Enum) String() string   { return TypeString(e, nil) }

// A Variant represents one variant of an enum type. A Variant is an Object
// whose Type is the enum type it belongs to. It is recorded in Info.Defs for
// its declaration and in Info.Uses for every T.Variant selector and every
// match pattern that names it.
type Variant struct {
	object
	index     int
	fields    []*Var // payload fields; nil if the variant has no payload
	hasParens bool   // declared as Name(...) rather than Name
}

// NewVariant returns a new enum variant. index is the position of the variant
// in its enum; fields is the payload, or nil for a payload-less variant.
// hasParens reports whether the variant was declared with a parameter list
// (possibly empty). The variant's type is set when the enclosing enum is
// created with NewEnum.
func NewVariant(pos syntax.Pos, pkg *Package, name string, index int, fields []*Var, hasParens bool) *Variant {
	if fields == nil && hasParens {
		fields = []*Var{}
	}
	return &Variant{object{pos: pos, pkg: pkg, name: name}, index, fields, hasParens}
}

// Index returns the position of the variant within its enum.
func (v *Variant) Index() int { return v.index }

// NumFields returns the number of payload fields.
func (v *Variant) NumFields() int { return len(v.fields) }

// Field returns the i'th payload field for 0 <= i < NumFields().
func (v *Variant) Field(i int) *Var { return v.fields[i] }

// HasPayload reports whether the variant was declared with a parameter list,
// i.e. whether it must be constructed and matched with parentheses.
func (v *Variant) HasPayload() bool { return v.hasParens }

func (v *Variant) String() string { return ObjectString(v, nil) }

// ----------------------------------------------------------------------------
// Implementation

func (e *Enum) markComplete() {
	if e.variants == nil {
		e.variants = make([]*Variant, 0)
	}
}

// constructorSig returns the signature of the variant constructor
// func(payload...) T, where T is the (possibly instantiated) enum type.
func (v *Variant) constructorSig(T Type) *Signature {
	params := make([]*Var, len(v.fields))
	for i, f := range v.fields {
		params[i] = NewParam(f.pos, f.pkg, f.name, f.typ)
	}
	sig := NewSignatureType(nil, nil, nil, NewTuple(params...), NewTuple(NewParam(v.pos, v.pkg, "", T)), false)
	sig.variantCtor = v
	return sig
}

// enumType type-checks the enum type literal e declared as the underlying
// type of def and populates typ.
func (check *Checker) enumType(typ *Enum, e *syntax.EnumType, def *TypeName) {
	typ.obj = def
	var variants []*Variant
	var vset objset
	for _, vd := range e.VariantList {
		name := vd.Name.Value
		if name == "_" {
			check.error(vd.Name, InvalidEnum, "enum variants must have a unique non-blank name")
			continue
		}
		var fields []*Var
		if vd.HasParens {
			_, fields, _ = check.collectParams(FieldVar, vd.FieldList)
			if fields == nil {
				fields = []*Var{}
			}
		}
		v := NewVariant(vd.Name.Pos(), check.pkg, name, len(variants), fields, vd.HasParens)
		v.typ = def.typ
		if alt := vset.insert(v); alt != nil {
			err := check.newError(InvalidEnum)
			err.addf(vd.Name, "%s redeclared", name)
			err.addAltDecl(alt)
			err.report()
			continue
		}
		check.recordDef(vd.Name, v)
		variants = append(variants, v)
	}
	typ.variants = variants
	typ.markComplete()
}

// enumTagType returns the type of the tag field used when lowering an enum
// with n variants.
func enumTagType(n int) Type {
	switch {
	case n <= 1<<8:
		return Typ[Uint8]
	case n <= 1<<16:
		return Typ[Uint16]
	}
	return Typ[Uint32]
}

// loweredStruct returns the struct type an enum is represented as: a tag
// field followed by the payload fields of every variant in declaration
// order. Sizes and alignments of enums are those of this struct, and the
// compiler lays enums out exactly like it.
func (e *Enum) loweredStruct() *Struct {
	fields := []*Var{NewField(nopos, nil, "tag", enumTagType(len(e.variants)), false)}
	for _, v := range e.variants {
		for _, f := range v.fields {
			fields = append(fields, NewField(f.pos, f.pkg, "", f.typ, false))
		}
	}
	return &Struct{fields: fields}
}

// isUntypedVariant reports whether t is the type of an untyped predeclared
// variant value (None or Err(e)), which, like untyped nil, takes its type
// from the assignment context.
func isUntypedVariant(t Type) bool {
	if b, _ := t.(*Basic); b != nil {
		return b.kind == UntypedNone || b.kind == UntypedErr
	}
	return false
}

// predeclaredEnumInstance reports whether t is an instantiation of the
// predeclared generic enum named by obj (universeOption or universeResult).
func predeclaredEnumInstance(t Type, obj *TypeName) bool {
	n := asNamed(t)
	return n != nil && n.Origin().obj == obj && n.TypeArgs().Len() == 1
}

// untypedVariantTarget reports whether the untyped variant value of type t
// (UntypedNone or UntypedErr) can take on type target.
func untypedVariantTarget(t, target Type) bool {
	switch t.(*Basic).kind {
	case UntypedNone:
		return predeclaredEnumInstance(target, universeOption)
	case UntypedErr:
		return predeclaredEnumInstance(target, universeResult)
	}
	return false
}

// untypedVariantKind returns "an Option" or "a Result" for the untyped
// variant value type t, for use in error messages.
func untypedVariantKind(t Type) string {
	if t.(*Basic).kind == UntypedNone {
		return "an Option"
	}
	return "a Result"
}

// isCgoGenerated reports whether pos is in a file generated by cgo.
func isCgoGenerated(pos syntax.Pos) bool {
	base := pos.Base()
	if base == nil {
		return false
	}
	name := base.Filename()
	if i := strings.LastIndexAny(name, `/\`); i >= 0 {
		name = name[i+1:]
	}
	return strings.HasPrefix(name, "_cgo_")
}
