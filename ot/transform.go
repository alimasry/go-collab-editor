package ot

import "fmt"

// Transform takes two concurrent operations a and b (both applied to the same
// document state) and returns aPrime and bPrime such that:
//
//	Apply(Apply(doc, a), bPrime) == Apply(Apply(doc, b), aPrime)
func Transform(a, b Operation) (aPrime, bPrime Operation, err error) {
	if a.BaseLen() != b.BaseLen() {
		return Operation{}, Operation{}, fmt.Errorf(
			"base lengths differ: a=%d, b=%d", a.BaseLen(), b.BaseLen())
	}

	var ap, bp []Component
	ia := newIter(a.Ops)
	ib := newIter(b.Ops)

	for ia.hasNext() || ib.hasNext() {
		// Both insert: a goes first (tie-break).
		if ia.peekType() == compInsert && ib.peekType() == compInsert {
			c := ia.take(0)
			ap = append(ap, Component{Insert: c.Insert})
			bp = append(bp, Component{Retain: len(c.Insert)})
			continue
		}
		// Only a inserts.
		if ia.peekType() == compInsert {
			c := ia.take(0)
			ap = append(ap, Component{Insert: c.Insert})
			bp = append(bp, Component{Retain: len(c.Insert)})
			continue
		}
		// Only b inserts.
		if ib.peekType() == compInsert {
			c := ib.take(0)
			bp = append(bp, Component{Insert: c.Insert})
			ap = append(ap, Component{Retain: len(c.Insert)})
			continue
		}

		// Both consume input. Take the shorter chunk.
		if !ia.hasNext() || !ib.hasNext() {
			return Operation{}, Operation{}, fmt.Errorf("transform ran out of operations")
		}

		n := min(ia.peekLen(), ib.peekLen())
		ca := ia.take(n)
		cb := ib.take(n)

		switch {
		case ca.IsRetain() && cb.IsRetain():
			ap = append(ap, Component{Retain: n})
			bp = append(bp, Component{Retain: n})
		case ca.IsDelete() && cb.IsRetain():
			ap = append(ap, Component{Delete: n})
		case ca.IsRetain() && cb.IsDelete():
			bp = append(bp, Component{Delete: n})
		case ca.IsDelete() && cb.IsDelete():
			// Both delete same chars — nothing to do.
		}
	}

	return Operation{Ops: compact(ap)}, Operation{Ops: compact(bp)}, nil
}

// compact merges adjacent components of the same type.
func compact(ops []Component) []Component {
	if len(ops) == 0 {
		return ops
	}
	var result []Component
	for _, c := range ops {
		if len(result) == 0 {
			result = append(result, c)
			continue
		}
		last := &result[len(result)-1]
		if c.IsRetain() && last.IsRetain() {
			last.Retain += c.Retain
		} else if c.IsDelete() && last.IsDelete() {
			last.Delete += c.Delete
		} else if c.IsInsert() && last.IsInsert() {
			last.Insert += c.Insert
		} else {
			result = append(result, c)
		}
	}
	return result
}

// compType identifies a component kind for the iterator.
type compType int

const (
	compNone compType = iota
	compRetain
	compInsert
	compDelete
)

// iter walks through operation components, allowing partial consumption.
type iter struct {
	ops    []Component
	index  int
	offset int
}

func newIter(ops []Component) *iter {
	return &iter{ops: ops}
}

func (it *iter) hasNext() bool {
	return it.index < len(it.ops)
}

func (it *iter) peekType() compType {
	if !it.hasNext() {
		return compNone
	}
	c := it.ops[it.index]
	switch {
	case c.IsInsert():
		return compInsert
	case c.IsDelete():
		return compDelete
	default:
		return compRetain
	}
}

func (it *iter) peekLen() int {
	if !it.hasNext() {
		return 0
	}
	c := it.ops[it.index]
	switch {
	case c.IsRetain():
		return c.Retain - it.offset
	case c.IsInsert():
		return len(c.Insert) - it.offset
	case c.IsDelete():
		return c.Delete - it.offset
	}
	return 0
}

// take consumes n units from the current component. For inserts, n=0 means take all.
func (it *iter) take(n int) Component {
	c := it.ops[it.index]
	remaining := it.peekLen()

	switch {
	case c.IsRetain():
		if n >= remaining {
			it.index++
			it.offset = 0
			return Component{Retain: remaining}
		}
		it.offset += n
		return Component{Retain: n}

	case c.IsInsert():
		if n == 0 || n >= remaining {
			s := c.Insert[it.offset:]
			it.index++
			it.offset = 0
			return Component{Insert: s}
		}
		s := c.Insert[it.offset : it.offset+n]
		it.offset += n
		return Component{Insert: s}

	case c.IsDelete():
		if n >= remaining {
			it.index++
			it.offset = 0
			return Component{Delete: remaining}
		}
		it.offset += n
		return Component{Delete: n}
	}

	it.index++
	return Component{}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
