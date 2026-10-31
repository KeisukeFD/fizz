package fizz

import (
	"reflect"

	"github.com/wI2L/fizz/openapi"
)

// config holds the settings built from the Options passed
// to New and NewFromEngine.
type config struct {
	tagNames         openapi.TagNames
	schemaNamer      func(reflect.Type) string
	pointersNullable bool
	annotationStyle  openapi.AnnotationStyle
	mediaType        string
	noBindHook       bool // consumed by P7.2b; no effect yet
}

// Option configures a Fizz instance. Options are passed to
// New and NewFromEngine and applied in order; nil options are ignored.
type Option func(*config)

// WithTagNames sets the names of the fizz-owned struct tags. A zero
// field keeps the default name and "-" disables that tag. The binder
// tags (path, query, header, enum, default, validate, explode) are
// fixed to tonic's and cannot be renamed.
func WithTagNames(n openapi.TagNames) Option {
	return func(c *config) { c.tagNames = n }
}

// WithSchemaNamer sets the function that names the component schema
// of a type. A nil function selects the default naming.
func WithSchemaNamer(fn func(reflect.Type) string) Option {
	return func(c *config) { c.schemaNamer = fn }
}

// WithPointersAreNullable controls whether a pointer type also accepts
// null in the generated schema. It defaults to true.
func WithPointersAreNullable(b bool) Option {
	return func(c *config) { c.pointersNullable = b }
}

// WithAnnotationStyle selects how use-site annotations are attached to
// a schema that is a $ref. It defaults to openapi.SiblingAnnotations.
func WithAnnotationStyle(s openapi.AnnotationStyle) Option {
	return func(c *config) { c.annotationStyle = s }
}

// WithoutBindHook opts out of the process-global tonic bind hook that
// fizz installs to bind deepObject query and cookie parameters. The
// hook itself is not implemented yet, so this option currently has no
// effect.
func WithoutBindHook() Option {
	return func(c *config) { c.noBindHook = true }
}

// WithDefaultMediaType sets the media type used for request bodies and
// responses. An empty string selects tonic.MediaType().
func WithDefaultMediaType(mediaType string) Option {
	return func(c *config) { c.mediaType = mediaType }
}
