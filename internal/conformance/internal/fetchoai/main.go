// Command fetchoai downloads the pinned official OpenAPI Initiative material
// into internal/conformance/oai and checks (or rewrites) MANIFEST.json.
//
// It is run by go generate from internal/conformance/oai. The sources are a
// fixed list in this file, each pinned by repository commit. The downloaded
// files are untrusted data: they are written to disk and hashed, never
// executed.
//
// By default every download is compared with the manifest and any difference
// is an error; nothing is written. With -update the files and the manifest
// are rewritten.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"time"
)

const rawBase = "https://raw.githubusercontent.com"

const (
	learnCommit = "82b82f372aa185a07ea11bc158619895ab21f741"
	oasTests    = "0f9aa49039215e2ed414cdefe35f718273b77976"
	oasSpec     = "4f86429f134f7e48b6351563549b4502fc69de52"
)

// source is a set of files of one repository at one commit.
type source struct {
	repo    string // owner/name on GitHub
	short   string // directory name under orig/
	commit  string
	license string // SPDX identifier
	files   []string
}

// licenseFile is an upstream LICENSE kept next to the manifest.
type licenseFile struct {
	src  int    // index into sources
	name string // file name under the output directory
}

var sources = []source{
	{
		repo: "OAI/learn.openapis.org", short: "learn", commit: learnCommit, license: "CC-BY-4.0",
		files: []string{
			"examples/v3.0/api-with-examples.yaml",
			"examples/v3.0/callback-example.yaml",
			"examples/v3.0/link-example.yaml",
			"examples/v3.0/petstore-expanded.yaml",
			"examples/v3.0/petstore.yaml",
			"examples/v3.0/uspto.yaml",
			"examples/v3.1/non-oauth-scopes.yaml",
			"examples/v3.1/tictactoe.yaml",
			"examples/v3.1/webhook-example.yaml",
			"examples/v3.2/3.2-query-example.yaml",
			"examples/v3.2/3.2-tags-example.yaml",
		},
	},
	{
		repo: "OAI/OpenAPI-Specification", short: "oas", commit: oasTests, license: "Apache-2.0",
		files: []string{
			"tests/schema/pass/comp_pathitems.yaml",
			"tests/schema/pass/info_summary.yaml",
			"tests/schema/pass/json_schema_dialect.yaml",
			"tests/schema/pass/license_identifier.yaml",
			"tests/schema/pass/mega.yaml",
			"tests/schema/pass/minimal_comp.yaml",
			"tests/schema/pass/minimal_hooks.yaml",
			"tests/schema/pass/minimal_paths.yaml",
			"tests/schema/pass/non-oauth-scopes.yaml",
			"tests/schema/pass/path_no_response.yaml",
			"tests/schema/pass/path_var_empty_pathitem.yaml",
			"tests/schema/pass/schema.yaml",
			"tests/schema/pass/servers.yaml",
			"tests/schema/pass/valid_schema_types.yaml",
			"tests/schema/pass/webhook-example.yaml",
			"tests/schema/fail/invalid_schema_types.yaml",
			"tests/schema/fail/no_containers.yaml",
			"tests/schema/fail/server_enum_empty.yaml",
			"tests/schema/fail/servers.yaml",
			"tests/schema/fail/unknown_container.yaml",
		},
	},
	{
		repo: "OAI/OpenAPI-Specification", short: "oas", commit: oasSpec, license: "Apache-2.0",
		files: []string{"versions/3.2.1.md"},
	},
}

// licenses lists the LICENSE files, fetched from the commit of the source
// at index src.
var licenses = []licenseFile{
	{src: 0, name: "LICENSE-learn-CC-BY-4.0"},
	{src: 1, name: "LICENSE-oas-Apache-2.0"},
}

// entry is one MANIFEST.json element.
type entry struct {
	Path       string `json:"path"` // relative to the output directory
	Repo       string `json:"repo"`
	Commit     string `json:"commit"`
	SourcePath string `json:"sourcePath"`
	SHA256     string `json:"sha256"`
	License    string `json:"license"`
}

// blob is a downloaded file, ready to be written.
type blob struct {
	entry entry
	data  []byte
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("fetchoai: ")
	out := flag.String("out", ".", "output directory (internal/conformance/oai)")
	base := flag.String("base", rawBase, "base URL of the raw file host")
	update := flag.Bool("update", false, "rewrite the files and MANIFEST.json instead of failing on a mismatch")
	flag.Parse()

	client := &http.Client{Timeout: 2 * time.Minute}
	blobs, err := fetchAll(client, *base)
	if err != nil {
		log.Fatal(err)
	}
	manifest := manifestOf(blobs)
	if *update {
		if err := write(*out, blobs, manifest); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("wrote %d files and MANIFEST.json\n", len(blobs))
		return
	}
	if err := verify(*out, manifest); err != nil {
		log.Fatalf("%v (use -update to rewrite)", err)
	}
	fmt.Printf("%d files match MANIFEST.json\n", len(blobs))
}

// fetchAll downloads every pinned file, sorted by output path.
func fetchAll(c *http.Client, base string) ([]blob, error) {
	var blobs []blob
	add := func(s source, srcPath, dst string) error {
		url := fmt.Sprintf("%s/%s/%s/%s", base, s.repo, s.commit, srcPath)
		data, err := download(c, url)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		blobs = append(blobs, blob{
			entry: entry{
				Path: dst, Repo: s.repo, Commit: s.commit, SourcePath: srcPath,
				SHA256: hex.EncodeToString(sum[:]), License: s.license,
			},
			data: data,
		})
		return nil
	}
	for _, s := range sources {
		for _, f := range s.files {
			if err := add(s, f, path.Join("orig", s.short, f)); err != nil {
				return nil, err
			}
		}
	}
	for _, l := range licenses {
		if err := add(sources[l.src], "LICENSE", l.name); err != nil {
			return nil, err
		}
	}
	sort.Slice(blobs, func(i, j int) bool { return blobs[i].entry.Path < blobs[j].entry.Path })
	for i := 1; i < len(blobs); i++ {
		if blobs[i].entry.Path == blobs[i-1].entry.Path {
			return nil, fmt.Errorf("duplicate output path %s", blobs[i].entry.Path)
		}
	}
	return blobs, nil
}

// download fetches url and fails on a non-200 status or an empty body.
func download(c *http.Client, url string) ([]byte, error) {
	resp, err := c.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: status %s", url, resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", url, err)
	}
	if len(body) == 0 {
		return nil, fmt.Errorf("GET %s: empty body", url)
	}
	return body, nil
}

// manifestOf returns the manifest entries of blobs, already sorted.
func manifestOf(blobs []blob) []entry {
	m := make([]entry, len(blobs))
	for i, b := range blobs {
		m[i] = b.entry
	}
	return m
}

// encodeManifest renders the manifest as indented JSON with a final newline.
func encodeManifest(m []entry) ([]byte, error) {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

// verify compares the freshly downloaded manifest with MANIFEST.json in out.
func verify(out string, fresh []entry) error {
	raw, err := os.ReadFile(filepath.Join(out, "MANIFEST.json"))
	if err != nil {
		return err
	}
	var have []entry
	if err := json.Unmarshal(raw, &have); err != nil {
		return fmt.Errorf("MANIFEST.json: %w", err)
	}
	want := map[string]entry{}
	for _, e := range have {
		want[e.Path] = e
	}
	var errs []error
	for _, e := range fresh {
		w, ok := want[e.Path]
		delete(want, e.Path)
		switch {
		case !ok:
			errs = append(errs, fmt.Errorf("%s: not in MANIFEST.json", e.Path))
		case w.SHA256 != e.SHA256:
			errs = append(errs, fmt.Errorf("%s: %s/%s hashes to %s, manifest has %s",
				e.Path, e.Repo, e.SourcePath, e.SHA256, w.SHA256))
		case w != e:
			errs = append(errs, fmt.Errorf("%s: manifest metadata differs", e.Path))
		}
	}
	for p := range want {
		errs = append(errs, fmt.Errorf("%s: in MANIFEST.json but no longer fetched", p))
	}
	return errors.Join(errs...)
}

// write stores the files and the manifest under out. Stale files of orig/
// are removed first so that orig/ always equals the manifest.
func write(out string, blobs []blob, m []entry) error {
	if err := os.RemoveAll(filepath.Join(out, "orig")); err != nil {
		return err
	}
	for _, b := range blobs {
		dst := filepath.Join(out, filepath.FromSlash(b.entry.Path))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dst, b.data, 0o644); err != nil {
			return err
		}
	}
	mb, err := encodeManifest(m)
	if err != nil {
		return err
	}
	if bytes.Contains(mb, []byte("null")) {
		return errors.New("manifest contains null")
	}
	return os.WriteFile(filepath.Join(out, "MANIFEST.json"), mb, 0o644)
}
