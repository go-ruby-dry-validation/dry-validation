// Copyright (c) the go-ruby-dry-validation/dry-validation authors
//
// SPDX-License-Identifier: BSD-3-Clause

package dryvalidation

// run coerces and validates a single value against the macro in the given mode.
// It returns the coerced value (or the original, kept for the output when
// coercion fails) and a message subtree rooted at the value (nil = valid).
func (mc *macro) run(val any, m mode) (coercedVal, *messageNode) {
	switch mc.kind {
	case macroFilled:
		return mc.runScalar(val, m, true)
	case macroValue:
		return mc.runScalar(val, m, false)
	case macroMaybe:
		if val == nil {
			return coercedVal{val: nil, set: true}, nil
		}
		return mc.runScalar(val, m, false)
	case macroArray:
		return mc.runArray(val, m)
	case macroHash:
		return mc.runHash(val, m)
	}
	return coercedVal{}, nil
}

// runScalar coerces val to the macro's type then checks predicates. When filled
// is true, an implicit filled? runs first: an empty/nil value short-circuits to
// "must be filled" before the type is even considered (matching the gem, where
// filled(:string) reports "must be filled" for "").
func (mc *macro) runScalar(val any, m mode, filled bool) (coercedVal, *messageNode) {
	if filled && !isFilled(val) {
		return leaf(val, false, "must be filled")
	}
	spec := resolveType(mc.typeName, m)
	coerced, err := spec.coerce(val)
	if err != nil {
		return leaf(val, false, spec.typeMsg)
	}
	for _, p := range mc.preds {
		if ok, msg := p.check(coerced); !ok {
			return leaf(coerced, true, msg)
		}
	}
	return coercedVal{val: coerced, set: true}, nil
}

// runArray coerces val to an array then coerces+validates each element. Element
// failures are recorded under their integer index (dry-schema keys array errors
// by index). A non-array value reports "must be an array" at this node.
func (mc *macro) runArray(val any, m mode) (coercedVal, *messageNode) {
	arr, ok := val.([]any)
	if !ok {
		return leaf(val, false, "must be an array")
	}
	node := newMessageNode()
	coercedElems := make([]any, len(arr))
	for i, e := range arr {
		cv, en := mc.arrayElem.run(e, m)
		if cv.set {
			coercedElems[i] = cv.val
		} else {
			coercedElems[i] = e
		}
		if en != nil && !en.empty() {
			mergeInto(node.child(i), en)
		}
	}
	if node.empty() {
		return coercedVal{val: coercedElems, set: true}, nil
	}
	return coercedVal{val: coercedElems, set: true}, node
}

// runHash coerces val to a hash then applies the nested schema. A non-hash value
// reports "must be a hash" at this node.
func (mc *macro) runHash(val any, m mode) (coercedVal, *messageNode) {
	if _, ok := asMap(val); !ok {
		return leaf(val, false, "must be a hash")
	}
	out := NewMap()
	node := newMessageNode()
	mc.nested.apply(val, out, node, nil)
	if node.empty() {
		return coercedVal{val: out, set: true}, nil
	}
	return coercedVal{val: out, set: true}, node
}

// leaf builds a single-message subtree for a scalar failure.
func leaf(val any, set bool, msg string) (coercedVal, *messageNode) {
	n := newMessageNode()
	n.addText(msg)
	return coercedVal{val: val, set: set}, n
}
