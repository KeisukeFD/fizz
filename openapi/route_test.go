package openapi

import (
	"strings"
	"testing"
)

func routeAdd(g *Generator, method, path, id string) (*Operation, error) {
	return g.AddOperation(path, method, "", nil, nil, &OperationInfo{ID: id, StatusCode: 200})
}

// The same path and method twice is an operation error that names both
// operations; the first operation stays.
func TestDuplicateRouteIsError(t *testing.T) {
	g := gen(t)
	first, err := routeAdd(g, "GET", "/x", "first")
	if err != nil {
		t.Fatal(err)
	}
	op, err := routeAdd(g, "get", "/x", "second")
	if err == nil || op != nil {
		t.Fatalf("op = %v, err = %v, want an error", op, err)
	}
	for _, s := range []string{"GET /x", `"first"`, "(second)"} {
		if !strings.Contains(err.Error(), s) {
			t.Errorf("error %q lacks %q", err, s)
		}
	}
	if g.api.Paths["/x"].GET != first {
		t.Error("the first operation was replaced")
	}
	if errs, _ := rboErrors(g); len(errs) != 1 {
		t.Errorf("errors = %v, want one", errs)
	}
	// Another method on the same path is fine.
	if _, err := routeAdd(g, "POST", "/x", "third"); err != nil {
		t.Errorf("another method: %v", err)
	}
	// A rejected operation does not reserve its ID.
	if g.OperationIDUsed("second") {
		t.Error("the ID of the rejected operation is reserved")
	}
}
