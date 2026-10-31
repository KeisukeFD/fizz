package openapi

import (
	"encoding/json"
	"net"
	"net/url"
	"reflect"
	"testing"
	"time"
	"unsafe"
	stduuid "uuid"

	gofrs "github.com/gofrs/uuid"
	gofrsv5 "github.com/gofrs/uuid/v5"
	google "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestInternalDataType tests that the internal
// data types have correct types and formats.
func TestInternalDataType(t *testing.T) {
	type formatType struct {
		Type   string
		Format string
	}
	tests := map[InternalDataType]formatType{
		TypeFloat:       {"number", "float"},
		TypeDouble:      {"number", "double"},
		TypeInteger:     {"integer", "int32"},
		TypeLong:        {"integer", "int64"},
		TypeString:      {"string", ""},
		TypeByte:        {"string", "byte"},
		TypeBinary:      {"string", "binary"},
		TypeDate:        {"string", "date"},
		TypeDateTime:    {"string", "date-time"},
		TypeDuration:    {"integer", "int64"},
		TypeIP:          {"string", ""},
		TypeURL:         {"string", "url"},
		TypePassword:    {"string", "password"},
		TypeComplex:     {"string", ""},
		TypeBoolean:     {"boolean", ""},
		TypeInt8:        {"integer", "int8"},
		TypeInt16:       {"integer", "int16"},
		TypeUint8:       {"integer", "uint8"},
		TypeUint16:      {"integer", "uint16"},
		TypeUint32:      {"integer", "uint32"},
		TypeUint64:      {"integer", "uint64"},
		TypeUnsupported: {},
	}
	for dt, ft := range tests {
		format, typ := dt.Format(), dt.Type()
		if format != ft.Format {
			t.Errorf("expected format %s for data type %s, got %s", ft.Format, dt.String(), format)
		}
		if typ != ft.Type {
			t.Errorf("expected type %s for data type %s, got %s", ft.Type, dt.String(), typ)
		}
	}
}

// TestPrimitiveGoTypeToDataType tests that a Go
// primitive type can be converted to a DataType..
func TestPrimitiveGoTypeToDataType(t *testing.T) {
	tests := map[any]DataType{
		int(0):       TypeLong,
		int8(1):      TypeInt8,
		int16(2):     TypeInt16,
		int32(3):     TypeInteger,
		int64(4):     TypeLong,
		new(int):     TypeLong,
		new(int64):   TypeLong,
		uint(0):      TypeUint64,
		uint8(1):     TypeUint8,
		uint16(2):    TypeUint16,
		uint32(3):    TypeUint32,
		uint64(4):    TypeUint64,
		new(uint):    TypeUint64,
		new(uint64):  TypeUint64,
		float32(1.0): TypeFloat,
		float64(2.0): TypeDouble,
		new(float32): TypeFloat,
		new(float64): TypeDouble,
		true:         TypeBoolean,
		false:        TypeBoolean,
		new(bool):    TypeBoolean,
		"string1":    TypeString,
		`string2`:    TypeString,
		new(string):  TypeString,
	}
	for i, dt := range tests {
		tof := reflect.TypeOf(i)
		dtt := DataTypeFromType(tof)
		if dtt != dt {
			t.Errorf("expected data type %s for Go type %s, got %s", dt, tof.String(), dtt)
		}
	}
}

// TestComplexGoTypeToInternalDataType tests that a complex
// Go type can be converted to a InternalDataType.
func TestComplexGoTypeToDataType(t *testing.T) {
	tests := map[reflect.Type]InternalDataType{
		rt([]byte{}):                 TypeByte,
		rt(time.Now()):               TypeDateTime,
		rt(5 * time.Second):          TypeDuration,
		rt(url.URL{}):                TypeComplex,
		rt(net.IP{}):                 TypeIP,
		rt(struct{}{}):               TypeComplex,
		rt([]string{}):               TypeComplex,
		rt([6]string{}):              TypeComplex,
		rt([...]string{}):            TypeComplex,
		rt(map[string]string{}):      TypeComplex,
		rt(map[int]string{}):         TypeComplex,
		rt(func() {}):                TypeUnsupported,
		rt(make(chan int)):           TypeUnsupported,
		rt(uintptr(0xFEED)):          TypeUnsupported,
		rt(unsafe.Pointer(new(int))): TypeUnsupported,
		rt(complex(1.0, float32(0))): TypeUnsupported,
		rt(complex(2.0, float64(0))): TypeUnsupported,
	}
	for tof, dt := range tests {
		dtt := DataTypeFromType(tof)
		if dtt != dt {
			t.Errorf("expected data type %s for Go type %s, got %s", dt, tof.String(), dtt)
		}
	}
}

// TestInternalDataTypeStringer tests that the internal
// data types implements the Stringer interface.
func TestInternalDataTypeStringer(t *testing.T) {
	for _, dt := range []InternalDataType{
		TypeFloat,
		TypeDouble,
		TypeInteger,
		TypeLong,
		TypeString,
		TypeByte,
		TypeBinary,
		TypeDate,
		TypeDateTime,
		TypeDuration,
		TypeIP,
		TypeURL,
		TypePassword,
		TypeComplex,
		TypeBoolean,
		TypeUnsupported,
	} {
		s, ss := dt.String(), datatypes[dt]
		if s != ss {
			t.Errorf("expected %s, got %s", ss, s)
		}
	}
	// Invalid DataType constant.
	udt := InternalDataType(255).String()
	if udt != "" {
		t.Errorf("expected invalid data type to have no string representation, got %s", udt)
	}
}

type UUIDv4 struct{}

func (*UUIDv4) Format() string { return "uuid" }
func (*UUIDv4) Type() string   { return "string" }

// TestCustomDataType tests that a custom type
// that implements the DataType interface can be
// used to get a type and format.
func TestCustomDataType(t *testing.T) {
	uuid := &UUIDv4{}

	dt := DataTypeFromType(reflect.TypeOf(uuid))
	assert.NotNil(t, dt)

	if v, ok := dt.(DataType); ok {
		assert.Equal(t, "uuid", v.Format())
		assert.Equal(t, "string", v.Type())
	} else {
		t.Error("expected type to implements the DataType interface")
	}
}

// TestStringToTimeType tests that a string can be
// converted to the type of a time.Time.
func TestStringToTimeType(t *testing.T) {
	now := time.Now()
	v, err := stringToType(now.Format(time.RFC3339), tofTime)
	if err != nil {
		t.Error(err)
	}
	vv, ok := v.(time.Time)
	if !ok {
		t.Errorf("expected converted value to be of type %T, got %T", now, v)
	}
	if vv.Equal(now) {
		t.Errorf("expected time to equal %s, got %s", now.String(), vv.String())
	}
}

// TestStringToDurationType tests that a string can be
// converted to the type of a time.Duration.
func TestStringToDurationType(t *testing.T) {
	dur := 60 * time.Minute
	v, err := stringToType(dur.String(), tofDuration)
	if err != nil {
		t.Error(err)
	}
	vv, ok := v.(int64)
	if !ok {
		t.Errorf("expected converted value to be of type int64, got %T", v)
	}
	if time.Duration(vv) != dur {
		t.Errorf("expected duration to equal %s, got %s", dur.String(), time.Duration(vv).String())
	}
}

// TestStringToType tests that a string can be
// converted to the kind of a reflect.Type value.
func TestStringToType(t *testing.T) {
	tests := map[string]struct {
		t reflect.Type
		v any
	}{
		"coucou": {rt(""), "coucou"},
		"0":      {rt(int8(0)), int64(0)},
		"1":      {rt(int16(0)), int64(1)},
		"2":      {rt(int32(0)), int64(2)},
		"3":      {rt(int64(0)), int64(3)},
		"4":      {rt(uint8(4)), uint64(4)},
		"5":      {rt(uint16(5)), uint64(5)},
		"6":      {rt(uint32(6)), uint64(6)},
		"7":      {rt(uint64(7)), uint64(7)},
		"true":   {rt(true), true},
		"f":      {rt(false), false},
		"8":      {rt(float32(8)), json.Number("8")},
		"9.1":    {rt(float64(9.1)), float64(9.1)},
	}
	for from, to := range tests {
		vv, err := stringToType(from, to.t)
		if err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(to.v, vv) {
			t.Errorf("expected %s to be converted to %s(%v), got %T(%v)", from, to.t.String(), to.v, vv, vv)
		}
	}
	// Expect error for unknown unsuported types
	// in conversion.
	_, err := stringToType("", reflect.TypeFor[[]string]())
	if err == nil {
		t.Errorf("expected error to be non-nil for unsupported type")
	}
}

// TypeDateTime tests that imported types
// are properly handled.
func TestImportedTypes(t *testing.T) {
	tests := []struct {
		name string
		typ  reflect.Type
	}{
		{"stdlib", reflect.TypeFor[stduuid.UUID]()},
		{"google", reflect.TypeFor[google.UUID]()},
		{"gofrs", reflect.TypeFor[gofrs.UUID]()},
		{"gofrs/v5", reflect.TypeFor[gofrsv5.UUID]()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dt := DataTypeFromType(tt.typ)
			assert.Equal(t, "string", dt.Type())
			assert.Equal(t, "uuid", dt.Format())
		})
	}
}
