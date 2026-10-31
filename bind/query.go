package bind

import (
	"encoding"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

var (
	// ErrNestedKey is returned (wrapped) for a deepObject key with more
	// than one bracket pair, such as filter[a][b].
	ErrNestedKey = errors.New("bind: nested deepObject keys are not supported")
	// ErrTooManyKeys is returned (wrapped) when WithMaxKeys is exceeded.
	ErrTooManyKeys = errors.New("bind: too many deepObject keys")
)

type queryConfig struct {
	maxKeys int
}

// QueryOption configures Query.
type QueryOption func(*queryConfig)

// WithMaxKeys caps the number of deepObject keys routed across one
// request. n <= 0 means no cap (net/url already limits a query to 10000
// parameters).
func WithMaxKeys(n int) QueryOption {
	return func(c *queryConfig) { c.maxKeys = n }
}

type route struct {
	key  string // decoded key between the brackets
	raw  string // raw query key, for error messages
	vals []string
}

// Query decodes the deepObject parameters of p from r's query string into
// dst, a non-nil pointer to the struct type p was planned for. Keys have
// the form name[key]. Only planned fields are written; a map or pointer
// field stays nil when none of its keys is present.
func Query(r *http.Request, dst any, p *Plan, opts ...QueryOption) error {
	if p == nil {
		return errors.New("bind: Query: nil plan")
	}
	rv := reflect.ValueOf(dst)
	if rv.Kind() != reflect.Pointer || rv.IsNil() || rv.Elem().Type() != p.t {
		return fmt.Errorf("bind: Query: dst must be a non-nil *%s, got %T", p.t, dst)
	}
	if len(p.fields) == 0 {
		return nil
	}
	var cfg queryConfig
	for _, o := range opts {
		o(&cfg)
	}
	q, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return fmt.Errorf("bind: invalid query string: %w", err)
	}
	rv = rv.Elem()

	rawKeys := make([]string, 0, len(q))
	for k := range q {
		rawKeys = append(rawKeys, k)
	}
	slices.Sort(rawKeys) // deterministic errors

	routes := make([][]route, len(p.fields))
	total := 0
	for _, raw := range rawKeys {
		for i := range p.fields {
			key, ok, err := parseKey(raw, p.fields[i].name)
			if err != nil {
				return err
			}
			if !ok {
				continue
			}
			total++
			if cfg.maxKeys > 0 && total > cfg.maxKeys {
				return fmt.Errorf("%w: more than %d keys (at %q)", ErrTooManyKeys, cfg.maxKeys, raw)
			}
			routes[i] = append(routes[i], route{key: key, raw: raw, vals: q[raw]})
			break
		}
	}
	for i := range p.fields {
		if len(routes[i]) == 0 {
			continue
		}
		if err := p.fields[i].apply(rv, routes[i]); err != nil {
			return err
		}
	}
	return nil
}

// parseKey reports whether raw belongs to parameter name and extracts the
// key between the brackets.
func parseKey(raw, name string) (key string, ok bool, err error) {
	if raw == name {
		return "", false, fmt.Errorf("bind: query parameter %q: deepObject parameter needs a key, want %s[key]", raw, name)
	}
	rest, found := strings.CutPrefix(raw, name+"[")
	if !found {
		return "", false, nil
	}
	end := strings.IndexByte(rest, ']')
	switch {
	case end < 0:
		return "", false, fmt.Errorf("bind: query parameter %q: unclosed bracket", raw)
	case end == 0:
		return "", false, fmt.Errorf("bind: query parameter %q: empty key", raw)
	case strings.IndexByte(rest[:end], '[') >= 0:
		return "", false, fmt.Errorf("bind: query parameter %q: malformed key", raw)
	}
	switch tail := rest[end+1:]; {
	case tail == "":
	case tail[0] == '[':
		return "", false, fmt.Errorf("query parameter %q: %w", raw, ErrNestedKey)
	default:
		return "", false, fmt.Errorf("bind: query parameter %q: unexpected %q after ]", raw, tail)
	}
	return rest[:end], true, nil
}

func (pf *planField) apply(rv reflect.Value, routes []route) error {
	fv := rv.FieldByIndex(pf.index)
	target := fv
	if pf.ptr {
		fv.Set(reflect.New(pf.typ))
		target = fv.Elem()
	}
	switch pf.kind {
	case kindMap:
		m := reflect.MakeMapWithSize(pf.typ, len(routes))
		for _, rt := range routes {
			elem := reflect.New(pf.typ.Elem()).Elem()
			if err := setValue(elem, rt); err != nil {
				return err
			}
			m.SetMapIndex(reflect.ValueOf(rt.key).Convert(pf.typ.Key()), elem)
		}
		target.Set(m)
	case kindStruct:
		target.SetZero()
		for _, rt := range routes {
			mem, ok := pf.members[rt.key]
			if !ok {
				continue
			}
			if err := setValue(target.Field(mem.index), rt); err != nil {
				return err
			}
		}
	}
	return nil
}

// setValue stores rt's values into v, a scalar or a slice of scalars.
func setValue(v reflect.Value, rt route) error {
	if v.Kind() == reflect.Slice && !implementsText(v.Type()) {
		s := reflect.MakeSlice(v.Type(), 0, len(rt.vals))
		for _, str := range rt.vals {
			e := reflect.New(v.Type().Elem()).Elem()
			if err := setScalar(e, str, rt.raw); err != nil {
				return err
			}
			s = reflect.Append(s, e)
		}
		v.Set(s)
		return nil
	}
	if len(rt.vals) != 1 {
		return fmt.Errorf("bind: query parameter %q: got %d values for a scalar, want 1", rt.raw, len(rt.vals))
	}
	return setScalar(v, rt.vals[0], rt.raw)
}

func implementsText(t reflect.Type) bool {
	return reflect.PointerTo(t).Implements(textUnmarshaler)
}

func setScalar(v reflect.Value, s, raw string) error {
	fail := func(err error) error {
		return fmt.Errorf("bind: query parameter %q: cannot convert %q to %s: %w", raw, s, v.Type(), err)
	}
	if v.CanAddr() && implementsText(v.Type()) {
		if err := v.Addr().Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(s)); err != nil {
			return fail(err)
		}
		return nil
	}
	switch v.Kind() {
	case reflect.String:
		v.SetString(s)
	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return fail(err)
		}
		v.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(s, 10, v.Type().Bits())
		if err != nil {
			return fail(err)
		}
		v.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(s, 10, v.Type().Bits())
		if err != nil {
			return fail(err)
		}
		v.SetUint(n)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(s, v.Type().Bits())
		if err != nil {
			return fail(err)
		}
		v.SetFloat(f)
	default:
		return fail(errors.New("unsupported type"))
	}
	return nil
}
