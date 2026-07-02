// Copyright (c) the go-ruby-dry-validation/dry-validation authors
//
// SPDX-License-Identifier: BSD-3-Clause

package dryvalidation

import (
	"math/big"
	"strconv"
	"strings"
	"time"
)

// rubyInspectVal renders a coerced value the way Ruby 4.0 Object#inspect does, so
// a Result's to_h / errors.to_h can be compared byte-for-byte with the gems'
// output in the oracle and golden tests.
func rubyInspectVal(v any) string {
	switch x := v.(type) {
	case nil:
		return "nil"
	case bool:
		if x {
			return "true"
		}
		return "false"
	case string:
		return strconv.Quote(x)
	case Symbol:
		return ":" + string(x)
	case int:
		return strconv.Itoa(x)
	case int32:
		return strconv.FormatInt(int64(x), 10)
	case int64:
		return strconv.FormatInt(x, 10)
	case *big.Int:
		return x.String()
	case float64:
		return formatRubyFloat(x)
	case Date:
		return "#<Date: " + x.String() + ">"
	case time.Time:
		return x.Format("2006-01-02 15:04:05 -0700")
	case []any:
		parts := make([]string, len(x))
		for i, e := range x {
			parts[i] = rubyInspectVal(e)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case *Map:
		return rubyInspectMap(x)
	}
	return ""
}

// rubyInspectMap renders a *Map the way Ruby 4.0 Hash#inspect does: `sym: v` for
// symbol keys, `int => v` for integer keys (array-index error keys).
func rubyInspectMap(m *Map) string {
	parts := make([]string, 0, m.Len())
	for _, p := range m.Pairs() {
		switch k := p.Key.(type) {
		case Symbol:
			parts = append(parts, string(k)+": "+rubyInspectVal(p.Val))
		default:
			parts = append(parts, rubyInspectVal(p.Key)+" => "+rubyInspectVal(p.Val))
		}
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func formatRubyFloat(f float64) string {
	s := strconv.FormatFloat(f, 'g', -1, 64)
	if !strings.ContainsAny(s, ".eEnN") {
		s += ".0"
	}
	return s
}
