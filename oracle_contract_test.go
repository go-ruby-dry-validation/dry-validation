// Copyright (c) the go-ruby-dry-validation/dry-validation authors
//
// SPDX-License-Identifier: BSD-3-Clause

package dryvalidation

import "testing"

// contractCase pairs a Go contract with the equivalent Ruby params + rules and an
// input, exercising the schema-then-rules pipeline and the rule-run gating.
type contractCase struct {
	name       string
	ns         string // "params" / "json"
	build      func(*Contract)
	rubyParams string
	rubyRules  string
	input      any
}

func TestOracleContract(t *testing.T) {
	if envSkipDefault() {
		t.Skip("DRYV_NO_ORACLE set")
	}
	bin := rubyBin(t)
	for _, c := range oracleContractCases() {
		t.Run(c.name, func(t *testing.T) {
			ct := buildContract(c)
			got := goSchemaOutcome(ct.Call(c.input))
			ns := c.ns
			if ns == "" {
				ns = "params"
			}
			want := rubyContractOutcome(t, bin, ns, c.rubyParams, c.rubyRules, c.input)
			if got != want {
				t.Errorf("case %s\n  go:   %q\n  ruby: %q", c.name, got, want)
			}
		})
	}
}

func buildContract(c contractCase) *Contract {
	ns := c.ns
	var schema *Schema
	// The schema builder is embedded in build via a closure convention: build
	// receives the contract, whose schema it must set first. We standardize by
	// having each case's build call ct.setSchema then add rules.
	_ = ns
	ct := &Contract{}
	c.build(ct)
	schema = ct.schema
	if schema == nil {
		panic("case " + c.name + " did not set a schema")
	}
	return ct
}

// setSchema attaches the schema a contract case builds (used by the oracle
// corpus; the public API is [NewContract]).
func (c *Contract) setSchema(s *Schema) { c.schema = s }

func oracleContractCases() []contractCase {
	return []contractCase{
		{
			name: "rule_ok",
			build: func(c *Contract) {
				c.setSchema(Params(func(b *Builder) {
					b.Required("email").Filled("string")
					b.Required("age").Filled("integer")
				}))
				c.Rule(func(rc *RuleContext) {
					if v, ok := rc.Value("age"); ok && lessThan(v, 18) {
						rc.KeyFailure("must be at least 18")
					}
				}, "age")
			},
			rubyParams: "required(:email).filled(:string)\nrequired(:age).filled(:integer)",
			rubyRules:  "rule(:age) do\nkey.failure(\"must be at least 18\") if value < 18\nend",
			input:      map[string]any{"email": "a@b", "age": "25"},
		},
		{
			name: "rule_fail",
			build: func(c *Contract) {
				c.setSchema(Params(func(b *Builder) {
					b.Required("email").Filled("string")
					b.Required("age").Filled("integer")
				}))
				c.Rule(func(rc *RuleContext) {
					if v, ok := rc.Value("age"); ok && lessThan(v, 18) {
						rc.KeyFailure("must be at least 18")
					}
				}, "age")
			},
			rubyParams: "required(:email).filled(:string)\nrequired(:age).filled(:integer)",
			rubyRules:  "rule(:age) do\nkey.failure(\"must be at least 18\") if value < 18\nend",
			input:      map[string]any{"email": "a@b", "age": "10"},
		},
		{
			name: "rule_runs_when_other_key_failed",
			build: func(c *Contract) {
				c.setSchema(Params(func(b *Builder) {
					b.Required("email").Filled("string")
					b.Required("age").Filled("integer")
				}))
				c.Rule(func(rc *RuleContext) {
					if v, ok := rc.Value("age"); ok && lessThan(v, 18) {
						rc.KeyFailure("must be at least 18")
					}
				}, "age")
			},
			rubyParams: "required(:email).filled(:string)\nrequired(:age).filled(:integer)",
			rubyRules:  "rule(:age) do\nkey.failure(\"must be at least 18\") if value < 18\nend",
			input:      map[string]any{"age": "10"},
		},
		{
			name: "rule_skipped_own_key_failed",
			build: func(c *Contract) {
				c.setSchema(Params(func(b *Builder) {
					b.Required("age").Filled("integer")
				}))
				c.Rule(func(rc *RuleContext) {
					rc.KeyFailure("custom age fail")
				}, "age")
			},
			rubyParams: "required(:age).filled(:integer)",
			rubyRules:  "rule(:age) do\nkey.failure(\"custom age fail\")\nend",
			input:      map[string]any{"age": "abc"},
		},
		{
			name: "multi_key_rule_gated",
			build: func(c *Contract) {
				c.setSchema(Params(func(b *Builder) {
					b.Required("a").Filled("integer")
					b.Required("b").Filled("integer")
				}))
				c.Rule(func(rc *RuleContext) {
					rc.KeyFailure("multi")
				}, "a", "b")
			},
			rubyParams: "required(:a).filled(:integer)\nrequired(:b).filled(:integer)",
			rubyRules:  "rule(:a, :b) do\nkey.failure(\"multi\")\nend",
			input:      map[string]any{"a": "1", "b": "abc"},
		},
		{
			name: "multi_key_rule_runs",
			build: func(c *Contract) {
				c.setSchema(Params(func(b *Builder) {
					b.Required("a").Filled("integer")
					b.Required("b").Filled("integer")
				}))
				c.Rule(func(rc *RuleContext) {
					rc.KeyFailure("multi")
				}, "a", "b")
			},
			rubyParams: "required(:a).filled(:integer)\nrequired(:b).filled(:integer)",
			rubyRules:  "rule(:a, :b) do\nkey.failure(\"multi\")\nend",
			input:      map[string]any{"a": "1", "b": "2"},
		},
		{
			name: "base_failure",
			build: func(c *Contract) {
				c.setSchema(Params(func(b *Builder) {
					b.Required("a").Filled("integer")
					b.Required("b").Filled("integer")
				}))
				c.RuleBase(func(rc *RuleContext) {
					av, _ := rc.Value("a")
					bv, _ := rc.Value("b")
					if ai, ok := toI(av); ok {
						if bi, ok2 := toI(bv); ok2 && ai+bi > 10 {
							rc.BaseFailure("a+b too big")
						}
					}
				})
			},
			rubyParams: "required(:a).filled(:integer)\nrequired(:b).filled(:integer)",
			rubyRules:  "rule do\nbase.failure(\"a+b too big\") if values[:a] + values[:b] > 10\nend",
			input:      map[string]any{"a": "6", "b": "7"},
		},
		{
			name: "nested_rule_at_path",
			build: func(c *Contract) {
				c.setSchema(Params(func(b *Builder) {
					b.Required("range").Hash(func(b *Builder) {
						b.Required("min").Filled("integer")
						b.Required("max").Filled("integer")
					})
				}))
				c.Rule(func(rc *RuleContext) {
					rng, _ := rc.Value("range")
					if m, ok := rng.(*Map); ok {
						mn, _ := m.Get(Symbol("min"))
						mx, _ := m.Get(Symbol("max"))
						if mi, ok := toI(mn); ok {
							if xi, ok2 := toI(mx); ok2 && xi < mi {
								rc.KeyFailureAt([]any{Symbol("range"), Symbol("max")}, "must be greater than min")
							}
						}
					}
				}, "range")
			},
			rubyParams: "required(:range).hash do\nrequired(:min).filled(:integer)\nrequired(:max).filled(:integer)\nend",
			rubyRules:  "rule(\"range.max\", \"range.min\") do\nkey([:range, :max]).failure(\"must be greater than min\") if values[:range][:max] < values[:range][:min]\nend",
			input:      map[string]any{"range": map[string]any{"min": "5", "max": "2"}},
		},
	}
}

func lessThan(v any, n int) bool { i, ok := toI(v); return ok && i < n }

func toI(v any) (int, bool) {
	switch x := v.(type) {
	case int:
		return x, true
	case int64:
		return int(x), true
	}
	return 0, false
}
