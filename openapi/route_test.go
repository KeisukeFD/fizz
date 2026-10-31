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

// Paths that differ only in their variable names are two keys that OAS
// forbids: the second route is an operation error naming both, and no
// second key is added.
func TestEquivalentPathTemplatesAreError(t *testing.T) {
	g := gen(t)
	if _, err := routeAdd(g, "GET", "/x/{a}", "get"); err != nil {
		t.Fatal(err)
	}
	op, err := routeAdd(g, "POST", "/x/{c}", "post")
	if err == nil || op != nil {
		t.Fatalf("op = %v, err = %v, want an error", op, err)
	}
	for _, s := range []string{"POST /x/{c}", "GET /x/{a}"} {
		if !strings.Contains(err.Error(), s) {
			t.Errorf("error %q lacks %q", err, s)
		}
	}
	if len(g.api.Paths) != 1 || g.api.Paths["/x/{c}"] != nil {
		t.Errorf("paths = %v, want only /x/{a}", g.api.Paths)
	}
	// Same name: the same key, another method.
	if _, err := routeAdd(g, "PUT", "/x/{a}", "put"); err != nil {
		t.Errorf("same template: %v", err)
	}
	// Different shape: not equivalent.
	if _, err := routeAdd(g, "GET", "/x/{a}/y/{b}", "deep"); err != nil {
		t.Errorf("different template: %v", err)
	}
	if _, err := routeAdd(g, "GET", "/x/lit", "lit"); err != nil {
		t.Errorf("literal segment: %v", err)
	}
}
