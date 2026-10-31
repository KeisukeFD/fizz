package specvalidate_test

import (
	"strings"
	"testing"

	"github.com/wI2L/fizz/internal/specvalidate"
)

const (
	pathID   = `{"name":"id","in":"path","required":true,"schema":{"type":"string"}}`
	queryQ   = `{"name":"q","in":"query","schema":{"type":"string"}}`
	qsParam  = `{"name":"qs","in":"querystring","content":{"application/x-www-form-urlencoded":{}}}`
	qsParam2 = `{"name":"qs2","in":"querystring","content":{"application/json":{}}}`
)

// opWith returns an operation body with the given parameters and extra fields.
func opWith(params string, extra string) string {
	s := `{` + okResp
	if params != "" {
		s += `,"parameters":[` + params + `]`
	}
	if extra != "" {
		s += `,` + extra
	}
	return s + `}`
}

// semCase is one lint scenario. want holds substrings that must each match
// a distinct problem "pointer: message"; an empty want means no problems.
type semCase struct {
	name string
	doc  []byte
	want []string
}

func TestLintSemantics(t *testing.T) {
	comps := func(params string) string { return `{"parameters":{` + params + `}}` }
	tests := []semCase{
		// (a) path templating
		{"a ok: operation param", doc(`{"/x/{id}":{"get":`+opWith(pathID, "")+`}}`, ""), nil},
		{"a ok: path item param", doc(`{"/x/{id}":{"parameters":[`+pathID+`],"get":`+opWith("", "")+`}}`, ""), nil},
		{"a ok: operation overrides path item", doc(`{"/x/{id}":{"parameters":[`+pathID+`],"get":`+opWith(`{"name":"id","in":"path","required":true,"content":{"text/plain":{}}}`, "")+`}}`, ""), nil},
		{"a ok: empty path item", doc(`{"/x/{id}":{}}`, ""), nil},
		{"a ok: $ref to components", doc(`{"/x/{id}":{"get":`+opWith(`{"$ref":"#/components/parameters/ID"}`, "")+`}}`, comps(`"ID":`+pathID)), nil},
		{"a ok: external $ref skipped", doc(`{"/x/{id}":{"get":`+opWith(`{"$ref":"other.json#/p"}`, "")+`}}`, ""), nil},
		{"a bad: no param", doc(`{"/x/{id}":{"get":`+opWith("", "")+`}}`, ""),
			[]string{"/paths/~1x~1{id}/get: path template expression {id} has no path parameter"}},
		{"a bad: only one operation has it", doc(`{"/x/{id}":{"get":`+opWith(pathID, "")+`,"put":`+opWith("", "")+`}}`, ""),
			[]string{"/paths/~1x~1{id}/put: path template expression {id}"}},
		{"a bad: $ref to components lacks it", doc(`{"/x/{id}":{"get":`+opWith(`{"$ref":"#/components/parameters/Q"}`, "")+`}}`, comps(`"Q":`+queryQ)),
			[]string{"{id} has no path parameter"}},
		{"a bad: path param not in template", doc(`{"/x":{"get":`+opWith(pathID, "")+`}}`, ""),
			[]string{"/paths/~1x/get/parameters/0: path parameter \"id\" does not appear"}},
		{"a bad: path item param not in template", doc(`{"/x":{"parameters":[`+pathID+`],"get":`+opWith("", "")+`}}`, ""),
			[]string{"/paths/~1x/parameters/0: path parameter \"id\" does not appear"}},
		{"a bad: path param not required", doc(`{"/x/{id}":{"get":`+opWith(`{"name":"id","in":"path","content":{"text/plain":{}}}`, "")+`}}`, ""),
			[]string{"/paths/~1x~1{id}/get/parameters/0: path parameter \"id\" must have required: true"}},
		{"a bad: expression twice", doc(`{"/x/{id}/{id}":{"get":`+opWith(pathID, "")+`}}`, ""),
			[]string{"appears more than once"}},

		// (b) unique (name, in)
		{"b ok: same name, different in", doc(op(opWith(queryQ+`,{"name":"q","in":"cookie","schema":{}}`, "")), ""), nil},
		{"b bad: duplicate in operation", doc(op(opWith(queryQ+`,`+queryQ, "")), ""),
			[]string{"/paths/~1x/get/parameters/1: duplicate parameter"}},
		{"b bad: duplicate in path item", doc(`{"/x":{"parameters":[`+queryQ+`,`+queryQ+`],"get":`+opWith("", "")+`}}`, ""),
			[]string{"/paths/~1x/parameters/1: duplicate parameter"}},
		{"b bad: duplicate through $ref", doc(op(opWith(queryQ+`,{"$ref":"#/components/parameters/Q"}`, "")), comps(`"Q":`+queryQ)),
			[]string{"/paths/~1x/get/parameters/1: duplicate parameter"}},
		{"b bad: header names are case-insensitive", doc(op(opWith(`{"name":"X-A","in":"header","schema":{}},{"name":"x-a","in":"header","schema":{}}`, "")), ""),
			[]string{"duplicate parameter"}},

		// (c) operationId
		{"c ok: distinct ids", doc(`{"/x":{"get":`+opWith("", `"operationId":"a"`)+`,"put":`+opWith("", `"operationId":"b"`)+`}}`, ""), nil},
		{"c bad: duplicate", doc(`{"/x":{"get":`+opWith("", `"operationId":"a"`)+`,"put":`+opWith("", `"operationId":"a"`)+`}}`, ""),
			[]string{"/paths/~1x/put/operationId: operationId \"a\" is already used at /paths/~1x/get/operationId"}},
		{"c bad: callback reuses an id", doc(op(opWith("", `"operationId":"a","callbacks":{"cb":{"{$request.body#/u}":{"post":`+opWith("", `"operationId":"a"`)+`}}}`)), ""),
			[]string{"/paths/~1x/get/callbacks/cb/{$request.body#~1u}/post/operationId: operationId \"a\""}},
		{"c bad: webhook reuses an id", func() []byte {
			d := doc(op(opWith("", `"operationId":"a"`)), "")
			return []byte(strings.Replace(string(d), `"paths"`, `"webhooks":{"w":{"post":`+opWith("", `"operationId":"a"`)+`}},"paths"`, 1))
		}(), []string{"/webhooks/w/post/operationId: operationId \"a\""}},

		// (d) schema xor content, one content entry
		{"d ok: content", doc(op(opWith(`{"name":"q","in":"query","content":{"application/json":{}}}`, "")), ""), nil},
		{"d bad: both", doc(op(opWith(`{"name":"q","in":"query","schema":{},"content":{"application/json":{}}}`, "")), ""),
			[]string{"/paths/~1x/get/parameters/0: parameter \"q\" has both schema and content"}},
		{"d bad: neither", doc(op(opWith(`{"name":"q","in":"query"}`, "")), ""),
			[]string{"has neither schema nor content"}},
		{"d bad: two content entries", doc(op(opWith(`{"name":"q","in":"query","content":{"a/b":{},"c/d":{}}}`, "")), ""),
			[]string{"/paths/~1x/get/parameters/0/content: parameter \"q\" content has 2 entries"}},
		{"d bad: component parameter", doc("", comps(`"Q":{"name":"q","in":"query","schema":{},"content":{"a/b":{}}}`)),
			[]string{"/components/parameters/Q: parameter \"q\" has both"}},

		// (e) in
		{"e bad: invalid in", doc(op(opWith(`{"name":"q","in":"body","schema":{}}`, "")), ""),
			[]string{"/paths/~1x/get/parameters/0/in: invalid parameter location \"body\""}},

		// (f) reserved headers
		{"f ok: other header", doc(op(opWith(`{"name":"X-Token","in":"header","schema":{}}`, "")), ""), nil},
		{"f ok: reserved name as query", doc(op(opWith(`{"name":"Accept","in":"query","schema":{}}`, "")), ""), nil},
		{"f bad: Accept", doc(op(opWith(`{"name":"Accept","in":"header","schema":{}}`, "")), ""),
			[]string{"/paths/~1x/get/parameters/0/name: header parameter \"Accept\""}},
		{"f bad: content-type lower case", doc(op(opWith(`{"name":"content-type","in":"header","schema":{}}`, "")), ""),
			[]string{"header parameter \"content-type\""}},
		{"f bad: AUTHORIZATION", doc(op(opWith(`{"name":"AUTHORIZATION","in":"header","schema":{}}`, "")), ""),
			[]string{"header parameter \"AUTHORIZATION\""}},

		// (g) querystring
		{"g ok: one querystring", doc(op(opWith(qsParam, "")), ""), nil},
		{"g bad: two querystring", doc(op(opWith(qsParam+`,`+qsParam2, "")), ""),
			[]string{"/paths/~1x/get/parameters/1: querystring parameter \"qs2\": an in: querystring parameter MUST NOT appear more than once"}},
		{"g bad: querystring without content", doc(op(opWith(`{"name":"qs","in":"querystring","schema":{}}`, "")), ""),
			[]string{"querystring parameter \"qs\" must use content"}},
		{"g bad: with query in the operation", doc(op(opWith(qsParam+`,`+queryQ, "")), ""),
			[]string{"/paths/~1x/get/parameters/1: query parameter \"q\" cannot be combined"}},
		{"g bad: query in the path item", doc(`{"/x":{"parameters":[`+queryQ+`],"get":`+opWith(qsParam, "")+`}}`, ""),
			[]string{"/paths/~1x/parameters/0: query parameter \"q\" cannot be combined"}},
		{"g bad: both in the path item, reported once", doc(`{"/x":{"parameters":[`+queryQ+`,`+qsParam+`],"get":`+opWith("", "")+`,"put":`+opWith("", "")+`}}`, ""),
			[]string{"/paths/~1x/parameters/0: query parameter \"q\" cannot be combined"}},

		// (h) colliding templates
		{"h ok: different shapes", doc(`{"/pets/{id}":{},"/pets/mine":{},"/{e}/me":{}}`, ""), nil},
		{"h bad: identical templates", doc(`{"/pets/{petId}":{},"/pets/{name}":{}}`, ""),
			[]string{"/paths/~1pets~1{petId}: path template is identical to \"/pets/{name}\""}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// The scenarios are about the lint, not the meta-schema; the
			// rules (d) and (e) overlap it on purpose.
			got := specvalidate.LintSemantics(tc.doc)
			if len(got) != len(tc.want) {
				t.Fatalf("got %d problems, want %d:\n%v", len(got), len(tc.want), got)
			}
			for i, w := range tc.want {
				if !strings.Contains(got[i].String(), w) {
					t.Errorf("problem %d = %q, want it to contain %q", i, got[i], w)
				}
			}
		})
	}
}

func TestLintSemanticsInvalidJSON(t *testing.T) {
	if got := specvalidate.LintSemantics([]byte(`{`)); len(got) != 1 {
		t.Fatalf("got %v, want one problem", got)
	}
}
