package fizz

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/wI2L/fizz/bind"
	"github.com/wI2L/fizz/openapi"
)

// ctxBindStateKey is the key of the request context value that holds the
// *bindState of a route.
type ctxBindStateKey struct{}

// bindState is the per-request binding state, put in the request context
// by the handler that Handle wraps around the tonic handler. The context,
// not gin's c.Set, carries it: it is not visible to the keys of the
// request.
type bindState struct {
	plan *bind.Plan // never nil, never empty
	ran  bool       // fizz's hook step has run for this request
}

// BindHook returns a tonic bind hook that calls next, then binds what tonic
// does not understand: the deepObject query parameters of the route (those
// tagged style:"deepObject") and its cookie parameters (scalar fields with
// a cookie tag). A cookie that is absent takes the field's default (the
// ",default=" option of the cookie tag, else the default tag); one that is
// present, even empty, keeps its raw value, which is not URL-decoded. An
// absent cookie with the ",required" option and no default is a binding
// error, as is an empty one; validate:"required" is also checked by
// tonic. If next fails, its error is returned unchanged. A
// binding error is returned as is: tonic wraps it in a tonic.BindError and
// passes it to the error hook, which answers 400 by default. A nil next
// stands for tonic.DefaultBindingHook.
//
// New installs it around the current hook (see InstallBindHook), so
// calling BindHook yourself is only needed when you set a hook after New:
//
//	tonic.SetBindHook(myHook)                // replaces fizz's step: deepObject and cookies stop binding
//	tonic.SetBindHook(fizz.BindHook(myHook)) // composes with it
//
// The hook does nothing for a request that is not served by a route
// registered with fizz, and it binds at most once per request, so
// wrapping it twice only costs a call.
//
// Every hook that BindHook returns is recognized by InstallBindHook.
//
//go:noinline
func BindHook(next tonic.BindHook) tonic.BindHook {
	if next == nil {
		next = tonic.DefaultBindingHook
	}
	// Keep this the only function literal that builds a fizz hook:
	// InstallBindHook identifies fizz's hooks by its code pointer.
	return func(c *gin.Context, i any) error {
		if err := next(c, i); err != nil {
			return err
		}
		if c.Request == nil {
			return nil
		}
		st, _ := c.Request.Context().Value(ctxBindStateKey{}).(*bindState)
		if st == nil || st.ran {
			return nil
		}
		st.ran = true
		if err := bind.Query(c.Request, i, st.plan); err != nil {
			return err
		}
		return bind.Cookies(c.Request, i, st.plan)
	}
}

var (
	installMu sync.Mutex
	// fizzHookPC is the code pointer shared by every hook BindHook returns.
	fizzHookPC = sync.OnceValue(func() uintptr {
		return reflect.ValueOf(BindHook(nil)).Pointer()
	})
)

// isFizzHook reports whether h was returned by BindHook.
func isFizzHook(h tonic.BindHook) bool {
	return h != nil && reflect.ValueOf(h).Pointer() == fizzHookPC()
}

// InstallBindHook wraps the current tonic bind hook with BindHook, unless
// the current hook is already one that BindHook returned. It is
// idempotent, safe for concurrent use, and called by New and NewFromEngine
// unless WithoutBindHook is passed.
//
// The tonic bind hook is process-global: this changes the hook of every
// tonic handler of the process, fizz's or not (the hook only acts on
// routes registered with fizz). A hook set with tonic.SetBindHook after
// the last New replaces fizz's: compose it with
// tonic.SetBindHook(fizz.BindHook(hook)) or call InstallBindHook again.
//
// Fizz recognizes its own hook by identity. If you wrap the hook that
// fizz installed inside a hook of your own and then call New again, fizz
// cannot see inside your closure and wraps once more. That is harmless:
// the deepObject parameters are bound once per request, but the extra
// layer costs a call.
func InstallBindHook() {
	installMu.Lock()
	defer installMu.Unlock()
	if cur := tonic.GetBindHook(); !isFizzHook(cur) {
		tonic.SetBindHook(BindHook(cur))
	}
}

// bindPlan returns the plan to attach to a route, or nil if the route has
// nothing for the hook to bind. A problem that makes the route's
// deepObject or cookie parameters unbindable is recorded as a warning.
func (g *RouterGroup) bindPlan(method, path, id string, hfunc *tonic.Route, inputModel any) *bind.Plan {
	it := hfunc.InputType()
	if it == nil {
		return nil
	}
	warn := func(msg string) {
		g.gen.RecordError(&openapi.Warning{Method: method, Path: path, ID: id, Msg: msg})
	}
	if inputModel != nil && deref(reflect.TypeOf(inputModel)) != deref(it) {
		warn(fmt.Sprintf("InputModel %s differs from the handler input %s: deepObject and cookie parameters are not bound", reflect.TypeOf(inputModel), it))
		return nil
	}
	p, err := bind.PlanFor(it, g.bindTags)
	if err != nil {
		warn(fmt.Sprintf("deepObject and cookie parameters are not bound: %v", err))
		return nil
	}
	if p.Empty() {
		return nil
	}
	return p
}

func deref(t reflect.Type) reflect.Type {
	for t != nil && t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}

// bindTagsOf returns the struct tag names that the planner reads. The
// query and default tags are tonic's; style and cookie follow n.
func bindTagsOf(n openapi.TagNames) bind.Tags {
	pick := func(set, def string) string {
		switch set {
		case "":
			return def
		case "-":
			return ""
		}
		return set
	}
	return bind.Tags{
		Query:   tonic.QueryTag,
		Cookie:  pick(n.Cookie, "cookie"),
		Style:   pick(n.Style, "style"),
		Default: tonic.DefaultTag,
	}
}

// warnBindHookNotRun logs, in gin debug mode, that fizz's hook did not
// run for a route that has a plan: another hook replaced it.
func warnBindHookNotRun(opID string) {
	if gin.IsDebugging() {
		fmt.Fprintf(gin.DefaultWriter, "[FIZZ-debug] WARNING: the tonic bind hook was replaced after fizz.New, so the deepObject and cookie parameters of operation %q are not bound; set it with tonic.SetBindHook(fizz.BindHook(next))\n", opID)
	}
}
