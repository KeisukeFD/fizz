package kitchensink

import (
	"errors"
	"strings"
	"testing"

	"github.com/KeisukeFD/fizz"
)

func TestCaseCustomNew(t *testing.T) {
	var made *fizz.Fizz
	c := Case{
		Name: "custom-new",
		New: func() *fizz.Fizz {
			made = fizz.New()
			return made
		},
		Register: func(f *fizz.Fizz) {
			if f != made {
				panic("Register got a fizz that Case.New did not build")
			}
		},
	}
	if problems := runCase(c); len(problems) != 0 {
		t.Fatalf("problems: %v", problems)
	}
	if made == nil {
		t.Fatal("Case.New was not called")
	}
}

type warn struct{ msg string }

func (w warn) Error() string   { return w.msg }
func (w warn) IsWarning() bool { return true }

type notWarn struct{ msg string }

func (w notWarn) Error() string   { return w.msg }
func (w notWarn) IsWarning() bool { return false }

func TestCaseWarnings(t *testing.T) {
	errs := []error{
		errors.New("plain mentions alpha"),
		notWarn{"flagged false mentions beta"},
		warn{"real warning mentions gamma"},
	}
	for _, tc := range []struct {
		sub  string
		fail bool
	}{
		{"gamma", false},
		{"alpha", true}, // an ordinary error is not a warning
		{"beta", true},  // IsWarning() == false is not a warning
		{"nothing", true},
	} {
		rec := &recorder{}
		checkWarnings(rec, errs, []string{tc.sub})
		if got := len(rec.msgs) > 0; got != tc.fail {
			t.Errorf("%q: failed=%v, want %v (%v)", tc.sub, got, tc.fail, rec.msgs)
		}
		if tc.fail && !strings.Contains(rec.msgs[0], tc.sub) {
			t.Errorf("%q: message %q does not name the substring", tc.sub, rec.msgs[0])
		}
	}
}
