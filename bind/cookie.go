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
//   - a cookie that is absent takes the default of the field, if it has
//     one: the ",default=" option of the cookie tag, which wins as it does
//     in tonic, or else the default tag value. Without a default, an
//     absent ",required" cookie is an error and any other leaves the
//     field alone.
//
// An empty cookie does not take the default, but without a default it
// does not satisfy ",required" either: tonic treats an empty query or
// header value as missing. Otherwise, required-ness is left to the
// router's validation. An error names the cookie.
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
			if raw == "" && cf.required && !cf.hasDef {
				return fmt.Errorf("bind: cookie %q: missing cookie parameter", cf.name)
			}
		} else if cf.hasDef {
			raw = cf.def
		} else if cf.required {
			return fmt.Errorf("bind: cookie %q: missing cookie parameter", cf.name)
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
