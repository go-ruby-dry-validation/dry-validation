// Copyright (c) the go-ruby-dry-validation/dry-validation authors
//
// SPDX-License-Identifier: BSD-3-Clause

package dryvalidation

// Contract is a dry-validation contract: a [Schema] (built with `params`/`json`)
// plus a list of custom rules that run after the schema against the coerced
// values. Build one with [NewContract], attach the schema, add rules with
// [Contract.Rule]/[Contract.RuleBase], then apply it with [Contract.Call].
//
// The rule bodies are the host / rbgo evaluation seam: the schema and its macro/
// predicate evaluation are deterministic Go here, while each rule's predicate
// logic is a Go closure (in the host, the compiled Ruby block) that inspects the
// coerced values through a [RuleContext] and calls Failure to add an error.
type Contract struct {
	schema *Schema
	rules  []*rule
}

// rule is one contract rule: the key path(s) it depends on and its body. keys is
// the list of dependency keys (a rule with no keys is a base rule). The body runs
// only when every dependency key passed schema validation.
type rule struct {
	keys []Symbol // top-level dependency keys (empty = base rule)
	// target is the path a Key.Failure attaches to (the last dependency key for a
	// keyed rule; nil for a base rule).
	target []any
	body   func(*RuleContext)
}

// NewContract creates a contract wrapping the given schema (built with [Params]
// or [JSON]).
func NewContract(schema *Schema) *Contract { return &Contract{schema: schema} }

// Rule registers a rule keyed on one or more top-level keys. The body runs only
// if every named key passed the schema; a [RuleContext.Key] failure attaches to
// the last named key (matching the gem, where `rule(:a, :b)` reports under :a's
// error path is actually the first — see below). dry-validation reports a keyed
// `key.failure` under the rule's first key.
func (c *Contract) Rule(body func(*RuleContext), keys ...Symbol) {
	var target []any
	if len(keys) > 0 {
		target = []any{keys[0]}
	}
	c.rules = append(c.rules, &rule{keys: keys, target: target, body: body})
}

// RuleBase registers a base rule (no dependency keys) whose failures attach to
// the base (nil) path — dry-validation's `rule do base.failure(...) end`. It runs
// unconditionally after the schema.
func (c *Contract) RuleBase(body func(*RuleContext)) {
	c.rules = append(c.rules, &rule{body: body})
}

// Call applies the contract: it runs the schema, then every rule whose
// dependency keys all passed the schema, collecting rule failures into the same
// errors tree. It returns the combined [Result].
func (c *Contract) Call(input any) *Result {
	out := NewMap()
	errs := newMessageNode()
	passed := map[Symbol]bool{}
	c.schema.apply(input, out, errs, passed)

	for _, r := range c.rules {
		if !r.canRun(passed) {
			continue
		}
		rc := &RuleContext{values: out, target: r.target}
		r.body(rc)
		for _, f := range rc.failures {
			errs.add(f.path, f.text)
		}
	}
	return &Result{output: out, errors: errs}
}

// canRun reports whether every dependency key of the rule passed schema
// validation (a base rule, with no keys, always runs).
func (r *rule) canRun(passed map[Symbol]bool) bool {
	for _, k := range r.keys {
		if ok, seen := passed[k]; seen && !ok {
			return false
		} else if !seen {
			// Key not declared/processed (e.g. absent optional): treat as not
			// passing so the rule does not fire on a value that isn't there.
			return false
		}
	}
	return true
}

// RuleContext is the seam a rule body operates through. It exposes the coerced
// values and records failures. In the host, rbgo runs the Ruby block and drives
// this context: reading Values and calling Key/Base .failure.
type RuleContext struct {
	values   *Map
	target   []any // default path for Key failures (the rule's first key)
	failures []ruleFailure
}

type ruleFailure struct {
	path []any
	text string
}

// Values returns the coerced output hash the rule inspects (dry-validation's
// `values`). Reading a key: use [RuleContext.Value].
func (rc *RuleContext) Values() *Map { return rc.values }

// Value returns the coerced value at a top-level key and whether it was present.
func (rc *RuleContext) Value(key Symbol) (any, bool) { return rc.values.Get(key) }

// KeyFailure adds a failure at the rule's default key path (dry-validation's
// `key.failure(text)`). For a base rule (no key) it falls back to the base path.
func (rc *RuleContext) KeyFailure(text string) {
	path := rc.target
	if path == nil {
		path = []any{nil}
	}
	rc.failures = append(rc.failures, ruleFailure{path: append([]any{}, path...), text: text})
}

// KeyFailureAt adds a failure at an explicit key path (dry-validation's
// `key([:a, :b]).failure(text)`), so a rule can report under a nested or
// different key than its dependency.
func (rc *RuleContext) KeyFailureAt(path []any, text string) {
	rc.failures = append(rc.failures, ruleFailure{path: append([]any{}, path...), text: text})
}

// BaseFailure adds a base failure under the nil path (dry-validation's
// `base.failure(text)`).
func (rc *RuleContext) BaseFailure(text string) {
	rc.failures = append(rc.failures, ruleFailure{path: []any{nil}, text: text})
}
