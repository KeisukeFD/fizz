package contract_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/KeisukeFD/fizz"
	"github.com/KeisukeFD/fizz/internal/conformance/contract"
	"github.com/KeisukeFD/fizz/internal/conformance/repro"
	"github.com/KeisukeFD/fizz/internal/conformance/repro/apiwithexamples"
	"github.com/KeisukeFD/fizz/internal/conformance/repro/baseuriretrieval"
	"github.com/KeisukeFD/fizz/internal/conformance/repro/petstore"
	"github.com/KeisukeFD/fizz/internal/conformance/repro/petstoreexpanded"
	"github.com/KeisukeFD/fizz/internal/conformance/repro/query32"
	"github.com/KeisukeFD/fizz/internal/conformance/repro/tags32"
	"github.com/KeisukeFD/fizz/internal/conformance/repro/uspto"
)

// TestContractFixture drives the fixture app from its own spec.
func TestContractFixture(t *testing.T) {
	f := contract.New()
	spec := repro.Serve(t, f)
	contract.Run(t, contract.Target{
		Handler: f,
		Spec:    spec,
		Echo:    true,
	})
}

// TestContractSliceDefault: a slice default is documented as the array
// tonic binds (review finding 4, P10.7c).
func TestContractSliceDefault(t *testing.T) {
	f := contract.NewSliceDefault()
	if err := f.Err(); err != nil {
		t.Fatalf("fizz errors: %v", err)
	}
	contract.Run(t, contract.Target{Handler: f, Spec: repro.Serve(t, f), Echo: true})
}

// TestContractBadParam: a map or struct parameter outside deepObject (P10.7b),
// and a slice or array header or path parameter (P10.7e, P10.7l), is a
// generation error, since tonic cannot bind it.
func TestContractBadParam(t *testing.T) {
	for _, variant := range contract.BadParams {
		t.Run(strings.ReplaceAll(variant, " ", "_"), func(t *testing.T) {
			f := contract.NewBadParam(variant)
			if f.Err() == nil {
				t.Errorf("%s parameter: f.Errors() = %v, want an error", variant, f.Errors())
			}
		})
	}
}

// TestContractByteHeader: a []byte header is a string, not a slice, so it
// is accepted (the P10.7e rule uses the same definition as isBindable).
func TestContractByteHeader(t *testing.T) {
	f := contract.NewByteHeader()
	if err := f.Err(); err != nil {
		t.Fatalf("fizz errors: %v", err)
	}
}

// TestContractNoDefaultHeaders: an optional non-string header without a
// default cannot be bound when absent, so the generator records a warning
// for each of them, and for no other (P10.7i).
func TestContractNoDefaultHeaders(t *testing.T) {
	f := contract.NewNoDefaultHeaders()
	if err := f.Err(); err != nil {
		t.Fatalf("fizz errors: %v", err)
	}
	var got []string
	for _, e := range f.Errors() {
		w, ok := e.(interface{ IsWarning() bool })
		if !ok || !w.IsWarning() {
			t.Errorf("unexpected error: %v", e)
			continue
		}
		got = append(got, e.Error())
	}
	var want []string
	for _, h := range contract.NoDefaultHeaders {
		want = append(want, fmt.Sprintf("absent optional header %q is bound as \"\" by tonic and returns 400; add a default or make it a string", h))
	}
	for _, w := range want {
		if !slices.ContainsFunc(got, func(g string) bool { return strings.Contains(g, w) }) {
			t.Errorf("missing warning %q in %q", w, got)
		}
	}
	if len(got) != len(want) {
		t.Errorf("got %d warnings, want %d: %q", len(got), len(want), got)
	}
}

// TestContractRepro drives the reproduction apps that have parameters.
// Their handlers do not echo: the contract is that spec-conformant
// requests are not rejected.
func TestContractRepro(t *testing.T) {
	apps := map[string]*fizz.Fizz{
		"petstore":         petstore.New(),
		"petstoreexpanded": petstoreexpanded.New(),
		"apiwithexamples":  apiwithexamples.New(),
		"uspto":            uspto.New(),
		"query32":          query32.New(),
		"tags32":           tags32.New(),
		"baseuriretrieval": baseuriretrieval.New(),
	}
	for name, f := range apps {
		t.Run(name, func(t *testing.T) {
			spec := repro.Serve(t, f)
			contract.Run(t, contract.Target{Handler: f, Spec: spec})
		})
	}
}
