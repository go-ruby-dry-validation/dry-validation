<p align="center"><img src="https://raw.githubusercontent.com/go-ruby-dry-validation/brand/main/social/go-ruby-dry-validation-dry-validation.png" alt="go-ruby-dry-validation/dry-validation" width="720"></p>

# dry-validation — go-ruby-dry-validation

[![Docs](https://img.shields.io/badge/docs-mkdocs--material-DC2626)](https://go-ruby-dry-validation.github.io/docs/)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.26.4%2B-00ADD8)](https://go.dev/dl/)
[![Coverage](https://img.shields.io/badge/coverage-100%25-1a7f37)](#tests--coverage)

**A pure-Go (no cgo) reimplementation of Ruby's
[dry-schema](https://dry-rb.org/gems/dry-schema/) and
[dry-validation](https://dry-rb.org/gems/dry-validation/) gems** — the schema /
contract layer that coerces and validates a hash of input against key-presence
rules, value macros, [dry-types](https://github.com/go-ruby-dry-types/dry-types)
coercion and dry-logic predicate constraints, then reports a nested errors hash
whose messages are **byte-identical to the gems' default English locale**
("is missing", "must be filled", "must be greater than 18", "must be a hash", …)
— **without any Ruby runtime**.

It is built directly on top of
[go-ruby-dry-types](https://github.com/go-ruby-dry-types/dry-types): every schema
key's coercion + constraint is a dry-types `Type`, and the schema composes those
into a whole-hash validator. It is the validation backend for
[go-embedded-ruby](https://github.com/go-embedded-ruby/ruby) but is a
**standalone, reusable** module.

> **What it is — and isn't.** The schema definition, coercion, macro/predicate
> evaluation, and error-message localization are fully deterministic and need
> **no interpreter**, so they live here as pure Go. The custom-`rule` predicate
> *bodies* of a dry-validation `Contract` are the host's job: rbgo runs the Ruby
> block, and this library exposes the coerced values and a `RuleContext` whose
> `Key`/`Base` failures add errors — the schema + built-in macro/predicate
> evaluation is the deterministic core.

## Features

Faithful port of dry-schema + dry-validation, validated against the
`dry-schema` / `dry-validation` gems on every supported platform:

- **Key presence** — `required(:k)` (absent ⇒ `"is missing"`) and `optional(:k)`
  (absent ⇒ dropped, no error).
- **Value macros** — `filled` (present + non-empty), `maybe` (nil passes
  through), `value` (coerce + predicates), `array(:elem, …)` (per-element coerce
  + validate, errors keyed by index), `hash do … end` / `schema do … end`
  (nested schema), and `array(:hash) do … end` (array of nested schemas).
- **Coercion via dry-types** — the `Params` namespace form-coerces strings to
  integer/float/bool/symbol/date/time; the `JSON` namespace uses JSON-native
  types. `:string` is strict in `Params` (a non-string reports `must be a
  string`, matching the gem).
- **dry-logic predicates** — `gt? gteq? lt? lteq? format? size? min_size?
  max_size? included_in? excluded_from? eql? not_eql? filled? empty? odd? even?
  true? false?`, each with its exact en-locale message.
- **Nested errors hash** — a `*Map` whose leaves are `[]any` of message strings,
  hash keys are `Symbol`, and array-element keys are `int`, plus a flat
  `[]Message` with full key paths (`result.errors.each`).
- **Contracts** — `Contract` wraps a schema and a list of `rule`s that run after
  the schema against the coerced values; a rule fires only when **all** its
  dependency keys passed the schema (a base rule always runs), matching the gem.

CGO-free, **100% test coverage**, `gofmt` + `go vet` clean, and green across the
six 64-bit Go targets (amd64, arm64, riscv64, loong64, ppc64le, s390x).

## Install

```sh
go get github.com/go-ruby-dry-validation/dry-validation
```

## Usage

```go
package main

import (
	"fmt"

	dv "github.com/go-ruby-dry-validation/dry-validation"
)

func main() {
	// Dry::Schema.Params { required(:email).filled(:string)
	//                      optional(:age).maybe(:integer)
	//                      required(:address).hash { required(:city).filled(:string) } }
	schema := dv.Params(func(b *dv.Builder) {
		b.Required("email").Filled("string")
		b.Optional("age").Maybe("integer")
		b.Required("address").Hash(func(b *dv.Builder) {
			b.Required("city").Filled("string")
		})
	})

	r := schema.Call(map[string]any{
		"email":   "a@b.com",
		"age":     "30", // Params coerces "30" → 30
		"address": map[string]any{},
	})

	fmt.Println(r.Success())               // false
	fmt.Println(r.ToH())                   // {email: "a@b.com", age: 30, address: {}}
	fmt.Println(r.Errors())                // {address: {city: ["is missing"]}}

	// A contract adds custom rules over the coerced values.
	c := dv.NewContract(schema)
	c.Rule(func(rc *dv.RuleContext) {
		if v, ok := rc.Value("age"); ok {
			if age, ok := v.(int64); ok && age < 18 {
				rc.KeyFailure("must be at least 18")
			}
		}
	}, "age")
}
```

## Error messages & paths

`Result.Errors()` renders the nested hash exactly as the gems' `errors.to_h`:

| Input                                            | `errors.to_h`                             |
| ------------------------------------------------ | ----------------------------------------- |
| `required(:email)` absent                        | `{email: ["is missing"]}`                 |
| `filled(:string)` on `""`                        | `{email: ["must be filled"]}`             |
| `filled(:integer, gt?: 18)` on `10`              | `{age: ["must be greater than 18"]}`      |
| `array(:string)` on `["a", 1]`                   | `{tags: {1 => ["must be a string"]}}`     |
| `hash do required(:city).filled end` missing key | `{address: {city: ["is missing"]}}`       |
| base rule failure                                | `{nil => ["…"]}`                          |

## API

```go
func Params(build func(*Builder)) *Schema // Dry::Schema.Params
func JSON(build func(*Builder)) *Schema   // Dry::Schema.JSON

func (*Builder) Required(name Symbol) *Key
func (*Builder) Optional(name Symbol) *Key

func (*Key) Filled(typeName string, preds ...Predicate)
func (*Key) Maybe(typeName string, preds ...Predicate)
func (*Key) Value(typeName string, preds ...Predicate)
func (*Key) Array(elemType string, elemPreds ...Predicate)
func (*Key) ArrayOf(build func(*Builder)) // array(:hash) do … end
func (*Key) Hash(build func(*Builder))    // hash do … end
func (*Key) Schema(build func(*Builder))  // schema do … end

func (*Schema) Call(input any) *Result

type Predicate struct { Name string; Arg any } // gt / format / size / …

func NewContract(schema *Schema) *Contract
func (*Contract) Rule(body func(*RuleContext), keys ...Symbol)
func (*Contract) RuleBase(body func(*RuleContext))
func (*Contract) Call(input any) *Result

type RuleContext struct{ /* … */ }
func (*RuleContext) Values() *Map
func (*RuleContext) Value(key Symbol) (any, bool)
func (*RuleContext) KeyFailure(text string)
func (*RuleContext) KeyFailureAt(path []any, text string)
func (*RuleContext) BaseFailure(text string)

type Result struct{ /* … */ }
func (*Result) ToH() *Map      // coerced output
func (*Result) Errors() *Map   // nested errors hash
func (*Result) Success() bool
func (*Result) Messages() []Message // flat, with paths
```

## Ruby value model

Inputs and coerced outputs use the same small, fixed set of Go types the
go-ruby-* ecosystem shares (re-exported from
[go-ruby-dry-types](https://github.com/go-ruby-dry-types/dry-types)), so a host
maps its object graph with no glue: `nil`, `bool`, `int64`/`*big.Int`, `float64`,
`string`, `Symbol`, `[]any`, `*Map` (ordered Hash), `Date`, `time.Time`.

## Tests & coverage

The suite pairs deterministic, ruby-free golden tests (which alone hold coverage
at 100%, so the qemu cross-arch and Windows lanes pass the gate) with a
**differential oracle**: schemas and contracts are run against valid + invalid
inputs here and by the system `ruby` with the `dry-schema` / `dry-validation`
gems (gated on `RUBY_VERSION >= "4.0"`), comparing the coerced output and the
full errors hash across filled / maybe / array / nested / predicate cases in both
the `Params` and `JSON` namespaces. The oracle skips itself where `ruby` or the
gems are absent.

```sh
COVERPKG=$(go list ./... | paste -sd, -)
go test -race -coverpkg="$COVERPKG" -coverprofile=cover.out ./...
go tool cover -func=cover.out | tail -1   # 100.0%
```

## License

BSD-3-Clause — see [LICENSE](LICENSE). Copyright the go-ruby-dry-validation/dry-validation authors.

## WebAssembly

Being pure Go (CGO=0), this library also compiles to **WebAssembly** — both
`GOOS=js GOARCH=wasm` (browser / Node.js) and `GOOS=wasip1 GOARCH=wasm` (WASI).
CI builds both targets on every push, alongside the six 64-bit native/qemu arches.

```sh
GOOS=js     GOARCH=wasm go build ./...   # browser / Node
GOOS=wasip1 GOARCH=wasm go build ./...   # WASI (wasmtime, wasmer, wasmedge, …)
```
