package openapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/KeisukeFD/fizz/openapi/internal/vtag"
)

// vkind is the class of a Go type that decides how a validate rule
// maps to a JSON Schema keyword.
type vkind int

const (
	kindOther    vkind = iota
	kindString         // string: length rules are minLength/maxLength
	kindInt            // signed integers
	kindUint           // unsigned integers
	kindFloat          // float32, float64
	kindDuration       // time.Duration, bounds in nanoseconds (D16)
	kindBool           // bool
	kindSlice          // slice or array other than []byte
	kindMap            // map
	kindBytes          // []byte: a contentEncoding string, no length rules
)

// derefType strips pointers from t.
func derefType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}

// isBytes reports whether t is a slice or array of bytes.
func isBytes(t reflect.Type) bool {
	return (t.Kind() == reflect.Slice || t.Kind() == reflect.Array) && t.Elem().Kind() == reflect.Uint8
}

func kindOf(t reflect.Type) vkind {
	t = derefType(t)
	if t == tofDuration {
		return kindDuration
	}
	switch t.Kind() {
	case reflect.String:
		return kindString
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return kindInt
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return kindUint
	case reflect.Float32, reflect.Float64:
		return kindFloat
	case reflect.Bool:
		return kindBool
	case reflect.Slice, reflect.Array:
		if isBytes(t) {
			return kindBytes
		}
		return kindSlice
	case reflect.Map:
		return kindMap
	}
	return kindOther
}

// rulePatterns are the anchored regexes of validator.v9 v9.30.0,
// regexes.go (alphaRegexString, alphaNumericRegexString,
// numericRegexString, hexadecimalRegexString), copied verbatim.
var rulePatterns = map[string]string{
	"alpha":       "^[a-zA-Z]+$",
	"alphanum":    "^[a-zA-Z0-9]+$",
	"numeric":     "^[-+]?[0-9]+(?:\\.[0-9]+)?$",
	"hexadecimal": "^[0-9a-fA-F]+$",
}

func ruleString(r vtag.Rule) string {
	if r.Param == "" {
		return r.Name
	}
	return r.Name + "=" + r.Param
}

// numberLiteral parses param as a number of kind k, and returns it
// in canonical JSON form. A duration is parsed with time.ParseDuration
// (falling back to a plain integer of nanoseconds).
func numberLiteral(k vkind, param string) (json.Number, error) {
	switch k {
	case kindInt:
		n, err := strconv.ParseInt(param, 10, 64)
		return json.Number(strconv.FormatInt(n, 10)), err
	case kindUint:
		n, err := strconv.ParseUint(param, 10, 64)
		return json.Number(strconv.FormatUint(n, 10)), err
	case kindDuration:
		if d, err := time.ParseDuration(param); err == nil {
			return Num(int64(d)), nil
		}
		n, err := strconv.ParseInt(param, 10, 64)
		return json.Number(strconv.FormatInt(n, 10)), err
	case kindFloat:
		f, err := strconv.ParseFloat(param, 64)
		if err == nil && (math.IsInf(f, 0) || math.IsNaN(f)) {
			err = errors.New("not a finite number")
		}
		return json.Number(strconv.FormatFloat(f, 'g', -1, 64)), err
	}
	return "", fmt.Errorf("not a number kind")
}

// setBound maps min/gte/max/lte/gt/lt (name) to a bound of s. Numbers
// use minimum/maximum (inclusive) and exclusiveMinimum/exclusiveMaximum;
// strings, slices and maps use their length keywords, where gt/lt
// shift the count by one.
func setBound(s *Schema, _ reflect.Type, k vkind, name, param string, fail func(error) bool) bool {
	lower := name == "min" || name == "gte" || name == "gt"
	exclusive := name == "gt" || name == "lt"
	switch k {
	case kindInt, kindUint, kindFloat, kindDuration:
		n, err := numberLiteral(k, param)
		if err != nil {
			return fail(err)
		}
		switch {
		case lower && exclusive:
			s.ExclusiveMinimum = n
		case lower:
			s.Minimum = n
		case exclusive:
			s.ExclusiveMaximum = n
		default:
			s.Maximum = n
		}
		return true
	case kindString, kindSlice, kindMap:
		n, err := strconv.ParseUint(param, 10, 63)
		if err != nil {
			return fail(err)
		}
		if exclusive {
			if lower {
				n++
			} else if n == 0 {
				return fail(errors.New("no length is below 0"))
			} else {
				n--
			}
		}
		var dst **uint64
		switch {
		case k == kindString && lower:
			dst = &s.MinLength
		case k == kindString:
			dst = &s.MaxLength
		case k == kindSlice && lower:
			dst = &s.MinItems
		case k == kindSlice:
			dst = &s.MaxItems
		case lower:
			dst = &s.MinProperties
		default:
			dst = &s.MaxProperties
		}
		*dst = u64(n)
		return true
	}
	return false
}

// setConst maps eq (and len on numbers) to const.
func setConst(s *Schema, k vkind, param string, fail func(error) bool) bool {
	switch k {
	case kindString:
		s.Const = param
	case kindInt, kindUint, kindFloat, kindDuration:
		n, err := numberLiteral(k, param)
		if err != nil {
			return fail(err)
		}
		s.Const = n
	case kindBool:
		b, err := strconv.ParseBool(param)
		if err != nil {
			return fail(err)
		}
		s.Const = b
	default:
		return false
	}
	return true
}

// setOneOf maps oneof to enum. Values are split on spaces, as
// validator does without quotes, and typed per field kind.
func setOneOf(s *Schema, k vkind, param string, fail func(error) bool) bool {
	switch k {
	case kindString, kindInt, kindUint, kindDuration:
	case kindFloat, kindBool, kindSlice, kindMap, kindBytes, kindOther:
		return false
	}
	vals := strings.Fields(param)
	if len(vals) == 0 {
		return fail(errors.New("no values"))
	}
	enum := make([]any, 0, len(vals))
	for _, v := range vals {
		if k == kindString {
			enum = append(enum, v)
			continue
		}
		n, err := numberLiteral(k, v)
		if err != nil {
			return fail(err)
		}
		enum = append(enum, n)
	}
	s.Enum = enum
	return true
}

// setFormat sets a string format; other kinds are left alone.
func setFormat(s *Schema, k vkind, format string) bool {
	if k != kindString {
		return false
	}
	s.Format = format
	return true
}

// cmpNum compares two JSON numbers exactly.
func cmpNum(a, b json.Number) int {
	x, okx := new(big.Rat).SetString(string(a))
	y, oky := new(big.Rat).SetString(string(b))
	if !okx || !oky {
		return 0
	}
	return x.Cmp(y)
}

// mergeSchema ANDs the keywords of src into dst: lower bounds keep
// the larger value, upper bounds the smaller one, and any other
// keyword already set to another value moves to an allOf entry.
func mergeSchema(dst, src *Schema) {
	num := func(d *json.Number, v json.Number, lower bool) {
		switch {
		case v == "":
		case *d == "":
			*d = v
		default:
			if c := cmpNum(v, *d); (lower && c > 0) || (!lower && c < 0) {
				*d = v
			}
		}
	}
	num(&dst.Minimum, src.Minimum, true)
	num(&dst.ExclusiveMinimum, src.ExclusiveMinimum, true)
	num(&dst.Maximum, src.Maximum, false)
	num(&dst.ExclusiveMaximum, src.ExclusiveMaximum, false)

	count := func(d **uint64, v *uint64, lower bool) {
		switch {
		case v == nil:
		case *d == nil:
			*d = v
		case (lower && *v > **d) || (!lower && *v < **d):
			*d = v
		}
	}
	count(&dst.MinLength, src.MinLength, true)
	count(&dst.MinItems, src.MinItems, true)
	count(&dst.MinProperties, src.MinProperties, true)
	count(&dst.MaxLength, src.MaxLength, false)
	count(&dst.MaxItems, src.MaxItems, false)
	count(&dst.MaxProperties, src.MaxProperties, false)

	extra := func(extra *Schema) { dst.AllOf = append(dst.AllOf, extra) }
	if src.Format != "" {
		if dst.Format == "" {
			dst.Format = src.Format
		} else if dst.Format != src.Format {
			extra(&Schema{Format: src.Format})
		}
	}
	if src.Pattern != "" {
		if dst.Pattern == "" {
			dst.Pattern = src.Pattern
		} else if dst.Pattern != src.Pattern {
			extra(&Schema{Pattern: src.Pattern})
		}
	}
	if src.Const != nil {
		if dst.Const == nil {
			dst.Const = src.Const
		} else if !reflect.DeepEqual(dst.Const, src.Const) {
			extra(&Schema{Const: src.Const})
		}
	}
	if len(src.Enum) > 0 {
		if len(dst.Enum) == 0 {
			dst.Enum = src.Enum
		} else if !reflect.DeepEqual(dst.Enum, src.Enum) {
			extra(&Schema{Enum: src.Enum})
		}
	}
	if src.Not != nil {
		if dst.Not == nil {
			dst.Not = src.Not
		} else if !reflect.DeepEqual(dst.Not, src.Not) {
			extra(&Schema{Not: src.Not})
		}
	}
	if src.UniqueItems {
		dst.UniqueItems = true
	}
}

// isString returns whether the given reflect type represents a string.
func isString(typ reflect.Type) bool { return typ.Kind() == reflect.String }

// isMap returns whether the given reflect type represents a string.
func isMap(typ reflect.Type) bool { return typ.Kind() == reflect.Map }

// isNumber returns whether the given reflect type
// represents a number.
func isNumber(typ reflect.Type) bool {
	switch typ.Kind() {
	case
		reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Float32,
		reflect.Float64:
		return true
	}
	return false
}
