package openapi

import (
	"encoding"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"net/netip"
	"reflect"
	"strconv"
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

// stringToType converts val to t's type and return the new value.
func stringToType(val string, t reflect.Type) (any, error) {
	// Compare type to know Golang types.
	// IT MUST BE EXECUTED BEFORE swithing over
	// primitives because a time.Duration is itself
	// an int64.
	if t.AssignableTo(tofTime) {
		return time.Parse(time.RFC3339, val)
	}
	if t.AssignableTo(tofDuration) {
		// A Duration is documented as an integer number of
		// nanoseconds, so a plain integer is accepted first.
		if ns, err := strconv.ParseInt(val, 10, 64); err == nil {
			return time.Duration(ns), nil
		}
		return time.ParseDuration(val)
	}
	switch t.Kind() {
	case reflect.Bool:
		// ParseBool returns an error if the value
		// is invalid and cannot be converted to a
		// boolean. We assume that invalid values
		// are always falsy.
		v, _ := strconv.ParseBool(val)
		return v, nil
	case reflect.String:
		return val, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.ParseInt(val, 10, t.Bits())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.ParseUint(val, 10, t.Bits())
	case reflect.Float32, reflect.Float64:
		return strconv.ParseFloat(val, t.Bits())
	}
	return nil, fmt.Errorf("unknown type %s", t.String())
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
