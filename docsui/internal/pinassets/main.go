// Command pinassets downloads the pinned docs UI assets into
// docsui/embedded/assets and checks (or rewrites) the SRI constants in
// package docsui.
//
// It is run by go generate from docsui/embedded. Every string constant in
// docsui/*.go named XxxJS or XxxCSS whose value is an https URL is a pinned
// asset; XxxJSIntegrity / XxxCSSIntegrity holds its sha384 SRI value.
//
// By default a hash that differs from the constant is an error. With
// -update the constant is rewritten in place.
package main

import (
	"crypto/sha512"
	"encoding/base64"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// asset is a pinned URL constant and its Integrity companion.
type asset struct {
	name      string // constant name, e.g. SwaggerUIJS
	url       string
	integrity string // current value of the Integrity constant
	file      string // docsui source file declaring the Integrity constant
	litStart  int    // byte offsets of the Integrity literal in file
	litEnd    int
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("pinassets: ")
	docsuiDir := flag.String("docsui", "..", "docsui package directory")
	assetsDir := flag.String("assets", "assets", "output directory for the downloaded assets")
	update := flag.Bool("update", false, "rewrite the Integrity constants instead of failing on a mismatch")
	flag.Parse()

	assets, err := findAssets(*docsuiDir)
	if err != nil {
		log.Fatal(err)
	}
	if len(assets) == 0 {
		log.Fatal("no pinned assets found")
	}
	if err := os.MkdirAll(*assetsDir, 0o755); err != nil {
		log.Fatal(err)
	}
	client := &http.Client{Timeout: 2 * time.Minute}
	sums := make(map[string]string, len(assets))
	for _, a := range assets {
		body, err := download(client, a.url)
		if err != nil {
			log.Fatalf("%s: %v", a.name, err)
		}
		sum := sri(body)
		if sum != a.integrity && !*update {
			log.Fatalf("%s: %s hashes to %s, constant %sIntegrity is %s (use -update to rewrite)",
				a.name, a.url, sum, a.name, a.integrity)
		}
		dst := filepath.Join(*assetsDir, path.Base(a.url))
		if err := os.WriteFile(dst, body, 0o644); err != nil {
			log.Fatal(err)
		}
		sums[a.name] = sum
		fmt.Printf("%s: %d bytes, %s\n", dst, len(body), sum)
	}
	if err := rewrite(assets, sums); err != nil {
		log.Fatal(err)
	}
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

// sri returns the SRI value (sha384-<base64>) of b.
func sri(b []byte) string {
	h := sha512.Sum384(b)
	return "sha384-" + base64.StdEncoding.EncodeToString(h[:])
}

// findAssets parses the non-test Go files of dir and returns the pinned
// assets, sorted by name.
func findAssets(dir string) ([]asset, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		return nil, err
	}
	type lit struct {
		val        string
		file       string
		start, end int
	}
	consts := map[string]lit{}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		fset := token.NewFileSet()
		af, err := parser.ParseFile(fset, f, src, 0)
		if err != nil {
			return nil, err
		}
		for _, d := range af.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok || gd.Tok != token.CONST {
				continue
			}
			for _, s := range gd.Specs {
				vs := s.(*ast.ValueSpec)
				for i, n := range vs.Names {
					if i >= len(vs.Values) {
						continue
					}
					bl, ok := vs.Values[i].(*ast.BasicLit)
					if !ok || bl.Kind != token.STRING {
						continue
					}
					v, err := strconv.Unquote(bl.Value)
					if err != nil {
						return nil, err
					}
					consts[n.Name] = lit{v, f, fset.Position(bl.Pos()).Offset, fset.Position(bl.End()).Offset}
				}
			}
		}
	}
	var out []asset
	for name, c := range consts {
		if !(strings.HasSuffix(name, "JS") || strings.HasSuffix(name, "CSS")) || !strings.HasPrefix(c.val, "https://") {
			continue
		}
		ic, ok := consts[name+"Integrity"]
		if !ok {
			return nil, fmt.Errorf("constant %sIntegrity not found", name)
		}
		out = append(out, asset{name: name, url: c.val, integrity: ic.val, file: ic.file, litStart: ic.start, litEnd: ic.end})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out, nil
}

// rewrite replaces the Integrity literals whose value changed and gofmts
// the touched files.
func rewrite(assets []asset, sums map[string]string) error {
	byFile := map[string][]asset{}
	for _, a := range assets {
		if sums[a.name] != a.integrity {
			byFile[a.file] = append(byFile[a.file], a)
		}
	}
	for f, as := range byFile {
		src, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		out, err := replaceLiterals(src, as, sums)
		if err != nil {
			return fmt.Errorf("%s: %w", f, err)
		}
		if err := os.WriteFile(f, out, 0o644); err != nil {
			return err
		}
		fmt.Printf("rewrote %s\n", f)
	}
	return nil
}

// replaceLiterals substitutes the literals located by as, back to front so
// that the offsets stay valid, and formats the result.
func replaceLiterals(src []byte, as []asset, sums map[string]string) ([]byte, error) {
	sort.Slice(as, func(i, j int) bool { return as[i].litStart > as[j].litStart })
	for _, a := range as {
		tail := append([]byte(strconv.Quote(sums[a.name])), src[a.litEnd:]...)
		src = append(src[:a.litStart:a.litStart], tail...)
	}
	return format.Source(src)
}
