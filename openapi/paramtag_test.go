package openapi

import "testing"

func TestParseParamTag(t *testing.T) {
	tests := []struct {
		tag      string
		name     string
		required bool
		def      string
		err      string
	}{
		{tag: "q", name: "q"},
		{tag: "q,required", name: "q", required: true},
		{tag: "q,default=5", name: "q", def: "5"},
		{tag: "q, required ,default=x", name: "q", required: true, def: "x"},
		{tag: "q,default=", name: "q"},
		{tag: "", name: ""},
		{tag: "q,omitempty", err: "malformed tag for param 'q': unknown option 'omitempty'"},
	}
	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			name, req, def, err := ParseParamTag(tt.tag)
			if tt.err != "" {
				if err == nil || err.Error() != tt.err {
					t.Fatalf("err = %v, want %q", err, tt.err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if name != tt.name || req != tt.required || def != tt.def {
				t.Errorf("got (%q, %v, %q), want (%q, %v, %q)", name, req, def, tt.name, tt.required, tt.def)
			}
		})
	}
}
