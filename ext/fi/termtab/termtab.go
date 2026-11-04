// Package termtab is the source and the kind "term_tab" of goda-fi.vim
// (https://github.com/Shougo/goda-fi.vim), as ddt.vim's ddu source and kind
// ddt_tab: the tab pages, with the directory of their last UI.
//
// Add them in your configuration of goda-fi.vim:
//
//	import "github.com/Shougo/goda-term.vim/ext/fi/termtab"
//
//	b.AddSource(termtab.NewSource())
//	b.AddKind(termtab.NewKind())
package termtab

import (
	"context"

	"github.com/Shougo/goda-fi.vim/fi"
	"github.com/Shougo/goda.vim/sdk"
)

// Name is that of the source and of the kind.
const Name = "term_tab"

// ActionData is the action of an item.
type ActionData struct {
	Cwd   string `json:"cwd"`
	TabNr int    `json:"tabNr"`
}

type source struct{}

// NewSource returns the source.
func NewSource() fi.Source { return source{} }

func (source) Name() string { return Name }
func (source) Kind() string { return Name }

func (source) Gather(ctx context.Context, h sdk.Host, _ fi.GatherArgs, emit func([]fi.Item) error) error {
	v, err := h.Call(ctx, "goda_term#fi#_tabs")
	if err != nil {
		return err
	}
	var tabs []struct {
		Word string `json:"word"`
		ActionData
	}
	if err := sdk.Decode(v, &tabs); err != nil {
		return err
	}
	items := make([]fi.Item, len(tabs))
	for i, t := range tabs {
		items[i] = fi.Item{Word: t.Word, Action: t.ActionData}
	}
	return emit(items)
}

type kind struct{}

// NewKind returns the kind.
func NewKind() fi.Kind { return kind{} }

func (kind) Name() string { return Name }

func data(items []fi.Item) []ActionData {
	var out []ActionData
	for _, it := range items {
		var d ActionData
		if it.Action != nil && sdk.Decode(it.Action, &d) == nil && d.TabNr > 0 {
			out = append(out, d)
		}
	}
	return out
}

// each runs the Vim function fn for each item.
func each(desc, fn string, args func(ActionData) []any) fi.Action {
	return fi.Action{
		Description: desc,
		Run: func(ctx context.Context, h sdk.Host, a fi.ActionArgs) (fi.ActionResult, error) {
			for _, d := range data(a.Items) {
				if _, err := h.Call(ctx, fn, args(d)...); err != nil {
					return fi.ActionResult{}, err
				}
			}
			return fi.ActionResult{}, nil
		},
	}
}

func (kind) Actions() map[string]fi.Action {
	tabnr := func(d ActionData) []any { return []any{d.TabNr} }
	return map[string]fi.Action{
		"delete": {
			Description: "Close the tab pages.",
			Run: func(ctx context.Context, h sdk.Host, a fi.ActionArgs) (fi.ActionResult, error) {
				tabnrs := []int{}
				for _, d := range data(a.Items) {
					tabnrs = append(tabnrs, d.TabNr)
				}
				_, err := h.Call(ctx, "goda_term#fi#_tab_delete", tabnrs)
				return fi.ActionResult{Flags: fi.Persist | fi.RefreshItems}, err
			},
		},
		"edit":   each("Change the directory of the tab page.", "goda_term#fi#_tab_edit", tabnr),
		"switch": each("Go to the tab page.", "goda_term#fi#_tab_switch", tabnr),
		"new": each("A new tab page in another directory.", "goda_term#fi#_tab_new",
			func(d ActionData) []any { return []any{d.TabNr, d.Cwd} }),
	}
}
