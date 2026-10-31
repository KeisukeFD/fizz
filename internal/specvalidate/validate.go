// Package specvalidate validates generated OpenAPI documents against the
// official OAS 3.2 JSON Schemas (targeting 3.2.1), vendored under schemas/3.2 and loaded
// offline. It is intended for use from tests.
package specvalidate

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

//go:embed schemas/3.2/*.json
var schemaFS embed.FS

const (
	idPrefix = "https://spec.openapis.org/oas/3.2/"
	// BaseURL is the $id of the schema-base document, which validates
	// Schema Objects against the OAS dialect.
	BaseURL = idPrefix + "schema-base/2026-08-30"
)

// DialectURL is the $id of the OAS 3.2 base dialect, the only OAS dialect URI
// a 3.2 document may use in jsonSchemaDialect or $schema. It is derived from
// the same table the offline loader uses.
var DialectURL = idPrefix + "dialect/" + schemaDates["dialect"]

// schemaDates maps each vendored document kind to the dated suffix of its
// $id. The kinds are revised independently, so the dates differ.
var schemaDates = map[string]string{
	"schema":      "2026-08-30",
	"schema-base": "2026-08-30",
	"dialect":     "2026-02-26",
	"meta":        "2026-02-26",
}

// embedLoader maps spec.openapis.org OAS 3.2 URLs to embedded files and
// refuses everything else, so no network access can happen.
type embedLoader struct{}

func (embedLoader) Load(url string) (any, error) {
	rest, ok := strings.CutPrefix(url, idPrefix)
	if !ok {
		return nil, fmt.Errorf("specvalidate: refusing to load %q (offline)", url)
	}
	kind, date, ok := strings.Cut(rest, "/")
	if !ok || schemaDates[kind] != date {
		return nil, fmt.Errorf("specvalidate: unknown schema %q", url)
	}
	f, err := schemaFS.Open("schemas/3.2/" + kind + "-" + date + ".json")
	if err != nil {
		return nil, fmt.Errorf("specvalidate: unknown schema %q: %w", url, err)
	}
	defer f.Close()
	return jsonschema.UnmarshalJSON(f)
}

var (
	once     sync.Once
	compiled *jsonschema.Schema
	errComp  error
	printer  = message.NewPrinter(language.English)
)

func schema() (*jsonschema.Schema, error) {
	once.Do(func() {
		c := jsonschema.NewCompiler()
		c.UseLoader(embedLoader{})
		compiled, errComp = c.Compile(BaseURL)
	})
	return compiled, errComp
}

// Validate checks doc (a JSON OpenAPI document) against the OAS 3.2
// schema-base. The error lists each failure with its instance pointer.
func Validate(doc []byte) error {
	sch, err := schema()
	if err != nil {
		return fmt.Errorf("specvalidate: compile schema: %w", err)
	}
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(doc))
	if err != nil {
		return fmt.Errorf("specvalidate: invalid JSON: %w", err)
	}
	err = sch.Validate(inst)
	if err == nil {
		return nil
	}
	var ve *jsonschema.ValidationError
	if !errors.As(err, &ve) {
		return err
	}
	var b strings.Builder
	b.WriteString("document is not a valid OpenAPI 3.2 description:")
	collect(&b, ve)
	return errors.New(b.String())
}

// collect appends one line per leaf cause: "<pointer>: <message>".
func collect(b *strings.Builder, ve *jsonschema.ValidationError) {
	if len(ve.Causes) == 0 {
		var ptr strings.Builder
		for _, seg := range ve.InstanceLocation {
			ptr.WriteString("/" + escape(seg))
		}
		fmt.Fprintf(b, "\n  %s: %s", ptr.String(), ve.ErrorKind.LocalizedString(printer))
		return
	}
	for _, c := range ve.Causes {
		collect(b, c)
	}
}
