package openapi

// SchemaOrRef represents a Schema that can be inlined
// or referenced in the API description.
type SchemaOrRef struct {
	*Schema
	*Reference
}

// MarshalYAML implements yaml.Marshaler for SchemaOrRef.
func (sor *SchemaOrRef) MarshalYAML() (any, error) {
	if sor.Schema != nil {
		return sor.Schema, nil
	}
	return sor.Reference, nil
}

// Schema represents the definition of input and output data
// types of the API.
type Schema struct {
	// The following properties are taken from the JSON Schema
	// definition but their definitions were adjusted to the
	// OpenAPI Specification.
	Type                 string                  `json:"type,omitempty" yaml:"type,omitempty"`
	AllOf                *SchemaOrRef            `json:"allOf,omitempty" yaml:"allOf,omitempty"`
	OneOf                *SchemaOrRef            `json:"oneOf,omitempty" yaml:"oneOf,omitempty"`
	AnyOf                *SchemaOrRef            `json:"anyOf,omitempty" yaml:"anyOf,omitempty"`
	Items                *SchemaOrRef            `json:"items,omitempty" yaml:"items,omitempty"`
	Properties           map[string]*SchemaOrRef `json:"properties,omitempty" yaml:"properties,omitempty"`
	AdditionalProperties *SchemaOrRef            `json:"additionalProperties,omitempty" yaml:"additionalProperties,omitempty"`
	Description          string                  `json:"description,omitempty" yaml:"description,omitempty"`
	Format               string                  `json:"format,omitempty" yaml:"format,omitempty"`
	Default              any                     `json:"default,omitempty" yaml:"default,omitempty"`
	Example              any                     `json:"example,omitempty" yaml:"example,omitempty"`

	// The following properties are taken directly from the
	// JSON Schema definition and follow the same specifications
	Title            string   `json:"title,omitempty" yaml:"title,omitempty"`
	MultipleOf       int      `json:"multipleOf,omitempty" yaml:"multipleOf,omitempty"`
	Maximum          int      `json:"maximum,omitempty" yaml:"maximum,omitempty"`
	ExclusiveMaximum bool     `json:"exclusiveMaximum,omitempty" yaml:"exclusiveMaximum,omitempty"`
	Minimum          int      `json:"minimum,omitempty" yaml:"minimum,omitempty"`
	ExclusiveMinimum bool     `json:"exclusiveMinimum,omitempty" yaml:"exclusiveMinimum,omitempty"`
	MaxLength        int      `json:"maxLength,omitempty" yaml:"maxLength,omitempty"`
	MinLength        int      `json:"minLength,omitempty" yaml:"minLength,omitempty"`
	Pattern          string   `json:"pattern,omitempty" yaml:"pattern,omitempty"`
	MaxItems         int      `json:"maxItems,omitempty" yaml:"maxItems,omitempty"`
	MinItems         int      `json:"minItems,omitempty" yaml:"minItems,omitempty"`
	UniqueItems      bool     `json:"uniqueItems,omitempty" yaml:"uniqueItems,omitempty"`
	MaxProperties    int      `json:"maxProperties,omitempty" yaml:"maxProperties,omitempty"`
	MinProperties    int      `json:"minProperties,omitempty" yaml:"minProperties,omitempty"`
	Required         []string `json:"required,omitempty" yaml:"required,omitempty"`
	Enum             []any    `json:"enum,omitempty" yaml:"enum,omitempty"`
	Nullable         bool     `json:"nullable,omitempty" yaml:"nullable,omitempty"`
	Deprecated       bool     `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
}
