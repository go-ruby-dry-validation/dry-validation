// Copyright (c) the go-ruby-dry-validation/dry-validation authors
//
// SPDX-License-Identifier: BSD-3-Clause

package dryvalidation

import (
	"math/big"
	"regexp"
	"testing"
	"time"
)

// eq renders a Result and compares its "to_h.inspect" + "errors.to_h.inspect"
// against the golden strings captured from the gems (see oracle_test.go for how
// those strings were produced). These deterministic goldens alone hold coverage
// at 100% so the no-ruby / qemu / Windows lanes pass the gate.
func eq(t *testing.T, r *Result, wantOut, wantErr string) {
	t.Helper()
	if g := rubyInspectMap(r.ToH()); g != wantOut {
		t.Errorf("to_h\n  got  %s\n  want %s", g, wantOut)
	}
	if g := rubyInspectMap(r.Errors()); g != wantErr {
		t.Errorf("errors\n  got  %s\n  want %s", g, wantErr)
	}
}

func TestGoldenFilledMaybeValue(t *testing.T) {
	s := Params(func(b *Builder) {
		b.Required("email").Filled("string")
		b.Optional("age").Maybe("integer")
	})
	eq(t, s.Call(map[string]any{"email": "a@b.com", "age": "30"}),
		`{email: "a@b.com", age: 30}`, `{}`)
	eq(t, s.Call(map[string]any{}), `{}`, `{email: ["is missing"]}`)
	eq(t, s.Call(map[string]any{"email": ""}), `{email: ""}`, `{email: ["must be filled"]}`)
	eq(t, s.Call(map[string]any{"email": "x", "age": nil}), `{email: "x", age: nil}`, `{}`)
	eq(t, s.Call(map[string]any{"email": "x", "age": "abc"}),
		`{email: "x", age: "abc"}`, `{age: ["must be an integer"]}`)

	// value(...) without filled: an empty string coerces fine and passes.
	s2 := Params(func(b *Builder) { b.Required("f").Value("string") })
	eq(t, s2.Call(map[string]any{"f": ""}), `{f: ""}`, `{}`)
	// non-string stays and reports the strict type error.
	eq(t, s2.Call(map[string]any{"f": 5}), `{f: 5}`, `{f: ["must be a string"]}`)
	// optional absent is skipped entirely.
	s3 := Params(func(b *Builder) {
		b.Optional("x").Filled("string")
		b.Required("y").Filled("string")
	})
	eq(t, s3.Call(map[string]any{"y": "v"}), `{y: "v"}`, `{}`)
}

func TestGoldenPredicates(t *testing.T) {
	cases := []struct {
		name    string
		pred    Predicate
		typ     string
		val     any
		wantOut string
		wantErr string
	}{
		{"gt", Predicate{"gt", 18}, "integer", "10", `{f: 10}`, `{f: ["must be greater than 18"]}`},
		{"gteq", Predicate{"gteq", 5}, "integer", "2", `{f: 2}`, `{f: ["must be greater than or equal to 5"]}`},
		{"lt", Predicate{"lt", 5}, "integer", "9", `{f: 9}`, `{f: ["must be less than 5"]}`},
		{"lteq", Predicate{"lteq", 5}, "integer", "9", `{f: 9}`, `{f: ["must be less than or equal to 5"]}`},
		{"format", Predicate{"format", `^\d+$`}, "string", "ab", `{f: "ab"}`, `{f: ["is in invalid format"]}`},
		{"included_str", Predicate{"included_in", []any{"a", "b", "c"}}, "string", "z", `{f: "z"}`, `{f: ["must be one of: a, b, c"]}`},
		{"included_int", Predicate{"included_in", []any{1, 2, 3}}, "integer", "9", `{f: 9}`, `{f: ["must be one of: 1, 2, 3"]}`},
		{"excluded", Predicate{"excluded_from", []any{1, 2}}, "integer", "1", `{f: 1}`, `{f: ["must not be one of: 1, 2"]}`},
		{"size", Predicate{"size", 3}, "string", "ab", `{f: "ab"}`, `{f: ["length must be 3"]}`},
		{"min_size", Predicate{"min_size", 2}, "string", "b", `{f: "b"}`, `{f: ["size cannot be less than 2"]}`},
		{"max_size", Predicate{"max_size", 3}, "string", "abcd", `{f: "abcd"}`, `{f: ["size cannot be greater than 3"]}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := Params(func(b *Builder) { b.Required("f").Value(c.typ, c.pred) })
			eq(t, s.Call(map[string]any{"f": c.val}), c.wantOut, c.wantErr)
		})
	}
	// passing predicate cases (the true branch of check).
	ok := Params(func(b *Builder) {
		b.Required("a").Value("integer", Predicate{"gt", 1})
		b.Required("b").Value("integer", Predicate{"gteq", 1})
		b.Required("c").Value("integer", Predicate{"lt", 9})
		b.Required("d").Value("integer", Predicate{"lteq", 9})
		b.Required("e").Value("string", Predicate{"format", `^\d+$`})
		b.Required("f").Value("string", Predicate{"included_in", []any{"x"}})
		b.Required("g").Value("integer", Predicate{"excluded_from", []any{9}})
		b.Required("h").Value("string", Predicate{"size", 1})
		b.Required("i").Value("string", Predicate{"min_size", 1})
		b.Required("j").Value("string", Predicate{"max_size", 1})
	})
	in := map[string]any{"a": "2", "b": "1", "c": "1", "d": "9", "e": "5",
		"f": "x", "g": "1", "h": "z", "i": "z", "j": "z"}
	if !ok.Call(in).Success() {
		t.Fatalf("expected all preds to pass: %s", rubyInspectMap(ok.Call(in).Errors()))
	}
}

func TestGoldenPredicateExtras(t *testing.T) {
	// filled? / empty? / eql / not_eql / true / false / odd / even predicates
	// (not reachable through the standard macro type paths but part of the
	// dry-logic set the predicate engine supports).
	type pc struct {
		p    Predicate
		v    any
		want bool
	}
	for _, c := range []pc{
		{Predicate{"filled", nil}, "x", true},
		{Predicate{"filled", nil}, "", false},
		{Predicate{"empty", nil}, "", true},
		{Predicate{"empty", nil}, "x", false},
		{Predicate{"eql", 5}, int64(5), true},
		{Predicate{"eql", "s"}, "s", true},
		{Predicate{"eql", 5}, int64(6), false},
		{Predicate{"not_eql", 5}, int64(6), true},
		{Predicate{"not_eql", 5}, int64(5), false},
		{Predicate{"true", nil}, true, true},
		{Predicate{"true", nil}, false, false},
		{Predicate{"false", nil}, false, true},
		{Predicate{"false", nil}, true, false},
		{Predicate{"odd", nil}, int64(3), true},
		{Predicate{"odd", nil}, int64(4), false},
		{Predicate{"even", nil}, int64(4), true},
		{Predicate{"even", nil}, int64(3), false},
		{Predicate{"unknown_pred", nil}, "x", true},
	} {
		if got, _ := c.p.check(c.v); got != c.want {
			t.Errorf("%s(%v) = %v, want %v", c.p.Name, c.v, got, c.want)
		}
	}
	// message strings for the extras.
	checkMsg(t, Predicate{"eql", "s"}, int64(1), `must be equal to "s"`)
	checkMsg(t, Predicate{"not_eql", 5}, int64(5), "cannot be equal to 5")
	checkMsg(t, Predicate{"true", nil}, false, "must be true")
	checkMsg(t, Predicate{"false", nil}, true, "must be false")
	checkMsg(t, Predicate{"odd", nil}, int64(2), "must be odd")
	checkMsg(t, Predicate{"even", nil}, int64(3), "must be even")
	checkMsg(t, Predicate{"filled", nil}, "", "must be filled")
	checkMsg(t, Predicate{"empty", nil}, "x", "cannot be defined")
}

func checkMsg(t *testing.T, p Predicate, v any, want string) {
	t.Helper()
	if _, msg := p.check(v); msg != want {
		t.Errorf("msg %s(%v) = %q, want %q", p.Name, v, msg, want)
	}
}

func TestGoldenNested(t *testing.T) {
	s := Params(func(b *Builder) {
		b.Required("address").Hash(func(b *Builder) {
			b.Required("city").Filled("string")
			b.Required("zip").Filled("string")
		})
	})
	eq(t, s.Call(map[string]any{"address": map[string]any{"city": "Paris", "zip": "75001"}}),
		`{address: {city: "Paris", zip: "75001"}}`, `{}`)
	eq(t, s.Call(map[string]any{"address": map[string]any{"city": "Paris"}}),
		`{address: {city: "Paris"}}`, `{address: {zip: ["is missing"]}}`)
	eq(t, s.Call(map[string]any{"address": "nope"}),
		`{address: "nope"}`, `{address: ["must be a hash"]}`)
	eq(t, s.Call(map[string]any{}), `{}`, `{address: ["is missing"]}`)

	// schema{} alias of hash{}
	s2 := Params(func(b *Builder) {
		b.Required("a").Schema(func(b *Builder) { b.Required("b").Filled("string") })
	})
	eq(t, s2.Call(map[string]any{"a": map[string]any{"b": "x"}}), `{a: {b: "x"}}`, `{}`)
}

func TestGoldenArray(t *testing.T) {
	s := Params(func(b *Builder) { b.Required("tags").Array("string") })
	eq(t, s.Call(map[string]any{"tags": []any{"a", "b"}}), `{tags: ["a", "b"]}`, `{}`)
	eq(t, s.Call(map[string]any{"tags": []any{"a", 1}}),
		`{tags: ["a", 1]}`, `{tags: {1 => ["must be a string"]}}`)
	eq(t, s.Call(map[string]any{"tags": "x"}), `{tags: "x"}`, `{tags: ["must be an array"]}`)

	s2 := Params(func(b *Builder) { b.Required("nums").Array("integer", Predicate{"gt", 0}) })
	eq(t, s2.Call(map[string]any{"nums": []any{"1", "2"}}), `{nums: [1, 2]}`, `{}`)
	eq(t, s2.Call(map[string]any{"nums": []any{"1", "-5"}}),
		`{nums: [1, -5]}`, `{nums: {1 => ["must be greater than 0"]}}`)

	s3 := Params(func(b *Builder) {
		b.Required("people").ArrayOf(func(b *Builder) { b.Required("name").Filled("string") })
	})
	eq(t, s3.Call(map[string]any{"people": []any{
		map[string]any{"name": "a"}, map[string]any{"name": ""},
	}}), `{people: [{name: "a"}, {name: ""}]}`, `{people: {1 => {name: ["must be filled"]}}}`)
	// array-of-hash with a non-hash element.
	eq(t, s3.Call(map[string]any{"people": []any{"nope"}}),
		`{people: ["nope"]}`, `{people: {0 => ["must be a hash"]}}`)
}

func TestGoldenTypesAndJSON(t *testing.T) {
	// bool / float / date / symbol / time / nil coercion + type messages.
	sb := Params(func(b *Builder) { b.Required("f").Filled("bool") })
	eq(t, sb.Call(map[string]any{"f": "true"}), `{f: true}`, `{}`)
	eq(t, sb.Call(map[string]any{"f": "notbool"}), `{f: "notbool"}`, `{f: ["must be boolean"]}`)

	sf := Params(func(b *Builder) { b.Required("f").Filled("float") })
	eq(t, sf.Call(map[string]any{"f": "3.5"}), `{f: 3.5}`, `{}`)
	eq(t, sf.Call(map[string]any{"f": "x"}), `{f: "x"}`, `{f: ["must be a float"]}`)

	// JSON namespace: integers native, strings not coerced.
	j := JSON(func(b *Builder) { b.Required("age").Filled("integer") })
	eq(t, j.Call(map[string]any{"age": int64(5)}), `{age: 5}`, `{}`)
	eq(t, j.Call(map[string]any{"age": "5"}), `{age: "5"}`, `{age: ["must be an integer"]}`)

	// unknown type name → passthrough, unknown type message.
	if got := typeMessage("weird"); got != "is in invalid type" {
		t.Errorf("typeMessage(weird)=%q", got)
	}
	if resolveType("weird", modeParams).typeMsg != "is in invalid type" {
		t.Error("resolveType weird")
	}
	// all remaining type-name message + coercion branches.
	for _, n := range []string{"symbol", "date", "time", "datetime", "date_time", "nil", "array", "hash"} {
		_ = paramsType(n)
		_ = jsonType(n)
		_ = typeMessage(n)
	}
	// passthrough type coerces anything through.
	if out, err := passthroughType().Call(42); err != nil || out != 42 {
		t.Errorf("passthrough=%v,%v", out, err)
	}
}

func TestGoldenContract(t *testing.T) {
	c := NewContract(Params(func(b *Builder) {
		b.Required("email").Filled("string")
		b.Required("age").Filled("integer")
	}))
	c.Rule(func(rc *RuleContext) {
		if v, ok := rc.Value("age"); ok {
			if i, ok := v.(int64); ok && i < 18 {
				rc.KeyFailure("must be at least 18")
			}
		}
	}, "age")

	eq(t, c.Call(map[string]any{"email": "a@b", "age": "25"}),
		`{email: "a@b", age: 25}`, `{}`)
	eq(t, c.Call(map[string]any{"email": "a@b", "age": "10"}),
		`{email: "a@b", age: 10}`, `{age: ["must be at least 18"]}`)
	// rule runs even though a different key (email) failed schema.
	eq(t, c.Call(map[string]any{"age": "10"}),
		`{age: 10}`, `{email: ["is missing"], age: ["must be at least 18"]}`)

	// rule skipped when its own key failed schema.
	c2 := NewContract(Params(func(b *Builder) { b.Required("age").Filled("integer") }))
	c2.Rule(func(rc *RuleContext) { rc.KeyFailure("custom age fail") }, "age")
	eq(t, c2.Call(map[string]any{"age": "abc"}), `{age: "abc"}`, `{age: ["must be an integer"]}`)
	eq(t, c2.Call(map[string]any{}), `{}`, `{age: ["is missing"]}`)

	// base rule.
	c3 := NewContract(Params(func(b *Builder) {
		b.Required("a").Filled("integer")
		b.Required("b").Filled("integer")
	}))
	c3.RuleBase(func(rc *RuleContext) {
		av, _ := rc.Value("a")
		bv, _ := rc.Value("b")
		if a, ok := av.(int64); ok {
			if bb, ok2 := bv.(int64); ok2 && a+bb > 10 {
				rc.BaseFailure("a+b too big")
			}
		}
	})
	eq(t, c3.Call(map[string]any{"a": "6", "b": "7"}),
		`{a: 6, b: 7}`, `{nil => ["a+b too big"]}`)

	// nested-path rule failure via KeyFailureAt.
	c4 := NewContract(Params(func(b *Builder) {
		b.Required("range").Hash(func(b *Builder) {
			b.Required("min").Filled("integer")
			b.Required("max").Filled("integer")
		})
	}))
	c4.Rule(func(rc *RuleContext) {
		rng, _ := rc.Value("range")
		m := rng.(*Map)
		mn, _ := m.Get(Symbol("min"))
		mx, _ := m.Get(Symbol("max"))
		if mx.(int64) < mn.(int64) {
			rc.KeyFailureAt([]any{Symbol("range"), Symbol("max")}, "must be greater than min")
		}
	}, "range")
	eq(t, c4.Call(map[string]any{"range": map[string]any{"min": "5", "max": "2"}}),
		`{range: {min: 5, max: 2}}`, `{range: {max: ["must be greater than min"]}}`)

	// multi-key rule gating: one key fails schema → rule skipped.
	c5 := NewContract(Params(func(b *Builder) {
		b.Required("a").Filled("integer")
		b.Required("b").Filled("integer")
	}))
	c5.Rule(func(rc *RuleContext) { rc.KeyFailure("multi") }, "a", "b")
	eq(t, c5.Call(map[string]any{"a": "1", "b": "abc"}),
		`{a: 1, b: "abc"}`, `{b: ["must be an integer"]}`)
	eq(t, c5.Call(map[string]any{"a": "1", "b": "2"}),
		`{a: 1, b: 2}`, `{a: ["multi"]}`)

	// KeyFailure on a base rule (no key) falls back to base path.
	c6 := NewContract(Params(func(b *Builder) { b.Required("a").Filled("integer") }))
	c6.RuleBase(func(rc *RuleContext) { rc.KeyFailure("base via key") })
	eq(t, c6.Call(map[string]any{"a": "1"}), `{a: 1}`, `{nil => ["base via key"]}`)

	// rule referencing an absent optional key does not fire.
	c7 := NewContract(Params(func(b *Builder) { b.Optional("x").Filled("integer") }))
	c7.Rule(func(rc *RuleContext) { rc.KeyFailure("should not run") }, "x")
	eq(t, c7.Call(map[string]any{}), `{}`, `{}`)

	// contract over non-hash input: schema marks keys failed, rules gated off.
	c8 := NewContract(Params(func(b *Builder) { b.Required("a").Filled("integer") }))
	c8.Rule(func(rc *RuleContext) { rc.KeyFailure("nope") }, "a")
	eq(t, c8.Call("not-a-hash"), `{}`, `{}`)
}

func TestGoldenResultAccessors(t *testing.T) {
	s := Params(func(b *Builder) {
		b.Required("a").Filled("integer", Predicate{"gt", 100})
		b.Required("b").Hash(func(b *Builder) { b.Required("c").Filled("string") })
	})
	r := s.Call(map[string]any{"a": "1", "b": map[string]any{}})
	if r.Success() {
		t.Fatal("expected failure")
	}
	if r.Output() != r.ToH() {
		t.Fatal("Output != ToH")
	}
	// Messages flattens paths in order.
	msgs := r.Messages()
	if len(msgs) != 2 {
		t.Fatalf("messages=%d %+v", len(msgs), msgs)
	}
	if msgs[0].Text != "must be greater than 100" || len(msgs[0].Path) != 1 ||
		msgs[0].Path[0] != Symbol("a") {
		t.Errorf("msg0=%+v", msgs[0])
	}
	if msgs[1].Text != "is missing" || len(msgs[1].Path) != 2 {
		t.Errorf("msg1=%+v", msgs[1])
	}
	// Success path: empty errors map, populated Values.
	r2 := s.Call(map[string]any{"a": "200", "b": map[string]any{"c": "x"}})
	if !r2.Success() {
		t.Fatalf("expected success: %s", rubyInspectMap(r2.Errors()))
	}
	if r2.Messages() != nil {
		t.Errorf("expected nil messages, got %+v", r2.Messages())
	}
	// RuleContext.Values accessor.
	c := NewContract(s)
	c.Rule(func(rc *RuleContext) {
		if rc.Values().Len() == 0 {
			rc.KeyFailure("empty")
		}
	}, "a")
	_ = c.Call(map[string]any{"a": "200", "b": map[string]any{"c": "x"}})
}

func TestGoldenValueModelHelpers(t *testing.T) {
	// NewMap re-export and Map behavior.
	m := NewMap()
	m.Set(Symbol("k"), 1)
	if v, ok := m.Get(Symbol("k")); !ok || v != 1 {
		t.Fatal("map set/get")
	}
	// asMap over the accepted input shapes.
	if _, ok := asMap(map[Symbol]any{"a": 1}); !ok {
		t.Error("asMap symbol map")
	}
	if _, ok := asMap(map[any]any{"a": 1}); !ok {
		t.Error("asMap any map")
	}
	if _, ok := asMap(NewMap()); !ok {
		t.Error("asMap *Map")
	}
	if _, ok := asMap(42); ok {
		t.Error("asMap non-map")
	}
	// input via *Map and map[Symbol]any into a schema.
	s := Params(func(b *Builder) { b.Required("a").Filled("string") })
	mm := NewMap()
	mm.Set(Symbol("a"), "x")
	eq(t, s.Call(mm), `{a: "x"}`, `{}`)
	eq(t, s.Call(map[Symbol]any{"a": "x"}), `{a: "x"}`, `{}`)

	// rendering helpers over the full value shapes (used by tests + inspection).
	if rubyInspectVal(nil) != "nil" || rubyInspectVal(true) != "true" ||
		rubyInspectVal(false) != "false" {
		t.Error("inspect bool/nil")
	}
	if rubyInspectVal(int(3)) != "3" || rubyInspectVal(int32(3)) != "3" ||
		rubyInspectVal(int64(3)) != "3" || rubyInspectVal(big.NewInt(3)) != "3" {
		t.Error("inspect ints")
	}
	if rubyInspectVal(3.5) != "3.5" || rubyInspectVal(3.0) != "3.0" {
		t.Error("inspect float")
	}
	if rubyInspectVal(Symbol("s")) != ":s" {
		t.Error("inspect symbol")
	}
	if rubyInspectVal([]any{1, "a"}) != `[1, "a"]` {
		t.Error("inspect array")
	}
	if rubyInspectVal(Date{Year: 2020, Month: 1, Day: 2}) != "#<Date: 2020-01-02>" {
		t.Error("inspect date")
	}
	tm := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	if rubyInspectVal(tm) == "" {
		t.Error("inspect time")
	}
	// Map with integer key renders `k => v`.
	im := NewMap()
	im.Set(0, []any{"e"})
	if rubyInspectMap(im) != `{0 => ["e"]}` {
		t.Errorf("int-key map=%q", rubyInspectMap(im))
	}
	if rubyInspectVal(im) == "" {
		t.Error("inspect map val")
	}
	// unrenderable value → "".
	if rubyInspectVal(struct{}{}) != "" {
		t.Error("inspect unknown")
	}
}

func TestGoldenPredicateHelpers(t *testing.T) {
	// numeric helpers across the int/big/float shapes + non-numeric.
	if _, ok := toFloat("x"); ok {
		t.Error("toFloat string")
	}
	for _, v := range []any{int(1), int32(1), int64(1), big.NewInt(1), 1.0} {
		if _, ok := toFloat(v); !ok {
			t.Errorf("toFloat %T", v)
		}
	}
	if _, ok := toInt("x"); ok {
		t.Error("toInt string")
	}
	// sizeOf over string/array/map/non-sized.
	if n, _ := sizeOf("ab"); n != 2 {
		t.Error("sizeOf string")
	}
	if n, _ := sizeOf([]any{1}); n != 1 {
		t.Error("sizeOf array")
	}
	m := NewMap()
	m.Set("a", 1)
	if n, _ := sizeOf(m); n != 1 {
		t.Error("sizeOf map")
	}
	if _, ok := sizeOf(42); ok {
		t.Error("sizeOf int")
	}
	// sizeCmp with non-sized value and non-int arg.
	if sizeCmp(42, 3, 0) {
		t.Error("sizeCmp non-sized")
	}
	if sizeCmp("ab", "x", 0) {
		t.Error("sizeCmp bad arg")
	}
	// cmp with non-numeric operands.
	if cmp("x", 1, func(int) bool { return true }) {
		t.Error("cmp non-numeric v")
	}
	if cmp(1, "x", func(int) bool { return true }) {
		t.Error("cmp non-numeric arg")
	}
	// formatOK non-string and *regexp.Regexp arg and bad string regexp.
	if formatOK(5, `x`) {
		t.Error("formatOK non-string")
	}
	if !formatOK("abc", regexp.MustCompile(`b`)) {
		t.Error("formatOK regexp arg")
	}
	if formatOK("abc", `(`) {
		t.Error("formatOK bad regexp")
	}
	if formatOK("abc", 5) {
		t.Error("formatOK bad arg type")
	}
	// includedIn with non-list arg.
	if includedIn("x", "notalist") {
		t.Error("includedIn non-list")
	}
	// intParity over shapes + non-int.
	if !intParity(int(3), 1) || !intParity(int32(3), 1) || !intParity(big.NewInt(-3), 1) {
		t.Error("intParity")
	}
	if intParity("x", 0) {
		t.Error("intParity non-int")
	}
	// isFilled shapes.
	if isFilled(nil) || isFilled("") || isFilled([]any{}) || isFilled(NewMap()) {
		t.Error("isFilled empties")
	}
	if !isFilled(5) || !isFilled("x") {
		t.Error("isFilled non-empty")
	}
	// valuesEqual across shapes.
	if !valuesEqual(int64(5), 5.0) || valuesEqual(int64(5), "x") {
		t.Error("valuesEqual numeric")
	}
	if !valuesEqual("a", "a") || valuesEqual("a", "b") || valuesEqual("a", 1) {
		t.Error("valuesEqual string")
	}
	if !valuesEqual(Symbol("a"), Symbol("a")) || valuesEqual(Symbol("a"), Symbol("b")) {
		t.Error("valuesEqual symbol")
	}
	if !valuesEqual(true, true) || valuesEqual(true, false) || valuesEqual(true, 1) {
		t.Error("valuesEqual bool")
	}
	if !valuesEqual(nil, nil) || valuesEqual(nil, 1) {
		t.Error("valuesEqual nil")
	}
	if !valuesEqual([]any{1, 2}, []any{1, 2}) || valuesEqual([]any{1}, []any{1, 2}) ||
		valuesEqual([]any{1}, "x") || valuesEqual([]any{1}, []any{2}) {
		t.Error("valuesEqual array")
	}
	if valuesEqual(struct{}{}, struct{}{}) {
		t.Error("valuesEqual unknown")
	}
	// rubyInt / rubyInspect / joinList / listElem branches.
	if rubyInt(int(1)) != "1" || rubyInt(int32(1)) != "1" || rubyInt(int64(1)) != "1" ||
		rubyInt(big.NewInt(1)) != "1" || rubyInt("x") != `"x"` {
		t.Error("rubyInt")
	}
	if rubyInspect("x") != `"x"` || rubyInspect(Symbol("s")) != ":s" ||
		rubyInspect(true) != "true" || rubyInspect(false) != "false" ||
		rubyInspect(nil) != "nil" || rubyInspect(5) != "5" {
		t.Error("rubyInspect")
	}
	if joinList("notalist") != `"notalist"` {
		t.Error("joinList non-list")
	}
	if joinList([]any{Symbol("s"), 1, "t"}) != "s, 1, t" {
		t.Errorf("joinList=%q", joinList([]any{Symbol("s"), 1, "t"}))
	}
	// sortStrings.
	ss := []string{"c", "a", "b"}
	sortStrings(ss)
	if ss[0] != "a" || ss[2] != "c" {
		t.Error("sortStrings")
	}
}

func TestGoldenBigIntCoercion(t *testing.T) {
	// gt/lt against a big.Int arg and value exercises the big.Int toFloat path.
	s := Params(func(b *Builder) {
		b.Required("f").Value("integer", Predicate{"gt", big.NewInt(10)})
	})
	eq(t, s.Call(map[string]any{"f": "5"}),
		`{f: 5}`, `{f: ["must be greater than 10"]}`)
}
