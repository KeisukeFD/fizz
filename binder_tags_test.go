package fizz

import (
	"testing"

	"github.com/loopfz/gadgeto/tonic"

	"github.com/wI2L/fizz/openapi"
)

func TestBinderTagsMatchTonic(t *testing.T) {
	b := openapi.BinderTags
	for _, c := range []struct{ name, got, want string }{
		{"Path", b.Path, tonic.PathTag},
		{"Query", b.Query, tonic.QueryTag},
		{"Header", b.Header, tonic.HeaderTag},
		{"Enum", b.Enum, tonic.EnumTag},
		{"Default", b.Default, tonic.DefaultTag},
		{"Validate", b.Validate, tonic.ValidationTag},
		{"Explode", b.Explode, tonic.ExplodeTag},
	} {
		if c.got != c.want {
			t.Errorf("%s = %q, tonic = %q", c.name, c.got, c.want)
		}
	}
}

func TestParseParamTagMatchesTonic(t *testing.T) {
	for _, tag := range []string{"q", "q,required", "q,default=5", "q, required ,default=x", "q,omitempty", "q,default=", ""} {
		t.Run(tag, func(t *testing.T) {
			name, _, _, err := openapi.ParseParamTag(tag)
			tname, terr := tonic.ParseTagKey(tag)
			if terr == nil && name != tname {
				t.Errorf("name = %q, tonic = %q", name, tname)
			}
			if (err == nil) != (terr == nil) || (err != nil && err.Error() != terr.Error()) {
				t.Errorf("err = %v, tonic = %v", err, terr)
			}
		})
	}
}

func TestGinPathTranslation(t *testing.T) {
	for in, want := range map[string]string{
		"/a/:id":      "/a/{id}",
		"/a/:id/b/:x": "/a/{id}/b/{x}",
		"/":           "/",
		"/a/*rest":    "/a/*rest", // unchanged until P4.3
		"/a/b":        "/a/b",
	} {
		if got := openAPIPath(in); got != want {
			t.Errorf("openAPIPath(%q) = %q, want %q", in, got, want)
		}
	}
}
