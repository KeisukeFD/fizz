package fizz

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"
	"github.com/wI2L/fizz/openapi"
)

const ctxOpenAPIOperation = "_ctx_openapi_operation"

// Primitive type helpers.
var (
	Integer  int32
	Long     int64
	Float    float32
	Double   float64
	String   string
	Byte     []byte
	Binary   []byte
	Boolean  bool
	DateTime time.Time
)

// Fizz is an abstraction of a Gin engine that wraps the
// routes handlers with Tonic and generates an OpenAPI
// 3.0 specification from it.
type Fizz struct {
	gen    *openapi.Generator
	engine *gin.Engine
	*RouterGroup
}

// RouterGroup is an abstraction of a Gin router group.
type RouterGroup struct {
	group       *gin.RouterGroup
	gen         *openapi.Generator
	Name        string
	Description string
}

// New creates a new Fizz wrapper for
// a default Gin engine.
func New(opts ...Option) *Fizz {
	return NewFromEngine(gin.New(), opts...)
}

// NewFromEngine creates a new Fizz wrapper
// from an existing Gin engine.
func NewFromEngine(e *gin.Engine, opts ...Option) *Fizz {
	var cfg config
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	// Create a new spec with the config
	// based on tonic internals.
	gen, _ := openapi.NewGenerator(
		&openapi.SpecGenConfig{
			ValidatorTag:      tonic.ValidationTag,
			PathLocationTag:   tonic.PathTag,
			QueryLocationTag:  tonic.QueryTag,
			HeaderLocationTag: tonic.HeaderTag,
			EnumTag:           tonic.EnumTag,
			DefaultTag:        tonic.DefaultTag,
			DefaultMediaType:  tonic.MediaType(),
		},
	)
	return &Fizz{
		engine: e,
		gen:    gen,
		RouterGroup: &RouterGroup{
			group: &e.RouterGroup,
			gen:   gen,
		},
	}
}

// ServeHTTP implements http.HandlerFunc for Fizz.
func (f *Fizz) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.engine.ServeHTTP(w, r)
}

// Engine returns the underlying Gin engine.
func (f *Fizz) Engine() *gin.Engine {
	return f.engine
}

// GinRouterGroup returns the underlying Gin router group.
func (f *Fizz) GinRouterGroup() *gin.RouterGroup {
	return f.group
}

// Generator returns the underlying OpenAPI generator.
func (f *Fizz) Generator() *openapi.Generator {
	return f.gen
}

// Errors returns the errors that may have occurred
// during the spec generation.
func (f *Fizz) Errors() []error {
	return f.gen.Errors()
}
