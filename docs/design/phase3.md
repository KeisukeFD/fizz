# Phase 3 design: struct fields and type→schema recursion

- Status: accepted (Opus, before P3.1a/P3.2)
- Scope: PLAN-v2 P3.1a, P3.1b and P3.2. P3.3–P3.6 build on these entry points.
- Sequencing: run serially as P3.1a → P3.1b → P3.2. The plan allows P3.2 in parallel, but the per-case goldens and the done ledger are shared, so parallel runs would conflict.

## 1. P3.1a: `typeFields` (openapi/typefields.go)

This is a port of the field selection in `encoding/json` v1 (`typeFields`, `dominantField` in src/encoding/json/encode.go). gin serializes with v1 semantics, so v1 is the reference.

```go
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

// typeFields returns the JSON members of struct type t, as the media type's
// tag (json, xml...) selects them, in encoding/json order. Cached per
// (t, tag) on the Generator.
func (g *Generator) typeFields(t reflect.Type, tag string) []field

// fieldsOf iterates typeFields(t, tag).
func (g *Generator) fieldsOf(t reflect.Type, tag string) iter.Seq[field]
```

**Algorithm (as in v1):**
- Walk embedded structs breadth-first by depth. A `visited map[reflect.Type]bool` is shared across all levels, so a type reached again at a deeper level is skipped. This fixes the embedded-cycle stack overflow in `flattenStructSchema`.
- Per field:
  - Unexported non-embedded: skip.
  - Embedded field of an unexported non-struct type: skip.
  - Embedded pointer to an unexported struct type: **not** skipped. v1 selects its fields and encodes them when the pointer is non-nil. Only decoding into it fails. (Corrected at P3.1a review.)
  - Tag `-` (exactly): skip. Tag `-,`: the name is `-`.
  - Invalid tag name: fall back to the Go name (v1 `isValidTag`).
  - Options are parsed with `strings.SplitSeq`. Recognised: `omitempty`, `omitzero`, and `string`, which is recorded only when the leaf kind is bool, int*, uint*, float* or string (v1 rule).
  - An embedded struct (or pointer to struct) **without** a tag name is descended into at depth+1. **With** a tag name it becomes a regular named field.
  - An embedded non-struct type is a field named by its type name.
- Name conflicts follow `dominantField`. Among the fields of the shallowest depth for a name:
  - exactly one → it wins;
  - several → the one tagged field wins if there is exactly one, else the name is dropped.
- Output order: by index sequence (declaration order), the same as v1.

**Integration (openapi/fields.go):**
- `flattenStructSchema(t, parent, schema)` keeps its signature, so schema.go is untouched.
- Its body iterates `g.fieldsOf(t, mediaTag)` and calls `newSchemaFromStructField(f.sf, required, f.name, t)` for each field. A new parameter or a small wrapper passes `f.quoted`.
- **`,string`:** the property schema becomes `{"type":"string"}`, and P3.6 annotates the format. Exact output follows the `fields/*` case fragments.
- `fieldNameFromTag` is deleted.
- The "recursive embedding" `FieldError` is no longer needed. The visited set handles the cycle, and the cycle cases expect no error unless a case fragment says otherwise.

**Red first:** un-skip the P3.1a cases in cases_fields.go, and drop `crashes(...)` from the embedded-cycle cases P3.1a owns. Also un-skip K06 `TestDifferentialJSONKeys`. The `FCycA` sample can be added to it once it no longer overflows.

## 2. P3.1b: param walker (openapi/params.go)

- `buildParamsRecursive` is replaced by a walk over `g.fieldsOf(t, …)`, with **tonic location semantics**.
- tonic (handler.go:176-191) recurses into **every** anonymous struct field, tagged or not. It binds by location tag (`path`, `query`, `header`), not by JSON name.
- So the walker uses typeFields' visited set and depth handling, but:
  - Location fields: the param name comes from the location tag via `ParseParamTag`, not from the JSON name. A field with `json:"-"` and a `query:"q"` tag **is** a param. This fixes the `json:"-"` → `""` leak.
  - Body fields (no location tag) follow plain JSON `typeFields` rules.
- Embedded cycles are cut by the visited set, which fixes the old :523 recursion.

## 3. P3.2: unified type→schema recursion (openapi/schema.go only)

```go
// schemaFor returns the schema of type t at a use site. It is the single
// recursion for top-level types, struct fields, slice/array elements and
// map values.
func (g *Generator) schemaFor(t reflect.Type) *Schema
```

**Steps:**
1. Strip pointers, recording `nullable` (any pointer level). Then apply the `Nullable` interface check exactly as `newSchemaFromType` does today. P3.5 owns nullability changes, so P3.2 keeps today's conditions.
2. Resolve in this order: overrides (`g.dataTypes`, `OverrideDataType`) → interfaces (`Typer`, `DataType`, `Exampler`, as `g.datatype` does today) → known types (`time.Time`, uuid…) → kind switch.
3. Kind switch:
   - struct → `newSchemaFromStruct` (component + `$ref`, unchanged);
   - slice/array → `{type: array, items: schemaFor(elem)}`, plus fixed `minItems`/`maxItems` for arrays;
   - map → `{type: object, additionalProperties: schemaFor(elem)}`. Non-string keys keep today's error, since the map-key policy is P3.3;
   - scalar → datatype type/format, with `[]byte` → base64 as today.
4. Nullable → `MakeNullable()`, as today.

**Replacements:**
- `newSchemaFromType` becomes a thin wrapper over `schemaFor`, so fields.go and params.go call sites are untouched.
- `buildSchemaRecursive` is deleted.
- Elements and map values now get known types, overrides and interfaces. This fixes `elem/datatype` and `elem/override-datatype`.

**Cycles through named non-struct types** (`type L []L`, `type M map[string]M`, `type A []B; type B map[string]A`):
- The Generator keeps an ordered stack of the named composites being built (kinds slice, array or map with `t.Name() != ""`) and a `promoted` set.
- On entry a type is pushed. If `schemaFor` reaches a type already on the stack, every named composite from that type up to the top of the stack (all members of the cycle) is promoted, and `{"$ref": "#/components/schemas/<typeName(t)>"}` is returned.
- When the call for a promoted type finishes, its schema is registered under `typeName(t)` and it returns the `$ref`. A promoted type met later anywhere returns the `$ref` directly. Non-promoted types are returned inline, as today.
- Rule 1, struct boundary: promotion applies only to cycles made purely of named composites. A named struct is already a component and cuts the cycle with a `$ref`, so `newSchemaFromStruct` saves the stack, flattens the struct with an empty one, and restores it. A cycle through a struct (`Z map[string]*Y`, `Y{K *Z}`) therefore promotes nothing.
- Rule 2, entry independence: all cycle members are promoted, so the component set and schemas are the same whichever type is entered first, and do not depend on route registration order.

**Element nullability:**
- Today `buildSchemaRecursive` drops element pointers, so `[]*T` has a non-nullable `items`.
- Under the unified path, `[]*string` gets `items: {type: [string, "null"]}`. That is consistent with D6 and with what encoding/json emits for nil elements.
- If an existing case fragment or golden asserts the old shape, P3.2 follows the case. Otherwise the change is accepted, and listed in the golden review.

**Red first:** un-skip `elem/datatype`, `elem/override-datatype`, and the `cycle/*` cases owned by P3.2. Drop `crashes(...)` from the slice-of-self and map-of-self cases.

## 4. Common rules

- ADR §4 "Phase 2 intermediate states" still applies:
  - each task is one commit and stays green;
  - every golden diff (per-case goldens, the whole kitchen sink, spec.json) is classified and reviewed by Opus before commit;
  - a case that passes early is relabelled to the current task;
  - regressions are fixed, never relabelled.
- No behaviour beyond the task. Validation tags are P4.1, default/enum/example conversion is P3.6, use-site annotations without component mutation are P3.4, and nullability policy is P3.5.
