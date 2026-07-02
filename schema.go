// Copyright (c) the go-ruby-dry-validation/dry-validation authors
//
// SPDX-License-Identifier: BSD-3-Clause

package dryvalidation

import drytypes "github.com/go-ruby-dry-types/dry-types"

// Schema is a dry-schema definition: an ordered list of key rules that, applied
// to an input hash, coerces the values and produces a [Result] with the coerced
// output and a nested errors hash. Build one with [Params] or [JSON] and the
// [Key] DSL, then apply it with [Schema.Call].
type Schema struct {
	keys []*keyRule
	mode mode
}

// keyRule is one key of a schema: its name, whether it is required or optional,
// and the value macro that coerces+validates the value under it.
type keyRule struct {
	name     Symbol
	required bool
	macro    *macro
}

// macro is the value rule attached to a key. Exactly one shape is active:
//   - kind filled/maybe/value: a scalar type + predicates.
//   - kind array: an element type + element predicates (each element coerced),
//     with an optional member schema (array(:hash) do ... end).
//   - kind hash/schema: a nested [Schema] applied to the value.
type macro struct {
	kind      macroKind
	typeName  string
	preds     []Predicate
	nested    *Schema // hash / schema, and array-of-hash member schema
	arrayElem *macro  // array element macro (type + preds)
}

type macroKind int

const (
	macroValue  macroKind = iota // value(...) — type + preds, no presence coercion
	macroFilled                  // filled(...) — value + implicit filled? predicate
	macroMaybe                   // maybe(...) — nil passes through, else value(...)
	macroArray                   // array(:elem, preds) [do member end]
	macroHash                    // hash do ... end / schema do ... end
)

// Params starts a Params-namespace schema (form-string coercion — the default
// dry-validation `params do` block). Call build to add keys, then finish.
func Params(build func(*Builder)) *Schema { return newSchema(modeParams, build) }

// JSON starts a JSON-namespace schema (JSON-native coercion — `json do`).
func JSON(build func(*Builder)) *Schema { return newSchema(modeJSON, build) }

func newSchema(m mode, build func(*Builder)) *Schema {
	s := &Schema{mode: m}
	b := &Builder{schema: s}
	if build != nil {
		build(b)
	}
	return s
}

// Builder is the DSL receiver inside a [Params]/[JSON] block: it collects key
// rules via [Builder.Required] and [Builder.Optional].
type Builder struct{ schema *Schema }

// Required declares a required key. The returned [Key] attaches the value macro;
// a required key absent from the input reports "is missing".
func (b *Builder) Required(name Symbol) *Key {
	kr := &keyRule{name: name, required: true}
	b.schema.keys = append(b.schema.keys, kr)
	return &Key{rule: kr, mode: b.schema.mode}
}

// Optional declares an optional key. An absent optional key is skipped entirely
// (no error, no output entry); a present one is coerced+validated like a
// required one.
func (b *Builder) Optional(name Symbol) *Key {
	kr := &keyRule{name: name, required: false}
	b.schema.keys = append(b.schema.keys, kr)
	return &Key{rule: kr, mode: b.schema.mode}
}

// Call applies the schema to input (any hash-shaped value: *Map, map[string]any,
// map[Symbol]any). It returns a [Result] carrying the coerced output *Map and the
// errors tree. A non-hash input yields an empty output and no per-key errors —
// the caller (a nested hash macro) reports the "must be a hash" type error.
func (s *Schema) Call(input any) *Result {
	out := drytypes.NewMap()
	errs := newMessageNode()
	s.apply(input, out, errs, nil)
	return &Result{output: out, errors: errs}
}

// apply runs every key rule of s against input, writing coerced values into out
// and messages into errs under prefix. It also records, per key, whether that
// key passed schema validation (used by the contract rule seam).
func (s *Schema) apply(input any, out *Map, errs *messageNode, passed map[Symbol]bool) {
	m, ok := asMap(input)
	if !ok {
		// Non-hash input: mark every required key as failed for the rule seam,
		// but emit no messages here (the enclosing macro emits the type error).
		for _, kr := range s.keys {
			markPassed(passed, kr.name, false)
		}
		return
	}
	for _, kr := range s.keys {
		val, present := lookupKey(m, kr.name)
		if !present {
			if kr.required {
				errs.child(kr.name).addText("is missing")
				markPassed(passed, kr.name, false)
			}
			continue
		}
		coerced, keyErrs := kr.macro.run(val, s.mode)
		if coerced.set {
			out.Set(kr.name, coerced.val)
		} else {
			out.Set(kr.name, val)
		}
		if keyErrs != nil && !keyErrs.empty() {
			mergeInto(errs.child(kr.name), keyErrs)
			markPassed(passed, kr.name, false)
		} else {
			markPassed(passed, kr.name, true)
		}
	}
}

func markPassed(passed map[Symbol]bool, name Symbol, ok bool) {
	if passed != nil {
		passed[name] = ok
	}
}

// lookupKey finds a schema key in an input map trying the symbol key then its
// string form (Params/JSON inputs use string keys; a host may pass symbol keys).
func lookupKey(m *Map, name Symbol) (any, bool) {
	if v, ok := m.Get(name); ok {
		return v, true
	}
	if v, ok := m.Get(string(name)); ok {
		return v, true
	}
	return nil, false
}

// mergeInto copies src's texts and children into dst (used to attach a macro's
// error subtree under a key node).
func mergeInto(dst, src *messageNode) {
	for _, t := range src.texts {
		dst.addText(t)
	}
	for _, c := range src.children {
		mergeInto(dst.child(c.key), c.node)
	}
}

// coercedVal is a coerced value with a flag for whether coercion actually
// produced one (vs. the original being kept because coercion failed).
type coercedVal struct {
	val any
	set bool
}
