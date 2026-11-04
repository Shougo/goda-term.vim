// Package shellhistory is the source "term_shell_history" of goda-fi.vim
// (https://github.com/Shougo/goda-fi.vim), as ddt.vim's ddu source
// ddt_shell_history: the command lines of history files (that of the UI
// shell, of zsh, ...), the newest first, with the actions execute, insert
// and edit for the session of the current buffer.
//
// Add it in your configuration of goda-fi.vim:
//
//	import "github.com/Shougo/goda-term.vim/ext/fi/shellhistory"
//
//	b.AddSource(shellhistory.New(shellhistory.Params{
//		Paths: []string{"~/.cache/ddt-shell-history"},
//	}))
package shellhistory

import (
	"bufio"
	"context"
	"os"
	"regexp"
	"slices"

	"github.com/Shougo/goda-fi.vim/fi"
	"github.com/Shougo/goda.vim/sdk"
)

// Name is that of the source.
const Name = "term_shell_history"

// Params are the params of the source.
type Params struct {
	// Limit is how many command lines of each file (default 500).
	Limit int `json:"limit"`
	// Paths are the history files (expand()ed; those that are not there
	// are skipped).
	Paths []string `json:"paths"`
}

// ActionData is the action of an item.
type ActionData struct {
	CommandLine string `json:"commandLine"`
}

type source struct{ p Params }

// New returns the source, with p as the default params.
func New(p Params) fi.Source { return source{p} }

func (source) Name() string { return Name }
func (source) Kind() string { return "base" }

func (s source) Gather(ctx context.Context, h sdk.Host, a fi.GatherArgs, emit func([]fi.Item) error) error {
	p := s.p
	if p.Limit == 0 {
		p.Limit = 500
	}
	if err := fi.Decode(a.SourceParams, &p); err != nil {
		return err
	}
	var items []fi.Item
	for _, path := range p.Paths {
		v, err := h.Call(ctx, "expand", path)
		if err != nil {
			return err
		}
		expanded, _ := v.(string)
		lines := History(expanded, p.Limit)
		slices.Reverse(lines)
		for _, l := range lines {
			items = append(items, fi.Item{Word: l, Action: ActionData{l}})
		}
	}
	return emit(items)
}

var zshExtended = regexp.MustCompile(`^: \d+:\d+;(.*)`)

// History is the command lines of the file path (zsh's extended history
// too), each once (where it was first), the last limit of them.
func History(path string, limit int) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	seen := map[string]bool{}
	var out []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if m := zshExtended.FindStringSubmatch(line); m != nil {
			line = m[1]
		}
		if line != "" && !seen[line] {
			seen[line] = true
			out = append(out, line)
		}
	}
	if limit > 0 && len(out) > limit {
		out = out[len(out)-limit:]
	}
	return out
}

func commandLines(items []fi.Item) []string {
	var out []string
	for _, it := range items {
		var d ActionData
		if it.Action != nil && sdk.Decode(it.Action, &d) == nil {
			out = append(out, d.CommandLine)
		}
	}
	return out
}

func action(desc, uiAction string, edit bool) fi.Action {
	return fi.Action{
		Description: desc,
		Run: func(ctx context.Context, h sdk.Host, a fi.ActionArgs) (fi.ActionResult, error) {
			for _, l := range commandLines(a.Items) {
				if _, err := h.Call(ctx, "goda_term#fi#_command_line", uiAction, l, edit); err != nil {
					return fi.ActionResult{}, err
				}
			}
			return fi.ActionResult{}, nil
		},
	}
}

func (source) Actions() map[string]fi.Action {
	return map[string]fi.Action{
		"edit":    action("Edit the command line, and execute it.", "send", true),
		"execute": action("Execute the command line.", "send", false),
		"insert":  action("Insert the command line.", "insert", false),
	}
}

var _ fi.SourceActioner = source{}
