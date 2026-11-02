package fizz

import (
	"reflect"

	"github.com/KeisukeFD/fizz/openapi"
)

// config holds the settings built from the Options passed
// to New and NewFromEngine.
type config struct {
	tagNames         openapi.TagNames
	schemaNamer      func(reflect.Type) string
	bodyNamer        func(opID string, t reflect.Type) string
	pointersNullable bool
	annotationStyle  openapi.AnnotationStyle
	mediaType        string
	idNamer          func(method, path, handlerName string) string
	noBindHook       bool
	hooks            openapi.Hooks
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

// WithRequestBodyNamer sets the function that names the component
// schema of a request body from the operation ID and the input type.
// A nil function, or an empty name, selects the default: the operation
// ID with its first letter upper-cased, followed by "Input".
func WithRequestBodyNamer(fn func(opID string, t reflect.Type) string) Option {
	return func(c *config) { c.bodyNamer = fn }
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

// WithoutBindHook stops New and NewFromEngine from calling
// InstallBindHook, which wraps the process-global tonic bind hook to
// bind deepObject query parameters and cookie parameters. Without the
// hook those parameters are documented but not bound, unless you install it yourself with
// InstallBindHook or tonic.SetBindHook(fizz.BindHook(next)). It does
// not remove a hook that an earlier New already installed.
func WithoutBindHook() Option {
	return func(c *config) { c.noBindHook = true }
}

// WithDefaultMediaType sets the media type used for request bodies and
// responses. An empty string selects tonic.MediaType().
func WithDefaultMediaType(mediaType string) Option {
	return func(c *config) { c.mediaType = mediaType }
}

// WithOperationIDNamer sets the function that derives the ID of an
// operation registered without fizz.ID. It receives the HTTP method,
// the OpenAPI path (for example "/users/{id}") and the full
// runtime name of the handler (for example
// "github.com/x/pkg.(*T).Method-fm"). It is not called when fizz.ID is set. An
// empty result selects the default naming. If the ID it returns is
// already used, the ID falls back to the method and path
// ("get_users_id"). A nil function selects the default naming.
func WithOperationIDNamer(fn func(method, path, handlerName string) string) Option {
	return func(c *config) { c.idNamer = fn }
}

// WithHooks sets the generator hooks. The hooks of several calls
// compose in order: the Schema hooks all run; the Field and Parameter
// hooks run until one skips; the Operation and Document hooks run until
// one fails. A nil member is ignored. See openapi.Hooks for what each
// hook receives and openapi.Generator.SetHooks for the restrictions.
func WithHooks(h openapi.Hooks) Option {
	return func(c *config) { c.hooks = composeHooks(c.hooks, h) }
}

// composeHooks returns hooks that call a, then b.
func composeHooks(a, b openapi.Hooks) openapi.Hooks {
	var out openapi.Hooks
	switch {
	case a.Schema == nil:
		out.Schema = b.Schema
	case b.Schema == nil:
		out.Schema = a.Schema
	default:
		out.Schema = func(ctx *openapi.SchemaContext, s *openapi.Schema) {
			a.Schema(ctx, s)
			b.Schema(ctx, s)
		}
	}
	switch {
	case a.Field == nil:
		out.Field = b.Field
	case b.Field == nil:
		out.Field = a.Field
	default:
		out.Field = func(ctx *openapi.FieldContext, s *openapi.Schema) bool {
			return a.Field(ctx, s) || b.Field(ctx, s)
		}
	}
	switch {
	case a.Parameter == nil:
		out.Parameter = b.Parameter
	case b.Parameter == nil:
		out.Parameter = a.Parameter
	default:
		out.Parameter = func(ctx *openapi.ParameterContext, p *openapi.Parameter) bool {
			return a.Parameter(ctx, p) || b.Parameter(ctx, p)
		}
	}
	switch {
	case a.Operation == nil:
		out.Operation = b.Operation
	case b.Operation == nil:
		out.Operation = a.Operation
	default:
		out.Operation = func(ctx *openapi.OperationContext, op *openapi.Operation) error {
			if err := a.Operation(ctx, op); err != nil {
				return err
			}
			return b.Operation(ctx, op)
		}
	}
	switch {
	case a.Document == nil:
		out.Document = b.Document
	case b.Document == nil:
		out.Document = a.Document
	default:
		out.Document = func(doc *openapi.OpenAPI) error {
			if err := a.Document(doc); err != nil {
				return err
			}
			return b.Document(doc)
		}
	}
	return out
}
