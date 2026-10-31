package contract

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/wI2L/fizz/internal/plan"
)

// Target is an app under contract test.
type Target struct {
	// Handler serves the app.
	Handler http.Handler
	// Spec is the document the app serves.
	Spec []byte
	// Echo tells that the handlers answer 200 with their input as JSON
	// (the bound values). The client then requires a 2xx and that the
	// echoed value of each parameter equals what it sent. Without Echo
	// it only requires that the answer is not a 400.
	Echo bool
	// Tolerate, without Echo, accepts a 400 that the handler's own logic
	// answers (such as "not found"), as opposed to a rejection of the
	// request itself. Optional.
	Tolerate func(status int, body []byte) bool
	// Fields maps "in:name" to the key of the echoed JSON object. The
	// default is the parameter name.
	Fields map[string]string
	// Pending maps "<op> <in>:<name>" to the plan task that fixes the
	// drift (a red-until-done case). <op> is the operationId, else
	// "METHOD path". The name "all" stands for the request that carries
	// every parameter, "required-only" for the one that carries the
	// required parameters only, and "<in>:<name>/default" for the omitted
	// default.
	Pending map[string]string
}

type operation struct {
	method, path, id string
	params           []map[string]any
	body             map[string]any // requestBody, resolved
}

func (o operation) key() string {
	if o.id != "" {
		return o.id
	}
	return o.method + " " + o.path
}

var methods = []string{"get", "put", "post", "delete", "options", "head", "patch", "trace", "query"}

// operations lists the operations of the document, with their parameters
// ($ref resolved, the operation's overriding those of the path item).
func operations(doc Doc) []operation {
	var out []operation
	paths, _ := doc["paths"].(map[string]any)
	for _, path := range slices.Sorted(maps.Keys(paths)) {
		item, _ := paths[path].(map[string]any)
		item = doc.Resolve(item)
		for _, m := range methods {
			op, ok := item[m].(map[string]any)
			if !ok {
				continue
			}
			o := operation{method: strings.ToUpper(m), path: path}
			o.id, _ = op["operationId"].(string)
			o.params = mergeParams(doc, item["parameters"], op["parameters"])
			if rb, ok := op["requestBody"].(map[string]any); ok {
				o.body = doc.Resolve(rb)
			}
			out = append(out, o)
		}
	}
	return out
}

func mergeParams(doc Doc, lists ...any) []map[string]any {
	var out []map[string]any
	for _, l := range lists {
		list, _ := l.([]any)
		for _, e := range list {
			m, ok := e.(map[string]any)
			if !ok {
				continue
			}
			m = doc.Resolve(m)
			i := slices.IndexFunc(out, func(x map[string]any) bool { return x["name"] == m["name"] && x["in"] == m["in"] })
			if i >= 0 {
				out[i] = m
			} else {
				out = append(out, m)
			}
		}
	}
	return out
}

// Run sends, for every operation that has parameters, the requests of the
// contract:
//
//   - one per parameter, which carries that parameter (and the required
//     ones), serialized as the document says;
//   - one that carries the required parameters only;
//   - for a parameter with a default, one that omits it (Echo only);
//   - one that carries every parameter, unless one of them is pending.
func Run(t *testing.T, tg Target) {
	t.Helper()
	var doc Doc
	if err := json.Unmarshal(tg.Spec, &doc); err != nil {
		t.Fatalf("decode the spec: %v", err)
	}
	for _, o := range operations(doc) {
		if len(o.params) == 0 {
			continue
		}
		c := &client{t: t, tg: tg, doc: doc, op: o}
		t.Run(strings.ReplaceAll(o.key(), " ", "_"), c.run)
	}
}

type client struct {
	t   *testing.T
	tg  Target
	doc Doc
	op  operation
}

func (c *client) pending(sub string) string { return c.tg.Pending[c.op.key()+" "+sub] }

func (c *client) sub(t *testing.T, name, sub string, fn func(tb testing.TB)) {
	t.Run(name, func(t *testing.T) {
		if fb := c.pending(sub); fb != "" {
			plan.Pending(t, fb, fn)
			return
		}
		fn(t)
	})
}

func pname(p map[string]any) string {
	n, _ := p["name"].(string)
	in, _ := p["in"].(string)
	return in + ":" + n
}

func (c *client) run(t *testing.T) {
	anyPending := c.pending("required-only") != ""
	c.sub(t, "required_only", "required-only", func(tb testing.TB) { c.checkRequiredOnly(tb) })
	for _, p := range c.op.params {
		key := pname(p)
		anyPending = anyPending || c.pending(key) != "" || c.pending(key+"/default") != ""
		c.sub(t, strings.NewReplacer(":", "_", "/", "_").Replace(key), key, func(tb testing.TB) { c.checkParam(tb, p) })
		if _, ok := schemaOf(p)["default"]; ok && c.tg.Echo && p["in"] != "path" {
			c.sub(t, strings.NewReplacer(":", "_", "/", "_").Replace(key)+"_default", key+"/default", func(tb testing.TB) { c.checkDefault(tb, p) })
		}
	}
	if len(c.op.params) > 1 && !anyPending {
		c.sub(t, "all", "all", func(tb testing.TB) { c.checkAll(tb) })
	}
}

func schemaOf(p map[string]any) map[string]any {
	s, _ := p["schema"].(map[string]any)
	return s
}

// supported reports why the parameter cannot be driven, "" when it can.
func supported(p map[string]any) string {
	if _, ok := p["content"]; ok {
		return "parameter has content, not a schema"
	}
	if p["in"] == "querystring" {
		return "in: querystring"
	}
	if schemaOf(p) == nil {
		return "parameter has no schema"
	}
	return ""
}

// checkParam sends the parameter at its sample (at every enum value, when
// it has an enum).
func (c *client) checkParam(tb testing.TB, p map[string]any) {
	tb.Helper()
	if why := supported(p); why != "" {
		tb.Skip(why)
	}
	values := []any{c.doc.ParamSample(p)}
	if list, _ := c.doc.Resolve(schemaOf(p))["enum"].([]any); len(list) > 1 {
		values = nil
		for _, v := range list {
			if v != nil {
				values = append(values, v)
			}
		}
	}
	for _, v := range values {
		c.exchange(tb, map[string]any{pname(p): v}, []map[string]any{p})
	}
}

// checkDefault omits the parameter: the server must bind the default the
// document states.
func (c *client) checkDefault(tb testing.TB, p map[string]any) {
	tb.Helper()
	if why := supported(p); why != "" {
		tb.Skip(why)
	}
	s := schemaOf(p)
	def := c.doc.Resolve(s)["default"]
	want, err := c.doc.Coerce(s, def)
	if err != nil {
		tb.Errorf("%s: the documented default %s is not a value of its schema: %v",
			pname(p), mustJSON(def), err)
		return
	}
	resp := c.send(tb, map[string]any{})
	if resp == nil {
		return
	}
	c.compare(tb, p, want, resp, "default omitted")
}

// checkRequiredOnly sends the request that carries the required parameters
// only: every optional parameter is absent, which the document allows.
func (c *client) checkRequiredOnly(tb testing.TB) {
	tb.Helper()
	c.send(tb, map[string]any{})
}

func (c *client) checkAll(tb testing.TB) {
	tb.Helper()
	set := map[string]any{}
	var checked []map[string]any
	for _, p := range c.op.params {
		if supported(p) != "" {
			continue
		}
		set[pname(p)] = c.doc.ParamSample(p)
		checked = append(checked, p)
	}
	c.exchange(tb, set, checked)
}

// exchange sends the request that carries set (plus the required
// parameters) and compares the echo of each of checked.
func (c *client) exchange(tb testing.TB, set map[string]any, checked []map[string]any) {
	tb.Helper()
	resp := c.send(tb, set)
	if resp == nil {
		return
	}
	for _, p := range checked {
		want, err := c.doc.Coerce(schemaOf(p), set[pname(p)])
		if err != nil {
			tb.Errorf("%s: sample %s does not fit its schema: %v", pname(p), mustJSON(set[pname(p)]), err)
			continue
		}
		c.compare(tb, p, want, resp, "sent "+mustJSON(set[pname(p)]))
	}
}

// response is what the server answered to a request.
type response struct {
	desc string // the request, as sent
	body []byte
}

// send builds the request of the operation, with the parameters of set and
// the required ones, sends it and checks the status. It returns nil if
// the check failed or the request could not be built.
func (c *client) send(tb testing.TB, set map[string]any) *response {
	tb.Helper()
	path := c.op.path
	var query, cookies []string
	header := http.Header{}
	for _, p := range c.op.params {
		if supported(p) != "" {
			continue
		}
		key := pname(p)
		v, ok := set[key]
		if !ok {
			if req, _ := p["required"].(bool); !req && p["in"] != "path" {
				continue
			}
			v = c.doc.ParamSample(p)
		}
		pl, err := Serialize(ParamOf(p), v)
		if err != nil {
			tb.Skipf("%s: %v", key, err)
		}
		name := p["name"].(string)
		switch p["in"] {
		case "path":
			path = strings.ReplaceAll(path, "{"+name+"}", pl.Path)
		case "query":
			if pl.Query != "" {
				query = append(query, pl.Query)
			}
		case "header":
			header.Set(name, pl.Header)
		case "cookie":
			cookies = append(cookies, pl.Cookie)
		}
	}
	target := path
	if len(query) > 0 {
		target += "?" + strings.Join(query, "&")
	}
	body := ""
	if c.op.body != nil {
		if b, ok := c.jsonBody(); ok {
			body = b
			header.Set("Content-Type", "application/json")
		} else if req, _ := c.op.body["required"].(bool); req {
			tb.Skip("the request body has no JSON media type")
		}
	}
	r := httptest.NewRequest(c.op.method, target, strings.NewReader(body))
	for k, vs := range header {
		r.Header[k] = vs
	}
	if len(cookies) > 0 {
		r.Header.Set("Cookie", strings.Join(cookies, "; "))
	}
	desc := c.op.method + " " + target
	for _, k := range slices.Sorted(maps.Keys(header)) {
		desc += fmt.Sprintf(" [%s: %s]", k, header.Get(k))
	}
	if len(cookies) > 0 {
		desc += " [Cookie: " + r.Header.Get("Cookie") + "]"
	}
	w := httptest.NewRecorder()
	c.tg.Handler.ServeHTTP(w, r)
	ok := w.Code != http.StatusBadRequest || (c.tg.Tolerate != nil && c.tg.Tolerate(w.Code, w.Body.Bytes()))
	if c.tg.Echo {
		ok = w.Code/100 == 2
	}
	if !ok {
		tb.Errorf("%s: status %d, want %s: the document says this request is valid; body %q",
			desc, w.Code, map[bool]string{true: "2xx", false: "not 400"}[c.tg.Echo], w.Body.String())
		return nil
	}
	return &response{desc: desc, body: w.Body.Bytes()}
}

// jsonBody samples the request body when the operation has a JSON media
// type.
func (c *client) jsonBody() (string, bool) {
	content, _ := c.op.body["content"].(map[string]any)
	for _, mt := range slices.Sorted(maps.Keys(content)) {
		if mt != "application/json" && !strings.HasSuffix(mt, "+json") {
			continue
		}
		m, _ := content[mt].(map[string]any)
		s, _ := m["schema"].(map[string]any)
		return mustJSON(c.doc.Sample(s)), true
	}
	return "", false
}

// compare checks the echoed value of p against want.
func (c *client) compare(tb testing.TB, p map[string]any, want any, resp *response, what string) {
	tb.Helper()
	if !c.tg.Echo {
		return
	}
	var got map[string]any
	if err := json.Unmarshal(resp.body, &got); err != nil {
		tb.Errorf("%s: the answer is not an echoed JSON object: %q", resp.desc, resp.body)
		return
	}
	key := pname(p)
	field, ok := c.tg.Fields[key]
	if !ok {
		field = p["name"].(string)
	}
	v, present := got[field]
	if !present {
		tb.Errorf("%s: the echo has no %q for %s: %s", resp.desc, field, key, resp.body)
		return
	}
	if !reflect.DeepEqual(v, want) {
		tb.Errorf("%s: %s bound %s, %s: want %s", resp.desc, key, mustJSON(v), what, mustJSON(want))
	}
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprint(v)
	}
	return string(b)
}
