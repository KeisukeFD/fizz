package openapi

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Pallinder/go-randomdata"
	"github.com/stretchr/testify/assert"

	"github.com/wI2L/fizz/internal/golden"
)

var genConfig = &SpecGenConfig{
	ValidatorTag:      BinderTags.Validate,
	PathLocationTag:   BinderTags.Path,
	QueryLocationTag:  BinderTags.Query,
	HeaderLocationTag: BinderTags.Header,
	EnumTag:           BinderTags.Enum,
	DefaultTag:        BinderTags.Default,
}

var rt = reflect.TypeOf

type (
	W struct {
		A, B string
	}
	u struct {
		S int
	}
	q  int
	ns string
	ni int
	X  struct {
		*X // ignored, recursive embedding
		*Y
		A string `validate:"required"`
		B *int
		C bool `deprecated:"true"`
		D []*Y
		E [3]*X
		F *X
		G *Y
		H map[int]*Y // ignored, unsupported keys type
		*u
		uu *u // ignored, unexported field
		q     // ignored, embedded field of non-struct type
		*Q
		*V `json:"data"`
		NS ns
		NI *ni
	}
	Y struct {
		H float32   `validate:"required"`
		I time.Time `format:"date"`
		J *uint8    `deprecated:"oui"` // invalid value, interpreted as false
		K *Z        `validate:"required"`
		N struct {
			Na, Nb string
			Nc     time.Duration
		}
		l int // ignored
		M int `json:"-"`
	}
	Z map[string]*Y
	Q struct {
		NnNnnN string `json:"nnNnnN"`
	}
	V struct {
		L int
	}
)

func (*X) TypeName() string { return "XXX" }
func (*W) Format() string   { return "wallet" }
func (*W) Type() string     { return "string" }
func (ns) Nullable() bool   { return true }
func (ni) Nullable() bool   { return false }

// TestStructFieldName tests that the name of a
// struct field can be correctly extracted.
func TestStructFieldName(t *testing.T) {
	type T struct {
		A  string `name:"A"`
		Ba string `name:""`
		AB string `name:"-"`
		B  struct{}
	}
	to := reflect.TypeFor[T]()

	assert.Equal(t, "A", fieldNameFromTag(to.Field(0), "name"))
	assert.Equal(t, "Ba", fieldNameFromTag(to.Field(1), "name"))
	assert.Equal(t, "", fieldNameFromTag(to.Field(2), "name"))
}

func TestAddTag(t *testing.T) {
	g := gen(t)

	// Append nil tag to ensure sort works
	// works even with non-addressable values.
	g.api.Tags = append(g.api.Tags, nil)

	g.AddTag("", "Test routes")
	assert.Len(t, g.API().Tags, 1)

	g.AddTag("Test", "Test routes")
	assert.Len(t, g.API().Tags, 2)

	tag := g.API().Tags[1]
	assert.NotNil(t, tag)
	assert.Equal(t, tag.Name, "Test")
	assert.Equal(t, tag.Description, "Test routes")

	// Update tag description.
	g.AddTag("Test", "Routes test")
	assert.Equal(t, tag.Description, "Routes test")

	// Add other tag, check sort order.
	g.AddTag("A", "")
	assert.Len(t, g.API().Tags, 3)
	tag = g.API().Tags[1]
	assert.Equal(t, "A", tag.Name)
}

// TestSchemaFromPrimitiveType tests that a schema
// can be created given a primitive input type.
func TestSchemaFromPrimitiveType(t *testing.T) {
	g := gen(t)

	// Use a pointer to primitive type to test
	// pointer dereference and property nullable.
	schema := g.newSchemaFromType(rt(new(int64)))

	// Ensure it is an inlined schema before
	// accessing properties for assertions.
	if schema.Ref != "" {
		t.Error("expected an inlined schema, got a schema reference")
	}
	assert.Equal(t, Types("integer", "null"), schema.Type)
	assert.Equal(t, "int64", schema.Format)
}

// TestSchemaFromInterface tests that a schema
// can be created for an interface{} value that
// represent *any* type.
func TestSchemaFromInterface(t *testing.T) {
	g := gen(t)

	schema := g.newSchemaFromType(tofEmptyInterface)
	assert.NotNil(t, schema)
	assert.Empty(t, schema.Type)
	assert.Empty(t, schema.Format)
	assert.NotEmpty(t, schema.Description)
}

// TestSchemaFromUnsupportedType tests that a schema
// cannot be created given an unsupported input type.
func TestSchemaFromUnsupportedType(t *testing.T) {
	g := gen(t)

	// Test with nil input.
	schema := g.newSchemaFromType(nil)
	assert.Nil(t, schema)

	// Test with unsupported input.
	schema = g.newSchemaFromType(rt(func() {}))
	assert.Nil(t, schema)
	assert.Len(t, g.Errors(), 1)
	assert.Implements(t, (*error)(nil), g.Errors()[0])
	assert.NotEmpty(t, g.Errors()[0])
}

// TestSchemaFromMapWithUnsupportedKeys tests that a
// schema cannot be created given a map type with
// unsupported key's type.
func TestSchemaFromMapWithUnsupportedKeys(t *testing.T) {
	g := gen(t)

	schema := g.newSchemaFromType(rt(map[int]string{}))
	assert.Nil(t, schema)
	assert.Len(t, g.Errors(), 1)
	assert.Implements(t, (*error)(nil), g.Errors()[0])
	assert.NotEmpty(t, g.Errors()[0].Error())
}

// TestSchemaFromComplex tests that a schema
// can be created from a complex type.
func TestSchemaFromComplex(t *testing.T) {
	g := gen(t)
	g.UseFullSchemaNames(false)

	sor := g.newSchemaFromType(rt(new(X)))
	assert.NotNil(t, sor)

	b, err := json.Marshal(sor)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, `{"$ref":"#/components/schemas/XXX"}`, string(b))

	schema := g.resolveSchema(sor)
	assert.NotNil(t, schema)

	actual, err := json.Marshal(schema)
	if err != nil {
		t.Error(err)
	}
	golden.Assert(t, "../testdata/schemas/X.json", actual)

	sor = g.API().Components.Schemas["Y"]
	schema = g.resolveSchema(sor)
	assert.NotNil(t, schema)

	actual, err = json.Marshal(schema)
	if err != nil {
		t.Error(err)
	}
	golden.Assert(t, "../testdata/schemas/Y.json", actual)
}

// TestNewSchemaFromStructErrors tests the errors
// case of generation of a schema from a struct.
func TestNewSchemaFromStructErrors(t *testing.T) {
	g := gen(t)

	// Invalid input.
	sor := g.newSchemaFromStruct(reflect.TypeFor[*string]())
	assert.Nil(t, sor)
}

// TestNewSchemaFromStructFieldExampleValues tests the
// case of setting example values.
func TestNewSchemaFromStructFieldExampleValues(t *testing.T) {
	g := gen(t)

	type T struct {
		A    string    `example:"value"`
		APtr *string   `example:"value"`
		B    int       `example:"1"`
		BPtr *int      `example:"1"`
		C    float64   `example:"0.1"`
		CPtr *float64  `example:"0.1"`
		D    bool      `example:"true"`
		DPtr *bool     `example:"true"`
		EPtr **bool    `example:"false"`
		FPtr ***uint16 `example:"128"`
	}
	typ := reflect.TypeFor[T]()

	// Field A contains string example.
	sor := g.newSchemaFromStructField(typ.Field(0), false, "A", typ)
	assert.NotNil(t, sor)
	assert.Equal(t, []any{"value"}, sor.Examples)

	// Field APtr contains pointer to string example.
	sor = g.newSchemaFromStructField(typ.Field(1), false, "APtr", typ)
	assert.NotNil(t, sor)
	assert.Equal(t, []any{"value"}, sor.Examples)

	// Field B contains int example.
	sor = g.newSchemaFromStructField(typ.Field(2), false, "B", typ)
	assert.NotNil(t, sor)
	assert.Equal(t, []any{int64(1)}, sor.Examples)

	// Field BPtr contains pointer to int example.
	sor = g.newSchemaFromStructField(typ.Field(3), false, "BPtr", typ)
	assert.NotNil(t, sor)
	assert.Equal(t, []any{int64(1)}, sor.Examples)

	// Field C contains float example.
	sor = g.newSchemaFromStructField(typ.Field(4), false, "C", typ)
	assert.NotNil(t, sor)
	assert.Equal(t, []any{0.1}, sor.Examples)

	// Field CPtr contains pointer to float example.
	sor = g.newSchemaFromStructField(typ.Field(5), false, "CPtr", typ)
	assert.NotNil(t, sor)
	assert.Equal(t, []any{0.1}, sor.Examples)

	// Field D contains boolean example.
	sor = g.newSchemaFromStructField(typ.Field(6), false, "D", typ)
	assert.NotNil(t, sor)
	assert.Equal(t, []any{true}, sor.Examples)

	// Field DPtr contains pointer to boolean example.
	sor = g.newSchemaFromStructField(typ.Field(7), false, "DPtr", typ)
	assert.NotNil(t, sor)
	assert.Equal(t, []any{true}, sor.Examples)

	// Field EPtr contains a double-pointer to boolean example.
	sor = g.newSchemaFromStructField(typ.Field(8), false, "EPtr", typ)
	assert.NotNil(t, sor)
	assert.Equal(t, []any{false}, sor.Examples)

	// Field FPtr contains a triple-pointer to uint16 value example.
	sor = g.newSchemaFromStructField(typ.Field(9), false, "FPtr", typ)
	assert.NotNil(t, sor)
	assert.Equal(t, []any{uint16(128)}, sor.Examples)
}

// TestNewSchemaFromStructFieldErrors tests the errors
// case of generation of a schema from a struct field.
func TestNewSchemaFromStructFieldErrors(t *testing.T) {
	g := gen(t)

	type T struct {
		A string `validate:"required" default:"foobar"`
		B int    `default:"foobaz"`
		C int    `enum:"a,1,c"`
		D bool   `example:"not-a-bool-value"`
	}
	typ := reflect.TypeFor[T]()

	// Field A is required and has a default value.
	sor := g.newSchemaFromStructField(typ.Field(0), true, "A", typ)
	assert.NotNil(t, sor)
	assert.Len(t, g.Errors(), 1)
	assert.Implements(t, (*error)(nil), g.Errors()[0])
	assert.NotEmpty(t, g.Errors()[0].Error())

	// Field B has default value that cannot be converted to type's type.
	sor = g.newSchemaFromStructField(typ.Field(1), false, "B", typ)
	assert.NotNil(t, sor)
	assert.Len(t, g.Errors(), 2)
	assert.Implements(t, (*error)(nil), g.Errors()[1])
	assert.NotEmpty(t, g.Errors()[1].Error())

	// Field C has enum values that cannot be converted to type's type.
	sor = g.newSchemaFromStructField(typ.Field(2), true, "C", typ)
	assert.NotNil(t, sor)
	// it generates two errors, one per value
	// that cannot be converted, here "a" and "b".
	assert.Len(t, g.Errors(), 4)
	assert.NotEmpty(t, g.Errors()[2].Error())
	assert.NotEmpty(t, g.Errors()[3].Error())

	// Field D has example value that cannot be parsed to bool.
	sor = g.newSchemaFromStructField(typ.Field(3), false, "D", typ)
	assert.NotNil(t, sor)
	assert.Len(t, g.Errors(), 5)
	assert.NotEmpty(t, g.Errors()[4].Error())
	// check that Name & Type of the error are set correctly
	fe, ok := g.Errors()[4].(*FieldError)
	assert.True(t, ok)
	assert.Equal(t, "D", fe.Name)
	assert.Equal(t, reflect.Bool, fe.Type.Kind())
}

func TestNewSchemaFromStructFieldFormat(t *testing.T) {
	g := gen(t)

	type T struct {
		A string `validate:"email" default:"foobar"`
	}
	typ := reflect.TypeFor[T]()

	// Field A is required and has a default value.
	sor := g.newSchemaFromStructField(typ.Field(0), true, "A", typ)
	assert.NotNil(t, sor)
	assert.Len(t, g.Errors(), 1)
	assert.Implements(t, (*error)(nil), g.Errors()[0])
	assert.NotEmpty(t, g.Errors()[0].Error())
	assert.Equal(t, sor.Format, "email")
}

func TestNewSchemaFromEnumField(t *testing.T) {
	g := gen(t)

	type T struct {
		A string      `enum:"a,b,c"`
		B int         `enum:"1,2,3"`
		C *string     `enum:"d,e,f"`
		D *int        `enum:"4,5,6"`
		E []string    `enum:"g,h,i"`
		F *[]string   `enum:"j,k,l"`
		G **string    `enum:"m,n,o"`
		H **[]string  `enum:"p,q,r"`
		I **[]float64 `enum:"7.0,8.1,9.2"`
	}

	tests := []struct {
		fname        string
		expectedEnum []any
		isSlice      bool
	}{
		{"A", []any{"a", "b", "c"}, false},
		{"B", []any{int64(1), int64(2), int64(3)}, false},
		{"C", []any{"d", "e", "f"}, false},
		{"D", []any{int64(4), int64(5), int64(6)}, false},
		{"E", []any{"g", "h", "i"}, true},
		{"F", []any{"j", "k", "l"}, true},
		{"G", []any{"m", "n", "o"}, false},
		{"H", []any{"p", "q", "r"}, false},
		{"I", []any{7.0, 8.1, 9.2}, false},
	}

	typ := reflect.TypeFor[T]()

	for i, tt := range tests {
		t.Run(tt.fname, func(t *testing.T) {
			sor := g.newSchemaFromStructField(typ.Field(i), true, tt.fname, typ)
			assert.NotNil(t, sor)
			var enum []any
			if tt.isSlice {
				enum = sor.Items.Enum
			} else {
				enum = sor.Enum
			}
			assert.Equal(t, tt.expectedEnum, enum)
		})

	}
}

// TestAddOperation tests that an operation can be added
// and generates the according specification.
func TestAddOperation(t *testing.T) {
	type InEmbed struct {
		D int      `query:"xd" enum:"1,2,3" default:"1"`
		E bool     `query:"e"`
		F *string  `json:"f" description:"This is F"`
		G []byte   `validate:"required"`
		H uint16   `binding:"-"`
		K []string `query:"k" enum:"aaa,bbb,ccc"`
	}
	type inEmbedPrivate struct {
		I string `query:"i"`
	}
	type h string
	type In struct {
		*In // ignored, recusrive embedding
		*InEmbed
		*inEmbedPrivate

		A int       `path:"a" description:"This is A" deprecated:"oui"`
		B time.Time `query:"b" validate:"required" description:"This is B"`
		C string    `header:"X-Test-C" description:"This is C" default:"test"`
		d int       // ignored, unexported field
		E int       `path:"a"` // ignored, duplicate of A
		F *string   `json:"f"` // ignored, duplicate of F in InEmbed
		G *inEmbedPrivate
		h // ignored, embedded field of non-struct type
	}
	type CustomError struct{}

	var Header string

	g := gen(t)
	g.UseFullSchemaNames(false)
	g.SetSortParams(true)

	path := "/test/{a}"

	infos := &OperationInfo{
		ID:          "CreateTest",
		StatusCode:  201,
		Summary:     "ABC",
		Description: "XYZ",
		Deprecated:  true,
		Responses: []*OperationResponse{
			{
				Code:        "400",
				Description: "Bad Request",
				Model:       CustomError{},
			},
			{
				Code:        "5XX",
				Description: "Server Errors",
			},
		},
		Headers: []*ResponseHeader{
			{
				Name:        "X-Test-Header",
				Description: "Test header",
				Model:       Header,
			},
			{
				Name:        "X-Test-Header-Alt",
				Description: "Test header alt",
			},
		},
	}
	_, err := g.AddOperation(path, "POST", "Test", reflect.TypeFor[*In](), reflect.TypeFor[Z](), infos)
	if err != nil {
		t.Error(err)
	}
	// Add another operation with no input/output type.
	// No parameters should be present, and a response
	// matching the default status code used by tonic
	// should be present with no content.
	_, err = g.AddOperation(path, "PUT", "Test", nil, nil, &OperationInfo{
		ID:          "UpdateTest",
		StatusCode:  204,
		Description: "Update a test.",
	})
	if err != nil {
		t.Error(err)
	}
	assert.Len(t, g.API().Paths, 1)

	item, ok := g.API().Paths[path]
	if !ok {
		t.Errorf("expected to found item for path %s", path)
	}
	assert.NotNil(t, item.POST)
	assert.NotNil(t, item.PUT)

	actual, err := json.Marshal(item)
	if err != nil {
		t.Error(err)
	}
	golden.Assert(t, "../testdata/schemas/path-item.json", actual)
	// Try to add the operation again with the same
	// identifier. Expected to fail.
	_, err = g.AddOperation(path, "POST", "Test", reflect.TypeFor[*In](), reflect.TypeFor[Z](), infos)
	assert.NotNil(t, err)

	// Add an operation with a bad input type.
	_, err = g.AddOperation("/", "GET", "", reflect.TypeFor[*string](), nil, nil)
	assert.NotNil(t, err)
}

// TestTypeName tests that the name of a type
// can be discovered.
func TestTypeName(t *testing.T) {
	g, err := NewGenerator(genConfig)
	if err != nil {
		t.Error(err)
	}
	// Typer interface.
	name := g.typeName(rt(new(X)))
	assert.Equal(t, "XXX", name)

	// Override. This has precedence
	// over the interface implementation.
	err = g.OverrideTypeName(rt(new(X)), "")
	assert.NotNil(t, err)
	assert.Equal(t, "XXX", g.typeName(rt(new(X))))

	g.OverrideTypeName(rt(new(X)), "xXx")
	assert.Equal(t, "xXx", g.typeName(rt(X{})))

	err = g.OverrideTypeName(rt(new(X)), "YYY")
	assert.NotNil(t, err)

	// Default.
	assert.Equal(t, "OpenapiY", g.typeName(rt(new(Y))))
	g.UseFullSchemaNames(false)
	assert.Equal(t, "Y", g.typeName(rt(Y{})))

	// Unnamed type.
	assert.Equal(t, "", g.typeName(rt(struct{}{})))
}

// TestSetInfo tests that the informations
// of the spec can be modified.
func TestSetInfo(t *testing.T) {
	g := gen(t)

	infos := &Info{
		Description: "Test",
	}
	g.SetInfo(infos)

	assert.NotNil(t, g.API().Info)
	assert.Equal(t, infos, g.API().Info)
}

// TestSetOperationByMethod tests that an operation
// is added to a path item accordingly to the given
// HTTP method.
func TestSetOperationByMethod(t *testing.T) {
	pi := &PathItem{}
	for method, ptr := range map[string]**Operation{
		"GET":     &pi.GET,
		"POST":    &pi.POST,
		"PUT":     &pi.PUT,
		"PATCH":   &pi.PATCH,
		"DELETE":  &pi.DELETE,
		"HEAD":    &pi.HEAD,
		"OPTIONS": &pi.OPTIONS,
		"TRACE":   &pi.TRACE,
	} {
		desc := randomdata.SillyName()
		op := &Operation{
			Description: desc,
		}
		setOperationBymethod(pi, op, method)
		assert.Equal(t, op, *ptr)
		assert.Equal(t, desc, (*ptr).Description)
	}
}

// TestSetOperationResponseError tests the various error
// cases that can occur while adding a response to an op.
func TestSetOperationResponseError(t *testing.T) {
	g := gen(t)
	op := &Operation{
		Responses: make(Responses),
	}
	err := g.setOperationResponse(op, "GET /test", reflect.TypeFor[*string](), "200", "application/json", "", nil, nil, nil)
	assert.Nil(t, err)
	assert.Equal(t, "OK", op.Responses["200"].Value.Description)

	err = g.setOperationResponse(op, "GET /test", reflect.TypeFor[*string](), "429", "application/json", "testDesc", nil, nil, nil)
	assert.Nil(t, err)
	assert.Equal(t, "testDesc", op.Responses["429"].Value.Description)

	// Add another response with same code.
	err = g.setOperationResponse(op, "GET /test", reflect.TypeFor[*int](), "200", "application/xml", "", nil, nil, nil)
	assert.NotNil(t, err)

	// Add invalid response code that cannot
	// be converted to an integer.
	err = g.setOperationResponse(op, "GET /test", reflect.TypeFor[*bool](), "two-hundred", "", "", nil, nil, nil)
	assert.NotNil(t, err)

	// Add out of range response code.
	err = g.setOperationResponse(op, "GET /test", reflect.TypeFor[*bool](), "777", "", "", nil, nil, nil)
	assert.NotNil(t, err)

	// Cannot set both example and examples
	err = g.setOperationResponse(op, "GET /test", reflect.TypeFor[*bool](), "404", "", "", nil, "notFoundExample", map[string]any{"badRequest": "message"})
	assert.NotNil(t, err)
}

// TestSetOperationResponseExample tests that
// one example is set correctly.
func TestSetOperationResponseExample(t *testing.T) {
	g := gen(t)
	op := &Operation{
		Responses: make(Responses),
	}

	error1 := map[string]any{"error": "message1"}

	err := g.setOperationResponse(op, "GET /test", reflect.TypeFor[*string](), "400", "application/json", "", nil, error1, nil)
	assert.Nil(t, err)

	// assert example set correctly
	mt := op.Responses["400"].Value.Content["application/json"].Value
	assert.Equal(t, error1, mt.Example)

	// examples should be empty
	assert.Nil(t, mt.Examples)
}

// TestSetOperationResponseExamples tests that
// multiple examples are set correctly.
func TestSetOperationResponseExamples(t *testing.T) {
	g := gen(t)
	op := &Operation{
		Responses: make(Responses),
	}

	error1 := map[string]any{"error": "message1"}
	error2 := map[string]any{"error": "message2"}

	err := g.setOperationResponse(op, "GET /test", reflect.TypeFor[*string](), "400", "application/json", "", nil, nil,
		map[string]any{
			"one": error1,
			"two": error2,
		},
	)
	assert.Nil(t, err)

	// assert examples set correctly
	mt := op.Responses["400"].Value.Content["application/json"].Value
	assert.Equal(t, 2, len(mt.Examples))
	assert.Equal(t, error1, mt.Examples["one"].Value.Value)
	assert.Equal(t, error2, mt.Examples["two"].Value.Value)

	// example should be empty
	assert.Nil(t, mt.Example)
}

// TestSetOperationParamsError tests the various error
// cases that can occur while adding parameters to an op.
func TestSetOperationParamsError(t *testing.T) {
	g := gen(t)
	op := &Operation{}

	// Use invalid input type for parameters.
	typ := reflect.TypeFor[[]string]()
	err := g.setOperationParams(op, typ, typ, false, "/")
	assert.NotNil(t, err)

	// Semantic error for path.
	type T struct {
		B string `path:"B"`
	}
	typ = reflect.TypeFor[T]()
	err = g.setOperationParams(op, typ, typ, false, "/{a}/{B}")
	assert.NotNil(t, err)
}

// TestParamLocationConflict tests that using conflicting
// locations in the tag of a parameter throws an error.
func TestParamLocationConflict(t *testing.T) {
	type T struct {
		A string `path:"a" query:"b"`
	}
	g := gen(t)

	_, err := g.paramLocation(reflect.TypeFor[T]().Field(0), reflect.TypeFor[T]())
	assert.NotNil(t, err)
}

// TestOverrideDataType tests that the data type
// of a type can be ovirriden manually.
func TestOverrideSchema(t *testing.T) {
	g := gen(t)

	// Type is mandatory.
	err := g.OverrideDataType(rt(W{}), "", "wallet")
	assert.NotNil(t, err)

	// Success.
	err = g.OverrideDataType(rt(&W{}), "string", "wallet")
	assert.Nil(t, err)

	// Data type already overidden.
	err = g.OverrideDataType(rt(&W{}), "string", "wallet")
	assert.NotNil(t, err)

	sor := g.newSchemaFromType(rt(W{}))
	assert.NotNil(t, sor)

	schema := g.resolveSchema(sor)
	assert.NotNil(t, schema)

	assert.Equal(t, Types("string"), schema.Type)
	assert.Equal(t, "wallet", schema.Format)
}

// TestNewGenWithoutConfig tests that creating a
// new generator without config fails.
func TestNewGenWithoutConfig(t *testing.T) {
	_, err := NewGenerator(nil)
	assert.NotNil(t, err)
}

// TestSetServers tests that a custom servers description
// can be added to the specification and is properly marshaled.
func TestSetServers(t *testing.T) {
	g := gen(t)

	servers := []*Server{
		{
			URL:         "https://dev.api.foo.bar/v1",
			Description: "Development server",
		},
		{
			URL:         "https://prod.api.foo.bar/{basePath}",
			Description: "Production server",
			Variables: map[string]*ServerVariable{
				"basePath": {
					Description: "Version of the API",
					Enum: []string{
						"v1", "v2", "beta",
					},
					Default: "v2",
				},
			},
		},
	}
	g.SetServers(servers)

	assert.NotNil(t, g.API().Servers)
	assert.Equal(t, servers, g.API().Servers)
}

type customUnit float64

func (c customUnit) ParseExample(v string) (any, error) {
	s, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return nil, err
	}
	return fmt.Sprintf("%.2f USD", s), nil
}

type customTime time.Time

func (c customTime) ParseExample(v string) (any, error) {
	t1, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return nil, err
	}
	return customTime(t1), nil
}

// TestGenerator_parseExampleValue tests the parsing of example values.
func TestGenerator_parseExampleValue(t *testing.T) {
	testCases := []struct {
		testName    string
		typ         reflect.Type
		inputValue  string
		outputValue any
	}{
		{
			"mapping to string",
			reflect.TypeFor[string](),
			"value",
			"value",
		},
		{
			"mapping pointer to string",
			reflect.PointerTo(reflect.TypeFor[string]()),
			"value",
			"value",
		},
		{
			"mapping to int8",
			reflect.TypeOf(int8(math.MaxInt8)),
			"127",
			int8(math.MaxInt8),
		},
		{
			"mapping pointer to int8",
			reflect.PointerTo(reflect.TypeOf(int8(math.MaxInt8))),
			"127",
			int8(math.MaxInt8),
		},
		{
			"mapping to int16",
			reflect.TypeOf(int16(math.MaxInt16)),
			"32767",
			int16(math.MaxInt16),
		},
		{
			"mapping pointer to int16",
			reflect.PointerTo(reflect.TypeOf(int16(math.MaxInt16))),
			"32767",
			int16(math.MaxInt16),
		},
		{
			"mapping to int32",
			reflect.TypeOf(int32(math.MaxInt32)),
			"2147483647",
			int32(math.MaxInt32),
		},
		{
			"mapping pointer to int32",
			reflect.PointerTo(reflect.TypeOf(int32(math.MaxInt32))),
			"2147483647",
			int32(math.MaxInt32),
		},
		{
			"mapping to int64",
			reflect.TypeOf(int64(math.MaxInt64)),
			"9223372036854775807",
			int64(math.MaxInt64),
		},
		{
			"mapping pointer to int64",
			reflect.PointerTo(reflect.TypeOf(int64(math.MaxInt64))),
			"9223372036854775807",
			int64(math.MaxInt64),
		},
		{
			"mapping to uint8",
			reflect.TypeOf(uint8(math.MaxUint8)),
			"255",
			uint8(math.MaxUint8),
		},
		{
			"mapping pointer to uint8",
			reflect.PointerTo(reflect.TypeOf(uint8(math.MaxUint8))),
			"255",
			uint8(math.MaxUint8),
		},
		{
			"mapping to uint16",
			reflect.TypeOf(uint16(math.MaxUint16)),
			"65535",
			uint16(math.MaxUint16),
		},
		{
			"mapping pointer to uint16",
			reflect.PointerTo(reflect.TypeOf(uint16(math.MaxUint16))),
			"65535",
			uint16(math.MaxUint16),
		},
		{
			"mapping to uint32",
			reflect.TypeOf(uint32(math.MaxUint32)),
			"4294967295",
			uint32(math.MaxUint32),
		},
		{
			"mapping pointer to uint32",
			reflect.PointerTo(reflect.TypeOf(uint32(math.MaxUint32))),
			"4294967295",
			uint32(math.MaxUint32),
		},
		{
			"mapping to uint64",
			reflect.TypeOf(uint64(math.MaxUint64)),
			"18446744073709551615",
			uint64(math.MaxUint64),
		},
		{
			"mapping pointer to uint64",
			reflect.PointerTo(reflect.TypeOf(uint64(math.MaxUint64))),
			"18446744073709551615",
			uint64(math.MaxUint64),
		},
		{
			"mapping to number",
			reflect.TypeFor[float64](),
			"1.23",
			1.23,
		},
		{
			"mapping pointer to number",
			reflect.PointerTo(reflect.TypeFor[float64]()),
			"1.23",
			1.23,
		},
		{
			"mapping to boolean",
			reflect.TypeOf(true),
			"true",
			true,
		},
		{
			"mapping pointer to boolean",
			reflect.PointerTo(reflect.TypeOf(true)),
			"true",
			true,
		},
		{
			"mapping to customUnit",
			reflect.TypeFor[customUnit](),
			"15",
			"15.00 USD",
		},
		{
			"mapping pointer to customUnit",
			reflect.PointerTo(reflect.TypeFor[customUnit]()),
			"20.00000",
			"20.00 USD",
		},
		{
			"mapping to customTime",
			reflect.TypeFor[customTime](),
			"2022-02-07T18:00:00+09:00",
			customTime(time.Date(2022, time.February, 7, 18, 0, 0, 0, time.FixedZone("", 9*3600))),
		},
		{
			"mapping pointer to customTime",
			reflect.PointerTo(reflect.TypeFor[customTime]()),
			"2022-02-07T18:00:00+09:00",
			customTime(time.Date(2022, time.February, 7, 18, 0, 0, 0, time.FixedZone("", 9*3600))),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			returned, err := parseExampleValue(tc.typ, tc.inputValue)
			assert.Nil(t, err)
			assert.Equal(t, tc.outputValue, returned)
		})
	}
}

// TestGenerator_parseExampleValueError tests that
// parseExampleValue raises error on unsupported type.
func TestGenerator_parseExampleValueError(t *testing.T) {
	_, err := parseExampleValue(reflect.TypeFor[map[string]string](), "whatever")
	assert.Error(t, err, "parseExampleValue does not support type")
}

func gen(t *testing.T) *Generator {
	g, err := NewGenerator(genConfig)
	if err != nil {
		t.Error(err)
	}
	g.UseFullSchemaNames(false)

	return g
}

// TestUnencodableExamplesDropped tests that examples that cannot be
// encoded as JSON are recorded in Errors() and dropped.
func TestUnencodableExamplesDropped(t *testing.T) {
	g := gen(t)
	op := &Operation{Responses: make(Responses)}

	err := g.setOperationResponse(op, "GET /nan", reflect.TypeFor[*string](), "400", "application/json", "", nil, nil,
		map[string]any{"bad": math.NaN(), "ch": make(chan int), "fn": func() {}, "ok": "fine"})
	assert.NoError(t, err)
	mt := op.Responses["400"].Value.Content["application/json"].Value
	assert.Len(t, mt.Examples, 1)
	assert.Contains(t, mt.Examples, "ok")
	assert.Len(t, g.Errors(), 3)
	for _, e := range g.Errors() {
		assert.Contains(t, e.Error(), "GET /nan")
		assert.Contains(t, e.Error(), "response 400")
	}

	err = g.setOperationResponse(op, "POST /x", reflect.TypeFor[*string](), "401", "application/json", "", nil,
		math.Inf(1), nil)
	assert.NoError(t, err)
	assert.Nil(t, op.Responses["401"].Value.Content["application/json"].Value.Example)
	assert.Len(t, g.Errors(), 4)
	assert.Contains(t, g.Errors()[3].Error(), "POST /x")

	// A generated document with these examples still encodes.
	_, err = MarshalJSON(g.API())
	assert.NoError(t, err)
}

type nanExample struct {
	V float64 `json:"v" example:"NaN"`
	W float64 `json:"w" example:"1.5"`
}

func TestUnencodableTagExampleDropped(t *testing.T) {
	g := gen(t)
	s := g.newSchemaFromType(reflect.TypeFor[nanExample]())
	_ = s
	var found bool
	for _, e := range g.Errors() {
		if strings.Contains(e.Error(), "not JSON-encodable") && strings.Contains(e.Error(), "field=v") {
			found = true
		}
	}
	assert.True(t, found, "errors: %v", g.Errors())
	sc := g.api.Components.Schemas[g.typeName(reflect.TypeFor[nanExample]())]
	assert.Empty(t, sc.Properties["v"].Examples)
	assert.Equal(t, []any{1.5}, sc.Properties["w"].Examples)
}
