// Copyright (c) the go-ruby-dry-validation/dry-validation authors
//
// SPDX-License-Identifier: BSD-3-Clause

package dryvalidation

// Result is the outcome of applying a [Schema] or a [Contract]: the coerced
// output hash plus the errors tree. It mirrors Dry::Schema::Result /
// Dry::Validation::Result (`.to_h`, `.errors`, `.success?`).
type Result struct {
	output *Map
	errors *messageNode
}

// ToH returns the coerced output hash (dry-schema's Result#to_h). Keys are the
// schema keys that were present in the input, in schema-declaration order, with
// their coerced values.
func (r *Result) ToH() *Map { return r.output }

// Output is an alias of [Result.ToH].
func (r *Result) Output() *Map { return r.output }

// Success reports whether validation produced no errors (Result#success?).
func (r *Result) Success() bool { return r.errors.empty() }

// Errors returns the nested errors hash the way Dry::Schema's
// `result.errors.to_h` renders it: a *Map whose leaves are []any of message
// strings, hash keys are [Symbol], and array-element keys are int. An empty
// result yields an empty *Map.
func (r *Result) Errors() *Map {
	h := r.errors.toH()
	if m, ok := h.(*Map); ok {
		return m
	}
	return NewMap()
}

// Messages returns the flat list of every failure with its full key path, in the
// order dry-validation's `result.errors.each` yields them.
func (r *Result) Messages() []Message { return r.errors.flatten(nil) }
