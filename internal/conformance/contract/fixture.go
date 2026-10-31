package contract

import (
	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/wI2L/fizz"
	"github.com/wI2L/fizz/openapi"
)

// The handlers of the fixture answer 200 with their input, which is the
// value that tonic and the bind hook of fizz bound. Every parameter field
// carries a json tag equal to the parameter name, so the echo key is the
// name (Target.Fields is not needed).

type scalarsIn struct {
	PS string  `path:"ps" json:"ps"`
	PI int     `path:"pi" json:"pi"`
	PN float64 `path:"pn" json:"pn"`
	PB bool    `path:"pb" json:"pb"`

	QS string  `query:"qs" json:"qs"`
	QI int     `query:"qi" json:"qi"`
	QN float64 `query:"qn" json:"qn"`
	QB bool    `query:"qb" json:"qb"`

	CS string  `cookie:"cs" json:"cs"`
	CI int     `cookie:"ci" json:"ci"`
	CN float64 `cookie:"cn" json:"cn"`
	CB bool    `cookie:"cb" json:"cb"`
}

// headerScalarsIn is apart from scalarsIn: tonic fails when an optional
// numeric or boolean header is absent (P10.7i).
type headerScalarsIn struct {
	HS string  `header:"X-Str" json:"X-Str"`
	HI int     `header:"X-Int" json:"X-Int"`
	HN float64 `header:"X-Num" json:"X-Num"`
	HB bool    `header:"X-Bool" json:"X-Bool"`
}

type arraysIn struct {
	Tags  []string `query:"tags" json:"tags"`
	IDs   []int    `query:"ids" json:"ids"`
	CSV   []string `query:"csv" explode:"false" json:"csv"`
	CSVN  []int    `query:"csvn" explode:"false" json:"csvn"`
	Space []string `query:"space" style:"spaceDelimited" json:"space"`
	Pipe  []string `query:"pipe" style:"pipeDelimited" json:"pipe"`
}

type defaultsIn struct {
	Limit    int    `query:"limit" default:"10" json:"limit"`
	Sort     string `query:"sort" default:"asc" json:"sort"`
	Debug    bool   `query:"debug" default:"true" json:"debug"`
	Lang     string `header:"X-Lang" default:"en" json:"X-Lang"`
	Theme    string `cookie:"theme,default=dark" json:"theme"`
	Currency string `cookie:"currency" default:"EUR" json:"currency"`
}

// sliceDefaultIn has slice defaults, which fizz rejects today (P10.7c).
type sliceDefaultIn struct {
	IDs   []int    `query:"ids" default:"1,2,3" json:"ids"`
	Names []string `query:"names" default:"a,b" json:"names"`
}

type filterOpts struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

type filterIn struct {
	Filter map[string]string `query:"filter" style:"deepObject" json:"filter"`
	Scores map[string]int    `query:"scores" style:"deepObject" json:"scores"`
	Opts   *filterOpts       `query:"opts" style:"deepObject" json:"opts"`
}

// One slice header per operation, so that the other one being absent does
// not fail the request (P10.7i).
type headerArrIn struct {
	Arr []string `header:"X-Arr" json:"X-Arr"`
}

type headerNumsIn struct {
	Nums []int `header:"X-Nums" json:"X-Nums"`
}

// headerEnumIn is apart from enumsIn for the same reason (P10.7i).
type headerEnumIn struct {
	Mode string `header:"X-Mode" enum:"fast,slow" json:"X-Mode"`
}

type enumsIn struct {
	Kind   string `path:"kind" enum:"a,b" json:"kind"`
	Color  string `query:"color" enum:"red,green,blue" json:"color"`
	Level  int    `query:"level" enum:"1,2,3" json:"level"`
	Flavor string `cookie:"flavor" enum:"sweet,salty" json:"flavor"`
}

type nullableIn struct {
	Q *string `query:"nq" json:"nq"`
	N *int    `query:"nn" json:"nn"`
	B *bool   `query:"nb" json:"nb"`
	H *string `header:"X-Nullable" json:"X-Nullable"`
	C *int    `cookie:"nc" json:"nc"`
}

// specialIn has values that need percent-encoding, and a reserved one.
type specialIn struct {
	SP string `path:"sp" example:"a b&c=d,e" json:"sp"`
	QS string `query:"qsp" example:"a b&c=d/e?é" json:"qsp"`
	RS string `query:"rs" allowReserved:"true" example:"a/b?c=d:e" json:"rs"`
	HS string `header:"X-Special" example:"a b&c=d/e" json:"X-Special"`
	CS string `cookie:"csp" example:"a b&c=d/e" json:"csp"`
}

type labelIn struct {
	PL string `path:"pl" style:"label" json:"pl"`
}

type matrixIn struct {
	PM string `path:"pm" style:"matrix" json:"pm"`
}

type pathArrIn struct {
	PA []string `path:"pa" json:"pa"`
}

// New returns the fixture: a fizz app with every parameter shape that fizz
// documents. It has no map or struct parameter outside deepObject (see
// NewBadParam).
func New() *fizz.Fizz {
	f := fizz.New()
	f.Generator().SetInfo(&openapi.Info{Title: "Contract fixture", Version: "1.0.0"})
	id := func(s string) []fizz.OperationOption { return []fizz.OperationOption{fizz.ID(s)} }

	f.GET("/scalars/:ps/:pi/:pn/:pb", id("scalars"), tonic.Handler(func(c *gin.Context, in *scalarsIn) (*scalarsIn, error) { return in, nil }, 200))
	f.GET("/arrays", id("arrays"), tonic.Handler(func(c *gin.Context, in *arraysIn) (*arraysIn, error) { return in, nil }, 200))
	f.GET("/defaults", id("defaults"), tonic.Handler(func(c *gin.Context, in *defaultsIn) (*defaultsIn, error) { return in, nil }, 200))
	f.GET("/filter", id("filter"), tonic.Handler(func(c *gin.Context, in *filterIn) (*filterIn, error) { return in, nil }, 200))
	f.GET("/headerscalars", id("headerscalars"), tonic.Handler(func(c *gin.Context, in *headerScalarsIn) (*headerScalarsIn, error) { return in, nil }, 200))
	f.GET("/headerarr", id("headerarr"), tonic.Handler(func(c *gin.Context, in *headerArrIn) (*headerArrIn, error) { return in, nil }, 200))
	f.GET("/headernums", id("headernums"), tonic.Handler(func(c *gin.Context, in *headerNumsIn) (*headerNumsIn, error) { return in, nil }, 200))
	f.GET("/headerenum", id("headerenum"), tonic.Handler(func(c *gin.Context, in *headerEnumIn) (*headerEnumIn, error) { return in, nil }, 200))
	f.GET("/enums/:kind", id("enums"), tonic.Handler(func(c *gin.Context, in *enumsIn) (*enumsIn, error) { return in, nil }, 200))
	f.GET("/nullable", id("nullable"), tonic.Handler(func(c *gin.Context, in *nullableIn) (*nullableIn, error) { return in, nil }, 200))
	f.GET("/special/:sp", id("special"), tonic.Handler(func(c *gin.Context, in *specialIn) (*specialIn, error) { return in, nil }, 200))
	f.GET("/label/:pl", id("label"), tonic.Handler(func(c *gin.Context, in *labelIn) (*labelIn, error) { return in, nil }, 200))
	f.GET("/matrix/:pm", id("matrix"), tonic.Handler(func(c *gin.Context, in *matrixIn) (*matrixIn, error) { return in, nil }, 200))
	f.GET("/patharr/:pa", id("patharr"), tonic.Handler(func(c *gin.Context, in *pathArrIn) (*pathArrIn, error) { return in, nil }, 200))
	return f
}

// NewSliceDefault returns an app with slice defaults (review finding 4,
// P10.7c): the default tag is split on commas by tonic, so the document
// should state [1, 2, 3], and fizz should not record an error.
func NewSliceDefault() *fizz.Fizz {
	f := fizz.New()
	f.Generator().SetInfo(&openapi.Info{Title: "Contract fixture: slice default", Version: "1.0.0"})
	f.GET("/slicedefault", []fizz.OperationOption{fizz.ID("slicedefault")}, tonic.Handler(func(c *gin.Context, in *sliceDefaultIn) (*sliceDefaultIn, error) { return in, nil }, 200))
	return f
}

type mapQueryIn struct {
	M map[string]string `query:"m" json:"m"`
}

type structQueryIn struct {
	S *filterOpts `query:"s" json:"s"`
}

type mapHeaderIn struct {
	M map[string]string `header:"X-Map" json:"X-Map"`
}

// BadParams lists the variants of NewBadParam.
var BadParams = []string{"map query", "struct query", "map header"}

// NewBadParam returns an app with one parameter that tonic cannot bind and
// that the document should not offer (P10.7b): a map or struct outside
// deepObject. The app has no spec error today, which is the drift.
func NewBadParam(variant string) *fizz.Fizz {
	f := fizz.New()
	f.Generator().SetInfo(&openapi.Info{Title: "Contract fixture: bad parameter", Version: "1.0.0"})
	id := []fizz.OperationOption{fizz.ID("bad")}
	switch variant {
	case "map query":
		f.GET("/bad", id, tonic.Handler(func(c *gin.Context, in *mapQueryIn) (*mapQueryIn, error) { return in, nil }, 200))
	case "struct query":
		f.GET("/bad", id, tonic.Handler(func(c *gin.Context, in *structQueryIn) (*structQueryIn, error) { return in, nil }, 200))
	case "map header":
		f.GET("/bad", id, tonic.Handler(func(c *gin.Context, in *mapHeaderIn) (*mapHeaderIn, error) { return in, nil }, 200))
	default:
		panic("contract: unknown variant " + variant)
	}
	return f
}
