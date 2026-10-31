package openapi

import (
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ooErr struct {
	Message string `json:"message"`
}

func ooAdd(t *testing.T, g *Generator, info *OperationInfo) *Operation {
	t.Helper()
	if info.ID == "" {
		info.ID = "oo"
	}
	info.StatusCode = 200
	op, err := g.AddOperation("/p", "GET", "grp", nil, nil, info)
	require.NoError(t, err)
	return op
}

func ooErrors(g *Generator) []error {
	var out []error
	for _, err := range g.Errors() {
		var w *Warning
		if !errors.As(err, &w) {
			out = append(out, err)
		}
	}
	return out
}

func TestOperationTagsAppendDedupe(t *testing.T) {
	g := gen(t)
	op := ooAdd(t, g, &OperationInfo{Tags: []string{"admin", "", "grp", "admin", "x"}})
	assert.Equal(t, []string{"grp", "admin", "x"}, op.Tags)
	assert.Empty(t, g.api.Tags, "operation tags are not declared at the root")
}

func TestOperationExternalDocs(t *testing.T) {
	g := gen(t)
	op := ooAdd(t, g, &OperationInfo{ExternalDocs: &ExternalDocs{URL: "https://x", Description: "More"}})
	assert.Equal(t, &ExternalDocs{URL: "https://x", Description: "More"}, op.ExternalDocs)
	assert.Empty(t, g.Errors())

	g = gen(t)
	op = ooAdd(t, g, &OperationInfo{ExternalDocs: &ExternalDocs{Description: "More"}})
	assert.Nil(t, op.ExternalDocs)
	errs := ooErrors(g)
	require.Len(t, errs, 1)
	var oe *OperationError
	assert.True(t, errors.As(errs[0], &oe))
}

func TestOperationExtensions(t *testing.T) {
	g := gen(t)
	op := ooAdd(t, g, &OperationInfo{Extensions: []*ExtensionInfo{
		{Key: "x-rate", Value: 1},
		{Key: "x-rate", Value: 10},
		{Key: "rate", Value: 1},
		{Key: "x-oai-a", Value: 1},
		{Key: "x-", Value: 1},
		{Key: "x-nan", Value: math.NaN()},
		{Key: "x-codeSamples", Value: "other"},
		{Key: "x-internal", Value: true},
	}})
	assert.Equal(t, map[string]any{"x-rate": 10}, op.Extensions)
	errs := ooErrors(g)
	require.Len(t, errs, 6)
	assert.True(t, errors.Is(errs[0], ErrInvalidExtensionKey))
	assert.True(t, errors.Is(errs[1], ErrReservedExtension))
	assert.True(t, errors.Is(errs[2], ErrInvalidExtensionKey))
	assert.Contains(t, errs[3].Error(), "x-nan")
	assert.True(t, errors.Is(errs[4], ErrInvalidExtensionKey), "typed key: %v", errs[4])
	assert.True(t, errors.Is(errs[5], ErrInvalidExtensionKey))
}

func TestOperationTypedCodeSamplesWin(t *testing.T) {
	g := gen(t)
	cs := &XCodeSample{Lang: "sh", Source: "curl"}
	op := ooAdd(t, g, &OperationInfo{
		XCodeSamples: []*XCodeSample{cs},
		Extensions:   []*ExtensionInfo{{Key: "x-codeSamples", Value: "other"}},
	})
	assert.Equal(t, []*XCodeSample{cs}, op.XCodeSamples)
	assert.NotContains(t, op.Extensions, "x-codeSamples")
	assert.Len(t, ooErrors(g), 1)
}

func TestOperationCustomizersDeferredInOrder(t *testing.T) {
	g := gen(t)
	var seen []string
	op := ooAdd(t, g, &OperationInfo{
		Summary: "s",
		Customizers: []func(*Operation) error{
			func(op *Operation) error { seen = append(seen, "1:"+op.Summary); op.Summary = "c1"; return nil },
			nil,
			func(op *Operation) error { seen = append(seen, "2:"+op.Summary); op.Summary = "c2"; return nil },
			func(op *Operation) error { op.Summary = "lost"; op.ID = "other"; return errors.New("boom") },
			func(op *Operation) error { op.ID = "changed"; return nil },
		},
		Extensions: []*ExtensionInfo{{Key: "x-a", Value: 1}},
	})
	assert.Equal(t, []string{"1:s", "2:c1"}, seen)
	assert.Equal(t, "c2", op.Summary)
	assert.Equal(t, "oo", op.ID)
	assert.EqualValues(t, "1", op.Extensions["x-a"]) // a clone holds json.Number
	var boom, warn int
	for _, err := range g.Errors() {
		var oe *OperationError
		var w *Warning
		switch {
		case errors.As(err, &w):
			warn++
		case errors.As(err, &oe) && oe.ID == "oo" && strings.Contains(err.Error(), "boom"):
			boom++
		}
	}
	assert.Equal(t, 1, boom)
	assert.Equal(t, 1, warn)
}

func TestOperationCustomizerRunsBeforeHook(t *testing.T) {
	g := gen(t)
	g.SetHooks(Hooks{Operation: func(_ *OperationContext, op *Operation) error {
		assert.Equal(t, "c", op.Summary)
		op.Summary = "hook"
		return nil
	}})
	op := ooAdd(t, g, &OperationInfo{Customizers: []func(*Operation) error{
		func(op *Operation) error { op.Summary = "c"; return nil },
	}})
	assert.Equal(t, "hook", op.Summary)
}

func TestOperationResponseContents(t *testing.T) {
	g := gen(t)
	op := ooAdd(t, g, &OperationInfo{Responses: []*OperationResponse{{
		Code:        "404",
		Description: "Not found",
		Model:       ooErr{},
		Examples:    map[string]any{"a": 1},
		Contents: []*ResponseContent{
			{MediaType: "application/xml", Model: ooErr{}},
			{MediaType: "text/plain"},
			{MediaType: "text/plain", Model: ooErr{}},
			{MediaType: "application/json", Model: 1},
		},
		Extensions: []*ExtensionInfo{
			{Key: "x-r", Value: "v"},
			{Key: "r", Value: 1},
			{Key: "x-oas-r", Value: 1},
		},
	}}})
	r := op.Responses["404"].Value
	require.Len(t, r.Content, 3)
	assert.NotNil(t, r.Content["application/json"].Value.Schema.Ref, "the first of a media type is kept")
	assert.NotNil(t, r.Content["application/xml"].Value.Schema)
	assert.Nil(t, r.Content["text/plain"].Value.Schema, "the first, schemaless, is kept")
	assert.Contains(t, r.Content["application/json"].Value.Examples, "a")
	assert.Equal(t, map[string]any{"x-r": "v"}, r.Extensions)
	errs := ooErrors(g)
	require.Len(t, errs, 4)
	assert.True(t, errors.Is(errs[2], ErrInvalidExtensionKey))
	assert.True(t, errors.Is(errs[3], ErrReservedExtension))
}

func TestOperationResponseExampleAndExamplesKeepExamples(t *testing.T) {
	g := gen(t)
	op := ooAdd(t, g, &OperationInfo{Responses: []*OperationResponse{{
		Code: "400", Model: ooErr{}, Example: "e", Examples: map[string]any{"a": 1}, KeepExamples: true,
	}}})
	mt := op.Responses["400"].Value.Content["application/json"].Value
	assert.Nil(t, mt.Example)
	assert.Contains(t, mt.Examples, "a")
	errs := ooErrors(g)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "mutually exclusive")
}

func TestOperationResponseDuplicateCodeFailsOperation(t *testing.T) {
	g := gen(t)
	_, err := g.AddOperation("/p", "GET", "", nil, reflect.TypeFor[ooErr](), &OperationInfo{
		ID: "dup", StatusCode: 200,
		Responses: []*OperationResponse{{Code: "200", Contents: []*ResponseContent{{MediaType: "text/plain"}}}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "response with code 200 already exists")
}
