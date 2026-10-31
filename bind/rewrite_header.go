package bind

import "net/http"

// RewriteHeader returns h with the headers that p plans for a default
// set: a header that is absent takes the value of its default tag, which is
// what the generator documents. Tonic reads an absent header as one empty
// value and never applies the default tag, so the request is what changes.
// It reports whether it changed anything. h is never modified: when a
// header is added, the result is a copy.
//
// A header that is present, even empty, is left as it is. A field whose
// header tag also has the ",default=" option is not planned: tonic applies
// that option itself, to an absent or empty header.
func RewriteHeader(h http.Header, p *Plan) (http.Header, bool) {
	if p == nil || len(p.headers) == 0 {
		return h, false
	}
	out, changed := h, false
	for i := range p.headers {
		hf := &p.headers[i]
		key := http.CanonicalHeaderKey(hf.name)
		if _, ok := out[key]; ok {
			continue
		}
		if !changed {
			out, changed = h.Clone(), true
			if out == nil {
				out = http.Header{}
			}
		}
		out[key] = []string{hf.def}
	}
	return out, changed
}
