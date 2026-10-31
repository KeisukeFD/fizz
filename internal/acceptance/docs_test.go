package acceptance

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/wI2L/fizz/internal/acceptance/at"
)

// Batch I (docs): I10, I11, I12, I15. Everything here reads files, so it
// compiles against today's tree. Each test is red until its doc task lands.

const bIRoot = "../.."

func bIRead(tb testing.TB, rel string) string {
	tb.Helper()
	b, err := os.ReadFile(filepath.Join(bIRoot, filepath.FromSlash(rel)))
	if err != nil {
		tb.Fatalf("read %s: %v", rel, err)
	}
	return string(b)
}

func bIContains(tb testing.TB, file, doc string, words ...string) {
	tb.Helper()
	for _, w := range words {
		if !strings.Contains(doc, w) {
			tb.Errorf("%s does not contain %q", file, w)
		}
	}
}

func bIAbsent(tb testing.TB, file, doc string, words ...string) {
	tb.Helper()
	for _, w := range words {
		if strings.Contains(doc, w) {
			tb.Errorf("%s must not contain %q", file, w)
		}
	}
}

// bIDedent strips the common leading indentation and surrounding blank lines.
func bIDedent(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.Trim(s, "\n")
	indent := -1
	for l := range strings.SplitSeq(s, "\n") {
		if strings.TrimSpace(l) == "" {
			continue
		}
		n := len(l) - len(strings.TrimLeft(l, " \t"))
		if indent < 0 || n < indent {
			indent = n
		}
	}
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		if indent > 0 && len(l) >= indent {
			l = l[indent:]
		}
		lines[i] = strings.TrimRight(l, " \t")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// bIExampleBodies returns the dedented body of every Example function in
// the root package's test files, without the trailing "// Output:" block.
func bIExampleBodies(tb testing.TB) map[string]string {
	tb.Helper()
	files, _ := filepath.Glob(filepath.Join(bIRoot, "*_test.go"))
	out := map[string]string{}
	for _, name := range files {
		src, err := os.ReadFile(name)
		if err != nil {
			tb.Fatal(err)
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, src, parser.ParseComments)
		if err != nil {
			tb.Fatalf("parse %s: %v", name, err)
		}
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Body == nil || fd.Recv != nil || !strings.HasPrefix(fd.Name.Name, "Example") {
				continue
			}
			body := string(src[fset.Position(fd.Body.Lbrace).Offset+1 : fset.Position(fd.Body.Rbrace).Offset])
			for _, marker := range []string{"// Unordered output:", "// Output:"} {
				if i := strings.Index(body, marker); i >= 0 {
					body = body[:i]
				}
			}
			out[fd.Name.Name] = bIDedent(body)
		}
	}
	return out
}

var bIMarkerRe = regexp.MustCompile("(?s)<!-- example: (\\w+) -->\\s*\n```go\n(.*?)\n```")

// bICheckSnippets asserts that every marked go block equals its Example
// body and returns the number of marked blocks.
func bICheckSnippets(tb testing.TB, file, doc string) int {
	tb.Helper()
	bodies := bIExampleBodies(tb)
	blocks := bIMarkerRe.FindAllStringSubmatch(doc, -1)
	for _, m := range blocks {
		want, ok := bodies[m[1]]
		if !ok {
			tb.Errorf("%s: marker names %s, which is not an Example in the root package", file, m[1])
			continue
		}
		if got := bIDedent(m[2]); got != want {
			tb.Errorf("%s: snippet for %s differs from the Example body\n--- snippet\n%s\n--- example\n%s", file, m[1], got, want)
		}
	}
	return len(blocks)
}

// I10
func TestBIREADMESnippetsMatchExamples(t *testing.T) {
	at.Pending(t, "P9.2", func(tb testing.TB) {
		doc := bIRead(tb, "README.md")
		if n := bICheckSnippets(tb, "README.md", doc); n < 8 {
			tb.Errorf("README.md has %d example-marked go blocks, want >= 8", n)
		}
		bIAbsent(tb, "README.md", doc, "validator.v8", "versions/3.0.1.md")
		bIContains(tb, "README.md", doc, "3.2.1", "11065", "ScalarUI")

		// One tags-table row per TagNames field (15) and BinderTags entry (7).
		var rows []string
		var headings []string
		for l := range strings.SplitSeq(doc, "\n") {
			switch {
			case strings.HasPrefix(strings.TrimSpace(l), "|"):
				rows = append(rows, strings.ToLower(l))
			case strings.HasPrefix(l, "#"):
				headings = append(headings, strings.ToLower(l))
			}
		}
		for _, tag := range []string{
			"description", "example", "format", "deprecated", "title", "pattern",
			"readonly", "writeonly", "nullable", "multipleof", "additionalproperties",
			"inline", "style", "allowreserved", "cookie", // TagNames
			"path", "query", "header", "enum", "default", "validate", "explode", // BinderTags
		} {
			found := false
			for _, r := range rows {
				if strings.Contains(r, "`"+tag+"`") {
					found = true
					break
				}
			}
			if !found {
				tb.Errorf("README.md tags table has no row for `%s`", tag)
			}
		}
		for _, sec := range []string{"interfaces", "hooks", "naming", "deepobject", "cookies", "docs ui", "yaml"} {
			found := false
			for _, h := range headings {
				if strings.Contains(h, sec) {
					found = true
					break
				}
			}
			if !found {
				tb.Errorf("README.md has no section heading containing %q", sec)
			}
		}
	})
}

// I11
func TestBIMigrationCoversTopics(t *testing.T) {
	at.Pending(t, "P9.1", func(tb testing.TB) {
		doc := bIRead(tb, "MIGRATION.md")
		bIContains(tb, "MIGRATION.md", doc,
			"SchemaOrRef", ".Value", ".Ref", "TypeSet", "json.Number", "Explode",
			"MarshalYAML", "SpecGenConfig", "TagNames", "BinderTags", "Errors()", "Err()",
			"InstallBindHook", "WithoutBindHook", "operationId", "DELETE", "Duration",
			"url.URL", "nullable", "oapi-codegen", "kin-openapi", "ogen", "DocumentHook",
			"snapshot", "1.26")
		bICheckSnippets(tb, "MIGRATION.md", doc)
	})
}

var bIVersionHeadRe = regexp.MustCompile(`(?m)^#+ .*\d+\.\d+\.\d+`)

// I12
func TestBIReleaseNotes(t *testing.T) {
	at.Pending(t, "P9.5", func(tb testing.TB) {
		doc := bIRead(tb, "CHANGELOG.md")
		i := strings.Index(doc, "v0.30.0")
		if i < 0 {
			tb.Fatal("CHANGELOG.md has no v0.30.0 section")
		}
		section := doc[i:]
		// The section ends at the next version heading.
		if j := bIVersionHeadRe.FindAllStringIndex(section, 2); len(j) == 2 {
			section = section[:j[1][0]]
		}
		bIContains(tb, "CHANGELOG.md v0.30.0", section,
			"Go 1.27", "1.26", "3.2.1", "MIGRATION", "bind hook", "oapi-codegen")
	})
}

var bIPinRe = regexp.MustCompile("\\|\\s*`([^`]+)@([^`]+)`\\s*\\|\\s*`([^`]+)`\\s*\\|\\s*`(sha384-[^`]+)`")

// I15. The pin table of docs/compat.md must match the docsui constants.
// docsui does not exist yet, so its sources are scanned as text.
func TestBIDocsCompatTable(t *testing.T) {
	at.Pending(t, "P8.4", func(tb testing.TB) {
		doc := bIRead(tb, "docs/compat.md")
		pins := bIPinRe.FindAllStringSubmatch(doc, -1)
		if len(pins) < 3 {
			tb.Fatalf("docs/compat.md pin table has %d rows, want >= 3", len(pins))
		}
		var src bytes.Buffer
		files, _ := filepath.Glob(filepath.Join(bIRoot, "docsui", "*.go"))
		for _, f := range files {
			if strings.HasSuffix(f, "_test.go") {
				continue
			}
			b, _ := os.ReadFile(f)
			src.Write(b)
		}
		if src.Len() == 0 {
			tb.Fatal("no docsui sources to compare the pins with")
		}
		for _, p := range pins {
			pkg, version, sum := p[1], p[2], p[4]
			if !bytes.Contains(src.Bytes(), []byte(sum)) {
				tb.Errorf("docsui has no constant for %s@%s (%s)", pkg, version, sum)
			}
			if !bytes.Contains(src.Bytes(), []byte(pkg+"@"+version)) {
				tb.Errorf("docsui does not reference %s@%s", pkg, version)
			}
		}
		bIContains(tb, "docs/compat.md", doc, "AllOfWrap", "11065")
	})
}
