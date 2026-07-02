// Copyright (c) the go-ruby-dry-validation/dry-validation authors
//
// SPDX-License-Identifier: BSD-3-Clause

package dryvalidation

// oracleSchemaCases is the differential corpus: each pairs a Go schema builder
// with the equivalent Ruby DSL body and an input, exercising filled/maybe/value/
// array/hash/nested + every predicate + coercion across Params and JSON.
func oracleSchemaCases() []oracleCase {
	P := func(name string, build func(*Builder), dsl string, input any) oracleCase {
		return oracleCase{name: name, ns: "Params", build: build, rubyDSL: dsl, input: input}
	}
	J := func(name string, build func(*Builder), dsl string, input any) oracleCase {
		return oracleCase{name: name, ns: "JSON", build: build, rubyDSL: dsl, input: input}
	}
	return []oracleCase{
		P("filled_valid",
			func(b *Builder) { b.Required("email").Filled("string") },
			`required(:email).filled(:string)`,
			map[string]any{"email": "a@b.com"}),
		P("filled_missing",
			func(b *Builder) { b.Required("email").Filled("string") },
			`required(:email).filled(:string)`,
			map[string]any{}),
		P("filled_empty",
			func(b *Builder) { b.Required("email").Filled("string") },
			`required(:email).filled(:string)`,
			map[string]any{"email": ""}),
		P("maybe_nil",
			func(b *Builder) { b.Optional("age").Maybe("integer") },
			`optional(:age).maybe(:integer)`,
			map[string]any{"age": nil}),
		P("maybe_coerced",
			func(b *Builder) { b.Optional("age").Maybe("integer") },
			`optional(:age).maybe(:integer)`,
			map[string]any{"age": "30"}),
		P("maybe_bad",
			func(b *Builder) { b.Optional("age").Maybe("integer") },
			`optional(:age).maybe(:integer)`,
			map[string]any{"age": "abc"}),
		P("optional_absent",
			func(b *Builder) {
				b.Optional("x").Filled("string")
				b.Required("y").Filled("string")
			},
			"optional(:x).filled(:string)\nrequired(:y).filled(:string)",
			map[string]any{"y": "v"}),
		P("value_int_gt",
			func(b *Builder) { b.Required("age").Filled("integer", Predicate{"gt", 18}) },
			`required(:age).filled(:integer, gt?: 18)`,
			map[string]any{"age": "10"}),
		P("value_int_gt_ok",
			func(b *Builder) { b.Required("age").Filled("integer", Predicate{"gt", 18}) },
			`required(:age).filled(:integer, gt?: 18)`,
			map[string]any{"age": "20"}),
		P("value_int_gteq",
			func(b *Builder) { b.Required("f").Value("integer", Predicate{"gteq", 5}) },
			`required(:f).value(:integer, gteq?: 5)`,
			map[string]any{"f": "2"}),
		P("value_int_lt",
			func(b *Builder) { b.Required("f").Value("integer", Predicate{"lt", 5}) },
			`required(:f).value(:integer, lt?: 5)`,
			map[string]any{"f": "9"}),
		P("value_int_lteq",
			func(b *Builder) { b.Required("f").Value("integer", Predicate{"lteq", 5}) },
			`required(:f).value(:integer, lteq?: 5)`,
			map[string]any{"f": "9"}),
		P("format_fail",
			func(b *Builder) { b.Required("f").Value("string", Predicate{"format", `^\d+$`}) },
			`required(:f).value(:string, format?: /^\d+$/)`,
			map[string]any{"f": "ab"}),
		P("format_ok",
			func(b *Builder) { b.Required("f").Value("string", Predicate{"format", `^\d+$`}) },
			`required(:f).value(:string, format?: /^\d+$/)`,
			map[string]any{"f": "123"}),
		P("included_str",
			func(b *Builder) {
				b.Required("f").Value("string", Predicate{"included_in", []any{"a", "b", "c"}})
			},
			`required(:f).value(:string, included_in?: %w[a b c])`,
			map[string]any{"f": "z"}),
		P("included_int",
			func(b *Builder) {
				b.Required("f").Value("integer", Predicate{"included_in", []any{1, 2, 3}})
			},
			`required(:f).value(:integer, included_in?: [1,2,3])`,
			map[string]any{"f": "9"}),
		P("excluded",
			func(b *Builder) {
				b.Required("f").Value("integer", Predicate{"excluded_from", []any{1, 2}})
			},
			`required(:f).value(:integer, excluded_from?: [1,2])`,
			map[string]any{"f": "1"}),
		P("size_exact",
			func(b *Builder) { b.Required("f").Value("string", Predicate{"size", 3}) },
			`required(:f).value(:string, size?: 3)`,
			map[string]any{"f": "ab"}),
		P("min_size",
			func(b *Builder) { b.Required("f").Value("string", Predicate{"min_size", 3}) },
			`required(:f).value(:string, min_size?: 3)`,
			map[string]any{"f": "ab"}),
		P("max_size",
			func(b *Builder) { b.Required("f").Value("string", Predicate{"max_size", 3}) },
			`required(:f).value(:string, max_size?: 3)`,
			map[string]any{"f": "abcd"}),
		P("bool_fail",
			func(b *Builder) { b.Required("f").Filled("bool") },
			`required(:f).filled(:bool)`,
			map[string]any{"f": "notbool"}),
		P("bool_ok",
			func(b *Builder) { b.Required("f").Filled("bool") },
			`required(:f).filled(:bool)`,
			map[string]any{"f": "true"}),
		P("float_ok",
			func(b *Builder) { b.Required("f").Filled("float") },
			`required(:f).filled(:float)`,
			map[string]any{"f": "3.5"}),
		P("float_fail",
			func(b *Builder) { b.Required("f").Filled("float") },
			`required(:f).filled(:float)`,
			map[string]any{"f": "x"}),
		P("nested_valid",
			func(b *Builder) {
				b.Required("address").Hash(func(b *Builder) {
					b.Required("city").Filled("string")
					b.Required("zip").Filled("string")
				})
			},
			"required(:address).hash do\nrequired(:city).filled(:string)\nrequired(:zip).filled(:string)\nend",
			map[string]any{"address": map[string]any{"city": "Paris", "zip": "75001"}}),
		P("nested_missing_key",
			func(b *Builder) {
				b.Required("address").Hash(func(b *Builder) {
					b.Required("city").Filled("string")
					b.Required("zip").Filled("string")
				})
			},
			"required(:address).hash do\nrequired(:city).filled(:string)\nrequired(:zip).filled(:string)\nend",
			map[string]any{"address": map[string]any{"city": "Paris"}}),
		P("nested_not_hash",
			func(b *Builder) {
				b.Required("address").Hash(func(b *Builder) {
					b.Required("city").Filled("string")
				})
			},
			"required(:address).hash do\nrequired(:city).filled(:string)\nend",
			map[string]any{"address": "nope"}),
		P("nested_missing_whole",
			func(b *Builder) {
				b.Required("address").Hash(func(b *Builder) {
					b.Required("city").Filled("string")
				})
			},
			"required(:address).hash do\nrequired(:city).filled(:string)\nend",
			map[string]any{}),
		P("array_valid",
			func(b *Builder) { b.Required("tags").Array("string") },
			`required(:tags).array(:string)`,
			map[string]any{"tags": []any{"a", "b"}}),
		P("array_bad_elem",
			func(b *Builder) { b.Required("tags").Array("string") },
			`required(:tags).array(:string)`,
			map[string]any{"tags": []any{"a", 1}}),
		P("array_not_array",
			func(b *Builder) { b.Required("tags").Array("string") },
			`required(:tags).array(:string)`,
			map[string]any{"tags": "x"}),
		P("array_pred_fail",
			func(b *Builder) { b.Required("nums").Array("integer", Predicate{"gt", 0}) },
			`required(:nums).array(:integer, gt?: 0)`,
			map[string]any{"nums": []any{"1", "-5"}}),
		P("array_of_hash",
			func(b *Builder) {
				b.Required("people").ArrayOf(func(b *Builder) {
					b.Required("name").Filled("string")
				})
			},
			"required(:people).array(:hash) do\nrequired(:name).filled(:string)\nend",
			map[string]any{"people": []any{
				map[string]any{"name": "a"},
				map[string]any{"name": ""},
			}}),
		P("extra_keys_dropped",
			func(b *Builder) { b.Required("a").Filled("string") },
			`required(:a).filled(:string)`,
			map[string]any{"a": "x", "b": "y"}),
		J("json_int_native",
			func(b *Builder) { b.Required("age").Filled("integer") },
			`required(:age).filled(:integer)`,
			map[string]any{"age": 5}),
		J("json_string_not_coerced",
			func(b *Builder) { b.Required("age").Filled("integer") },
			`required(:age).filled(:integer)`,
			map[string]any{"age": "5"}),
		J("json_string_valid",
			func(b *Builder) { b.Required("name").Filled("string", Predicate{"min_size", 2}) },
			`required(:name).filled(:string, min_size?: 2)`,
			map[string]any{"name": "bob"}),
		P("multi_pred_first_reported",
			func(b *Builder) {
				b.Required("f").Filled("string", Predicate{"min_size", 3}, Predicate{"format", `^\d+$`})
			},
			`required(:f).filled(:string, min_size?: 3, format?: /^\d+$/)`,
			map[string]any{"f": "a"}),
	}
}
