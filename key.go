// Copyright (c) the go-ruby-dry-validation/dry-validation authors
//
// SPDX-License-Identifier: BSD-3-Clause

package dryvalidation

// Key is the value-macro attachment point returned by [Builder.Required] /
// [Builder.Optional]. Call exactly one macro method on it (Filled, Maybe, Value,
// Array, Hash, Schema) to define how the key's value is coerced and validated.
type Key struct {
	rule *keyRule
	mode mode
}

// Filled requires the value to be present and non-empty, coerced to typeName (a
// dry-types type name like "string"/"integer", or "" for any), then satisfy the
// predicates. It is dry-schema's `filled(:type, pred?: arg)`.
func (k *Key) Filled(typeName string, preds ...Predicate) {
	k.rule.macro = &macro{kind: macroFilled, typeName: typeName, preds: preds}
}

// Maybe permits nil (nil passes through unchanged); a non-nil value is coerced to
// typeName and checked against the predicates. It is `maybe(:type, ...)`.
func (k *Key) Maybe(typeName string, preds ...Predicate) {
	k.rule.macro = &macro{kind: macroMaybe, typeName: typeName, preds: preds}
}

// Value coerces the value to typeName and checks the predicates without the
// implicit filled? of [Key.Filled]. It is `value(:type, pred?: arg)`.
func (k *Key) Value(typeName string, preds ...Predicate) {
	k.rule.macro = &macro{kind: macroValue, typeName: typeName, preds: preds}
}

// Array coerces the value to an array and coerces+validates each element as
// elemType with elemPreds (`array(:elem, pred?: arg)`). For an array of hashes,
// follow with [Key.Each] on the returned member builder; use [Key.ArrayOf] for
// the block form.
func (k *Key) Array(elemType string, elemPreds ...Predicate) {
	k.rule.macro = &macro{
		kind:      macroArray,
		typeName:  "array",
		arrayElem: &macro{kind: macroValue, typeName: elemType, preds: elemPreds},
	}
}

// ArrayOf coerces the value to an array and validates each element against a
// nested member schema built by build — dry-schema's
// `array(:hash) do ... end`. The elements are coerced to hashes and each run
// through the member schema.
func (k *Key) ArrayOf(build func(*Builder)) {
	member := newSchema(k.mode, build)
	k.rule.macro = &macro{
		kind:      macroArray,
		typeName:  "array",
		arrayElem: &macro{kind: macroHash, typeName: "hash", nested: member},
		nested:    member,
	}
}

// Hash coerces the value to a hash and applies a nested schema built by build —
// dry-schema's `hash do ... end`. A non-hash value reports "must be a hash".
func (k *Key) Hash(build func(*Builder)) {
	k.rule.macro = &macro{kind: macroHash, typeName: "hash", nested: newSchema(k.mode, build)}
}

// Schema is an alias of [Key.Hash] (`schema do ... end`): a nested schema over
// the value.
func (k *Key) Schema(build func(*Builder)) { k.Hash(build) }
