package bind

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
)

// Cookies decodes the cookie parameters of p from r into dst, a non-nil
// pointer to the struct type p was planned for. For each planned field:
//
//   - a cookie that is present is converted to the field type, from its
//     raw value (net/http does not URL-decode it); an empty value is a
//     value, and converts like any other (to "" for a string);
//   - a cookie that is absent takes the field's default tag value, if it
//     has one, and leaves the field alone otherwise.
//
// Whether a cookie is required is left to the router's validation. An
// error names the cookie.
func Cookies(r *http.Request, dst any, p *Plan) error {
	if p == nil {
		return errors.New("bind: Cookies: nil plan")
	}
	rv := reflect.ValueOf(dst)
	if rv.Kind() != reflect.Pointer || rv.IsNil() || rv.Elem().Type() != p.t {
		return fmt.Errorf("bind: Cookies: dst must be a non-nil *%s, got %T", p.t, dst)
	}
	rv = rv.Elem()
	for i := range p.cookies {
		cf := &p.cookies[i]
		var raw string
		if c, err := r.Cookie(cf.name); err == nil {
			raw = c.Value
		} else if cf.hasDef {
			raw = cf.def
		} else {
			continue
		}
		fv := rv.FieldByIndex(cf.index)
		target := fv
		if cf.ptr {
			target = reflect.New(cf.typ).Elem()
		}
		if err := setScalar(target, raw, "cookie", cf.name); err != nil {
			return err
		}
		if cf.ptr {
			fv.Set(target.Addr())
		}
	}
	return nil
}
