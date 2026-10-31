//go:build acceptance

package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/wI2L/fizz/internal/plan"
)

var bAFixedBy = regexp.MustCompile(`^FixedBy: (P\d+\.\d+[a-z]?(?:\+P\d+\.\d+[a-z]?)*)$`)

// G-STUB: every declaration of a zz_stub_acceptance.go file names an
// open task in its doc comment.
func TestStubsBelongToOpenTasks(t *testing.T) {
	root := bARoot(t)
	fset := token.NewFileSet()
	var files int
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != "zz_stub_acceptance.go" {
			return nil
		}
		files++
		rel, _ := filepath.Rel(root, path)
		f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			t.Errorf("%s: %v", rel, err)
			return nil
		}
		check := func(pos token.Pos, name string, doc *ast.CommentGroup) {
			where := fset.Position(pos).String()
			if doc == nil {
				t.Errorf("%s: %s has no doc comment", where, name)
				return
			}
			lines := strings.Split(strings.TrimSpace(doc.Text()), "\n")
			m := bAFixedBy.FindStringSubmatch(lines[len(lines)-1])
			if m == nil {
				t.Errorf("%s: %s: doc comment must end with %q", where, name, "FixedBy: Pn.m")
				return
			}
			if plan.IsDone(m[1]) {
				t.Errorf("%s: %s: stub of done task %s must be deleted", where, name, m[1])
			}
		}
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				check(d.Pos(), d.Name.Name, d.Doc)
			case *ast.GenDecl:
				if d.Tok == token.IMPORT {
					continue
				}
				for _, spec := range d.Specs {
					doc := d.Doc
					if len(d.Specs) > 1 || d.Lparen.IsValid() {
						doc = nil // grouped blocks are not allowed: each spec needs its own doc
					}
					switch s := spec.(type) {
					case *ast.TypeSpec:
						if s.Doc != nil {
							doc = s.Doc
						}
						check(s.Pos(), s.Name.Name, doc)
					case *ast.ValueSpec:
						if s.Doc != nil {
							doc = s.Doc
						}
						check(s.Pos(), s.Names[0].Name, doc)
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if files == 0 {
		t.Fatal("no zz_stub_acceptance.go found")
	}
}
