// Package vtag parses go-playground/validator v9 `validate` struct tags.
//
// The grammar mirrors validator.v9 parseFieldTagsRecursive: the tag is split
// on ',' into items; an item containing '|' is an OR group; a rule is
// "name" or "name=param". Whitespace is not trimmed, exactly as in v9. In a
// param, the sequences 0x2C and 0x7C stand for ',' and '|'.
//
// Unlike v9, Parse does not know which rule names exist, so unknown names are
// accepted; v9's structonly, nostructlevel and isdefault are ordinary rules
// here. Malformed tags yield an error instead of a panic.
package vtag

import (
	"errors"
	"fmt"
	"strings"
)

// Rule is a single validation rule such as "min=1".
type Rule struct {
	Name  string // "min", "required", "oneof", ...
	Param string // text after the first '=', with 0x2C and 0x7C unescaped; "" if none
}

// Group is an OR group: the rule passes if any alternative passes ("a|b").
type Group []Rule

// Level is the set of rules applying to one nesting depth of a value.
type Level struct {
	OmitEmpty bool    // "omitempty" at this level
	Rules     []Group // AND of OR groups, in tag order (omitempty, dive, keys, endkeys excluded)
	Keys      *Level  // map-key rules from "keys ... endkeys"; only on a level entered by dive
	Dive      *Level  // element/value rules after "dive"; nil if no dive
}

// Tag is a parsed validate tag.
type Tag struct {
	Skip bool  // tag is exactly "-"
	Root Level // field-level rules
}

// Has reports whether any group of l (not its Keys or Dive levels) contains a
// rule named name. It is safe to call on a nil Level.
func (l *Level) Has(name string) bool {
	if l == nil {
		return false
	}
	for _, g := range l.Rules {
		for _, r := range g {
			if r.Name == name {
				return true
			}
		}
	}
	return false
}

var paramUnescaper = strings.NewReplacer("0x2C", ",", "0x7C", "|")

// Parse parses a validate tag. The empty tag yields an empty Tag and "-"
// yields Skip.
func Parse(tag string) (*Tag, error) {
	switch tag {
	case "":
		return &Tag{}, nil
	case "-":
		return &Tag{Skip: true}, nil
	}

	t := &Tag{}
	cur := &t.Root  // level receiving rules
	var keys *Level // non-nil while inside keys ... endkeys
	prevDive := false

	for item := range strings.SplitSeq(tag, ",") {
		wasDive := prevDive
		prevDive = false

		if item == "" {
			return nil, errors.New("vtag: empty rule")
		}
		target := cur
		if keys != nil {
			target = keys
		}

		if strings.Contains(item, "|") {
			var group Group
			for alt := range strings.SplitSeq(item, "|") {
				rule, err := parseRule(alt)
				if err != nil {
					return nil, err
				}
				switch rule.Name {
				case "omitempty", "dive", "keys", "endkeys":
					return nil, fmt.Errorf("vtag: %q cannot be used in an OR group (%q)", rule.Name, item)
				}
				group = append(group, rule)
			}
			target.Rules = append(target.Rules, group)
			continue
		}

		switch item {
		case "omitempty":
			target.OmitEmpty = true
		case "dive":
			if keys != nil {
				return nil, errors.New("vtag: dive is not allowed inside keys")
			}
			cur.Dive = &Level{}
			cur = cur.Dive
			prevDive = true
		case "keys":
			if keys != nil {
				return nil, errors.New("vtag: keys is not allowed inside keys")
			}
			if !wasDive {
				return nil, errors.New("vtag: keys must immediately follow dive")
			}
			cur.Keys = &Level{}
			keys = cur.Keys
		case "endkeys":
			if keys == nil {
				return nil, errors.New("vtag: endkeys without keys")
			}
			keys = nil
		default:
			rule, err := parseRule(item)
			if err != nil {
				return nil, err
			}
			target.Rules = append(target.Rules, Group{rule})
		}
	}
	if keys != nil {
		return nil, errors.New("vtag: keys without endkeys")
	}
	return t, nil
}

func parseRule(s string) (Rule, error) {
	name, param, _ := strings.Cut(s, "=")
	if name == "" {
		return Rule{}, fmt.Errorf("vtag: empty rule name in %q", s)
	}
	return Rule{Name: name, Param: paramUnescaper.Replace(param)}, nil
}
