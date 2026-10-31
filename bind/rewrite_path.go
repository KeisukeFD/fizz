package bind

import (
	"fmt"
	"strings"
)

// RewritePath returns the value of the path parameter name in the form
// that tonic binds, and whether name is a parameter of p that needs one.
// A label parameter ".v" becomes "v", and a matrix parameter ";name=v"
// (or ";name" for an empty value) becomes "v". A value that does not have
// the form of its style is an error.
func RewritePath(name, value string, p *Plan) (string, bool, error) {
	if p == nil {
		return value, false, nil
	}
	for i := range p.paths {
		pf := &p.paths[i]
		if pf.name != name {
			continue
		}
		if !pf.matrix {
			v, ok := strings.CutPrefix(value, ".")
			if !ok {
				return "", false, fmt.Errorf("bind: path parameter %q: label style wants a leading \".\", got %q", name, value)
			}
			return v, true, nil
		}
		if rest, ok := strings.CutPrefix(value, ";"+name); ok {
			if rest == "" {
				return "", true, nil
			}
			if v, ok := strings.CutPrefix(rest, "="); ok {
				return v, true, nil
			}
		}
		return "", false, fmt.Errorf("bind: path parameter %q: matrix style wants \";%s=value\", got %q", name, name, value)
	}
	return value, false, nil
}
