package kitchensink

import (
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/wI2L/fizz"
	"github.com/wI2L/fizz/internal/kitchensink/a/models"
	bmodels "github.com/wI2L/fizz/internal/kitchensink/b/models"
	"github.com/wI2L/fizz/internal/kitchensink/generic"
	"github.com/wI2L/fizz/openapi"
)

// Types and handlers for the operation, naming and document cases.
// Every identifier starts with O so sibling cases_*.go files cannot clash.

type OBodyA struct {
	A string `json:"a"`
}
type OBodyB struct {
	B string `json:"b"`
}

// OXInput has the name the default request body naming gives to the
// operation with ID "oX".
type OXInput struct {
	FromGoType string `json:"fromGoType"`
}
type OXBody struct {
	FromBody string `json:"fromBody"`
}

type OTyperA struct {
	A string `json:"a"`
}
type OTyperB struct {
	B string `json:"b"`
}

func (OTyperA) TypeName() string { return "OShared" }
func (OTyperB) TypeName() string { return "OShared" }

type OEmpty struct{}

func oCreateA(*gin.Context, *OBodyA) error                   { return nil }
func oCreateB(*gin.Context, *OBodyB) error                   { return nil }
func oGetXInput(*gin.Context, *OEmpty) (*OXInput, error)     { return &OXInput{}, nil }
func oPostXBody(*gin.Context, *OXBody) error                 { return nil }
func oGetItemA(*gin.Context, *OEmpty) (*models.Item, error)  { return nil, nil }
func oGetItemB(*gin.Context, *OEmpty) (*bmodels.Item, error) { return nil, nil }
func oGetTyperA(*gin.Context, *OEmpty) (*OTyperA, error)     { return nil, nil }
func oGetTyperB(*gin.Context, *OEmpty) (*OTyperB, error)     { return nil, nil }
func oGetPage(*gin.Context, *OEmpty) (*generic.Page[models.Item], error) {
	return nil, nil
}
func oGetPair(*gin.Context, *OEmpty) (*generic.Pair[string, int], error) {
	return nil, nil
}
func oGenericHandler[T any](*gin.Context, *OEmpty) (*T, error) { return nil, nil }
func oPlain(*gin.Context, *OEmpty) error                       { return nil }

type OSvc struct{}

func (*OSvc) List(*gin.Context, *OEmpty) error { return nil }

// oMutator mutates the operation it gets from the context.
func oMutator(c *gin.Context, _ *OEmpty) error {
	op, err := fizz.OperationFromContext(c)
	if err != nil {
		return err
	}
	op.Summary = "mutated by handler"
	op.Tags = append(op.Tags, "mutated")
	return nil
}

type OUser struct {
	Name string `json:"name"`
}
type OHolder struct {
	Owner OUser `json:"owner" description:"the owner"`
}

// OExample has a field hidden from JSON that a YAML encoder would show.
type OExample struct {
	Visible string `json:"visible"`
	Secret  string `json:"-" yaml:"secret"`
}

func oGetHolder(*gin.Context, *OEmpty) (*OHolder, error) { return nil, nil }

func opts(o ...fizz.OperationOption) []fizz.OperationOption { return o }

// serve performs a GET on f and returns the body.
func serve(f *fizz.Fizz, path string) string {
	w := httptest.NewRecorder()
	f.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w.Body.String()
}

const (
	ctJSON  = "application~1json"
	schemas = "/components/schemas/"
)

func init() {
	register(
		Case{
			// Op IDs createUser and CreateUser both become CreateUserInput:
			// the second body overwrites the first.
			Name: "ops/body-name-collision-op-ids",
			Register: func(f *fizz.Fizz) {
				f.POST("/a", opts(fizz.ID("createUser")), tonic.Handler(oCreateA, 201))
				f.POST("/b", opts(fizz.ID("CreateUser")), tonic.Handler(oCreateB, 201))
			},
			Expect: []Fragment{
				{Pointer: schemas + "CreateUserInput/properties/a", JSON: `{"type":"string"}`},
				{Pointer: schemas + "CreateUserInput/properties/b", Absent: true},
			},
			Errors:  []string{"CreateUserInput"},
			FixedBy: "P4.6",
		},
		Case{
			// The Go type OXInput is the component KitchensinkOXInput, and
			// the body of operation "kitchensinkOX" gets the same name: the
			// Go type is kept and an error is recorded.
			Name: "ops/body-name-collision-go-type",
			Register: func(f *fizz.Fizz) {
				f.GET("/x", opts(fizz.ID("getX")), tonic.Handler(oGetXInput, 200))
				f.POST("/x", opts(fizz.ID("kitchensinkOX")), tonic.Handler(oPostXBody, 201))
			},
			Expect: []Fragment{
				{Pointer: schemas + "KitchensinkOXInput/properties/fromGoType", JSON: `{"type":"string"}`},
				{Pointer: schemas + "KitchensinkOXInput/properties/fromBody", Absent: true},
			},
			Errors:  []string{"KitchensinkOXInput"},
			FixedBy: "P4.6",
		},
		Case{
			// Names must match ^[A-Za-z0-9._-]+$.
			Name: "ops/generic-component-names",
			Register: func(f *fizz.Fizz) {
				f.GET("/page", opts(fizz.ID("getPage")), tonic.Handler(oGetPage, 200))
				f.GET("/pair", opts(fizz.ID("getPair")), tonic.Handler(oGetPair, 200))
			},
			Expect: []Fragment{
				{Pointer: "/paths/~1page/get/responses/200/content/" + ctJSON + "/schema/$ref", JSON: `"#/components/schemas/GenericPage_models.Item"`},
				{Pointer: "/paths/~1pair/get/responses/200/content/" + ctJSON + "/schema/$ref", JSON: `"#/components/schemas/GenericPair_string_int"`},
			},
			FixedBy: "P4.5",
		},
		Case{
			Name: "ops/same-name-packages",
			Register: func(f *fizz.Fizz) {
				f.GET("/a", opts(fizz.ID("getA")), tonic.Handler(oGetItemA, 200))
				f.GET("/b", opts(fizz.ID("getB")), tonic.Handler(oGetItemB, 200))
			},
			// First type wins, the second is reported and not merged.
			Expect: []Fragment{
				{Pointer: schemas + "ModelsItem/properties/id", JSON: `{"type":"integer"}`},
				{Pointer: schemas + "ModelsItem/properties/sku", Absent: true},
			},
			Errors:  []string{"ModelsItem"},
			FixedBy: "P4.5",
		},
		Case{
			Name: "ops/typer-name-collision",
			Register: func(f *fizz.Fizz) {
				f.GET("/a", opts(fizz.ID("getA")), tonic.Handler(oGetTyperA, 200))
				f.GET("/b", opts(fizz.ID("getB")), tonic.Handler(oGetTyperB, 200))
			},
			Expect: []Fragment{
				{Pointer: schemas + "OShared/properties/a", JSON: `{"type":"string"}`},
				{Pointer: schemas + "OShared/properties/b", Absent: true},
			},
			Errors:  []string{"OShared"},
			FixedBy: "P4.5",
		},
		Case{
			// Default ID of a generic handler is "]" today.
			Name: "ops/default-id-generic-handler",
			Register: func(f *fizz.Fizz) {
				f.GET("/g", nil, tonic.Handler(oGenericHandler[OUser], 200))
			},
			Expect:  []Fragment{{Pointer: "/paths/~1g/get/operationId", JSON: `"oGenericHandler"`}},
			FixedBy: "P4.7",
		},
		Case{
			// Default ID of a method value ends in "-fm" today.
			Name: "ops/default-id-method-value",
			Register: func(f *fizz.Fizz) {
				f.GET("/m", nil, tonic.Handler((&OSvc{}).List, 200))
			},
			Expect:  []Fragment{{Pointer: "/paths/~1m/get/operationId", JSON: `"List"`}},
			FixedBy: "P4.7",
		},
		Case{
			Name: "ops/response-descriptions",
			Register: func(f *fizz.Fizz) {
				f.GET("/r", opts(
					fizz.ID("getR"),
					fizz.Response("default", "", nil, nil, nil),
					fizz.Response("4XX", "", nil, nil, nil),
					fizz.Response("299", "", nil, nil, nil),
				), tonic.Handler(oPlain, 204))
			},
			Expect: []Fragment{
				{Pointer: "/paths/~1r/get/responses/default/description", JSON: `"Default response"`},
				{Pointer: "/paths/~1r/get/responses/4XX/description", JSON: `"4XX response"`},
				{Pointer: "/paths/~1r/get/responses/299/description", JSON: `"Status 299"`},
			},
			FixedBy: "P4.4",
		},
		Case{
			// An empty examples map must not create a media type.
			Name: "ops/response-empty-examples",
			Register: func(f *fizz.Fizz) {
				f.GET("/e", opts(
					fizz.ID("getE"),
					fizz.ResponseWithExamples("400", "bad", nil, nil, map[string]any{}),
				), tonic.Handler(oPlain, 204))
			},
			Expect:  []Fragment{{Pointer: "/paths/~1e/get/responses/400/content", Absent: true}},
			FixedBy: "P4.4",
		},
		Case{
			// Panics today (fizz.go Handle) instead of recording an error.
			Name: "ops/response-code-clash-default-status",
			Register: func(f *fizz.Fizz) {
				f.GET("/c", opts(
					fizz.ID("getC"),
					fizz.Response("200", "again", nil, nil, nil),
				), tonic.Handler(oPlain, 200))
			},
			Errors:  []string{"200"},
			FixedBy: "P4.8",
		},
		Case{
			Name: "ops/security-nil-scopes",
			Register: func(f *fizz.Fizz) {
				f.Generator().SetSecurityRequirement([]*openapi.SecurityRequirement{
					{"bearer": nil},
				})
			},
			Expect:  []Fragment{{Pointer: "/security", JSON: `[{"bearer":[]}]`}},
			FixedBy: "P2.3",
		},
		Case{
			Name: "ops/oauth-flow-without-scopes",
			Register: func(f *fizz.Fizz) {
				f.Generator().SetSecuritySchemes(map[string]*openapi.SecuritySchemeOrRef{
					"oauth": {SecurityScheme: &openapi.SecurityScheme{
						Type: "oauth2",
						Flows: &openapi.OAuthFlows{ClientCredentials: &openapi.OAuthFlow{
							TokenURL: "https://example.com/token",
						}},
					}},
				})
			},
			Expect: []Fragment{{
				Pointer: "/components/securitySchemes/oauth/flows/clientCredentials",
				JSON:    `{"tokenUrl":"https://example.com/token","scopes":{}}`,
			}},
			FixedBy: "P2.3",
		},
		Case{
			// A later Group with an empty description wipes the first.
			Name: "ops/group-tag-empty-description-overwrite",
			Register: func(f *fizz.Fizz) {
				f.Group("/g1", "grp", "First description")
				f.Group("/g2", "grp", "")
			},
			Expect:  []Fragment{{Pointer: "/tags/0/description", JSON: `"First description"`}},
			FixedBy: "P6.5",
		},
		Case{
			// The brief asks for an error on the empty overwrite; the plan
			// (P6.5) only warns on a conflicting non-empty text. This case
			// covers the latter.
			Name: "ops/group-tag-conflicting-description",
			Register: func(f *fizz.Fizz) {
				f.Group("/g1", "grp", "First description")
				f.Group("/g2", "grp", "Second description")
			},
			Expect:  []Fragment{{Pointer: "/tags/0/description", JSON: `"First description"`}},
			Errors:  []string{"grp"},
			FixedBy: "P6.5",
		},
		Case{
			// The handler mutates the operation it is given. The document
			// must not change: the context holds a snapshot.
			Name: "ops/operation-from-context-snapshot",
			Register: func(f *fizz.Fizz) {
				f.GET("/op", opts(fizz.ID("getOp"), fizz.Summary("original")), tonic.Handler(oMutator, 204))
				serve(f, "/op")
			},
			Expect: []Fragment{
				{Pointer: "/paths/~1op/get/summary", JSON: `"original"`},
				{Pointer: "/paths/~1op/get/tags", Absent: true},
			},
			FixedBy: "P5.1",
		},
		Case{
			// Handle wraps the variadic slice in place: the second route
			// receives an already wrapped handler and is undocumented.
			Name: "ops/handle-shared-handlers-slice",
			Register: func(f *fizz.Fizz) {
				shared := []gin.HandlerFunc{tonic.Handler(oPlain, 204)}
				f.GET("/one", opts(fizz.ID("one")), shared...)
				f.GET("/two", opts(fizz.ID("two")), shared...)
			},
			Expect: []Fragment{
				{Pointer: "/paths/~1one/get/operationId", JSON: `"one"`},
				{Pointer: "/paths/~1two/get/operationId", JSON: `"two"`},
			},
			FixedBy: "P4.9",
		},
		Case{
			// A json:"-" field must not appear in the YAML document.
			Name: "ops/yaml-json-parity-hidden-field",
			Register: func(f *fizz.Fizz) {
				ex := OExample{Visible: "visible-value", Secret: "TOP-SECRET-VALUE"}
				f.GET("/y", opts(
					fizz.ID("getY"),
					fizz.Response("400", "bad", nil, nil, ex),
				), tonic.Handler(oPlain, 204))
				info := &openapi.Info{Title: "t", Version: "0"}
				f.GET("/openapi.json", nil, f.OpenAPI(info, "json"))
				f.GET("/openapi.yaml", nil, f.OpenAPI(info, "yaml"))
				// The harness only builds JSON, so the YAML check lives
				// here and reports by panicking.
				y, j := serve(f, "/openapi.yaml"), serve(f, "/openapi.json")
				switch {
				case strings.Contains(j, "TOP-SECRET-VALUE"):
					panic("json leaks a json:\"-\" field")
				case strings.Contains(y, "TOP-SECRET-VALUE"):
					panic("yaml leaks a json:\"-\" field")
				case !strings.Contains(y, "visible-value"):
					panic("yaml lost the visible field")
				}
			},
			Expect: []Fragment{
				{Pointer: "/paths/~1y/get/responses/400/content/" + ctJSON + "/example", JSON: `{"visible":"visible-value"}`},
			},
			FixedBy: "P2.5",
		},
		Case{
			// Use-site description next to a $ref (valid in 3.1+).
			Name: "ops/ref-with-description-sibling",
			Register: func(f *fizz.Fizz) {
				f.GET("/h", opts(fizz.ID("getH")), tonic.Handler(oGetHolder, 200))
			},
			Expect: []Fragment{{
				Pointer: schemas + "KitchensinkOHolder/properties/owner",
				JSON:    `{"$ref":"#/components/schemas/KitchensinkOUser","description":"the owner"}`,
			}},
			FixedBy: "P3.4",
		},
		Case{
			// Both fields nil must be reported, never emitted as {}.
			Name: "ops/orref-both-nil",
			Register: func(f *fizz.Fizz) {
				f.Generator().SetSecuritySchemes(map[string]*openapi.SecuritySchemeOrRef{
					"empty": {},
				})
				f.Generator().API() // validation runs at build
			},
			Expect:  []Fragment{{Pointer: "/components/securitySchemes/empty", Absent: true}},
			Errors:  []string{"empty"},
			FixedBy: "P4.8",
		},
	)
}
