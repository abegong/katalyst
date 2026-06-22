package project

import (
	"fmt"
	"strings"

	"github.com/abegong/katalyst/internal/config"
)

// UsageError signals an exit-code-2 condition: an unknown or
// wrong-depth selector. The cmd layer maps it to a usage exit.
type UsageError struct{ Msg string }

func (e *UsageError) Error() string { return e.Msg }

// Selector identifies a target by depth (see docs/explanation/domain-model.md):
//
//	<collection>          → one collection (Item == "")
//	<collection>/<item>   → one item
//
// The first segment is always a collection; a bare token is never an item.
type Selector struct {
	Raw        string
	Collection string
	Item       string
}

// IsItem reports whether the selector addresses a single item.
func (s Selector) IsItem() bool { return s.Item != "" }

// Depth is 1 for a collection selector, 2 for an item selector.
func (s Selector) Depth() int {
	if s.IsItem() {
		return 2
	}
	return 1
}

// ParseSelector splits a raw selector into its segments and validates its
// shape. It rejects empty selectors, empty segments, and depth > 2.
func ParseSelector(raw string) (Selector, error) {
	if raw == "" {
		return Selector{}, &UsageError{Msg: "empty selector"}
	}
	parts := strings.Split(raw, "/")
	if len(parts) > 2 {
		return Selector{}, &UsageError{Msg: fmt.Sprintf("invalid selector %q: too deep (expected <collection> or <collection>/<item>)", raw)}
	}
	for _, p := range parts {
		if p == "" {
			return Selector{}, &UsageError{Msg: fmt.Sprintf("invalid selector %q: empty segment", raw)}
		}
	}
	s := Selector{Raw: raw, Collection: parts[0]}
	if len(parts) == 2 {
		s.Item = parts[1]
	}
	return s, nil
}

// Resolution is the expansion of one or more selectors for the blessed
// verbs (check, fix).
type Resolution struct {
	// Items to process, de-duplicated, in resolution order.
	Items []Item
	// Scan holds collections that were selected wholesale (empty or
	// collection-level selectors), so callers can scan them for unmatched
	// references. De-duplicated, in resolution order.
	Scan []config.Collection
}

// Resolve expands selectors into items. With no selectors, it selects
// every collection (and all their items). Unknown collections/items and
// malformed selectors return a *UsageError.
func (p *Project) Resolve(selectors []string) (*Resolution, error) {
	res := &Resolution{}
	seenItem := map[string]bool{}
	seenScan := map[string]bool{}

	addItem := func(it Item) {
		if seenItem[it.Path] {
			return
		}
		seenItem[it.Path] = true
		res.Items = append(res.Items, it)
	}
	addScan := func(c config.Collection) {
		if seenScan[c.Name] {
			return
		}
		seenScan[c.Name] = true
		res.Scan = append(res.Scan, c)
	}

	// No selectors → the whole project.
	if len(selectors) == 0 {
		for _, c := range p.cfg.Collections {
			addScan(c)
			items, err := p.Items(c)
			if err != nil {
				return nil, err
			}
			for _, it := range items {
				addItem(it)
			}
		}
		return res, nil
	}

	for _, raw := range selectors {
		sel, err := ParseSelector(raw)
		if err != nil {
			return nil, err
		}
		c, ok := p.cfg.Collection(sel.Collection)
		if !ok {
			return nil, &UsageError{Msg: fmt.Sprintf("unknown collection %q", sel.Collection)}
		}
		if sel.IsItem() {
			it, err := p.ItemAt(sel.Collection, sel.Item)
			if err != nil {
				return nil, err
			}
			addItem(it)
			continue
		}
		// Collection-level selector: all its items, and mark for scanning.
		addScan(c)
		items, err := p.Items(c)
		if err != nil {
			return nil, err
		}
		for _, it := range items {
			addItem(it)
		}
	}
	return res, nil
}
