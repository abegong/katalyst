package inspect

import "unicode/utf8"

// maxValueSet caps how many distinct values object_field_values will enumerate.
// Above it, only the cardinality is reported — a wide-open field is not an enum
// candidate, so listing every value adds noise.
const maxValueSet = 12

// ObjectFieldFrequency reports, per frontmatter key, how many files contain it.
// Presence over N is the signal an agent weighs when deciding required vs
// optional — the inspector reports the count, never the decision.
type ObjectFieldFrequency struct{}

func (ObjectFieldFrequency) Name() string { return "object_field_frequency" }

func (ObjectFieldFrequency) Inspect(c Corpus) Evidence {
	present := map[string]int{}
	for _, f := range c.Files {
		for k := range meta(f) {
			present[k]++
		}
	}
	data := make(map[string]any, len(present))
	for k, n := range present {
		data[k] = map[string]any{"present": n}
	}
	return Evidence{Inspector: "object_field_frequency", Scope: c.Scope, N: len(c.Files), Data: data}
}

// ObjectFieldTypes reports, per key, the histogram of observed value types. A
// key seen as both string and integer is reported as both, not first-wins, so
// an agent can spot an inconsistent field.
type ObjectFieldTypes struct{}

func (ObjectFieldTypes) Name() string { return "object_field_types" }

func (ObjectFieldTypes) Inspect(c Corpus) Evidence {
	hist := map[string]map[string]int{}
	eachValue(c, func(k string, v any) {
		if hist[k] == nil {
			hist[k] = map[string]int{}
		}
		hist[k][jsonType(v)]++
	})
	data := make(map[string]any, len(hist))
	for k, h := range hist {
		types := make(map[string]any, len(h))
		for typ, n := range h {
			types[typ] = n
		}
		data[k] = map[string]any{"types": types}
	}
	return Evidence{Inspector: "object_field_types", Scope: c.Scope, N: len(c.Files), Data: data}
}

// ObjectFieldValues reports, per key, the number of distinct scalar values and
// — when that set is small enough to be an enum candidate — the values
// themselves with their counts. Array and object values are not scalars and so
// never enter the value set.
type ObjectFieldValues struct{}

func (ObjectFieldValues) Name() string { return "object_field_values" }

func (ObjectFieldValues) Inspect(c Corpus) Evidence {
	type acc struct {
		distinct  map[string]int
		nonScalar bool
	}
	accs := map[string]*acc{}
	eachValue(c, func(k string, v any) {
		a := accs[k]
		if a == nil {
			a = &acc{distinct: map[string]int{}}
			accs[k] = a
		}
		if s, ok := scalarString(v); ok {
			a.distinct[s]++
		} else {
			a.nonScalar = true
		}
	})
	data := make(map[string]any, len(accs))
	for k, a := range accs {
		entry := map[string]any{"cardinality": len(a.distinct)}
		if !a.nonScalar && len(a.distinct) > 0 && len(a.distinct) <= maxValueSet {
			values := make(map[string]any, len(a.distinct))
			for s, n := range a.distinct {
				values[s] = n
			}
			entry["values"] = values
		}
		data[k] = entry
	}
	return Evidence{Inspector: "object_field_values", Scope: c.Scope, N: len(c.Files), Data: data}
}

// ObjectFieldNumericRange reports, per key with numeric observations, the
// observed min and max and how many values were numeric.
type ObjectFieldNumericRange struct{}

func (ObjectFieldNumericRange) Name() string { return "object_field_numeric_range" }

func (ObjectFieldNumericRange) Inspect(c Corpus) Evidence {
	type rng struct {
		min, max float64
		count    int
	}
	ranges := map[string]*rng{}
	eachValue(c, func(k string, v any) {
		f, ok := toFloat(v)
		if !ok {
			return
		}
		r := ranges[k]
		if r == nil {
			ranges[k] = &rng{min: f, max: f, count: 1}
			return
		}
		if f < r.min {
			r.min = f
		}
		if f > r.max {
			r.max = f
		}
		r.count++
	})
	data := make(map[string]any, len(ranges))
	for k, r := range ranges {
		data[k] = map[string]any{"min": r.min, "max": r.max, "count": r.count}
	}
	return Evidence{Inspector: "object_field_numeric_range", Scope: c.Scope, N: len(c.Files), Data: data}
}

// ObjectFieldStringLength reports, per key with string observations, the
// observed minimum and maximum length (in runes) and how many values were
// strings.
type ObjectFieldStringLength struct{}

func (ObjectFieldStringLength) Name() string { return "object_field_string_length" }

func (ObjectFieldStringLength) Inspect(c Corpus) Evidence {
	type lens struct {
		min, max, count int
	}
	all := map[string]*lens{}
	eachValue(c, func(k string, v any) {
		s, ok := v.(string)
		if !ok {
			return
		}
		n := utf8.RuneCountInString(s)
		l := all[k]
		if l == nil {
			all[k] = &lens{min: n, max: n, count: 1}
			return
		}
		if n < l.min {
			l.min = n
		}
		if n > l.max {
			l.max = n
		}
		l.count++
	})
	data := make(map[string]any, len(all))
	for k, l := range all {
		data[k] = map[string]any{"min_length": l.min, "max_length": l.max, "count": l.count}
	}
	return Evidence{Inspector: "object_field_string_length", Scope: c.Scope, N: len(c.Files), Data: data}
}
