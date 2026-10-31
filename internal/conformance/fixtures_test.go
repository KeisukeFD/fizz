package conformance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wI2L/fizz/internal/specvalidate"
)

const fixtureDir = "oai/upgraded/oas/tests/schema/"

func fixtures(t *testing.T, kind string) []string {
	t.Helper()
	files, err := filepath.Glob(fixtureDir + kind + "/*.json")
	if err != nil || len(files) == 0 {
		t.Fatalf("no %s fixtures (err=%v)", kind, err)
	}
	return files
}

// TestOfficialPassFixtures requires every upgraded upstream pass fixture to
// be free of Validate problems.
func TestOfficialPassFixtures(t *testing.T) {
	for _, f := range fixtures(t, "pass") {
		t.Run(strings.TrimSuffix(filepath.Base(f), ".json"), func(t *testing.T) {
			b, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			if err := specvalidate.Validate(b); err != nil {
				t.Errorf("pass fixture rejected: %v", err)
			}
		})
	}
}

// failReason is one expected rejection: the instance pointer and a keyword
// that the failure message at that pointer must contain.
type failReason struct{ pointer, keyword string }

// failFixtures maps every upstream fail fixture to the reasons for which it
// must be rejected. Every reason is derived from the fixture's comment in
// oai/orig and its name; a rejection for any other reason (for example only
// the version) leaves a reason unmatched and fails the test.
var failFixtures = map[string][]failReason{
	// Schema values must be booleans or objects, not null, a number or an array.
	"invalid_schema_types": {
		{"/components/schemas/invalid_array", "want boolean or object"},
		{"/components/schemas/invalid_null", "want boolean or object"},
		{"/components/schemas/invalid_number", "want boolean or object"},
	},
	// At least one of paths, components and webhooks is required.
	"no_containers": {
		{"", "missing property 'paths'"},
		{"", "missing property 'components'"},
		{"", "missing property 'webhooks'"},
	},
	// A server variable enum must not be empty.
	"server_enum_empty": {
		{"/servers/0/variables/var/enum", "minItems"},
	},
	// servers must be an array, not an object.
	"servers": {
		{"/servers", "want array"},
	},
	// overlays is not a top-level field of an OpenAPI Object.
	"unknown_container": {
		{"/overlays", "false schema"},
	},
}

// validUnder32 lists fail fixtures that are valid under 3.2 because 3.2
// relaxed the rule, with the reason and the 3.2 section. A fixture listed
// here must validate; an entry that is still rejected fails the test.
// Empty: every upstream fail fixture is still invalid under 3.2.1.
var validUnder32 = map[string]struct{ reason, section string }{}

// failLines parses Validate's "<pointer>: <message>" lines.
func failLines(err error) map[string][]string {
	out := map[string][]string{}
	_, body, _ := strings.Cut(err.Error(), ":\n")
	for line := range strings.SplitSeq(body, "\n") {
		line = strings.TrimSpace(line)
		ptr, msg, ok := strings.Cut(line, ": ")
		if !ok {
			continue
		}
		out[ptr] = append(out[ptr], msg)
	}
	return out
}

// TestOfficialFailFixtures requires every upstream fail fixture to be
// rejected for its intended reason.
func TestOfficialFailFixtures(t *testing.T) {
	seen := map[string]bool{}
	for _, f := range fixtures(t, "fail") {
		name := strings.TrimSuffix(filepath.Base(f), ".json")
		seen[name] = true
		t.Run(name, func(t *testing.T) {
			b, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			err = specvalidate.Validate(b)
			if rel, ok := validUnder32[name]; ok {
				if err != nil {
					t.Errorf("listed in validUnder32 (%s, %s) but rejected: %v", rel.reason, rel.section, err)
				}
				return
			}
			want, ok := failFixtures[name]
			if !ok {
				t.Fatalf("fail fixture %s has no entry in failFixtures or validUnder32", name)
			}
			if err == nil {
				t.Fatal("fail fixture accepted")
			}
			lines := failLines(err)
			for _, w := range want {
				found := false
				for _, msg := range lines[w.pointer] {
					if strings.Contains(msg, w.keyword) {
						found = true
					}
				}
				if !found {
					t.Errorf("no rejection at %q containing %q; got: %v", w.pointer, w.keyword, err)
				}
			}
		})
	}
	for name := range failFixtures {
		if !seen[name] {
			t.Errorf("failFixtures entry %s matches no fixture", name)
		}
	}
	for name := range validUnder32 {
		if !seen[name] {
			t.Errorf("validUnder32 entry %s matches no fixture", name)
		}
	}
}

// semanticExceptions lists corpus documents that trip a semantic lint
// because of an upstream bug, with the reason. The value is the exact set of
// problems expected, so a change in either direction fails the test.
var semanticExceptions = map[string]struct {
	problems []string
	reason   string
}{
	// The spec shows a Path Item / Operation fragment without its path. The
	// extractor (upgrade.Wrap) wraps it under the placeholder path /example,
	// which has no {id} / {petId} template expression, so the path parameter
	// "does not appear in the path template". Not a spec or lint bug: an
	// artifact of wrapping the fragment.
	"upgraded/spectext/path-item-object-example-1.json": {
		[]string{`/paths/~1example/parameters/0: path parameter "id" does not appear in the path template "/example"`},
		"fragment wrapped under placeholder path /example",
	},
	"upgraded/spectext/operation-object-example-1.json": {
		[]string{`/paths/~1example/get/parameters/0: path parameter "petId" does not appear in the path template "/example"`},
		"fragment wrapped under placeholder path /example",
	},
}

// TestLintSemanticsOnCorpus requires the learn examples, the pass fixtures
// and the spec-text documents to produce no semantic problems.
func TestLintSemanticsOnCorpus(t *testing.T) {
	c := buildCorpus(t)
	n := 0
	for name, doc := range c.Files {
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		if !strings.HasPrefix(name, "upgraded/learn/") &&
			!strings.HasPrefix(name, "upgraded/oas/tests/schema/pass/") &&
			!strings.HasPrefix(name, "upgraded/spectext/") {
			continue
		}
		n++
		t.Run(strings.TrimSuffix(strings.TrimPrefix(name, "upgraded/"), ".json"), func(t *testing.T) {
			var got []string
			for _, p := range specvalidate.LintSemantics(doc) {
				got = append(got, p.String())
			}
			if exc, ok := semanticExceptions[name]; ok {
				if strings.Join(got, "\n") != strings.Join(exc.problems, "\n") {
					t.Errorf("exception (%s) no longer matches:\n got: %q\nwant: %q", exc.reason, got, exc.problems)
				}
				return
			}
			if len(got) > 0 {
				t.Errorf("semantic problems:\n%s", strings.Join(got, "\n"))
			}
		})
	}
	if n == 0 {
		t.Fatal("no corpus documents")
	}
	for name := range semanticExceptions {
		if _, ok := c.Files[name]; !ok {
			t.Errorf("semanticExceptions entry %s matches no corpus document", name)
		}
	}
}
