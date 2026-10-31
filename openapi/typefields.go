package openapi

import (
	"cmp"
	"iter"
	"reflect"
	"slices"
	"strings"
	"unicode"
)

// field is one JSON member of a struct type, after embedding promotion.
type field struct {
	name      string              // JSON member name
	tagged    bool                // name came from the tag
	index     []int               // path from the root struct (reflect.Value.FieldByIndex)
	typ       reflect.Type        // the field's type, as declared (not dereferenced)
	sf        reflect.StructField // the leaf struct field (tags, docs)
	omitEmpty bool
	omitZero  bool
	quoted    bool // ",string" option
}

// typeFieldsKey identifies a cached typeFields result.
type typeFieldsKey struct {
	t   reflect.Type
	tag string
}

// typeFields returns the JSON members of struct type t, as the media type's
// tag (json, xml...) selects them, in encoding/json order. The result is
// cached per (t, tag) on the Generator and must not be modified.
func (g *Generator) typeFields(t reflect.Type, tag string) []field {
	key := typeFieldsKey{t, tag}
	if fs, ok := g.fieldCache[key]; ok {
		return fs
	}
	fs := computeTypeFields(t, tag)
	if g.fieldCache == nil {
		g.fieldCache = make(map[typeFieldsKey][]field)
	}
	g.fieldCache[key] = fs
	return fs
}

// fieldsOf iterates typeFields(t, tag).
func (g *Generator) fieldsOf(t reflect.Type, tag string) iter.Seq[field] {
	return func(yield func(field) bool) {
		for _, f := range g.typeFields(t, tag) {
			if !yield(f) {
				return
			}
		}
	}
}

// computeTypeFields is a port of typeFields in encoding/json (v1).
func computeTypeFields(t reflect.Type, tagName string) []field {
	// Anonymous fields to explore at the current level and the next.
	var current []field
	next := []field{{typ: t}}

	// Count of queued names for current level and the next.
	var count, nextCount map[reflect.Type]int

	// Types already visited at an earlier level.
	visited := map[reflect.Type]bool{}

	// Fields found.
	var fields []field

	for len(next) > 0 {
		current, next = next, current[:0]
		count, nextCount = nextCount, map[reflect.Type]int{}

		for _, f := range current {
			if visited[f.typ] {
				continue
			}
			visited[f.typ] = true

			// Scan f.typ for fields to include.
			for i := range f.typ.NumField() {
				sf := f.typ.Field(i)
				if sf.Anonymous {
					et := sf.Type
					if et.Kind() == reflect.Pointer {
						et = et.Elem()
					}
					if !sf.IsExported() && et.Kind() != reflect.Struct {
						// Ignore embedded fields of unexported non-struct types.
						continue
					}
					// Do not ignore embedded fields of unexported struct types
					// since they may have exported fields.
				} else if !sf.IsExported() {
					// Ignore unexported non-embedded fields.
					continue
				}
				tag := sf.Tag.Get(tagName)
				if tag == "-" {
					continue
				}
				name, opts, _ := strings.Cut(tag, ",")
				if !isValidTag(name) {
					name = ""
				}
				index := make([]int, len(f.index)+1)
				copy(index, f.index)
				index[len(f.index)] = i

				ft := sf.Type
				if ft.Name() == "" && ft.Kind() == reflect.Pointer {
					// Follow pointer.
					ft = ft.Elem()
				}

				var omitEmpty, omitZero, quoted bool
				for o := range strings.SplitSeq(opts, ",") {
					switch o {
					case "omitempty":
						omitEmpty = true
					case "omitzero":
						omitZero = true
					case "string":
						// Only strings, floats, integers, and booleans can be quoted.
						switch ft.Kind() {
						case reflect.Bool,
							reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
							reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
							reflect.Float32, reflect.Float64,
							reflect.String:
							quoted = true
						}
					}
				}

				// Record found field and index sequence.
				if name != "" || !sf.Anonymous || ft.Kind() != reflect.Struct {
					tagged := name != ""
					if name == "" {
						name = sf.Name
					}
					fld := field{
						name:      name,
						tagged:    tagged,
						index:     index,
						typ:       sf.Type,
						sf:        sf,
						omitEmpty: omitEmpty,
						omitZero:  omitZero,
						quoted:    quoted,
					}
					fields = append(fields, fld)
					if count[f.typ] > 1 {
						// If there were multiple instances, add a second,
						// so that the annihilation code will see a duplicate.
						fields = append(fields, fld)
					}
					continue
				}

				// Record new anonymous struct to explore in next round.
				nextCount[ft]++
				if nextCount[ft] == 1 {
					next = append(next, field{name: ft.Name(), index: index, typ: ft})
				}
			}
		}
	}

	slices.SortFunc(fields, func(a, b field) int {
		// Sort by name, breaking ties with depth, then with
		// "name came from tag", then with index sequence.
		if c := strings.Compare(a.name, b.name); c != 0 {
			return c
		}
		if c := cmp.Compare(len(a.index), len(b.index)); c != 0 {
			return c
		}
		if a.tagged != b.tagged {
			if a.tagged {
				return -1
			}
			return +1
		}
		return slices.Compare(a.index, b.index)
	})

	// Delete all fields that are hidden by the Go rules for embedded fields,
	// except that fields with tags are promoted.
	out := fields[:0]
	for advance, i := 0, 0; i < len(fields); i += advance {
		// One iteration per name.
		fi := fields[i]
		for advance = 1; i+advance < len(fields); advance++ {
			if fields[i+advance].name != fi.name {
				break
			}
		}
		if advance == 1 { // Only one field with this name
			out = append(out, fi)
			continue
		}
		if dominant, ok := dominantField(fields[i : i+advance]); ok {
			out = append(out, dominant)
		}
	}
	fields = out
	slices.SortFunc(fields, func(a, b field) int {
		return slices.Compare(a.index, b.index)
	})
	return fields
}

// dominantField looks through the fields, all of which are known to have the
// same name, to find the single field that dominates the others using Go's
// embedding rules, modified by the presence of tags. If there are multiple
// top-level fields, the boolean will be false: this condition is an error in
// Go and we skip all the fields.
func dominantField(fields []field) (field, bool) {
	// The fields are sorted in increasing index-length order, then by
	// presence of tag, so the first field is the dominant one. Only the error
	// case remains: two fields at the top level, both tagged or neither.
	if len(fields) > 1 && len(fields[0].index) == len(fields[1].index) && fields[0].tagged == fields[1].tagged {
		return field{}, false
	}
	return fields[0], true
}

// isValidTag reports whether s is a valid tag name, as encoding/json does.
func isValidTag(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		switch {
		case strings.ContainsRune("!#$%&()*+-./:;<=>?@[]^_{|}~ ", c):
			// Backslash and quote chars are reserved, but
			// otherwise any punctuation chars are allowed
			// in a tag name.
		case !unicode.IsLetter(c) && !unicode.IsDigit(c):
			return false
		}
	}
	return true
}
