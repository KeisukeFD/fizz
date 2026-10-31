package openapi

import "encoding/json"

// SecuritySchemeOrRef represents a SecurityScheme that can be inlined
// or referenced in the API description.
type SecuritySchemeOrRef struct {
	*SecurityScheme
	*Reference
}

// MarshalYAML implements yaml.Marshaler for SecuritySchemeOrRef.
func (sor *SecuritySchemeOrRef) MarshalYAML() (any, error) {
	if sor.SecurityScheme != nil {
		return sor.SecurityScheme, nil
	}
	return sor.Reference, nil
}

// SecurityScheme represents a security scheme that can be used by an operation.
type SecurityScheme struct {
	Type             string      `json:"type,omitempty" yaml:"type,omitempty"`
	Scheme           string      `json:"scheme,omitempty" yaml:"scheme,omitempty"`
	BearerFormat     string      `json:"bearerFormat,omitempty" yaml:"bearerFormat,omitempty"`
	Description      string      `json:"description,omitempty" yaml:"description,omitempty"`
	In               string      `json:"in,omitempty" yaml:"in,omitempty"`
	Name             string      `json:"name,omitempty" yaml:"name,omitempty"`
	OpenIDConnectURL string      `json:"openIdConnectUrl,omitempty" yaml:"openIdConnectUrl,omitempty"`
	Flows            *OAuthFlows `json:"flows,omitempty" yaml:"flows,omitempty"`
}

// OAuthFlows represents all the supported OAuth flows.
type OAuthFlows struct {
	Implicit          *OAuthFlow `json:"implicit,omitempty" yaml:"implicit,omitempty"`
	Password          *OAuthFlow `json:"password,omitempty" yaml:"password,omitempty"`
	ClientCredentials *OAuthFlow `json:"clientCredentials,omitempty" yaml:"clientCredentials,omitempty"`
	AuthorizationCode *OAuthFlow `json:"authorizationCode,omitempty" yaml:"authorizationCode,omitempty"`
}

// OAuthFlow represents an OAuth security scheme.
type OAuthFlow struct {
	AuthorizationURL string            `json:"authorizationUrl,omitempty" yaml:"authorizationUrl,omitempty"`
	TokenURL         string            `json:"tokenUrl,omitempty" yaml:"tokenUrl,omitempty"`
	RefreshURL       string            `json:"refreshUrl,omitempty" yaml:"refreshUrl,omitempty"`
	Scopes           map[string]string `json:"scopes,omitempty" yaml:"scopes,omitempty"`
}

// MarshalYAML implements yaml.Marshaler for OAuthFlow.
func (f OAuthFlow) MarshalYAML() ([]byte, error) {
	type flow OAuthFlow
	if f.Scopes == nil {
		// The field is REQUIRED and MAY be empty according to the spec.
		f.Scopes = map[string]string{}
	}
	return json.Marshal(flow(f))
}

// SecurityRequirement represents the security object in the API specification.
type SecurityRequirement map[string][]string
