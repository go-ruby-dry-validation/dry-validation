// Copyright (c) the go-ruby-dry-validation/dry-validation authors
//
// SPDX-License-Identifier: BSD-3-Clause

package dryvalidation

import drytypes "github.com/go-ruby-dry-types/dry-types"

// Message is one validation failure: the message text and the path of keys from
// the root of the input to the offending value. A path element is a [Symbol] for
// a hash key or an int for an array index; the empty path (a base error) is
// reported by the gem under the nil key.
type Message struct {
	// Text is the failure message, byte-identical to the gem's en locale
	// (e.g. "is missing", "must be filled", "must be greater than 18").
	Text string
	// Path is the key/index path to the value (Symbol or int elements). A base
	// error carries the single-element path []any{nil}.
	Path []any
}

// messageNode is the internal tree the errors hash is built from: leaf texts at
// this node plus child nodes keyed by Symbol (hash) or int (array index),
// preserving insertion order.
type messageNode struct {
	texts    []string
	children []messageChild
	index    map[any]int
}

type messageChild struct {
	key  any // Symbol or int
	node *messageNode
}

func newMessageNode() *messageNode { return &messageNode{index: map[any]int{}} }

// addText appends a leaf message at this node (deduplicating identical repeats,
// which dry-schema does when the same rule fails twice).
func (n *messageNode) addText(text string) {
	for _, t := range n.texts {
		if t == text {
			return
		}
	}
	n.texts = append(n.texts, text)
}

// child returns the child node for key, creating it in insertion order.
func (n *messageNode) child(key any) *messageNode {
	if i, ok := n.index[key]; ok {
		return n.children[i].node
	}
	c := newMessageNode()
	n.index[key] = len(n.children)
	n.children = append(n.children, messageChild{key: key, node: c})
	return c
}

// empty reports whether the node carries no messages anywhere.
func (n *messageNode) empty() bool {
	if len(n.texts) > 0 {
		return false
	}
	for _, c := range n.children {
		if !c.node.empty() {
			return false
		}
	}
	return true
}

// add records a message at the given path, building intermediate nodes.
func (n *messageNode) add(path []any, text string) {
	cur := n
	for _, p := range path {
		cur = cur.child(p)
	}
	cur.addText(text)
}

// toH renders the node as dry-schema's errors.to_h shape: a leaf-only node
// becomes a []any of its texts; a node with children becomes an ordered *Map
// whose values are recursively rendered. A node carrying both texts and children
// (which the gem never produces for a single key) prefers the texts.
func (n *messageNode) toH() any {
	if len(n.children) == 0 {
		return textsSlice(n.texts)
	}
	m := drytypes.NewMap()
	for _, c := range n.children {
		if c.node.empty() {
			continue
		}
		m.Set(c.key, c.node.toH())
	}
	return m
}

// textsSlice copies the texts into an []any (the gem's array-of-strings leaf).
func textsSlice(texts []string) []any {
	out := make([]any, len(texts))
	for i, t := range texts {
		out[i] = t
	}
	return out
}

// flatten walks the tree in insertion order producing the flat [Message] list
// dry-validation's Result#errors.each yields, each with its full path.
func (n *messageNode) flatten(prefix []any) []Message {
	var out []Message
	for _, t := range n.texts {
		p := make([]any, len(prefix))
		copy(p, prefix)
		out = append(out, Message{Text: t, Path: p})
	}
	for _, c := range n.children {
		out = append(out, c.node.flatten(append(append([]any{}, prefix...), c.key))...)
	}
	return out
}
