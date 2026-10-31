package openapi

import (
	"encoding"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"net"
	"net/netip"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var (
	tofDataType = reflect.TypeFor[DataType]()
	tofNullable = reflect.TypeFor[Nullable]()

	// Native.
	tofTime           = reflect.TypeFor[time.Time]()
	tofDuration       = reflect.TypeFor[time.Duration]()
	tofByteSlice      = reflect.TypeFor[[]byte]()
	tofEmptyInterface = reflect.TypeFor[any]()

	tofTextMarshaler = reflect.TypeFor[encoding.TextMarshaler]()
	tofJSONMarshaler = reflect.TypeFor[json.Marshaler]()
)

// knownTypes maps the types that have a fixed schema, whatever their
// kind or the methods they implement, to their data type. It is
// consulted before the Marshaler policy and the kind-derived mapping.
// url.URL is deliberately absent: it is documented as an object.
var knownTypes = map[reflect.Type]DataType{
	tofTime:                            TypeDateTime,
	tofDuration:                        TypeDuration,
	reflect.TypeFor[json.RawMessage](): TypeAny,
	reflect.TypeFor[net.IP]():          TypeIP,
	reflect.TypeFor[netip.Addr]():      TypeIP,
	reflect.TypeFor[netip.Prefix]():    TypeIP,
	reflect.TypeFor[big.Int]():         &OverridedDataType{typ: "integer"},
	reflect.TypeFor[big.Float]():       &OverridedDataType{typ: "string"},
	reflect.TypeFor[big.Rat]():         &OverridedDataType{typ: "string"},
}

// implementsAny reports whether t or *t implements the interface iface,
// so that pointer-receiver methods are honoured.
func implementsAny(t, iface reflect.Type) bool {
	return t.Implements(iface) || reflect.PointerTo(t).Implements(iface)
}

// isTextMarshaler reports whether t or *t implements encoding.TextMarshaler.
func isTextMarshaler(t reflect.Type) bool { return implementsAny(t, tofTextMarshaler) }

// isJSONMarshaler reports whether t or *t implements json.Marshaler.
func isJSONMarshaler(t reflect.Type) bool { return implementsAny(t, tofJSONMarshaler) }

// importedUUIDs holds the "PkgPath.Name" of the known UUID types.
// They are matched by name so that the openapi package imports none
// of the third-party UUID modules.
var importedUUIDs = map[string]struct{}{
	"uuid.UUID":                     {}, // standard library
	"github.com/gofrs/uuid.UUID":    {},
	"github.com/gofrs/uuid/v5.UUID": {},
	"github.com/google/uuid.UUID":   {},
}

var _ DataType = (*InternalDataType)(nil)
var _ DataType = (*OverridedDataType)(nil)

// Typer is the interface implemented
// by the types that can describe themselves.
type Typer interface {
	TypeName() string
}

// DataType is the interface implemented by types
// that can describe their OAS3 data type and format.
type DataType interface {
	Type() string
	Format() string
}

// Exampler is the interface implemented by custom types
// that can parse example values.
type Exampler interface {
	ParseExample(v string) (any, error)
}

// Nullable is the interface implemented by the types
// that can be nullable.
type Nullable interface {
	Nullable() bool
}

// InternalDataType represents an internal type.
type InternalDataType int

// OverridedDataType represents a data type
// which details override the default generation.
type OverridedDataType struct {
	format string
	typ    string
}

// Format implements DataType for OverridedDataType.
func (dt *OverridedDataType) Format() string { return dt.format }

// Type implements DataType for OverridedDataType.
func (dt *OverridedDataType) Type() string { return dt.typ }

// Type constants.
const (
	TypeInteger InternalDataType = iota
	TypeLong
	TypeFloat
	TypeDouble
	TypeString
	TypeByte
	TypeBinary
	TypeBoolean
	TypeDate
	TypeDateTime
	TypeDuration
	TypeIP
	TypeURL
	TypePassword
	TypeAny

	// TypeComplex represents non-primitive types like
	// Go struct, for which a schema must be generated.
	TypeComplex

	// Imported data types.
	TypeUUID

	// Sized integers. Unsigned types carry a minimum of 0.
	TypeInt8
	TypeInt16
	TypeUint8
	TypeUint16
	TypeUint32
	TypeUint64

	TypeUnsupported
)

// String implements fmt.Stringer for DataType.
func (dt InternalDataType) String() string {
	if 0 <= dt && dt < InternalDataType(len(datatypes)) {
		return datatypes[dt]
	}
	return ""
}

// Type returns the type corresponding to the DataType.
func (dt InternalDataType) Type() string {
	if 0 <= dt && dt < InternalDataType(len(types)) {
		return types[dt]
	}
	return ""
}

// Format returns the format corresponding to the DataType.
func (dt InternalDataType) Format() string {
	if 0 <= dt && dt < InternalDataType(len(formats)) {
		return formats[dt]
	}
	return ""
}

// DataTypeFromType returns a DataType for the given type.
//
// The order is: types that describe themselves (DataType, on the value
// or the pointer receiver), the known types, UUIDs, the Marshaler policy
// (a TextMarshaler is a string), then the kind of the type.
func DataTypeFromType(t reflect.Type) DataType {
	// If the type implement the DataType interface,
	// return a new instance of the type.
	if t.Kind() == reflect.Pointer {
		if t.Implements(tofDataType) {
			return reflect.New(t.Elem()).Interface().(DataType)
		}
		// Dereference any pointer.
		return DataTypeFromType(t.Elem())
	}
	if implementsAny(t, tofDataType) {
		return reflect.New(t).Interface().(DataType)
	}
	if dt, ok := knownTypes[t]; ok {
		return dt
	}
	switch t {
	case tofByteSlice:
		return TypeByte
	case tofEmptyInterface:
		return TypeAny
	}
	// Treat imported types.
	if dt := isImportedType(t); dt != nil {
		return dt
	}
	// Marshaler policy: MarshalText (with or without MarshalJSON)
	// makes the type a string.
	if isTextMarshaler(t) {
		return TypeString
	}
	// Switch over primitive types.
	switch t.Kind() {
	case reflect.Int64, reflect.Int:
		return TypeLong
	case reflect.Int32:
		return TypeInteger
	case reflect.Int16:
		return TypeInt16
	case reflect.Int8:
		return TypeInt8
	case reflect.Uint, reflect.Uint64:
		return TypeUint64
	case reflect.Uint32:
		return TypeUint32
	case reflect.Uint16:
		return TypeUint16
	case reflect.Uint8:
		return TypeUint8
	case reflect.Float32:
		return TypeFloat
	case reflect.Float64:
		return TypeDouble
	case reflect.Bool:
		return TypeBoolean
	case reflect.String:
		return TypeString
	case reflect.Map, reflect.Struct, reflect.Array, reflect.Slice:
		return TypeComplex
	default:
		// reflect.Func, reflect.Chan, reflect.Uintptr, reflect.UnsafePointer,
		// reflect.Interface, reflect.Complex64, reflect.Complex128, ...
		return TypeUnsupported
	}
}

// isUnsigned reports whether dt is one of the unsigned integer types.
func isUnsigned(dt DataType) bool {
	switch dt {
	case TypeUint8, TypeUint16, TypeUint32, TypeUint64:
		return true
	}
	return false
}

func isImportedType(t reflect.Type) DataType {
	// uuid, github.com/gofrs/uuid, github.com/google/uuid
	if _, ok := importedUUIDs[t.PkgPath()+"."+t.Name()]; ok {
		return TypeUUID
	}
	return nil
}

// stringToType converts val, the value of a default or enum tag of a
// body field, to t's type. Pointers are dereferenced. See convertValue.
func stringToType(val string, t reflect.Type) (any, error) {
	return convertValue(val, t, false)
}

// convertValue converts val to a value that encodes, as JSON, like a
// value of type t does. Pointers are dereferenced.
//
// A Duration is documented as an integer number of nanoseconds, and is
// stored as an int64, never as a time.Duration. A parameter accepts only
// that integer (tonic cannot parse "1s" there, D16); a body field also
// accepts a ParseDuration string.
//
// An invalid bool or a non-finite number is an error. A float32 is
// emitted as the literal that was written, as a json.Number, so that it
// is not widened to a float64 (0.1 stays 0.1).
func convertValue(val string, t reflect.Type, param bool) (any, error) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	// Compare type to know Golang types.
	// IT MUST BE EXECUTED BEFORE swithing over
	// primitives because a time.Duration is itself
	// an int64.
	if t.AssignableTo(tofTime) {
		return time.Parse(time.RFC3339, val)
	}
	if t.AssignableTo(tofDuration) {
		return durationNanoseconds(val, param)
	}
	switch t.Kind() {
	case reflect.Bool:
		return strconv.ParseBool(val)
	case reflect.String:
		return val, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.ParseInt(val, 10, t.Bits())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.ParseUint(val, 10, t.Bits())
	case reflect.Float32, reflect.Float64:
		v, err := parseFloat(val, t.Bits())
		if err == nil && !finite(val) {
			return nil, fmt.Errorf("%q is not a finite number", val)
		}
		return v, err
	}
	return nil, fmt.Errorf("unknown type %s", t.String())
}

// durationNanoseconds parses val as an integer number of nanoseconds or,
// unless param is set, as a time.ParseDuration string.
func durationNanoseconds(val string, param bool) (int64, error) {
	ns, err := strconv.ParseInt(val, 10, 64)
	if err == nil {
		return ns, nil
	}
	if param {
		return 0, fmt.Errorf("a parameter duration must be an integer number of nanoseconds: %w", err)
	}
	d, derr := time.ParseDuration(val)
	if derr != nil {
		return 0, derr
	}
	return int64(d), nil
}

// parseFloat parses val as a finite float of the given size. A float32
// is returned as the literal of val (a json.Number), a float64 as a
// float64.
func parseFloat(val string, bits int) (any, error) {
	f, err := strconv.ParseFloat(val, bits)
	if err != nil {
		return nil, err
	}
	if bits != 32 {
		return f, nil
	}
	if lit := strings.TrimSpace(val); json.Valid([]byte(lit)) {
		return json.Number(lit), nil
	}
	return json.Number(strconv.FormatFloat(f, 'g', -1, 32)), nil
}

// finite reports whether the float literal val is neither infinite nor NaN.
func finite(val string) bool {
	f, err := strconv.ParseFloat(val, 64)
	return err == nil && !math.IsInf(f, 0) && !math.IsNaN(f)
}

// wireForm returns v as the string that a field with the ",string"
// option carries: the JSON text of v, as a JSON string value.
func wireForm(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

var datatypes = [...]string{
	TypeInteger:     "Integer",
	TypeLong:        "Long",
	TypeFloat:       "Float",
	TypeDouble:      "Double",
	TypeString:      "String",
	TypeByte:        "Byte",
	TypeBinary:      "Binary",
	TypeBoolean:     "Boolean",
	TypeDate:        "Date",
	TypeDateTime:    "DateTime",
	TypeDuration:    "Duration",
	TypeIP:          "IP Address",
	TypeURL:         "URL (Uniform Resource Locator)",
	TypePassword:    "Password",
	TypeUnsupported: "Unsupported",
	TypeComplex:     "Complex",
	TypeUUID:        "UUID",
	TypeInt8:        "Int8",
	TypeInt16:       "Int16",
	TypeUint8:       "Uint8",
	TypeUint16:      "Uint16",
	TypeUint32:      "Uint32",
	TypeUint64:      "Uint64",
}

var types = [...]string{
	TypeInteger:  "integer",
	TypeLong:     "integer",
	TypeFloat:    "number",
	TypeDouble:   "number",
	TypeString:   "string",
	TypeByte:     "string",
	TypeBinary:   "string",
	TypeBoolean:  "boolean",
	TypeDate:     "string",
	TypeDateTime: "string",
	TypeDuration: "integer",
	TypeIP:       "string",
	TypeURL:      "string",
	TypePassword: "string",
	TypeComplex:  "string",
	TypeUUID:     "string",
	TypeInt8:     "integer",
	TypeInt16:    "integer",
	TypeUint8:    "integer",
	TypeUint16:   "integer",
	TypeUint32:   "integer",
	TypeUint64:   "integer",
}

var formats = [...]string{
	TypeInteger:  "int32",
	TypeLong:     "int64",
	TypeFloat:    "float",
	TypeDouble:   "double",
	TypeString:   "",
	TypeByte:     "byte",
	TypeBinary:   "binary",
	TypeBoolean:  "",
	TypeDate:     "date",
	TypeDateTime: "date-time",
	TypeDuration: "int64",
	TypeIP:       "",
	TypeURL:      "url",
	TypePassword: "password",
	TypeComplex:  "",
	TypeUUID:     "uuid",
	TypeInt8:     "int8",
	TypeInt16:    "int16",
	TypeUint8:    "uint8",
	TypeUint16:   "uint16",
	TypeUint32:   "uint32",
	TypeUint64:   "uint64",
}
