// Copyright (c) the go-ruby-dry-validation/dry-validation authors
//
// SPDX-License-Identifier: BSD-3-Clause

package dryvalidation

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// rubyBin locates a `ruby` (>= 4.0) with the dry-schema/dry-validation gems once,
// and skips the oracle otherwise (the qemu cross-arch lanes, Windows, and any
// host without the gems). The deterministic golden tests alone hold coverage at
// 100%, so the no-ruby lanes still pass the gate.
func rubyBin(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath("ruby")
	if err != nil {
		t.Skip("ruby not on PATH; skipping dry-validation oracle")
	}
	if exec.Command(path, "-e",
		`exit((RUBY_VERSION.split(".").first.to_i >= 4) ? 0 : 3)`).Run() != nil {
		t.Skip("ruby < 4.0; skipping dry-validation oracle")
	}
	if exec.Command(path, "-e", `require "dry-schema"; require "dry-validation"`).Run() != nil {
		t.Skip("dry-schema/dry-validation gems not installed; skipping oracle")
	}
	return path
}

// rubySchemaOutcome runs a dry-schema definition (rubyDSL is the body of a
// Dry::Schema.<ns> block) against input and returns the canonical
// "OUT <to_h.inspect>\nERR <errors.to_h.inspect>" string.
func rubySchemaOutcome(t *testing.T, bin, ns, rubyDSL string, input any) string {
	t.Helper()
	in, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	script := `
$stdout.binmode
require "dry-schema"
require "json"
input = JSON.parse(STDIN.read)
schema = Dry::Schema.` + ns + ` do
` + rubyDSL + `
end
r = schema.call(input)
print "OUT "; print r.to_h.inspect; print "\n"
print "ERR "; print r.errors.to_h.inspect
`
	return runRuby(t, bin, script, in)
}

// rubyContractOutcome runs a dry-validation contract (rubyParams is the params
// block body, rubyRules the rule declarations) against input.
func rubyContractOutcome(t *testing.T, bin, ns, rubyParams, rubyRules string, input any) string {
	t.Helper()
	in, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	script := `
$stdout.binmode
require "dry-validation"
require "json"
input = JSON.parse(STDIN.read)
klass = Class.new(Dry::Validation::Contract) do
  ` + ns + ` do
` + rubyParams + `
  end
` + rubyRules + `
end
r = klass.new.call(input)
print "OUT "; print r.to_h.inspect; print "\n"
print "ERR "; print r.errors.to_h.inspect
`
	return runRuby(t, bin, script, in)
}

func runRuby(t *testing.T, bin, script string, stdin []byte) string {
	t.Helper()
	cmd := exec.Command(bin, "-e", script)
	cmd.Stdin = strings.NewReader(string(stdin))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("ruby error: %v\n%s", err, out)
	}
	return strings.TrimSpace(string(out))
}

// goSchemaOutcome renders a Go [Result] in the same "OUT ...\nERR ..." shape.
func goSchemaOutcome(r *Result) string {
	return "OUT " + rubyInspectMap(r.ToH()) + "\nERR " + rubyInspectMap(r.Errors())
}

// oracleCase pairs a Go schema-builder with the equivalent Ruby DSL and an input.
type oracleCase struct {
	name    string
	ns      string // "Params" / "JSON"
	build   func(*Builder)
	rubyDSL string
	input   any
}

func envSkipDefault() bool { return os.Getenv("DRYV_NO_ORACLE") != "" }

func TestOracleSchema(t *testing.T) {
	if envSkipDefault() {
		t.Skip("DRYV_NO_ORACLE set")
	}
	bin := rubyBin(t)
	cases := oracleSchemaCases()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ns := c.ns
			if ns == "" {
				ns = "Params"
			}
			got := goSchemaOutcome(Schemas[ns](c.build).Call(c.input))
			want := rubySchemaOutcome(t, bin, ns, c.rubyDSL, c.input)
			if got != want {
				t.Errorf("case %s\n  go:   %q\n  ruby: %q", c.name, got, want)
			}
		})
	}
}

// Schemas selects the constructor by namespace name for the oracle.
var Schemas = map[string]func(func(*Builder)) *Schema{
	"Params": Params,
	"JSON":   JSON,
}
