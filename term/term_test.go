package term

import (
	"context"
	"reflect"
	"testing"

	"github.com/Shougo/goda.vim/sdk"
)

type fakeUI struct{ name string }

func (u fakeUI) Name() string  { return u.name }
func (u fakeUI) New() Instance { return &fakeInstance{} }

type fakeInstance struct{ redraws []Args }

func (f *fakeInstance) Redraw(_ context.Context, _ sdk.Host, a Args) error {
	f.redraws = append(f.redraws, a)
	return nil
}
func (f *fakeInstance) GetInput(context.Context, sdk.Host, Args) (string, error) { return "in", nil }
func (f *fakeInstance) Actions() map[string]Action {
	return map[string]Action{"act": func(context.Context, sdk.Host, Args, Params) error { return nil }}
}

type fakeHost struct{}

func (fakeHost) Meta() sdk.Meta                                    { return sdk.Meta{Host: sdk.HostNone} }
func (fakeHost) Call(context.Context, string, ...any) (any, error) { return nil, nil }
func (fakeHost) Notify(context.Context, string, ...any) error      { return nil }
func (fakeHost) Eval(context.Context, string) (any, error)         { return nil, nil }
func (fakeHost) Cmd(context.Context, string) error                 { return nil }

func TestResolve(t *testing.T) {
	b := New().AddUI(fakeUI{"shell"})
	b.PatchGlobal(Options{UI: "shell", UIParams: map[string]Params{
		"shell": {"prompt": "%", "aliases": map[string]any{"ls": "ls -F"}},
	}})
	b.PatchLocal("x", Options{UIParams: map[string]Params{"shell": {"split": "vertical"}}})
	p, err := b.Plugin()
	if err != nil {
		t.Fatal(err)
	}
	o, err := p.resolve(optionMap{"name": "x", "uiParams": map[string]any{
		"shell": map[string]any{"aliases": map[string]any{"ll": "ls -l"}},
	}}, patches{Global: optionMap{"uiParams": map[string]any{"shell": map[string]any{"prompt": "$"}}}})
	if err != nil {
		t.Fatal(err)
	}
	want := Params{"prompt": "$", "split": "vertical", "aliases": map[string]any{"ll": "ls -l"}}
	if o.Name != "x" || o.UI != "shell" || !reflect.DeepEqual(o.UIParams["shell"], want) {
		t.Errorf("%+v", o)
	}
	// The default name.
	if o, _ := p.resolve(nil, patches{}); o.Name != "default" {
		t.Errorf("name %q", o.Name)
	}
}

func TestSessions(t *testing.T) {
	p, err := New().AddUI(fakeUI{"shell"}).Plugin()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	h := p.Handlers()
	if _, err := h["start"](ctx, fakeHost{}, []any{map[string]any{"name": "a", "ui": "shell"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := h["start"](ctx, fakeHost{}, []any{map[string]any{"name": "a", "ui": "shell"}}); err != nil {
		t.Fatal(err)
	}
	if got := len(p.sessions["a"].instance.(*fakeInstance).redraws); got != 2 {
		t.Errorf("one instance redrawn twice: %d", got)
	}
	if _, err := h["start"](ctx, fakeHost{}, []any{map[string]any{"ui": "none"}}); err == nil {
		t.Error("an unknown ui")
	}
	if _, err := h["uiAction"](ctx, fakeHost{}, []any{"a", "act", nil}); err != nil {
		t.Error(err)
	}
	if _, err := h["uiAction"](ctx, fakeHost{}, []any{"a", "nope", nil}); err == nil {
		t.Error("an unknown action")
	}
	if v, _ := h["getInput"](ctx, fakeHost{}, []any{"a"}); v != "in" {
		t.Errorf("input %v", v)
	}
	if v, _ := h["getInput"](ctx, fakeHost{}, []any{"b"}); v != "" {
		t.Errorf("no session: %v", v)
	}
	// Twice the same UI is an error of the configuration.
	if _, err := New().AddUI(fakeUI{"shell"}).AddUI(fakeUI{"shell"}).Plugin(); err == nil {
		t.Error("a ui twice")
	}
}

func TestDecodeParams(t *testing.T) {
	type P struct {
		A string `json:"a"`
		B int    `json:"b"`
	}
	p, err := DecodeParams(Params{"b": int64(3)}, P{A: "x", B: 1})
	if err != nil || p != (P{"x", 3}) {
		t.Errorf("%+v %v", p, err)
	}
	if _, err := DecodeParams(Params{"b": "no"}, P{}); err == nil {
		t.Error("invalid")
	}
}
