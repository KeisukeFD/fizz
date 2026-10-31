package fizz

import "testing"

func TestHandlerBaseName(t *testing.T) {
	tests := []struct{ in, pkg, name string }{
		{"github.com/x/pkg.Handler", "pkg", "Handler"},
		{"github.com/x/pkg.(*T).Method-fm", "pkg", "Method"},
		{"github.com/x/pkg.T.Method-fm", "pkg", "Method"},
		{"github.com/x/pkg.Handler[...]", "pkg", "Handler"},
		{"github.com/x/pkg.(*T[...]).Method-fm", "pkg", "Method"},
		{"github.com/x/pkg.outer.func1", "pkg", "outer"},
		{"github.com/x/pkg.outer.func1.2", "pkg", "outer"},
		{"github.com/x/pkg.(*T).M.func1", "pkg", "M"},
		{"main.main", "main", "main"},
		{"main.run.func3", "main", "run"},
		{"gopkg.in/yaml%2ev3.Handler", "yaml%2ev3", "Handler"},
		{"bare", "", "bare"},
	}
	for _, tt := range tests {
		pkg, name := handlerBaseName(tt.in)
		if pkg != tt.pkg || name != tt.name {
			t.Errorf("handlerBaseName(%q) = %q, %q; want %q, %q", tt.in, pkg, name, tt.pkg, tt.name)
		}
	}
}

func TestMethodPathID(t *testing.T) {
	tests := []struct{ method, path, want string }{
		{"GET", "/users/{id}", "get_users_id"},
		{"POST", "/", "post"},
		{"DELETE", "/a/:b/*c", "delete_a_b_c"},
	}
	for _, tt := range tests {
		if got := methodPathID(tt.method, tt.path); got != tt.want {
			t.Errorf("methodPathID(%q, %q) = %q; want %q", tt.method, tt.path, got, tt.want)
		}
	}
}

func TestDefaultOperationIDFallback(t *testing.T) {
	taken := map[string]bool{}
	has := func(id string) bool { return taken[id] }
	h := "github.com/x/models.Get-fm"
	for i, want := range []string{"Get", "models.Get", "get_a_id"} {
		got := defaultOperationID("GET", "/a/{id}", h, nil, has)
		if got != want {
			t.Fatalf("#%d: got %q, want %q", i, got, want)
		}
		taken[got] = true
	}
	namer := func(m, p, h string) string { return "Get" }
	if got := defaultOperationID("GET", "/b", h, namer, has); got != "get_b" {
		t.Errorf("namer collision: got %q", got)
	}
}
