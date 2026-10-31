package bind

import (
	"encoding"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// Tags names the struct tags the planner reads. Each field is the tag key
// (for example "query"), not a tag value. It is owned by this package so
// that bind does not depend on any router or on the openapi package.
type Tags struct {
	Query, Cookie, Style, Default string
}

// Plan is the precomputed decoding recipe for one struct type. It is
// immutable and safe for concurrent use. Obtain one with PlanFor.
type Plan struct {
	t      reflect.Type // the struct type, never a pointer
	fields []planField  // deepObject query fields, in declaration order
}

type planKind int

const (
	kindMap planKind = iota
	kindStruct
)

// planField is one deepObject query parameter.
type planField struct {
	name  string // query parameter name
	index []int  // path to the field (through embedded structs)
	ptr   bool   // the field is a pointer to the map or struct
	kind  planKind
	typ   reflect.Type // the map or struct type (pointer stripped)
	// members maps a json name to the struct member, for kindStruct.
	members map[string]member
}

type member struct {
	index int
	typ   reflect.Type
}

type planKey struct {
	t    reflect.Type
	tags Tags
}

type planEntry struct {
	plan *Plan
	err  error
}

var planCache sync.Map // planKey -> planEntry

// PlanFor returns the decoding plan for t, which must be a struct or a
// pointer to one. Only exported fields whose Tags.Query tag names a
// parameter and whose Tags.Style tag is exactly "deepObject" are planned;
// every other field is left to the router's own binder. Results are cached
// per (t, tags), so repeated calls return the same *Plan.
func PlanFor(t reflect.Type, tags Tags) (*Plan, error) {
	if t == nil {
		return nil, fmt.Errorf("bind: PlanFor: nil type")
	}
	key := planKey{t, tags}
	if v, ok := planCache.Load(key); ok {
		e := v.(planEntry)
		return e.plan, e.err
	}
	p, err := buildPlan(t, tags)
	v, _ := planCache.LoadOrStore(key, planEntry{p, err})
	e := v.(planEntry)
	return e.plan, e.err
}

// Empty reports whether p plans no parameter, so that Query would leave
// every field alone. A nil Plan is empty.
func (p *Plan) Empty() bool { return p == nil || len(p.fields) == 0 }

func buildPlan(t reflect.Type, tags Tags) (*Plan, error) {
	st := t
	if st.Kind() == reflect.Pointer {
		st = st.Elem()
	}
	if st.Kind() != reflect.Struct {
		return nil, fmt.Errorf("bind: PlanFor: %s is not a struct or pointer to struct", t)
	}
	p := &Plan{t: st}
	if tags.Query == "" || tags.Style == "" {
		return p, nil
	}
	if err := p.collect(st, nil, tags); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *Plan) collect(st reflect.Type, prefix []int, tags Tags) error {
	for i := range st.NumField() {
		f := st.Field(i)
		index := append(append([]int(nil), prefix...), i)
		if !f.IsExported() {
			continue
		}
		if f.Tag.Get(tags.Style) != "deepObject" {
			// Embedded structs without a query tag have promoted fields.
			if f.Anonymous && f.Type.Kind() == reflect.Struct && f.Tag.Get(tags.Query) == "" {
				if err := p.collect(f.Type, index, tags); err != nil {
					return err
				}
			}
			continue
		}
		name, _, _ := strings.Cut(f.Tag.Get(tags.Query), ",")
		if name == "" || name == "-" {
			continue
		}
		pf, err := planDeepObject(f, name)
		if err != nil {
			return err
		}
		pf.index = index
		p.fields = append(p.fields, pf)
	}
	return nil
}

func planDeepObject(f reflect.StructField, name string) (planField, error) {
	pf := planField{name: name}
	typ := f.Type
	if typ.Kind() == reflect.Pointer {
		pf.ptr = true
		typ = typ.Elem()
	}
	pf.typ = typ
	switch typ.Kind() {
	case reflect.Map:
		if typ.Key().Kind() != reflect.String {
			return pf, fmt.Errorf("bind: field %s: deepObject map key must be a string, got %s", f.Name, typ.Key())
		}
		if !isValueType(typ.Elem()) {
			return pf, fmt.Errorf("bind: field %s: deepObject map value %s must be a scalar or a slice of scalars", f.Name, typ.Elem())
		}
		pf.kind = kindMap
	case reflect.Struct:
		pf.kind = kindStruct
		pf.members = make(map[string]member)
		for i := range typ.NumField() {
			sf := typ.Field(i)
			if !sf.IsExported() {
				continue
			}
			jn, _, _ := strings.Cut(sf.Tag.Get("json"), ",")
			if jn == "-" {
				continue
			}
			if jn == "" {
				jn = sf.Name
			}
			if !isValueType(sf.Type) {
				return pf, fmt.Errorf("bind: field %s: deepObject struct member %s has type %s, want a scalar or a slice of scalars", f.Name, sf.Name, sf.Type)
			}
			pf.members[jn] = member{index: i, typ: sf.Type}
		}
	default:
		return pf, fmt.Errorf("bind: field %s: deepObject type %s must be a map[string]T, a struct or a pointer to one", f.Name, f.Type)
	}
	return pf, nil
}

var textUnmarshaler = reflect.TypeFor[encoding.TextUnmarshaler]()

// isScalar reports whether t can be decoded from one string.
func isScalar(t reflect.Type) bool {
	if reflect.PointerTo(t).Implements(textUnmarshaler) {
		return true
	}
	switch t.Kind() {
	case reflect.String, reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	}
	return false
}

// isValueType reports whether t is a scalar or a slice of scalars.
func isValueType(t reflect.Type) bool {
	if isScalar(t) {
		return true
	}
	return t.Kind() == reflect.Slice && isScalar(t.Elem())
}
