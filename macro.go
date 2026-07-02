// Copyright (c) the go-ruby-dry-validation/dry-validation authors
//
// SPDX-License-Identifier: BSD-3-Clause

package dryvalidation

import (
	drytypes "github.com/go-ruby-dry-types/dry-types"
)

// mode is the coercion namespace a schema runs in: Params (form-string coercion,
// the default and the one dry-validation's `params do` block uses) or JSON
// (JSON-native coercion, `json do`).
type mode int

const (
	modeParams mode = iota
	modeJSON
)

// typeSpec resolves a schema type name (:integer, :string, :bool, …) to its
// dry-types coercion for the active mode plus the en-locale "must be a/an X"
// type-failure message. A coercion returning an error means the value is not of
// (and cannot be coerced to) that type.
type typeSpec struct {
	// coerce runs the dry-types coercion; a non-nil error is a type failure.
	coerce func(any) (any, error)
	// typeMsg is the message reported when coercion fails ("must be an integer").
	typeMsg string
}

// resolveType returns the typeSpec for a type name under a mode. An unknown name
// (or the empty name, meaning "no type — any value") yields a pass-through spec.
func resolveType(name string, m mode) typeSpec {
	if name == "" {
		return typeSpec{coerce: func(v any) (any, error) { return v, nil }}
	}
	var t drytypes.Type
	switch m {
	case modeJSON:
		t = jsonType(name)
	default:
		t = paramsType(name)
	}
	return typeSpec{coerce: t.Call, typeMsg: typeMessage(name)}
}

// paramsType maps a type name to its Params-namespace dry-types type (the
// form-coercion the gem's Params schema uses). Nil / any pass through.
func paramsType(name string) drytypes.Type {
	switch name {
	case "integer":
		return drytypes.ParamsInteger()
	case "float":
		return drytypes.ParamsFloat()
	case "string":
		// dry-schema's Params :string is strict — it does not stringify a
		// non-string input; `filled(:string)` on 5 reports "must be a string".
		return drytypes.StrictString()
	case "bool":
		return drytypes.ParamsBool()
	case "symbol":
		return drytypes.ParamsSymbol()
	case "date":
		return drytypes.ParamsDate()
	case "time":
		return drytypes.ParamsTime()
	case "date_time", "datetime":
		return drytypes.ParamsDateTime()
	case "array":
		return drytypes.StrictArray()
	case "hash":
		return drytypes.StrictHash()
	case "nil":
		return drytypes.ParamsNil()
	}
	return passthroughType()
}

// jsonType maps a type name to its JSON-namespace dry-types type. JSON coercion
// does not turn strings into integers/floats/bools (those must already be the
// native JSON type), so the strict types back the numeric/bool names.
func jsonType(name string) drytypes.Type {
	switch name {
	case "integer":
		return drytypes.StrictInteger()
	case "float":
		return drytypes.StrictFloat()
	case "string":
		return drytypes.StrictString()
	case "bool":
		return drytypes.StrictBool()
	case "symbol":
		return drytypes.JSONSymbol()
	case "date":
		return drytypes.JSONDate()
	case "time":
		return drytypes.JSONTime()
	case "date_time", "datetime":
		return drytypes.JSONDateTime()
	case "array":
		return drytypes.StrictArray()
	case "hash":
		return drytypes.StrictHash()
	case "nil":
		return drytypes.StrictNil()
	}
	return passthroughType()
}

func passthroughType() drytypes.Type { return drytypes.NominalString() }

// typeMessage is the en-locale type-failure message for a type name, matching
// dry-schema's default errors ("must be an integer", "must be a string", …).
func typeMessage(name string) string {
	switch name {
	case "integer":
		return "must be an integer"
	case "float":
		return "must be a float"
	case "string":
		return "must be a string"
	case "bool":
		return "must be boolean"
	case "symbol":
		return "must be a symbol"
	case "date":
		return "must be a date"
	case "time":
		return "must be a time"
	case "date_time", "datetime":
		return "must be a date-time"
	case "array":
		return "must be an array"
	case "hash":
		return "must be a hash"
	case "nil":
		return "cannot be defined"
	}
	return "is in invalid type"
}
