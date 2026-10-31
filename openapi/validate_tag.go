package openapi

import (
	"fmt"
	"reflect"

	"github.com/wI2L/fizz/openapi/internal/vtag"
)

// updateSchemaValidation fills the JSON Schema validation keywords of
// schema from the validate tag of sf (go-playground/validator v9
// grammar, parsed by package vtag).
//
// Rules before the first "dive" constrain the container itself; rules
// after it constrain "items" (slice, array) or "additionalProperties"
// (map), recursively for nested dives, and "keys...endkeys" rules
// constrain "propertyNames". An item or value schema that is a $ref is
// never modified: it is copied at the use site, as annotations are.
//
// When two rules set the same keyword, both must hold, as in
// validator: lower bounds keep the larger value, upper bounds the
// smaller one, and any other conflicting keyword moves to an allOf.
//
// A malformed tag is recorded as a *FieldError and no rule is applied.
func (g *Generator) updateSchemaValidation(schema *Schema, sf reflect.StructField) *Schema {
	ts := sf.Tag.Get(BinderTags.Validate)
	if ts == "" {
		return schema
	}
	tag, err := vtag.Parse(ts)
	if err != nil {
		g.fieldErr(sf, fmt.Sprintf("invalid validate tag %q", ts), err)
		return schema
	}
	if tag.Skip {
		return schema
	}
	g.applyLevel(schema, sf.Type, &tag.Root, sf)
	return schema
}

// fieldErr records a *FieldError for a validate tag problem.
func (g *Generator) fieldErr(sf reflect.StructField, msg string, err error) {
	g.error(&FieldError{
		Message:  msg,
		Name:     sf.Name,
		Field:    sf.Name,
		Type:     sf.Type,
		TypeName: g.typeName(sf.Type),
		Err:      err,
	})
}

// applyLevel applies the rules of l to s, which describes a value of
// type t, then descends into the dive and keys levels.
func (g *Generator) applyLevel(s *Schema, t reflect.Type, l *vtag.Level, sf reflect.StructField) {
	t = derefType(t)
	for _, grp := range l.Rules {
		g.applyGroup(s, t, grp, sf)
	}
	if l.Dive == nil {
		return
	}
	kind := t.Kind()
	switch {
	case isBytes(t) || (kind != reflect.Slice && kind != reflect.Array && kind != reflect.Map):
		g.fieldErr(sf, "validate tag: dive on a value that is not a slice, array or map", nil)
	case kind == reflect.Map:
		if k := l.Dive.Keys; k != nil {
			pn := &Schema{}
			g.applyLevel(pn, t.Key(), k, sf)
			if !reflect.DeepEqual(pn, &Schema{}) {
				s.PropertyNames = pn
			}
		}
		cur := &Schema{}
		if s.AdditionalProperties != nil && s.AdditionalProperties.Bool == nil {
			cur = g.useSite(s.AdditionalProperties)
		}
		g.applyElem(cur, t.Elem(), l.Dive, sf)
		if !reflect.DeepEqual(cur, &Schema{}) && !reflect.DeepEqual(cur, s.AdditionalProperties) {
			s.AdditionalProperties = cur
		}
	default:
		cur := &Schema{}
		if s.Items != nil {
			cur = g.useSite(s.Items)
		}
		g.applyElem(cur, t.Elem(), l.Dive, sf)
		if !reflect.DeepEqual(cur, &Schema{}) && !reflect.DeepEqual(cur, s.Items) {
			s.Items = cur
		}
	}
}

// applyElem applies the element level l to cur. The Keys of l were
// already consumed by the map above.
func (g *Generator) applyElem(cur *Schema, et reflect.Type, l *vtag.Level, sf reflect.StructField) {
	inner := *l
	inner.Keys = nil
	g.applyLevel(cur, et, &inner, sf)
}

// useSite returns a copy of an item or value schema that can receive
// keywords without touching a shared component: a reference is
// wrapped in an allOf in the AllOfWrap style, and otherwise copied
// (keywords become siblings of the $ref).
func (g *Generator) useSite(s *Schema) *Schema {
	if s.Ref != "" && g.annotationStyle == AllOfWrap {
		ref := *s
		return &Schema{AllOf: []*Schema{&ref}}
	}
	return s.clone()
}

// applyGroup applies one AND-ed rule group to s. A group of several
// alternatives (a|b) becomes an anyOf of single-keyword schemas, only
// if every alternative maps to a keyword; otherwise it is dropped.
func (g *Generator) applyGroup(s *Schema, t reflect.Type, grp vtag.Group, sf reflect.StructField) {
	if len(grp) == 1 {
		tmp := &Schema{}
		if g.mapRule(tmp, t, grp[0], sf) {
			mergeSchema(s, tmp)
		}
		return
	}
	alts := make([]*Schema, 0, len(grp))
	for _, r := range grp {
		tmp := &Schema{}
		if !g.mapRule(tmp, t, r, sf) {
			return
		}
		alts = append(alts, tmp)
	}
	if len(s.AnyOf) == 0 {
		s.AnyOf = alts
		return
	}
	// s already has an anyOf (nullable reference): AND the new one.
	s.AllOf = append(s.AllOf, &Schema{AnyOf: alts})
}

// mapRule translates r, applied to a value of type t, into the
// keywords of s. It reports whether any keyword was produced. Rules
// with an invalid parameter are recorded as a *FieldError and dropped.
func (g *Generator) mapRule(s *Schema, t reflect.Type, r vtag.Rule, sf reflect.StructField) bool {
	k := kindOf(t)
	fail := func(err error) bool {
		g.fieldErr(sf, fmt.Sprintf("validate rule %q cannot be applied", ruleString(r)), err)
		return false
	}
	bothBounds := func(dst *Schema) bool {
		return setBound(dst, t, k, "min", r.Param, fail) && setBound(dst, t, k, "max", r.Param, fail)
	}
	switch r.Name {
	case "min", "gte", "max", "lte", "gt", "lt":
		return setBound(s, t, k, r.Name, r.Param, fail)
	case "len":
		switch k {
		case kindString, kindSlice, kindMap:
			return bothBounds(s)
		}
		return setConst(s, k, r.Param, fail)
	case "eq":
		switch k {
		case kindSlice, kindMap:
			return bothBounds(s)
		}
		return setConst(s, k, r.Param, fail)
	case "ne":
		not := &Schema{}
		var ok bool
		switch k {
		case kindSlice, kindMap:
			ok = bothBounds(not)
		default:
			ok = setConst(not, k, r.Param, fail)
		}
		if ok {
			s.Not = not
		}
		return ok
	case "oneof":
		return setOneOf(s, k, r.Param, fail)
	case "email":
		return setFormat(s, k, "email")
	case "url", "uri":
		return setFormat(s, k, "uri")
	case "uuid", "uuid3", "uuid4", "uuid5":
		return setFormat(s, k, "uuid")
	case "ipv4", "ipv6", "hostname":
		return setFormat(s, k, r.Name)
	case "alpha", "alphanum", "numeric", "hexadecimal":
		if k != kindString {
			return false
		}
		s.Pattern = rulePatterns[r.Name]
		return true
	case "unique":
		if k != kindSlice {
			return false
		}
		s.UniqueItems = true
		return true
	}
	// omitempty, required, isdefault, structonly, nostructlevel and
	// unknown rules have no keyword.
	return false
}
