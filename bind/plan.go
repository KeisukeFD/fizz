package bind

import (
	"encoding"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

// Tags names the struct tags the planner reads. Each field is the tag key
// (for example "query"), not a tag value. It is owned by this package so
// that bind does not depend on any router or on the openapi package.
type Tags struct {
	// Path, Header and Explode are the tonic tags of path and header
	// parameters and of the explode option. Empty disables the planning of
	// path parameters or of header defaults, and reads every explode as
	// unset.
	Query, Path, Header, Cookie, Style, Explode, Default string
}

// Plan is the precomputed decoding recipe for one struct type. It is
// immutable and safe for concurrent use. Obtain one with PlanFor.
type Plan struct {
	t      reflect.Type // the struct type, never a pointer
	fields []planField  // deepObject query fields, in declaration order
	// cookies are the cookie fields, in declaration order.
	cookies []cookieField
	// lists are the spaceDelimited and pipeDelimited query arrays.
	lists []listField
	// paths are the label and matrix path parameters.
	paths []pathField
	// headers are the scalar header fields that have a default tag.
	headers []headerField
}

// headerField is a header parameter whose absence takes the default tag.
type headerField struct {
	name string // header name
	def  string // the default tag value, never empty
}

// listField is a query array documented as spaceDelimited or
// pipeDelimited with explode=false.
type listField struct {
	name  string // query parameter name
	delim string // " " or "|"
	// tonicExploded reports whether tonic reads the field as one value per
	// occurrence (its explode tag is not false). Otherwise tonic splits
	// one value on commas.
	tonicExploded bool
}

// pathField is a scalar path parameter documented as label or matrix.
type pathField struct {
	name   string
	matrix bool // matrix, else label
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

// cookieField is one scalar cookie parameter.
type cookieField struct {
	name   string       // cookie name
	index  []int        // path to the field (through embedded structs)
	ptr    bool         // the field is a pointer to the scalar
	typ    reflect.Type // the scalar type (pointer stripped)
	def    string       // the default tag value, when hasDef
	hasDef bool
	// required is the ",required" option of the cookie tag. A default
	// makes the cookie optional, as the generator documents (D15).
	required bool
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
// pointer to one. Only exported fields are planned: those whose
// Tags.Query tag names a parameter and whose Tags.Style tag is exactly
// "deepObject", and scalar fields (or pointers to scalars) with a
// Tags.Cookie tag. Every other field is left to the router's own binder.
// Results are cached per (t, tags), so repeated calls return the same
// *Plan.
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

// Empty reports whether p plans no parameter, so that Query and Cookies
// would leave every field alone. A nil Plan is empty.
func (p *Plan) Empty() bool {
	return p == nil || len(p.fields) == 0 && len(p.cookies) == 0 && len(p.lists) == 0 && len(p.paths) == 0 && len(p.headers) == 0
}

func buildPlan(t reflect.Type, tags Tags) (*Plan, error) {
	st := t
	if st.Kind() == reflect.Pointer {
		st = st.Elem()
	}
	if st.Kind() != reflect.Struct {
		return nil, fmt.Errorf("bind: PlanFor: %s is not a struct or pointer to struct", t)
	}
	p := &Plan{t: st}
	if err := p.collect(st, nil, tags, map[reflect.Type]bool{st: true}); err != nil {
		return nil, err
	}
	return p, nil
}

// collect plans the fields of st. Like the generator's parameter walk, it
// descends into every anonymous struct field, or pointer to struct, whatever
// its tags and whether or not its type is exported; such a field is never a
// parameter itself. active holds the struct types being walked, so that
// embedding cycles end. An unexported embedded pointer cannot be allocated
// through reflection, so its fields are left alone; the generator reports
// them instead of documenting them.
func (p *Plan) collect(st reflect.Type, prefix []int, tags Tags, active map[reflect.Type]bool) error {
	for i := range st.NumField() {
		f := st.Field(i)
		index := append(append([]int(nil), prefix...), i)
		if f.Anonymous {
			et := f.Type
			if et.Kind() == reflect.Pointer {
				if !f.IsExported() {
					continue
				}
				et = et.Elem()
			}
			if et.Kind() != reflect.Struct || active[et] {
				continue
			}
			active[et] = true
			err := p.collect(et, index, tags, active)
			delete(active, et)
			if err != nil {
				return err
			}
			continue
		}
		if !f.IsExported() {
			continue
		}
		if tags.Cookie != "" {
			if v, ok := f.Tag.Lookup(tags.Cookie); ok {
				p.addCookie(f, v, index, tags)
				continue
			}
		}
		if tags.Header != "" {
			if v, ok := f.Tag.Lookup(tags.Header); ok {
				p.addHeader(f, v, tags)
				continue
			}
		}
		style := f.Tag.Get(tags.Style)
		if tags.Path != "" && (style == "label" || style == "matrix") {
			p.addPath(f, style, tags)
			continue
		}
		if style == "spaceDelimited" || style == "pipeDelimited" {
			p.addList(f, style, tags)
			continue
		}
		if style != "deepObject" {
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

// fieldByIndex is rv.FieldByIndex, allocating the nil pointers it crosses.
// The plan only crosses exported embedded pointers.
func fieldByIndex(rv reflect.Value, index []int) reflect.Value {
	for n, i := range index {
		if n > 0 && rv.Kind() == reflect.Pointer {
			if rv.IsNil() {
				rv.Set(reflect.New(rv.Type().Elem()))
			}
			rv = rv.Elem()
		}
		rv = rv.Field(i)
	}
	return rv
}

// addCookie plans a cookie field. The generator rejects a cookie parameter
// that is not a scalar, so such a field is left alone here.
func (p *Plan) addCookie(f reflect.StructField, tag string, index []int, tags Tags) {
	name, options, _ := strings.Cut(tag, ",")
	if name == "" || name == "-" {
		return
	}
	cf := cookieField{name: name, index: index, typ: f.Type}
	if cf.typ.Kind() == reflect.Pointer {
		cf.ptr = true
		cf.typ = cf.typ.Elem()
	}
	if !isScalar(cf.typ) {
		return
	}
	if tags.Default != "" {
		cf.def, cf.hasDef = f.Tag.Lookup(tags.Default)
	}
	// As in tonic's extractors, the ",default=" option wins over the
	// default tag, and an empty option is not a default. The generator
	// reports an unknown option, so it is ignored here.
	for o := range strings.SplitSeq(options, ",") {
		o = strings.TrimSpace(o)
		if o == "required" {
			cf.required = true
		} else if v, ok := strings.CutPrefix(o, "default="); ok && v != "" {
			cf.def, cf.hasDef = v, true
		}
	}
	p.cookies = append(p.cookies, cf)
}

// addHeader plans a header field that has a default tag. It mirrors the
// generator: a header is one value, so a slice is not planned (the
// generator rejects it), and an empty default is no default. The
// ",default=" option of the header tag is applied by tonic, to an absent or
// empty header, and wins over the tag, so such a field is left alone.
func (p *Plan) addHeader(f reflect.StructField, tag string, tags Tags) {
	name, options, _ := strings.Cut(tag, ",")
	if name == "" || name == "-" || tags.Default == "" {
		return
	}
	def := f.Tag.Get(tags.Default)
	if def == "" {
		return
	}
	t := f.Type
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if !isScalar(t) {
		return
	}
	for o := range strings.SplitSeq(options, ",") {
		if v, ok := strings.CutPrefix(strings.TrimSpace(o), "default="); ok && v != "" {
			return
		}
	}
	p.headers = append(p.headers, headerField{name: name, def: def})
}

// addList plans a spaceDelimited or pipeDelimited query array. It mirrors
// the generator (openapi/params.go): the style needs a slice or array, and
// the array is documented unexploded unless an explode tag says otherwise
// (any value that is not a false boolean).
func (p *Plan) addList(f reflect.StructField, style string, tags Tags) {
	name, _, _ := strings.Cut(f.Tag.Get(tags.Query), ",")
	if name == "" || name == "-" {
		return
	}
	t := f.Type
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if (t.Kind() != reflect.Slice && t.Kind() != reflect.Array) || t.Elem().Kind() == reflect.Uint8 {
		return
	}
	explode, set := "", false
	if tags.Explode != "" {
		explode, set = f.Tag.Lookup(tags.Explode)
	}
	b, err := strconv.ParseBool(explode)
	tonicExploded := !(set && err == nil && !b)
	if set && tonicExploded {
		return // documented exploded: the repeated form that tonic reads
	}
	delim := " "
	if style == "pipeDelimited" {
		delim = "|"
	}
	p.lists = append(p.lists, listField{name: name, delim: delim, tonicExploded: tonicExploded})
}

// addPath plans a label or matrix path parameter. Only scalars are
// planned: tonic reads a path parameter as one value, so it cannot bind an
// array from it.
func (p *Plan) addPath(f reflect.StructField, style string, tags Tags) {
	name, _, _ := strings.Cut(f.Tag.Get(tags.Path), ",")
	if name == "" || name == "-" {
		return
	}
	t := f.Type
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if !isScalar(t) {
		return
	}
	p.paths = append(p.paths, pathField{name: name, matrix: style == "matrix"})
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
