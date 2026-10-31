package archtest

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// G-STUB: every stub file is behind the acceptance build tag, so that
// no stub can leak into the default build.
func TestStubFilesAreTagged(t *testing.T) {
	root := bARoot(t)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != "zz_stub_acceptance.go" {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		sc := bufio.NewScanner(f)
		if !sc.Scan() || sc.Text() != "//go:build acceptance" {
			t.Errorf("%s: first line must be //go:build acceptance", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
