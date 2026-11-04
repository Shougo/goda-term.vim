package termtab

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/Shougo/goda-fi.vim/fi"
	"github.com/Shougo/goda.vim/sdk"
)

type host struct{ calls []string }

func (h *host) Meta() sdk.Meta { return sdk.Meta{} }
func (h *host) Call(_ context.Context, fn string, args ...any) (any, error) {
	h.calls = append(h.calls, fmt.Sprintln(fn, args)[:len(fmt.Sprintln(fn, args))-1])
	if fn == "goda_term#fi#_tabs" {
		return []any{
			map[string]any{"word": "/a", "cwd": "/a", "tabNr": int64(1)},
			map[string]any{"word": "[no goda-term] /b", "cwd": "/b", "tabNr": int64(2)},
		}, nil
	}
	return nil, nil
}
func (h *host) Notify(context.Context, string, ...any) error { return nil }
func (h *host) Eval(context.Context, string) (any, error)    { return nil, nil }
func (h *host) Cmd(context.Context, string) error            { return nil }

func TestTermTab(t *testing.T) {
	var items []fi.Item
	h := &host{}
	err := NewSource().Gather(context.Background(), h, fi.GatherArgs{},
		func(it []fi.Item) error { items = append(items, it...); return nil })
	if err != nil || len(items) != 2 || items[1].Word != "[no goda-term] /b" {
		t.Fatalf("%+v %v", items, err)
	}
	acts := NewKind().Actions()
	h.calls = nil
	res, err := acts["delete"].Run(context.Background(), h, fi.ActionArgs{Items: items})
	if err != nil || res.Flags != fi.Persist|fi.RefreshItems {
		t.Errorf("delete %+v %v", res, err)
	}
	if _, err := acts["new"].Run(context.Background(), h, fi.ActionArgs{Items: items[1:]}); err != nil {
		t.Fatal(err)
	}
	if _, err := acts["switch"].Run(context.Background(), h, fi.ActionArgs{Items: items[:1]}); err != nil {
		t.Fatal(err)
	}
	want := []string{"goda_term#fi#_tab_delete [[1 2]]", "goda_term#fi#_tab_new [2 /b]", "goda_term#fi#_tab_switch [1]"}
	if !slices.Equal(h.calls, want) {
		t.Errorf("calls %q", h.calls)
	}
}
