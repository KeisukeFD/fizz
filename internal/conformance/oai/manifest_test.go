package oai

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

type entry struct {
	Path       string `json:"path"`
	Repo       string `json:"repo"`
	Commit     string `json:"commit"`
	SourcePath string `json:"sourcePath"`
	SHA256     string `json:"sha256"`
	License    string `json:"license"`
}

// TestManifestMatchesFiles checks, offline, that every MANIFEST.json entry
// exists with the recorded sha256 and that orig/ holds no unlisted file.
// Files outside orig/ (such as upgraded/) are not covered.
func TestManifestMatchesFiles(t *testing.T) {
	raw, err := os.ReadFile("MANIFEST.json")
	if err != nil {
		t.Fatal(err)
	}
	var m []entry
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if len(m) == 0 {
		t.Fatal("MANIFEST.json is empty")
	}
	listed := map[string]bool{}
	for i, e := range m {
		if i > 0 && m[i-1].Path >= e.Path {
			t.Errorf("manifest not sorted/unique at %s", e.Path)
		}
		if e.Repo == "" || e.Commit == "" || e.SourcePath == "" || e.License == "" {
			t.Errorf("%s: incomplete entry %+v", e.Path, e)
		}
		listed[e.Path] = true
		data, err := os.ReadFile(filepath.FromSlash(e.Path))
		if err != nil {
			t.Errorf("%s: %v", e.Path, err)
			continue
		}
		sum := sha256.Sum256(data)
		if got := hex.EncodeToString(sum[:]); got != e.SHA256 {
			t.Errorf("%s: sha256 %s, manifest %s", e.Path, got, e.SHA256)
		}
	}
	var unlisted []string
	err = filepath.WalkDir("orig", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && !listed[filepath.ToSlash(p)] {
			unlisted = append(unlisted, filepath.ToSlash(p))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(unlisted)
	for _, p := range unlisted {
		t.Errorf("%s: file under orig/ is not in MANIFEST.json", p)
	}
}
