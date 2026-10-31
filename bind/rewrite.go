package bind

import (
	"fmt"
	"net/url"
	"strings"
)

// RewriteQuery returns rawQuery with the spaceDelimited and pipeDelimited
// arrays of p in the form that tonic reads: one pair per element. It
// reports whether it changed anything. The delimiter is applied after URL
// decoding, so "a%20b", "a+b" and "a b" are the elements a and b of a
// spaceDelimited array; empty elements are dropped, as tonic does. The
// other pairs are kept as they are.
//
// A field whose explode tag is false is read by tonic as one
// comma-separated value, so its elements are joined with a comma, and an
// element that contains a comma is an error.
func RewriteQuery(rawQuery string, p *Plan) (string, bool, error) {
	if p == nil || len(p.lists) == 0 || rawQuery == "" {
		return rawQuery, false, nil
	}
	var out []string
	changed := false
	pending := map[string][]string{} // tonic-unexploded fields: elements, by name
	slot := map[string]int{}         // and the position of their pair in out
	for pair := range strings.SplitSeq(rawQuery, "&") {
		k, v, _ := strings.Cut(pair, "=")
		key, err := url.QueryUnescape(k)
		var lf *listField
		if err == nil {
			lf = p.list(key)
		}
		if lf == nil {
			out = append(out, pair)
			continue
		}
		val, err := url.QueryUnescape(v)
		if err != nil {
			out = append(out, pair) // tonic drops it too
			continue
		}
		changed = true
		var elems []string
		for e := range strings.SplitSeq(val, lf.delim) {
			if e != "" {
				elems = append(elems, e)
			}
		}
		if lf.tonicExploded {
			for _, e := range elems {
				out = append(out, url.QueryEscape(key)+"="+url.QueryEscape(e))
			}
			continue
		}
		for _, e := range elems {
			if strings.Contains(e, ",") {
				return "", false, fmt.Errorf("bind: query parameter %q: element %q contains a comma, which its explode:\"false\" tag reads as a separator", key, e)
			}
		}
		if _, ok := slot[key]; !ok {
			slot[key] = len(out)
			out = append(out, "")
		}
		pending[key] = append(pending[key], elems...)
	}
	if !changed {
		return rawQuery, false, nil
	}
	for key, i := range slot {
		if elems := pending[key]; len(elems) > 0 {
			out[i] = url.QueryEscape(key) + "=" + url.QueryEscape(strings.Join(elems, ","))
		}
	}
	kept := out[:0]
	for _, e := range out {
		if e != "" {
			kept = append(kept, e)
		}
	}
	q := strings.Join(kept, "&")
	return q, q != rawQuery, nil
}

func (p *Plan) list(name string) *listField {
	for i := range p.lists {
		if p.lists[i].name == name {
			return &p.lists[i]
		}
	}
	return nil
}
