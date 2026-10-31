// Command upgradeoai regenerates internal/conformance/oai/upgraded and the
// generated section of oai/NOTICE from the pristine files under oai/orig.
//
// It is run by go generate from internal/conformance/oai. It is offline: it
// only parses files under orig/ (data, never executed) and writes the
// upgraded/ tree and NOTICE.
package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/wI2L/fizz/internal/conformance/upgrade"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("upgradeoai: ")
	if _, err := os.Stat("orig"); err != nil {
		log.Fatalf("run from internal/conformance/oai (no orig/ here): %v", err)
	}
	c, err := upgrade.Build(os.DirFS("."))
	if err != nil {
		log.Fatal(err)
	}
	if err := os.RemoveAll(upgrade.UpgradedDir); err != nil {
		log.Fatal(err)
	}
	for name, data := range c.Files {
		p := filepath.FromSlash(name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			log.Fatal(err)
		}
		if err := os.WriteFile(p, data, 0o644); err != nil {
			log.Fatal(err)
		}
	}
	notice, err := os.ReadFile("NOTICE")
	if err != nil {
		log.Fatal(err)
	}
	out, err := upgrade.ApplyNotice(notice, c.Notice)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile("NOTICE", out, 0o644); err != nil {
		log.Fatal(err)
	}
	log.Printf("wrote %d files, %d spec-text blocks excluded", len(c.Files), len(c.Excluded))
}
