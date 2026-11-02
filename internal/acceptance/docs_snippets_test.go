package acceptance

import (
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The snippet checker of the documentation (I10, I11). A go block of a
// markdown file that is preceded by the marker
//
//	<!-- example: ExampleX -->
//
// must equal the body of the Example function ExampleX, as gofmt prints
// it, without the "// Output:" comment and dedented by one tab. The
// Example functions are looked up in every *_test.go file of the repo, so
// that a snippet is always code that compiles and runs.

var (
	snippetMarkerRe = regexp.MustCompile(`<!-- example: (\w+) -->`)
	snippetBlockRe  = regexp.MustCompile("(?s)<!-- example: (\\w+) -->[ \t]*\n```go\n(.*?)\n```")
	outputCommentRe = regexp.MustCompile(`(?i)^//\s*(unordered )?output:`)
)

// exampleBodies returns the formatted bodies of the Example functions of
// every *_test.go file under the repo root, by function name. A name
// that is declared in several packages has one entry per distinct body.
func exampleBodies(tb testing.TB) map[string][]string {
	tb.Helper()
	out := map[string][]string{}
	err := filepath.WalkDir(bIRoot, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch n := d.Name(); {
			case p == bIRoot:
			case n == "testdata", n == "node_modules", strings.HasPrefix(n, "."):
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, "_test.go") {
			return nil
		}
		src, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, p, src, parser.ParseComments)
		if err != nil {
			tb.Errorf("parse %s: %v", p, err)
			return nil
		}
		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Recv != nil || fd.Body == nil || !strings.HasPrefix(fd.Name.Name, "Example") {
				continue
			}
			body, err := exampleBody(fset, src, f, fd)
			if err != nil {
				tb.Errorf("%s: %s: %v", p, fd.Name.Name, err)
				continue
			}
			if !contains(out[fd.Name.Name], body) {
				out[fd.Name.Name] = append(out[fd.Name.Name], body)
			}
		}
		return nil
	})
	if err != nil {
		tb.Fatalf("walk %s: %v", bIRoot, err)
	}
	return out
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// exampleBody returns the statements of fd as gofmt prints them, one tab
// removed, without the trailing Output comment.
func exampleBody(fset *token.FileSet, src []byte, f *ast.File, fd *ast.FuncDecl) (string, error) {
	start := fset.Position(fd.Body.Lbrace).Offset + 1
	end := fset.Position(fd.Body.Rbrace).Offset
	for _, cg := range f.Comments {
		if cg.Pos() < fd.Body.Lbrace || cg.Pos() > fd.Body.Rbrace {
			continue
		}
		if outputCommentRe.MatchString(cg.List[0].Text) {
			end = fset.Position(cg.Pos()).Offset
			break
		}
	}
	wrapped := "package p\n\nfunc _() {" + string(src[start:end]) + "\n}\n"
	formatted, err := format.Source([]byte(wrapped))
	if err != nil {
		return "", err
	}
	lines := strings.Split(strings.TrimRight(string(formatted), "\n"), "\n")
	// Drop "package p", the blank line, "func _() {" and the closing "}".
	if len(lines) < 4 {
		return "", nil
	}
	lines = lines[3 : len(lines)-1]
	for i, l := range lines {
		lines[i] = strings.TrimPrefix(l, "\t")
	}
	return strings.TrimSpace(strings.Join(lines, "\n")), nil
}

// checkSnippets asserts that every marked go block of doc equals the body
// of its Example, and that every marker has a block. It returns the number
// of marked blocks.
func checkSnippets(tb testing.TB, file, doc string) int {
	tb.Helper()
	bodies := exampleBodies(tb)
	blocks := snippetBlockRe.FindAllStringSubmatch(doc, -1)
	if n := len(snippetMarkerRe.FindAllString(doc, -1)); n != len(blocks) {
		tb.Errorf("%s: %d example markers but %d go blocks right after one", file, n, len(blocks))
	}
	for _, m := range blocks {
		name, got := m[1], strings.TrimSpace(m[2])
		want, ok := bodies[name]
		switch {
		case !ok:
			tb.Errorf("%s: marker names %s, which is not an Example function of the repo", file, name)
		case len(want) > 1:
			tb.Errorf("%s: marker names %s, which is declared with different bodies in several packages", file, name)
		case got != want[0]:
			tb.Errorf("%s: snippet of %s differs from the Example body\n--- snippet\n%s\n--- example\n%s", file, name, got, want[0])
		}
	}
	return len(blocks)
}

// tableFirstCells returns the lower-cased first cell of each markdown
// table row of doc.
func tableFirstCells(doc string) []string {
	var cells []string
	for l := range strings.SplitSeq(doc, "\n") {
		l = strings.TrimSpace(l)
		if !strings.HasPrefix(l, "|") {
			continue
		}
		first, _, _ := strings.Cut(strings.TrimPrefix(l, "|"), "|")
		cells = append(cells, strings.ToLower(strings.TrimSpace(first)))
	}
	return cells
}

func checkREADME(tb testing.TB) {
	tb.Helper()
	doc := bIRead(tb, "README.md")
	if n := checkSnippets(tb, "README.md", doc); n < 8 {
		tb.Errorf("README.md has %d example-marked go blocks, want >= 8", n)
	}
	bIAbsent(tb, "README.md", doc, "validator.v8", "versions/3.0.1.md")
	bIContains(tb, "README.md", doc, "3.2.1", "11065", "ScalarUI")

	// One tags-table row per TagNames field (15) and BinderTags entry (7).
	cells := tableFirstCells(doc)
	for _, tag := range []string{
		"description", "example", "format", "deprecated", "title", "pattern",
		"readonly", "writeonly", "nullable", "multipleof", "additionalproperties",
		"inline", "style", "allowreserved", "cookie", // TagNames
		"path", "query", "header", "enum", "default", "validate", "explode", // BinderTags
	} {
		if !contains(cells, "`"+tag+"`") {
			tb.Errorf("README.md tags table has no row for `%s`", tag)
		}
	}
	var headings []string
	for l := range strings.SplitSeq(doc, "\n") {
		if strings.HasPrefix(l, "#") {
			headings = append(headings, strings.ToLower(l))
		}
	}
	for _, sec := range []string{"interfaces", "hooks", "naming", "deepobject", "cookies", "docs ui", "yaml"} {
		if !anyContains(headings, sec) {
			tb.Errorf("README.md has no section heading containing %q", sec)
		}
	}
}

func anyContains(list []string, sub string) bool {
	for _, s := range list {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func checkMigration(tb testing.TB) {
	tb.Helper()
	doc := bIRead(tb, "MIGRATION.md")
	bIContains(tb, "MIGRATION.md", doc,
		"SchemaOrRef", ".Value", ".Ref", "TypeSet", "json.Number", "Explode",
		"MarshalYAML", "SpecGenConfig", "TagNames", "BinderTags", "Errors()", "Err()",
		"InstallBindHook", "WithoutBindHook", "operationId", "DELETE", "Duration",
		"url.URL", "nullable", "oapi-codegen", "kin-openapi", "ogen", "DocumentHook",
		"snapshot", "1.26")
	checkSnippets(tb, "MIGRATION.md", doc)
}

// TestSnippetCheckerSelf pins the behaviour of the checker on a fixture:
// an Example body is found, formatted and compared.
func TestSnippetCheckerSelf(t *testing.T) {
	bodies := exampleBodies(t)
	got := bodies["ExampleNew_options"]
	if len(got) != 1 || !strings.HasPrefix(got[0], "f := fizz.New(") || strings.Contains(got[0], "Output") {
		t.Fatalf("ExampleNew_options body = %q", got)
	}
	var bad testing.TB = &fakeTB{}
	doc := "<!-- example: ExampleDoesNotExist -->\n```go\nx := 1\n```\n"
	checkSnippets(bad, "fixture.md", doc)
	if !bad.Failed() {
		t.Fatalf("a marker naming a missing Example must fail;")
	}
}

// fakeTB records failures of the checker without failing the test.
type fakeTB struct {
	testing.TB
	failed bool
}

func (f *fakeTB) Helper()               {}
func (f *fakeTB) Errorf(string, ...any) { f.failed = true }
func (f *fakeTB) Fatalf(string, ...any) { f.failed = true }
func (f *fakeTB) Failed() bool          { return f.failed }
func (f *fakeTB) Cleanup(func())        {}
func (f *fakeTB) Name() string          { return "fake" }
func (f *fakeTB) Logf(string, ...any)   {}

// I10
func TestREADMESnippetsMatchExamples(t *testing.T) {
	checkREADME(t)
}

// I11
func TestMigrationCoversTopics(t *testing.T) {
	checkMigration(t)
}
