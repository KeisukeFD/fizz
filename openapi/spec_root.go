package openapi

// OpenAPI represents the root document object of
// an OpenAPI document.
type OpenAPI struct {
	OpenAPI           string                 `json:"openapi" yaml:"openapi"`
	Self              string                 `json:"$self,omitzero" yaml:"$self,omitempty"`
	Info              *Info                  `json:"info" yaml:"info"`
	JSONSchemaDialect string                 `json:"jsonSchemaDialect,omitzero" yaml:"jsonSchemaDialect,omitempty"`
	Servers           []*Server              `json:"servers,omitempty" yaml:"servers,omitempty"`
	Paths             Paths                  `json:"paths,omitzero" yaml:"paths,omitempty"`
	Webhooks          map[string]*PathItem   `json:"webhooks,omitempty" yaml:"webhooks,omitempty"`
	Components        *Components            `json:"components,omitempty" yaml:"components,omitempty"`
	Tags              []*Tag                 `json:"tags,omitempty" yaml:"tags,omitempty"`
	Security          []*SecurityRequirement `json:"security,omitempty" yaml:"security,omitempty"`
	XTagGroups        []*XTagGroup           `json:"x-tagGroups,omitempty" yaml:"x-tagGroups,omitempty"`

	// Extensions holds the "x-" specification extensions. Keys are
	// emitted sorted after the typed members. Use SetExtension to
	// validate keys.
	Extensions map[string]any `json:",embed" yaml:",inline"`
}

// Components holds a set of reusable objects for different
// aspects of the specification.
type Components struct {
	Schemas         map[string]*Schema              `json:"schemas,omitempty" yaml:"schemas,omitempty"`
	Responses       map[string]*ResponseOrRef       `json:"responses,omitempty" yaml:"responses,omitempty"`
	Parameters      map[string]*ParameterOrRef      `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	Examples        map[string]*ExampleOrRef        `json:"examples,omitempty" yaml:"examples,omitempty"`
	RequestBodies   map[string]*RequestBodyOrRef    `json:"requestBodies,omitempty" yaml:"requestBodies,omitempty"`
	Headers         map[string]*HeaderOrRef         `json:"headers,omitempty" yaml:"headers,omitempty"`
	SecuritySchemes map[string]*SecuritySchemeOrRef `json:"securitySchemes,omitempty" yaml:"securitySchemes,omitempty"`
	Links           map[string]*LinkOrRef           `json:"links,omitempty" yaml:"links,omitempty"`
	Callbacks       map[string]*CallbackOrRef       `json:"callbacks,omitempty" yaml:"callbacks,omitempty"`
	PathItems       map[string]*PathItem            `json:"pathItems,omitempty" yaml:"pathItems,omitempty"`
	MediaTypes      map[string]*MediaTypeOrRef      `json:"mediaTypes,omitempty" yaml:"mediaTypes,omitempty"`

	// Extensions holds the "x-" specification extensions. Keys are
	// emitted sorted after the typed members. Use SetExtension to
	// validate keys.
	Extensions map[string]any `json:",embed" yaml:",inline"`
}

// Info represents the metadata of an API.
type Info struct {
	Title          string   `json:"title" yaml:"title"`
	Summary        string   `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description    string   `json:"description,omitempty" yaml:"description,omitempty"`
	TermsOfService string   `json:"termsOfService,omitempty" yaml:"termsOfService,omitempty"`
	Contact        *Contact `json:"contact,omitempty" yaml:"contact,omitempty"`
	License        *License `json:"license,omitempty" yaml:"license,omitempty"`
	Version        string   `json:"version" yaml:"version"`
	XLogo          *XLogo   `json:"x-logo,omitempty" yaml:"x-logo,omitempty"`

	// Extensions holds the "x-" specification extensions. Keys are
	// emitted sorted after the typed members. Use SetExtension to
	// validate keys.
	Extensions map[string]any `json:",embed" yaml:",inline"`
}

// Contact represents the the contact informations
// exposed for an API.
type Contact struct {
	Name  string `json:"name,omitempty" yaml:"name,omitempty"`
	URL   string `json:"url,omitempty" yaml:"url,omitempty"`
	Email string `json:"email,omitempty" yaml:"email,omitempty"`

	// Extensions holds the "x-" specification extensions. Keys are
	// emitted sorted after the typed members. Use SetExtension to
	// validate keys.
	Extensions map[string]any `json:",embed" yaml:",inline"`
}

// License represents the license informations
// exposed for an API.
type License struct {
	Name       string `json:"name" yaml:"name"`
	Identifier string `json:"identifier,omitempty" yaml:"identifier,omitempty"`
	URL        string `json:"url,omitempty" yaml:"url,omitempty"`

	// Extensions holds the "x-" specification extensions. Keys are
	// emitted sorted after the typed members. Use SetExtension to
	// validate keys.
	Extensions map[string]any `json:",embed" yaml:",inline"`
}

// Server represents a server.
type Server struct {
	URL         string                     `json:"url" yaml:"url"`
	Name        string                     `json:"name,omitempty" yaml:"name,omitempty"`
	Description string                     `json:"description,omitempty" yaml:"description,omitempty"`
	Variables   map[string]*ServerVariable `json:"variables,omitempty" yaml:"variables,omitempty"`

	// Extensions holds the "x-" specification extensions. Keys are
	// emitted sorted after the typed members. Use SetExtension to
	// validate keys.
	Extensions map[string]any `json:",embed" yaml:",inline"`
}

// ServerVariable represents a server variable for server
// URL template substitution.
type ServerVariable struct {
	Enum        []string `json:"enum,omitempty" yaml:"enum,omitempty"`
	Default     string   `json:"default" yaml:"default"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`

	// Extensions holds the "x-" specification extensions. Keys are
	// emitted sorted after the typed members. Use SetExtension to
	// validate keys.
	Extensions map[string]any `json:",embed" yaml:",inline"`
}

// Paths represents the relative paths to the individual
// endpoints and their operations.
type Paths map[string]*PathItem

// PathItem describes the operations available on a single
// API path.
type PathItem struct {
	Ref         string            `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Summary     string            `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	GET         *Operation        `json:"get,omitempty" yaml:"get,omitempty"`
	PUT         *Operation        `json:"put,omitempty" yaml:"put,omitempty"`
	POST        *Operation        `json:"post,omitempty" yaml:"post,omitempty"`
	DELETE      *Operation        `json:"delete,omitempty" yaml:"delete,omitempty"`
	OPTIONS     *Operation        `json:"options,omitempty" yaml:"options,omitempty"`
	HEAD        *Operation        `json:"head,omitempty" yaml:"head,omitempty"`
	PATCH       *Operation        `json:"patch,omitempty" yaml:"patch,omitempty"`
	TRACE       *Operation        `json:"trace,omitempty" yaml:"trace,omitempty"`
	Servers     []*Server         `json:"servers,omitempty" yaml:"servers,omitempty"`
	Parameters  []*ParameterOrRef `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	Query       *Operation        `json:"query,omitempty" yaml:"query,omitempty"`

	// AdditionalOperations holds operations for HTTP methods
	// that have no dedicated field, keyed by method name.
	AdditionalOperations map[string]*Operation `json:"additionalOperations,omitempty" yaml:"additionalOperations,omitempty"`

	// Extensions holds the "x-" specification extensions. Keys are
	// emitted sorted after the typed members. Use SetExtension to
	// validate keys.
	Extensions map[string]any `json:",embed" yaml:",inline"`
}

// Tag represents the metadata of a single tag.
type Tag struct {
	Name         string        `json:"name" yaml:"name"`
	Summary      string        `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description  string        `json:"description,omitempty" yaml:"description,omitempty"`
	ExternalDocs *ExternalDocs `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`
	Parent       string        `json:"parent,omitempty" yaml:"parent,omitempty"`
	Kind         string        `json:"kind,omitempty" yaml:"kind,omitempty"`

	// Extensions holds the "x-" specification extensions. Keys are
	// emitted sorted after the typed members. Use SetExtension to
	// validate keys.
	Extensions map[string]any `json:",embed" yaml:",inline"`
}
