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
	"github.com/KeisukeFD/fizz/internal/plan"
)

// TestContractFixture drives the fixture app from its own spec.
func TestContractFixture(t *testing.T) {
	f := contract.New()
	spec := repro.Serve(t, f)
	contract.Run(t, contract.Target{
		Handler: f,
		Spec:    spec,
		Echo:    true,
		Pending: fixturePending(),
	})
}

// TestContractSliceDefault: a slice default is documented as the array
// tonic binds (review finding 4, P10.7c).
func TestContractSliceDefault(t *testing.T) {
	f := contract.NewSliceDefault()
	if err := f.Err(); err != nil {
		plan.Pending(t, "P10.7c", func(tb testing.TB) {
			tb.Fatalf("fizz errors: %v", err)
		})
		return
	}
	contract.Run(t, contract.Target{Handler: f, Spec: repro.Serve(t, f), Echo: true})
}

// TestContractBadParam: a map or struct parameter outside deepObject (P10.7b),
// and a slice or array header or path parameter (P10.7e, P10.7l), is a
// generation error, since tonic cannot bind it.
func TestContractBadParam(t *testing.T) {
	for _, variant := range contract.BadParams {
		t.Run(strings.ReplaceAll(variant, " ", "_"), func(t *testing.T) {
			plan.Pending(t, contract.BadParamTasks[variant], func(tb testing.TB) {
				f := contract.NewBadParam(variant)
				if f.Err() == nil {
					tb.Errorf("%s parameter: f.Errors() = %v, want an error", variant, f.Errors())
				}
			})
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

// fixturePending lists the drifts that the fixture reproduces, each with
// the plan task that fixes it. Keys are "<operationId> <in>:<name>" (see
// contract.Target.Pending).
func fixturePending() map[string]string {
	p := map[string]string{
		// P10.7j: spaceDelimited and pipeDelimited are documented, and
		// tonic splits on commas only.
		"arrays query:space": "P10.7j",
		"arrays query:pipe":  "P10.7j",

		// P10.7k: label and matrix path parameters are bound with their
		// leading delimiter.
		"label path:pl":  "P10.7k",
		"matrix path:pm": "P10.7k",

		// P10.7m: a form cookie is documented as percent-encoded, and the
		// bind hook does not decode it.
		"special cookie:csp": "P10.7m",
	}
	return p
}
