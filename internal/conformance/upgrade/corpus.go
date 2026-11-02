package upgrade

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/KeisukeFD/fizz/internal/specvalidate"
)

// Paths below are relative to the internal/conformance/oai directory.
const (
	specTextPath = "orig/oas/versions/3.2.1.md"
	// UpgradedDir is where the generated corpus lives.
	UpgradedDir = "upgraded"
	// NoticeMarker starts the generated section of NOTICE.
	NoticeMarker = "Modified copies\n---------------\n"
)

// Entry describes one generated file: where it came from and what the
// converter changed.
type Entry struct {
	File    string   `json:"file"`
	Source  string   `json:"source"`
	Wrapper string   `json:"wrapper,omitempty"` // spec text only: what the fragment was wrapped as
	Changes []Change `json:"changes"`
}

// Exclusion records a spec-text block that is not part of the corpus.
type Exclusion struct {
	Name     string `json:"name"`
	Source   string `json:"source"`
	Heading  string `json:"heading"`
	Category string `json:"category"`
	Reason   string `json:"reason"`
}

// Exclusion categories.
const (
	CategoryInvalidYAML     = "invalid-yaml"
	CategoryUnclassified    = "unclassified"
	CategoryFailsValidation = "fails-validation"
)

// Corpus is the full converter output, held in memory.
type Corpus struct {
	Files    map[string][]byte // path relative to the oai directory, e.g. "upgraded/learn/..."
	Entries  []Entry           // sorted by File; also serialized as CHANGES.json
	Excluded []Exclusion       // spec-text blocks left out; also EXCLUDED.json
	Notice   string            // generated NOTICE section, starting with NoticeMarker
}

// Build runs the converter over the vendored material in oai (an fs rooted
// at internal/conformance/oai). It reads only under orig/ and never touches
// the network or executes anything it reads.
func Build(oai fs.FS) (*Corpus, error) {
	c := &Corpus{Files: map[string][]byte{}}
	for _, root := range []string{"orig/learn/examples", "orig/oas/tests/schema"} {
		err := fs.WalkDir(oai, root, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(p, ".yaml") {
				return err
			}
			return c.addDocument(oai, p)
		})
		if err != nil {
			return nil, fmt.Errorf("upgrade %s: %w", root, err)
		}
	}
	if err := c.addSpecText(oai); err != nil {
		return nil, err
	}
	sort.Slice(c.Entries, func(i, j int) bool { return c.Entries[i].File < c.Entries[j].File })
	sort.Slice(c.Excluded, func(i, j int) bool { return c.Excluded[i].Name < c.Excluded[j].Name })

	var err error
	if c.Files[UpgradedDir+"/CHANGES.json"], err = EncodeJSON(c.Entries); err != nil {
		return nil, err
	}
	excl := c.Excluded
	if excl == nil {
		excl = []Exclusion{}
	}
	if c.Files[UpgradedDir+"/EXCLUDED.json"], err = EncodeJSON(excl); err != nil {
		return nil, err
	}
	c.Notice = c.notice()
	return c, nil
}

func (c *Corpus) addDocument(oai fs.FS, p string) error {
	src, err := fs.ReadFile(oai, p)
	if err != nil {
		return err
	}
	doc, err := DecodeYAML(src)
	if err != nil {
		return fmt.Errorf("%s: %w", p, err)
	}
	doc, changes, err := Upgrade(doc)
	if err != nil {
		return fmt.Errorf("%s: %w", p, err)
	}
	out, err := EncodeJSON(doc)
	if err != nil {
		return fmt.Errorf("%s: %w", p, err)
	}
	file := UpgradedDir + "/" + strings.TrimSuffix(strings.TrimPrefix(p, "orig/"), ".yaml") + ".json"
	c.Files[file] = out
	c.Entries = append(c.Entries, Entry{File: file, Source: p, Changes: nonNil(changes)})
	return nil
}

func (c *Corpus) addSpecText(oai fs.FS) error {
	md, err := fs.ReadFile(oai, specTextPath)
	if err != nil {
		return err
	}
	for _, b := range ExtractBlocks(md) {
		source := fmt.Sprintf("%s:L%d", specTextPath, b.Line)
		exclude := func(category, reason string) {
			c.Excluded = append(c.Excluded, Exclusion{Name: b.Name, Source: source, Heading: b.Heading, Category: category, Reason: reason})
		}
		v, err := DecodeYAML([]byte(b.Text))
		if err != nil {
			exclude(CategoryInvalidYAML, firstLine(err.Error()))
			continue
		}
		w, reason := Wrap(b.Heading, BlockTitle(b.Name), v)
		if reason != "" {
			exclude(CategoryUnclassified, reason)
			continue
		}
		doc, changes, err := Upgrade(w.Doc)
		if err != nil {
			return fmt.Errorf("%s: %w", b.Name, err)
		}
		out, err := EncodeJSON(doc)
		if err != nil {
			return fmt.Errorf("%s: %w", b.Name, err)
		}
		if err := specvalidate.Validate(out); err != nil {
			reason := "fails validation after wrapping as " + w.Kind + ": " + summarize(err)
			if note, ok := exclusionNotes[b.Name]; ok {
				reason = note + " (" + reason + ")"
			}
			exclude(CategoryFailsValidation, reason)
			continue
		}
		file := UpgradedDir + "/spectext/" + b.Name + ".json"
		c.Files[file] = out
		wrapper := w.Kind
		if wrapper == "document" {
			wrapper = ""
		}
		c.Entries = append(c.Entries, Entry{File: file, Source: source, Wrapper: wrapper, Changes: nonNil(changes)})
	}
	return nil
}

// exclusionNotes explains, by block name, why an upstream block fails
// validation. The block content is never modified to make it pass.
var exclusionNotes = map[string]string{
	"examples-of-api-base-url-determination-1": "upstream spec text bug: a full document with none of paths, components, webhooks",
}

func nonNil(c []Change) []Change {
	if c == nil {
		return []Change{}
	}
	return c
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return strings.TrimSpace(line)
}

// summarize reduces a specvalidate error to its first few problem lines plus
// a count of the others.
func summarize(err error) string {
	lines := strings.Split(err.Error(), "\n")
	var probs []string
	for _, l := range lines[1:] {
		if l = strings.TrimSpace(l); l != "" {
			probs = append(probs, l)
		}
	}
	if len(probs) == 0 {
		return firstLine(err.Error())
	}
	const max = 4
	if len(probs) <= max {
		return strings.Join(probs, "; ")
	}
	return fmt.Sprintf("%s (and %d more)", strings.Join(probs[:max], "; "), len(probs)-max)
}

// notice renders the generated NOTICE section.
func (c *Corpus) notice() string {
	var b strings.Builder
	b.WriteString(NoticeMarker)
	b.WriteString(`
The upgraded/ directory contains copies of the files above that were
modified by a deterministic converter (internal/conformance/upgrade, run by
"go generate" through internal/conformance/internal/upgradeoai). These are
modified versions, not the originals. As required by CC BY 4.0 section
3(a)(1)(B) and Apache-2.0 section 4(b), this section states the changes made
to each file. The exact, machine-readable change list (JSON pointer, rule,
before, after) is upgraded/CHANGES.json.

Changes applied to every file:
  - converted from YAML to JSON (sorted keys, two-space indent);
  - "openapi" set to 3.2.1 when it had another value;
  - unquoted YAML scalars in info.version were made strings, preserving the
    source text (1.0 became "1.0");
  - a jsonSchemaDialect or Schema Object $schema naming an OAS 3.1 dialect
    (https://spec.openapis.org/oas/3.1/dialect/...) became the OAS 3.2
    dialect URI. Other $schema values, such as JSON Schema 2020-12, are kept.

Changes applied only to openapi 3.0.x documents (OAI 3.1 migration guidance),
inside Schema Objects only:
  - "nullable: true" became a "null" entry in "type" (and in "enum" when
    present); "nullable: false" was removed;
  - boolean "exclusiveMinimum"/"exclusiveMaximum" with "minimum"/"maximum"
    became the numeric form;
  - in multipart and form-urlencoded bodies, "type: string, format: binary"
    became "contentMediaType: application/octet-stream" and "format: base64"
    became "contentEncoding: base64".

Specification text (orig/oas/versions/3.2.1.md): each yaml block was parsed
and wrapped, unmodified, in a minimal OpenAPI 3.2.1 document (an "info"
object and the container named below). Blocks that could not be wrapped or
that do not validate are not copied; see upgraded/EXCLUDED.json.

Files:
`)
	for _, e := range c.Entries {
		fmt.Fprintf(&b, "  %s\n      from %s\n      %s\n", e.File, e.Source, e.summary())
	}
	return b.String()
}

func (e Entry) summary() string {
	var parts []string
	counts := map[string]int{}
	var rules []string
	for _, ch := range e.Changes {
		if ch.Rule == "openapi-version" {
			parts = append(parts, fmt.Sprintf("openapi %v -> %v", ch.Before, ch.After))
			continue
		}
		if counts[ch.Rule] == 0 {
			rules = append(rules, ch.Rule)
		}
		counts[ch.Rule]++
	}
	sort.Strings(rules)
	for _, r := range rules {
		parts = append(parts, fmt.Sprintf("%dx %s", counts[r], r))
	}
	if e.Wrapper != "" {
		parts = append([]string{"wrapped as " + e.Wrapper + ", content unmodified"}, parts...)
	}
	if len(parts) == 0 {
		return "re-encoded as JSON, content unchanged"
	}
	return strings.Join(parts, "; ")
}

// ApplyNotice returns existing with everything from NoticeMarker to the end
// replaced by section. Applying it twice gives the same bytes. It fails when
// the marker is absent.
func ApplyNotice(existing []byte, section string) ([]byte, error) {
	i := bytes.Index(existing, []byte(NoticeMarker))
	if i < 0 {
		return nil, errors.New("NOTICE has no \"Modified copies\" section")
	}
	out := append([]byte{}, existing[:i]...)
	return append(out, section...), nil
}
