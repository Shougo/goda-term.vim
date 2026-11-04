package shellhistory

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Shougo/goda-fi.vim/fi"
	"github.com/Shougo/goda.vim/sdk"
)

type host struct{ calls []string }

func (h *host) Meta() sdk.Meta { return sdk.Meta{} }
func (h *host) Call(_ context.Context, fn string, args ...any) (any, error) {
	h.calls = append(h.calls, fmt.Sprintln(fn, args)[:len(fmt.Sprintln(fn, args))-1])
	if fn == "expand" {
		return args[0], nil
	}
	return nil, nil
}
func (h *host) Notify(context.Context, string, ...any) error { return nil }
func (h *host) Eval(context.Context, string) (any, error)    { return nil, nil }
func (h *host) Cmd(context.Context, string) error            { return nil }

func TestGather(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	z := filepath.Join(dir, "zsh")
	if err := os.WriteFile(a, []byte("ls\ngit status\nls\n\ngit diff"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(z, []byte(": 1710000000:0;make\n: 1710000001:0;make test\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := History(a, 2); !slices.Equal(got, []string{"git status", "git diff"}) {
		t.Errorf("limit %q", got)
	}
	var items []fi.Item
	err := New(Params{}).Gather(context.Background(), &host{}, fi.GatherArgs{
		SourceParams: fi.Params{"paths": []any{a, z, filepath.Join(dir, "none")}},
	}, func(it []fi.Item) error { items = append(items, it...); return nil })
	if err != nil {
		t.Fatal(err)
	}
	var words []string
	for _, it := range items {
		words = append(words, it.Word)
	}
	if !slices.Equal(words, []string{"git diff", "git status", "ls", "make test", "make"}) {
		t.Errorf("items %q", words)
	}
	// The actions: the UI action of the session of the current buffer.
	h := &host{}
	acts := New(Params{}).(fi.SourceActioner).Actions()
	if _, err := acts["edit"].Run(context.Background(), h, fi.ActionArgs{Items: items[:1]}); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(h.calls, []string{"goda_term#fi#_command_line [send git diff true]"}) {
		t.Errorf("calls %q", h.calls)
	}
}
