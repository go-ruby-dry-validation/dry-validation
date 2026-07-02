// Copyright (c) the go-ruby-dry-validation/dry-validation authors
//
// SPDX-License-Identifier: BSD-3-Clause

package dryvalidation

import "testing"

// TestCoverageInternalBranches drives the internal error-tree, resolver and
// helper branches the public golden/oracle tests don't reach on their own, so the
// deterministic suite alone holds the 100% coverage gate.
func TestCoverageInternalBranches(t *testing.T) {
	// addText dedup: the same text added twice keeps one copy.
	n := newMessageNode()
	n.addText("dup")
	n.addText("dup")
	if len(n.texts) != 1 {
		t.Fatalf("addText dedup: %v", n.texts)
	}
	// child returns the existing node on the second call (index-hit branch).
	c1 := n.child(Symbol("k"))
	c2 := n.child(Symbol("k"))
	if c1 != c2 {
		t.Fatal("child did not return existing node")
	}
	// toH skips an empty child node.
	root := newMessageNode()
	root.child(Symbol("empty")) // created but carries no message
	root.child(Symbol("real")).addText("msg")
	m, ok := root.toH().(*Map)
	if !ok || m.Len() != 1 {
		t.Fatalf("toH empty-child skip: %v", root.toH())
	}
	if _, present := m.Get(Symbol("empty")); present {
		t.Fatal("empty child should be omitted")
	}

	// resolveType("") is the any/passthrough spec (empty-name branch).
	spec := resolveType("", modeParams)
	if out, err := spec.coerce(99); err != nil || out != 99 {
		t.Fatalf("empty type spec: %v %v", out, err)
	}
	if spec.typeMsg != "" {
		t.Fatalf("empty type msg: %q", spec.typeMsg)
	}
	// jsonType/paramsType over every type name (all switch arms) + the
	// default (unknown name) → passthrough.
	for _, name := range []string{
		"integer", "float", "string", "bool", "symbol", "date", "time",
		"datetime", "date_time", "array", "hash", "nil", "weird",
	} {
		if jsonType(name) == nil || paramsType(name) == nil {
			t.Fatalf("type %q resolves nil", name)
		}
	}

	// absMod2 negative path.
	if absMod2(-3) != 1 || absMod2(-4) != 0 {
		t.Fatal("absMod2 negative")
	}
	// intParity over a negative int64.
	if !intParity(int64(-3), 1) {
		t.Fatal("intParity negative int64")
	}

	// run over an out-of-range macro kind hits the default (defensive) branch.
	bad := &macro{kind: macroKind(99)}
	cv, en := bad.run("x", modeParams)
	if cv.set || en != nil {
		t.Fatalf("run default branch: %+v %v", cv, en)
	}
}
