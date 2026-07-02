// Copyright (c) the go-ruby-dry-validation/dry-validation authors
//
// SPDX-License-Identifier: BSD-3-Clause

package dryvalidation

import (
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	drytypes "github.com/go-ruby-dry-types/dry-types"
)

// Predicate is one dry-logic value constraint attached to a schema key
// (`gt?: 18`, `format?: /.../`, `size?: 3`, …). Name is the predicate without
// the trailing `?`; Arg is its argument (nil for the arity-1 predicates).
type Predicate struct {
	Name string
	Arg  any
}

// check evaluates the predicate against the coerced value v, returning whether it
// holds and, on failure, the en-locale message dry-schema reports for it.
func (p Predicate) check(v any) (bool, string) {
	switch p.Name {
	case "gt":
		return cmp(v, p.Arg, func(d int) bool { return d > 0 }), "must be greater than " + rubyInt(p.Arg)
	case "gteq":
		return cmp(v, p.Arg, func(d int) bool { return d >= 0 }), "must be greater than or equal to " + rubyInt(p.Arg)
	case "lt":
		return cmp(v, p.Arg, func(d int) bool { return d < 0 }), "must be less than " + rubyInt(p.Arg)
	case "lteq":
		return cmp(v, p.Arg, func(d int) bool { return d <= 0 }), "must be less than or equal to " + rubyInt(p.Arg)
	case "format":
		return formatOK(v, p.Arg), "is in invalid format"
	case "size":
		return sizeCmp(v, p.Arg, 0), "length must be " + rubyInt(p.Arg)
	case "min_size":
		return sizeCmp(v, p.Arg, -1), "size cannot be less than " + rubyInt(p.Arg)
	case "max_size":
		return sizeCmp(v, p.Arg, 1), "size cannot be greater than " + rubyInt(p.Arg)
	case "included_in":
		return includedIn(v, p.Arg), "must be one of: " + joinList(p.Arg)
	case "excluded_from":
		return !includedIn(v, p.Arg), "must not be one of: " + joinList(p.Arg)
	case "filled":
		return isFilled(v), "must be filled"
	case "empty":
		return !isFilled(v), "cannot be defined"
	case "eql":
		return valuesEqual(v, p.Arg), "must be equal to " + rubyInspect(p.Arg)
	case "not_eql":
		return !valuesEqual(v, p.Arg), "cannot be equal to " + rubyInspect(p.Arg)
	case "true":
		return v == true, "must be true"
	case "false":
		return v == false, "must be false"
	case "odd":
		return intParity(v, 1), "must be odd"
	case "even":
		return intParity(v, 0), "must be even"
	}
	// Unknown predicates always pass (the gem would have raised at build time).
	return true, ""
}

// cmp returns whether sign(v-arg) satisfies ok, for numeric v and arg.
func cmp(v, arg any, ok func(int) bool) bool {
	vf, vok := toFloat(v)
	af, aok := toFloat(arg)
	if !vok || !aok {
		return false
	}
	switch {
	case vf < af:
		return ok(-1)
	case vf > af:
		return ok(1)
	default:
		return ok(0)
	}
}

func toFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case int:
		return float64(x), true
	case int32:
		return float64(x), true
	case int64:
		return float64(x), true
	case *big.Int:
		f, _ := new(big.Float).SetInt(x).Float64()
		return f, true
	case float64:
		return x, true
	}
	return 0, false
}

func sizeOf(v any) (int, bool) {
	switch x := v.(type) {
	case string:
		return utf8.RuneCountInString(x), true
	case []any:
		return len(x), true
	case *Map:
		return x.Len(), true
	}
	return 0, false
}

// sizeCmp checks size predicates. mode 0=exact, -1=min, 1=max.
func sizeCmp(v, arg any, mode int) bool {
	n, ok := sizeOf(v)
	if !ok {
		return false
	}
	want, wok := toInt(arg)
	if !wok {
		return false
	}
	switch mode {
	case -1:
		return n >= want
	case 1:
		return n <= want
	default:
		return n == want
	}
}

func toInt(v any) (int, bool) {
	f, ok := toFloat(v)
	return int(f), ok
}

func formatOK(v, arg any) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	switch r := arg.(type) {
	case *regexp.Regexp:
		return r.MatchString(s)
	case string:
		re, err := regexp.Compile(r)
		return err == nil && re.MatchString(s)
	}
	return false
}

func includedIn(v, arg any) bool {
	list, ok := arg.([]any)
	if !ok {
		return false
	}
	for _, e := range list {
		if valuesEqual(v, e) {
			return true
		}
	}
	return false
}

func intParity(v any, want int) bool {
	switch x := v.(type) {
	case int:
		return absMod2(int64(x)) == want
	case int32:
		return absMod2(int64(x)) == want
	case int64:
		return absMod2(x) == want
	case *big.Int:
		return int(new(big.Int).Abs(x).Bit(0)) == want
	}
	return false
}

func absMod2(x int64) int {
	if x < 0 {
		x = -x
	}
	return int(x % 2)
}

// isFilled reports Ruby presence: non-nil, non-empty string/array/hash.
func isFilled(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case string:
		return x != ""
	case []any:
		return len(x) != 0
	case *Map:
		return x.Len() != 0
	}
	return true
}

// valuesEqual compares two Ruby values for `==` across the shapes predicates see.
func valuesEqual(a, b any) bool {
	if af, aok := toFloat(a); aok {
		if bf, bok := toFloat(b); bok {
			return af == bf
		}
		return false
	}
	switch x := a.(type) {
	case string:
		y, ok := b.(string)
		return ok && x == y
	case Symbol:
		y, ok := b.(Symbol)
		return ok && x == y
	case bool:
		y, ok := b.(bool)
		return ok && x == y
	case nil:
		return b == nil
	case []any:
		y, ok := b.([]any)
		if !ok || len(x) != len(y) {
			return false
		}
		for i := range x {
			if !valuesEqual(x[i], y[i]) {
				return false
			}
		}
		return true
	}
	return false
}

// rubyInt renders an integer/float argument the way it appears in a dry-schema
// message ("greater than 18"): integers plain, floats via inspect.
func rubyInt(arg any) string {
	switch x := arg.(type) {
	case int:
		return strconv.Itoa(x)
	case int32:
		return strconv.FormatInt(int64(x), 10)
	case int64:
		return strconv.FormatInt(x, 10)
	case *big.Int:
		return x.String()
	}
	return rubyInspect(arg)
}

// joinList renders an included_in/excluded_from list as "a, b, c" (bare values,
// no quotes) the way the en messages interpolate a list.
func joinList(arg any) string {
	list, ok := arg.([]any)
	if !ok {
		return rubyInspect(arg)
	}
	parts := make([]string, len(list))
	for i, e := range list {
		parts[i] = listElem(e)
	}
	return strings.Join(parts, ", ")
}

// listElem renders one list element the way the message interpolation does:
// strings and symbols bare, numbers plain.
func listElem(e any) string {
	switch x := e.(type) {
	case string:
		return x
	case Symbol:
		return string(x)
	default:
		return rubyInt(x)
	}
}

// rubyInspect renders a value the way Ruby Object#inspect does, used for eql
// message interpolation (strings quoted).
func rubyInspect(v any) string {
	switch x := v.(type) {
	case string:
		return strconv.Quote(x)
	case Symbol:
		return ":" + string(x)
	case bool:
		if x {
			return "true"
		}
		return "false"
	case nil:
		return "nil"
	default:
		return rubyInt(x)
	}
}

// asMap normalizes a hash-shaped input into an ordered *Map. It accepts *Map,
// map[string]any, map[Symbol]any and map[any]any (sorting the plain-map variants
// for determinism); anything else returns (nil, false) so callers raise the
// proper "must be a hash" error.
func asMap(v any) (*Map, bool) {
	switch h := v.(type) {
	case *Map:
		return h, true
	case map[string]any:
		m := drytypes.NewMap()
		keys := make([]string, 0, len(h))
		for k := range h {
			keys = append(keys, k)
		}
		sortStrings(keys)
		for _, k := range keys {
			m.Set(k, h[k])
		}
		return m, true
	case map[Symbol]any:
		m := drytypes.NewMap()
		keys := make([]string, 0, len(h))
		for k := range h {
			keys = append(keys, string(k))
		}
		sortStrings(keys)
		for _, k := range keys {
			m.Set(Symbol(k), h[Symbol(k)])
		}
		return m, true
	case map[any]any:
		m := drytypes.NewMap()
		for k, val := range h {
			m.Set(k, val)
		}
		return m, true
	}
	return nil, false
}

// sortStrings is an allocation-free insertion sort for the small key slices
// asMap builds (avoids importing sort for one call site).
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}
