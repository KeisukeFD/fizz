package fizz

import (
	"context"
	"errors"
	"path"
	"reflect"
	"runtime"
	"slices"
	"sync"

	"github.com/KeisukeFD/fizz/openapi"
	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"
)

// Group creates a new group of routes.
func (g *RouterGroup) Group(path, name, description string, handlers ...gin.HandlerFunc) *RouterGroup {
	// Create the tag in the specification
	// for this groups.
	g.gen.AddTag(name, description)

	return &RouterGroup{
		gen:         g.gen,
		idNamer:     g.idNamer,
		bindTags:    g.bindTags,
		noBindHook:  g.noBindHook,
		group:       g.group.Group(path, handlers...),
		Name:        name,
		Description: description,
	}
}

// Use adds middleware to the group.
func (g *RouterGroup) Use(handlers ...gin.HandlerFunc) {
	g.group.Use(handlers...)
}

// GinRouterGroup returns the underlying Gin router group.
func (g *RouterGroup) GinRouterGroup() *gin.RouterGroup {
	return g.group
}

// GET is a shortcut to register a new handler with the GET method.
func (g *RouterGroup) GET(path string, infos []OperationOption, handlers ...gin.HandlerFunc) *RouterGroup {
	return g.Handle(path, "GET", infos, handlers...)
}

// POST is a shortcut to register a new handler with the POST method.
func (g *RouterGroup) POST(path string, infos []OperationOption, handlers ...gin.HandlerFunc) *RouterGroup {
	return g.Handle(path, "POST", infos, handlers...)
}

// PUT is a shortcut to register a new handler with the PUT method.
func (g *RouterGroup) PUT(path string, infos []OperationOption, handlers ...gin.HandlerFunc) *RouterGroup {
	return g.Handle(path, "PUT", infos, handlers...)
}

// PATCH is a shortcut to register a new handler with the PATCH method.
func (g *RouterGroup) PATCH(path string, infos []OperationOption, handlers ...gin.HandlerFunc) *RouterGroup {
	return g.Handle(path, "PATCH", infos, handlers...)
}

// DELETE is a shortcut to register a new handler with the DELETE method.
func (g *RouterGroup) DELETE(path string, infos []OperationOption, handlers ...gin.HandlerFunc) *RouterGroup {
	return g.Handle(path, "DELETE", infos, handlers...)
}

// OPTIONS is a shortcut to register a new handler with the OPTIONS method.
func (g *RouterGroup) OPTIONS(path string, infos []OperationOption, handlers ...gin.HandlerFunc) *RouterGroup {
	return g.Handle(path, "OPTIONS", infos, handlers...)
}

// HEAD is a shortcut to register a new handler with the HEAD method.
func (g *RouterGroup) HEAD(path string, infos []OperationOption, handlers ...gin.HandlerFunc) *RouterGroup {
	return g.Handle(path, "HEAD", infos, handlers...)
}

// TRACE is a shortcut to register a new handler with the TRACE method.
func (g *RouterGroup) TRACE(path string, infos []OperationOption, handlers ...gin.HandlerFunc) *RouterGroup {
	return g.Handle(path, "TRACE", infos, handlers...)
}

// Handle registers a new request handler that is wrapped
// with Tonic and documented in the OpenAPI specification.
func (g *RouterGroup) Handle(path, method string, infos []OperationOption, handlers ...gin.HandlerFunc) *RouterGroup {
	// Work on a copy: the variadic slice may be the caller's own
	// (hs...), and wrapping below must not write into it.
	handlers = slices.Clone(handlers)

	oi := &openapi.OperationInfo{}
	for _, info := range infos {
		info(oi)
	}
	type wrap struct {
		h gin.HandlerFunc
		r *tonic.Route
	}
	var wrapped []wrap

	// Find the handlers wrapped with Tonic.
	for _, h := range handlers {
		r, err := tonic.GetRouteByHandler(h)
		if err == nil {
			wrapped = append(wrapped, wrap{h: h, r: r})
		}
	}
	// Check that no more that one tonic-wrapped handler
	// is registered for this operation.
	if len(wrapped) > 1 {
		// The operation is not documented, the route is still served.
		g.gen.RecordError(&openapi.OperationError{
			Method: method,
			Path:   openAPIPath(joinPaths(g.group.BasePath(), path)),
			ID:     oi.ID,
			Err:    errors.New("multiple tonic-wrapped handlers used for the operation"),
		})
		g.group.Handle(method, path, handlers...)
		return g
	}
	// If we have a tonic-wrapped handler, generate the
	// specification of this operation.
	if len(wrapped) == 1 {
		hfunc := wrapped[0].r

		oi.StatusCode = hfunc.GetDefaultStatusCode()

		// Set an input type if provided.
		it := hfunc.InputType()
		if oi.InputModel != nil {
			it = reflect.TypeOf(oi.InputModel)
		}

		// Consolidate path for OpenAPI spec.
		operationPath := joinPaths(g.group.BasePath(), path)

		// Set an operation ID if none is provided. A default ID
		// that is already taken falls back (see defaultOperationID);
		// an explicit one never does and fails in AddOperation.
		if oi.ID == "" {
			h := hfunc.GetHandler()
			name := ""
			if fn := runtime.FuncForPC(h.Pointer()); fn != nil {
				name = fn.Name()
			}
			oi.ID = defaultOperationID(method, openAPIPath(operationPath), name, g.idNamer, g.gen.OperationIDUsed)
		}

		oi.WildcardParams = wildcardParams(operationPath)

		// Add operation to the OpenAPI spec.
		// On failure the generator has recorded the error (see
		// Errors) and the route is registered with the original
		// handlers, undocumented.
		operation, err := g.gen.AddOperation(openAPIPath(operationPath), method, g.Name, it, hfunc.OutputType(), oi)
		// If an operation was generated for the handler,
		// wrap the Tonic-wrapped handled with a closure
		// to inject it into the Gin context.
		if err == nil && operation != nil {
			// The handlers get a frozen copy of the operation, in
			// the request context (not in c.Keys): the live one
			// belongs to the document.
			snap, cerr := operation.Clone()
			if cerr != nil {
				g.gen.RecordError(&openapi.OperationError{Method: method, Path: openAPIPath(operationPath), ID: oi.ID, Err: cerr})
			}
			plan := g.bindPlan(method, openAPIPath(operationPath), oi.ID, hfunc, oi.InputModel)
			var warnOnce sync.Once
			opID, quiet := oi.ID, g.noBindHook
			for i, h := range handlers {
				if cerr == nil && funcEqual(h, wrapped[0].h) {
					orig := h // copy the original func
					handlers[i] = func(c *gin.Context) {
						ctx := withOperation(c.Request.Context(), snap)
						var st *bindState
						if plan != nil {
							st = &bindState{plan: plan}
							ctx = context.WithValue(ctx, ctxBindStateKey{}, st)
						}
						c.Request = c.Request.WithContext(ctx)
						orig(c)
						if st != nil && !st.ran && !quiet {
							warnOnce.Do(func() { warnBindHookNotRun(opID) })
						}
					}
				}
			}
		}
	}
	// Register the handlers with Gin underlying group.
	g.group.Handle(method, path, handlers...)

	return g
}

// withOperation returns a context that holds the frozen operation op.
func withOperation(ctx context.Context, op *openapi.Operation) context.Context {
	return context.WithValue(ctx, ctxOperationKey{}, op)
}

// OperationFromContext returns the OpenAPI operation of the route
// that is handling the request, or an error if none is found. ctx is
// the request context (c.Request.Context()) or a *gin.Context, which
// is read through its request even if gin's ContextWithFallback is off.
//
// The operation is a deep copy of the one documented when the route was
// registered: each call returns a new one, and changing it affects
// neither the other callers nor the specification.
func OperationFromContext(ctx context.Context) (*openapi.Operation, error) {
	if c, ok := ctx.(*gin.Context); ok {
		if c.Request == nil {
			return nil, errors.New("operation not found")
		}
		ctx = c.Request.Context()
	}
	v := ctx.Value(ctxOperationKey{})
	if v == nil {
		return nil, errors.New("operation not found")
	}
	op, ok := v.(*openapi.Operation)
	if !ok {
		return nil, errors.New("invalid type: not an operation")
	}
	return op.Clone()
}

func joinPaths(abs, rel string) string {
	if rel == "" {
		return abs
	}
	final := path.Join(abs, rel)
	as := lastChar(rel) == '/' && lastChar(final) != '/'
	if as {
		return final + "/"
	}
	return final
}

func lastChar(str string) uint8 {
	if str == "" {
		panic("empty string")
	}
	return str[len(str)-1]
}

func funcEqual(f1, f2 any) bool {
	v1 := reflect.ValueOf(f1)
	v2 := reflect.ValueOf(f2)

	if v1.Kind() == reflect.Func && v2.Kind() == reflect.Func { // prevent panic on call to Pointer()
		return runtime.FuncForPC(v1.Pointer()).Entry() == runtime.FuncForPC(v2.Pointer()).Entry()
	}
	return false
}
