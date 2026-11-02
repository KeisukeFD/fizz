package kitchensink

import (
	"strings"
	"testing"

	"github.com/KeisukeFD/fizz"
)

const doc = `{"a":{"b/c":[1,{"d":"x"}],"n":null}}`

func failures(frags ...Fragment) []string {
	r := &recorder{}
	AssertFragments(r, []byte(doc), frags)
	return r.msgs
}

func TestFragmentAssert(t *testing.T) {
	tests := []struct {
		name    string
		frag    Fragment
		wantErr string // substring; "" means must pass
	}{
		{"match", Fragment{Pointer: "/a/b~1c/1/d", JSON: `"x"`}, ""},
		{"semantic object", Fragment{Pointer: "/a", JSON: `{"n":null,"b/c":[1,{"d":"x"}]}`}, ""},
		{"whole doc", Fragment{Pointer: "", JSON: doc}, ""},
		{"null value", Fragment{Pointer: "/a/n", JSON: `null`}, ""},
		{"wrong value", Fragment{Pointer: "/a/b~1c/1/d", JSON: `"y"`}, "want:"},
		{"wrong nested names pointer", Fragment{Pointer: "/a", JSON: `{"n":null,"b/c":[1,{"d":"z"}]}`}, `"/b~1c/1/d"`},
		{"missing pointer", Fragment{Pointer: "/a/zzz", JSON: `1`}, "not found"},
		{"bad index", Fragment{Pointer: "/a/b~1c/9", JSON: `1`}, "not found"},
		{"absent ok", Fragment{Pointer: "/a/zzz", Absent: true}, ""},
		{"absent but present", Fragment{Pointer: "/a/n", Absent: true}, "must be absent"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msgs := failures(tc.frag)
			switch {
			case tc.wantErr == "" && len(msgs) != 0:
				t.Errorf("unexpected failure: %v", msgs)
			case tc.wantErr != "" && !strings.Contains(strings.Join(msgs, "\n"), tc.wantErr):
				t.Errorf("want failure containing %q, got %v", tc.wantErr, msgs)
			}
		})
	}
}

func TestRunnerOutcomes(t *testing.T) {
	boom := func(*fizz.Fizz) { panic("boom") }
	ok := func(*fizz.Fizz) {}
	tests := []struct {
		name string
		c    Case
		want string // substring of the single problem; "" means none
	}{
		{"panic fails", Case{Register: boom}, "boom"},
		{"expected panic", Case{Register: boom, Panics: "boom"}, ""},
		{"missing expected panic", Case{Register: ok, Panics: "boom"}, "must panic"},
		{"missing error", Case{Register: ok, Errors: []string{"nope"}}, "no entry"},
		{"bad fragment", Case{Register: ok, Expect: []Fragment{{Pointer: "/openapi", JSON: `"9"`}}}, "/openapi"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := strings.Join(runCase(tc.c), "\n")
			if (tc.want == "") != (got == "") || !strings.Contains(got, tc.want) {
				t.Errorf("want problem %q, got %q", tc.want, got)
			}
		})
	}
}
