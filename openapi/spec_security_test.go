package openapi

import (
	"encoding/json"
	"testing"
)

func TestSecuritySchemeMarshal(t *testing.T) {
	tests := []struct {
		name string
		in   *SecurityScheme
		want string
	}{
		{"zero", &SecurityScheme{Type: "http"}, `{"type":"http"}`},
		{"deprecated", &SecurityScheme{Type: "http", Deprecated: true}, `{"type":"http","deprecated":true}`},
		{"metadata url", &SecurityScheme{Type: "oauth2", OAuth2MetadataURL: "https://a/.well-known"}, `{"type":"oauth2","oauth2MetadataUrl":"https://a/.well-known"}`},
		{
			"device flow",
			&SecurityScheme{Type: "oauth2", Flows: &OAuthFlows{DeviceAuthorization: &OAuthFlow{
				DeviceAuthorizationURL: "https://a/device", TokenURL: "https://a/token", Scopes: map[string]string{"r": "read"},
			}}},
			`{"type":"oauth2","flows":{"deviceAuthorization":{"deviceAuthorizationUrl":"https://a/device","tokenUrl":"https://a/token","scopes":{"r":"read"}}}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := json.Marshal(tt.in)
			if err != nil {
				t.Fatal(err)
			}
			if string(b) != tt.want {
				t.Errorf("got  %s\nwant %s", b, tt.want)
			}
		})
	}
}
