package openapi

import "testing"

func TestOmitEmptyContainers(t *testing.T) {
	tests := []struct {
		name         string
		api          OpenAPI
		paths, comps bool // expected non-nil
	}{
		{"nothing", OpenAPI{Paths: Paths{}, Components: &Components{Schemas: map[string]*Schema{}}}, true, false},
		{"nil paths and components", OpenAPI{}, true, false},
		{"paths only", OpenAPI{Paths: Paths{"/x": {}}, Components: &Components{}}, true, false},
		{"components only", OpenAPI{Paths: Paths{}, Components: &Components{Schemas: map[string]*Schema{"S": {}}}}, false, true},
		{"webhooks only", OpenAPI{Paths: Paths{}, Webhooks: map[string]*PathItem{}, Components: &Components{}}, false, false},
		{"empty components with an extension", OpenAPI{Paths: Paths{}, Components: &Components{Extensions: map[string]any{"x-a": 1}}}, false, true},
		{"both", OpenAPI{Paths: Paths{"/x": {}}, Components: &Components{Schemas: map[string]*Schema{"S": {}}}}, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := tt.api
			omitEmptyContainers(&api)
			if got := api.Paths != nil; got != tt.paths {
				t.Errorf("paths non-nil = %v, want %v", got, tt.paths)
			}
			if got := api.Components != nil; got != tt.comps {
				t.Errorf("components non-nil = %v, want %v", got, tt.comps)
			}
		})
	}
}
