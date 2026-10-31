package docsui

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"
)

//go:embed page.tmpl
var pageSrc string

var pages = template.Must(template.New("page").Parse(pageSrc))

// defaultTitle is the page title when WithTitle is not used.
const defaultTitle = "API Documentation"

// Option configures a docs UI handler.
type Option func(*config)

type config struct {
	title    string
	uiConfig map[string]any
	nonce    string
	assets   bool // WithAssetURLs was used
	js, css  string
	jsInt    string
	cssInt   string
}

// WithTitle sets the HTML title of the page. It is escaped.
func WithTitle(title string) Option {
	return func(c *config) { c.title = title }
}

// WithUIConfig merges m into the configuration object passed to the UI
// (SwaggerUIBundle or Scalar.createApiReference). The values must be
// JSON-encodable. The spec URL is always set from the specURL argument
// and, for Swagger UI, validatorUrl is always null: the public
// validator does not understand OpenAPI 3.2 and would leak the URL.
func WithUIConfig(m map[string]any) Option {
	return func(c *config) {
		if c.uiConfig == nil {
			c.uiConfig = make(map[string]any, len(m))
		}
		for k, v := range m {
			c.uiConfig[k] = v
		}
	}
}

// WithCSPNonce sets the nonce attribute on every script of the page,
// for use under a Content-Security-Policy.
func WithCSPNonce(nonce string) Option {
	return func(c *config) { c.nonce = nonce }
}

// WithAssetURLs overrides the URL of the UI script (js) and style sheet
// (css). integrity holds the Subresource Integrity values of js, then
// css. Without integrity values the page carries no integrity
// attribute: an override drops SRI unless the caller supplies it. An
// empty css means no style sheet, which is what Scalar needs, as its
// script bundles its styles.
func WithAssetURLs(js, css string, integrity ...string) Option {
	return func(c *config) {
		c.assets = true
		c.js, c.css = js, css
		c.jsInt, c.cssInt = "", ""
		if len(integrity) > 0 {
			c.jsInt = integrity[0]
		}
		if len(integrity) > 1 {
			c.cssInt = integrity[1]
		}
	}
}

// pageData is the data of the page templates.
type pageData struct {
	Title                     string
	Nonce                     string
	JS, CSS                   string
	JSIntegrity, CSSIntegrity string
	Config                    template.JS
}

// render builds the handler of the template name. ui is the UI
// configuration, already merged with the user's; the remaining
// arguments are the pinned assets, used unless WithAssetURLs is given.
func render(name string, cfg *config, ui map[string]any, js, css, jsInt, cssInt string) http.Handler {
	d := pageData{Title: cfg.title, Nonce: cfg.nonce, JS: js, CSS: css, JSIntegrity: jsInt, CSSIntegrity: cssInt}
	if cfg.assets {
		d.JS, d.CSS, d.JSIntegrity, d.CSSIntegrity = cfg.js, cfg.css, cfg.jsInt, cfg.cssInt
	}
	if d.Title == "" {
		d.Title = defaultTitle
	}
	// json.Marshal escapes <, > and & so the config cannot close the
	// script element it is embedded in.
	cj, err := json.Marshal(ui)
	var body bytes.Buffer
	if err == nil {
		d.Config = template.JS(cj)
		err = pages.ExecuteTemplate(&body, name, d)
	}
	if err != nil {
		msg := "docsui: " + err.Error()
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, msg, http.StatusInternalServerError)
		})
	}
	page := body.Bytes()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		h := w.Header()
		h.Set("Content-Type", "text/html; charset=utf-8")
		h.Set("Content-Length", strconv.Itoa(len(page)))
		w.WriteHeader(http.StatusOK)
		if r.Method == http.MethodGet {
			_, _ = w.Write(page)
		}
	})
}

func newConfig(opts []Option) *config {
	c := &config{}
	for _, o := range opts {
		o(c)
	}
	return c
}

// uiConfig returns the user's UI configuration with url set to specURL.
func uiConfig(c *config, specURL string) map[string]any {
	m := make(map[string]any, len(c.uiConfig)+3)
	for k, v := range c.uiConfig {
		m[k] = v
	}
	m["url"] = specURL
	return m
}
