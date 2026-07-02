// Copyright (c) the go-ruby-dry-validation/dry-validation authors
//
// SPDX-License-Identifier: BSD-3-Clause

// Package dryvalidation is a pure-Go (CGO-free) MRI-faithful reimplementation of
// Ruby's dry-schema and dry-validation gems: a schema/contract DSL that coerces
// and validates a hash of input against key-presence rules, value macros
// (filled/maybe/value/each/array/hash/schema), dry-types coercion and dry-logic
// predicate constraints, then reports a nested errors hash whose messages are
// byte-identical to the gems' default English (en) locale.
//
// The type layer under every schema key is the sibling package
// github.com/go-ruby-dry-types/dry-types: a schema key's coercion+constraint is a
// dry-types [drytypes.Type], and the schema composes those into a whole-hash
// validator.
//
// # Ruby value model
//
// Inputs and coerced outputs use the same small, fixed set of Go types the
// go-ruby-* ecosystem shares (see [github.com/go-ruby-dry-types/dry-types]), so a
// host (go-embedded-ruby / rbgo) maps its object graph with no glue:
//
//	Ruby            Go
//	----            --
//	nil             nil
//	true / false    bool
//	Integer         int64, *big.Int
//	Float           float64
//	String          string
//	Symbol          Symbol
//	Array           []any
//	Hash            *Map (ordered)
//	Date            Date
//	Time / DateTime Time
//
// # Rule seam
//
// The custom-`rule` predicate bodies of a dry-validation Contract are the host /
// rbgo evaluation seam: rbgo runs the Ruby block, and this package exposes the
// coerced values, a schema-success view, and a [RuleContext] whose Key/Base
// .failure methods add errors. The schema and its built-in macro/predicate
// evaluation are the deterministic core here.
package dryvalidation

import (
	drytypes "github.com/go-ruby-dry-types/dry-types"
)

// Symbol is a Ruby Symbol (`:name`), re-exported from dry-types so keys and
// values share one representation across the stack.
type Symbol = drytypes.Symbol

// Map is the insertion-ordered Ruby Hash re-exported from dry-types; schema
// coercion yields a *Map so key order round-trips.
type Map = drytypes.Map

// Pair is one entry of an ordered [Map].
type Pair = drytypes.Pair

// Date is a Ruby Date, re-exported from dry-types (the coercion target of
// :date-typed keys).
type Date = drytypes.Date

// NewMap returns an empty ordered [Map].
func NewMap() *Map { return drytypes.NewMap() }
