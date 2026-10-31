package openapi

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
)

// componentNameRe is the set of names OpenAPI allows for the keys of
// the components maps.
var componentNameRe = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

func checkComponentName(kind, name string) error {
	if !componentNameRe.MatchString(name) {
		return fmt.Errorf("openapi: invalid %s component name %q: it must match %s", kind, name, componentNameRe)
	}
	return nil
}

// RegisterSchema adds a hand-written schema to the components under
// name, which must match ^[A-Za-z0-9._-]+$. The name goes through the
// type registry: if it already belongs to a Go type or to another
// registered schema, a *CollisionError is returned and the first owner
// is kept. The schema is deep-copied, so the caller may change s
// afterwards. Errors are returned, not recorded in Errors.
func (g *Generator) RegisterSchema(name string, s *Schema) error {
	defer g.mutate()()
	if s == nil {
		return errors.New("openapi: RegisterSchema: nil schema")
	}
	if err := checkComponentName("schema", name); err != nil {
		return err
	}
	if err := jsonEncodable(s); err != nil {
		return fmt.Errorf("openapi: schema %q is not JSON-encodable: %w", name, err)
	}
	if kept, ok := g.registry.owner(name); ok {
		return &CollisionError{Name: name, Kept: kept, Rejected: manualSchemaOwner}
	}
	g.registry.claim(name, manualSchemaOwner)
	g.api.Components.Schemas[name] = copySchemaTree(s)
	return nil
}

// RegisterType generates the component schema of the named type t, as
// if a route used it, so that it appears in the document even if no
// route does. A pointer type is dereferenced. It returns an error for
// a type that has no component (unnamed, or inlined), and a
// *CollisionError, not recorded, if the name of t belongs to another
// type or to a registered schema.
func (g *Generator) RegisterType(t reflect.Type) error {
	defer g.mutate()()
	if t == nil {
		return errors.New("openapi: RegisterType: nil type")
	}
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	name := g.typeName(t)
	if name == "" {
		return fmt.Errorf("openapi: RegisterType: %s is not a named type", t)
	}
	if kept, ok := g.registry.owner(name); ok && kept != t {
		return &CollisionError{Name: name, Kept: kept, Rejected: t}
	}
	snap := g.snapshot()
	g.newSchemaFromType(t)
	if owner, ok := g.registry.owner(name); !ok || owner != t || g.api.Components.Schemas[name] == nil {
		snap.restore(g)
		return fmt.Errorf("openapi: RegisterType: %s has no component schema", t)
	}
	return nil
}

// RegisterParameter adds p to components/parameters under name. The
// name rules are those of RegisterSchema, in a namespace of their own;
// a name already used by a parameter component is an error. p is
// deep-copied.
func (g *Generator) RegisterParameter(name string, p *Parameter) error {
	defer g.mutate()()
	g.api.Components.Parameters = nonNil(g.api.Components.Parameters)
	return registerComponent(g.api.Components.Parameters, "parameter", name, p)
}

// RegisterResponse adds r to components/responses under name. See
// RegisterParameter.
func (g *Generator) RegisterResponse(name string, r *Response) error {
	defer g.mutate()()
	g.api.Components.Responses = nonNil(g.api.Components.Responses)
	return registerComponent(g.api.Components.Responses, "response", name, r)
}

// RegisterExample adds e to components/examples under name. See
// RegisterParameter.
func (g *Generator) RegisterExample(name string, e *Example) error {
	defer g.mutate()()
	g.api.Components.Examples = nonNil(g.api.Components.Examples)
	return registerComponent(g.api.Components.Examples, "example", name, e)
}

// RegisterRequestBody adds rb to components/requestBodies under name.
// See RegisterParameter.
func (g *Generator) RegisterRequestBody(name string, rb *RequestBody) error {
	defer g.mutate()()
	if rb != nil && rb.Content == nil {
		return fmt.Errorf("openapi: requestBody %q has nil content: a request body needs content (an empty map is allowed)", name)
	}
	g.api.Components.RequestBodies = nonNil(g.api.Components.RequestBodies)
	return registerComponent(g.api.Components.RequestBodies, "requestBody", name, rb)
}

// RegisterHeader adds h to components/headers under name. See
// RegisterParameter.
func (g *Generator) RegisterHeader(name string, h *Header) error {
	defer g.mutate()()
	g.api.Components.Headers = nonNil(g.api.Components.Headers)
	return registerComponent(g.api.Components.Headers, "header", name, h)
}

func nonNil[V any](m map[string]V) map[string]V {
	if m == nil {
		return make(map[string]V)
	}
	return m
}

// registerComponent stores a deep copy of v in m. The copy goes through
// the encoder of the document, so a value that cannot be encoded is an
// error. The caller holds mu.
func registerComponent[T any](m map[string]*OrRef[T], kind, name string, v *T) error {
	if v == nil {
		return fmt.Errorf("openapi: nil %s", kind)
	}
	if err := checkComponentName(kind, name); err != nil {
		return err
	}
	if _, ok := m[name]; ok {
		return fmt.Errorf("openapi: %s component %q is already registered", kind, name)
	}
	c, err := deepClone(v)
	if err != nil {
		return fmt.Errorf("openapi: %s %q cannot be copied: %w", kind, name, err)
	}
	m[name] = &OrRef[T]{Value: c}
	return nil
}

// SetExtension sets the root specification extension key to value,
// the last writer winning. The key follows the rules of
// OpenAPI.SetExtension (ErrInvalidExtensionKey, ErrReservedExtension),
// and the value must be encodable with the encoder of the document.
func (g *Generator) SetExtension(key string, value any) error {
	defer g.mutate()()
	var scratch map[string]any
	if err := setExtension(&scratch, key, nil, "x-tagGroups"); err != nil {
		return err
	}
	if err := jsonEncodable(value); err != nil {
		return fmt.Errorf("openapi: extension %q is not JSON-encodable: %w", key, err)
	}
	return g.api.SetExtension(key, copyAny(value))
}

// SetTag adds t to the tags of the document, or replaces the tag of the
// same name in place, whatever its description. A nil tag or one
// without a name is ignored. The tag is deep-copied; a tag that cannot
// be copied is dropped and an error recorded.
func (g *Generator) SetTag(t *Tag) {
	defer g.mutate()()
	if t == nil || t.Name == "" {
		return
	}
	c, err := deepClone(t)
	if err != nil {
		g.error(fmt.Errorf("tag %q: %w", t.Name, err))
		return
	}
	for i, tag := range g.api.Tags {
		if tag != nil && tag.Name == t.Name {
			g.api.Tags[i] = c
			return
		}
	}
	g.appendTag(c)
}

// SetTagGroups sets the x-tagGroups extension of the document. The
// groups are deep-copied.
func (g *Generator) SetTagGroups(groups []*XTagGroup) {
	defer g.mutate()()
	if groups == nil {
		g.api.XTagGroups = nil
		return
	}
	c, err := deepClone(&groups)
	if err != nil {
		g.error(fmt.Errorf("tag groups: %w", err))
		return
	}
	g.api.XTagGroups = *c
}

// SetSelf sets the $self URI of the document.
func (g *Generator) SetSelf(uri string) {
	defer g.mutate()()
	g.api.Self = uri
}

// SetJSONSchemaDialect sets the jsonSchemaDialect of the document.
func (g *Generator) SetJSONSchemaDialect(uri string) {
	defer g.mutate()()
	g.api.JSONSchemaDialect = uri
}

// SetExternalDocs sets the root externalDocs of the document. The
// value is deep-copied; nil removes it.
func (g *Generator) SetExternalDocs(d *ExternalDocs) {
	defer g.mutate()()
	if d == nil {
		g.api.ExternalDocs = nil
		return
	}
	c, err := deepClone(d)
	if err != nil {
		g.error(fmt.Errorf("external docs: %w", err))
		return
	}
	g.api.ExternalDocs = c
}
