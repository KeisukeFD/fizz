package openapi

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
)

// applyParameters applies the Parameter options ps to op, whose inferred
// parameters are already built, in order. A parameter is identified by
// its name and location: it is merged into the one that has the same,
// else appended. Problems are recorded and skip the parameter that
// causes them.
func (g *Generator) applyParameters(op *Operation, path string, ps []*Parameter) {
	if len(ps) == 0 {
		return
	}
	for _, p := range ps {
		if p == nil {
			continue
		}
		switch {
		case p.Name == "":
			g.error(errors.New("Parameter: the parameter has no name"))
			continue
		case p.In == "":
			g.error(fmt.Errorf(`Parameter: %q has no "in" location`, p.Name))
			continue
		case p.In == "path" && !pathHasParam(path, p.Name):
			g.error(fmt.Errorf("Parameter: path parameter %q is not in the path template %q", p.Name, path))
			continue
		}
		// The parameter is copied so that no later merge or hook
		// writes into the value the option holds.
		c, err := p.Clone()
		if err != nil {
			g.error(fmt.Errorf("Parameter: %q: %w", p.Name, err))
			continue
		}
		if !g.mergeParameter(op, c) {
			op.Parameters = append(op.Parameters, &ParameterOrRef{Value: c})
		}
	}
	if g.sortParams {
		slices.SortStableFunc(op.Parameters, g.compareParams)
	}
}

// pathHasParam reports whether the path template has the parameter name.
func pathHasParam(path, name string) bool {
	for _, m := range paramsInPathRe.FindAllStringSubmatch(path, -1) {
		if m[1] == name {
			return true
		}
	}
	return false
}

// mergeParameter merges p into the parameter of op that has its name and
// location, and reports whether there is one. That is also the case when
// the match is a reference: it is left unchanged and an error is recorded.
func (g *Generator) mergeParameter(op *Operation, p *Parameter) bool {
	for _, pr := range op.Parameters {
		if pr == nil {
			continue
		}
		if pr.Ref != nil {
			// A reference is matched by the name and location of the
			// component it points to; it cannot be changed in place.
			if r := g.componentParameter(pr.Ref.Ref); r != nil && r.Name == p.Name && r.In == p.In {
				g.error(fmt.Errorf("Parameter: %q (%s) is a $ref (%s): it cannot be merged", p.Name, p.In, pr.Ref.Ref))
				return true
			}
			continue
		}
		dst := pr.Value
		if dst == nil || dst.Name != p.Name || dst.In != p.In {
			continue
		}
		if p.Description != "" {
			dst.Description = p.Description
		}
		dst.Required = dst.Required || p.Required
		dst.Deprecated = dst.Deprecated || p.Deprecated
		dst.AllowEmptyValue = dst.AllowEmptyValue || p.AllowEmptyValue
		dst.AllowReserved = dst.AllowReserved || p.AllowReserved
		if p.Schema != nil {
			dst.Schema = p.Schema
		}
		if p.Style != "" {
			dst.Style = p.Style
		}
		if p.Explode != nil {
			dst.Explode = p.Explode
		}
		if p.Example != nil {
			dst.Example = p.Example
		}
		if p.Examples != nil {
			dst.Examples = p.Examples
		}
		if p.Content != nil {
			dst.Content = p.Content
		}
		if len(p.Extensions) > 0 && dst.Extensions == nil {
			dst.Extensions = make(map[string]any, len(p.Extensions))
		}
		maps.Copy(dst.Extensions, p.Extensions)
		return true
	}
	return false
}

// componentParameter returns the parameter of components/parameters that
// ref points to, or nil.
func (g *Generator) componentParameter(ref string) *Parameter {
	name, ok := strings.CutPrefix(ref, "#/components/parameters/")
	if !ok || g.api.Components == nil {
		return nil
	}
	if pr := g.api.Components.Parameters[name]; pr != nil {
		return pr.Value
	}
	return nil
}
