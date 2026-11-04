// Package terminal is the UI "terminal": a terminal job of Vim or Neovim, as
// ddt-ui-terminal.
package terminal

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sync"

	"github.com/Shougo/goda.vim/sdk"

	"github.com/Shougo/goda-term.vim/term"
)

// Params are the params of the UI.
type Params struct {
	// Command is the job (required).
	Command []string `json:"command"`
	// Cwd is its directory (default: the current directory).
	Cwd string `json:"cwd"`
	// ExtraTermOptions are more options of term_start() (Vim) or jobstart()
	// (Neovim).
	ExtraTermOptions map[string]any `json:"extraTermOptions"`
	// FloatingBorder is the border of a floating window (Neovim).
	FloatingBorder any `json:"floatingBorder"`
	// PromptPattern is the prompt of the job (a Vim regexp): nextPrompt,
	// executeLine, ... need it.
	PromptPattern string `json:"promptPattern"`
	// Split is how the window opens: "" (the current window),
	// "horizontal", "vertical", "farleft", "farright", "floating".
	Split string `json:"split"`
	// StartInsert enters Terminal-Job mode (Neovim; Vim always does).
	StartInsert bool `json:"startInsert"`
	WinCol      int  `json:"winCol"`
	WinHeight   int  `json:"winHeight"`
	WinRow      int  `json:"winRow"`
	WinWidth    int  `json:"winWidth"`
}

// DefaultParams are those of ddt-ui-terminal.
func DefaultParams() Params {
	return Params{
		Command:          []string{},
		ExtraTermOptions: map[string]any{},
		FloatingBorder:   "",
		WinCol:           50,
		WinHeight:        15,
		WinRow:           20,
		WinWidth:         80,
	}
}

type ui struct{}

// New returns the UI.
func New() term.UI { return ui{} }

func (ui) Name() string { return "terminal" }

func (ui) New() term.Instance { return &instance{bufnr: -1, jobid: -1, pid: -1} }

type instance struct {
	mu                sync.Mutex
	bufnr, jobid, pid int
}

func (t *instance) state() (bufnr, jobid, pid int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.bufnr, t.jobid, t.pid
}

func params(a term.Args) (Params, error) { return term.DecodeParams(a.Params, DefaultParams()) }

func toInt(v any) int {
	switch n := v.(type) {
	case int64:
		return int(n)
	case float64:
		return int(n)
	case int:
		return n
	}
	return -1
}

func isDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

func (t *instance) Redraw(ctx context.Context, h sdk.Host, a term.Args) error {
	p, err := params(a)
	if err != nil {
		return err
	}
	cwd := p.Cwd
	if cwd == "" {
		cwd = a.Context.Cwd
	}
	if v, err := h.Call(ctx, "expand", cwd); err == nil {
		cwd, _ = v.(string)
	}
	if !isDir(cwd) {
		v, err := h.Call(ctx, "confirm", cwd+" is not a directory.  Create?", "&Yes\n&No\n&Cancel")
		if err != nil || toInt(v) != 1 {
			return err
		}
		if err := os.MkdirAll(cwd, 0o755); err != nil {
			return err
		}
	}
	bufnr, _, pid := t.state()
	exists := false
	if bufnr > 0 {
		v, err := h.Call(ctx, "bufexists", bufnr)
		if err != nil {
			return err
		}
		exists = toInt(v) == 1
	}
	if exists {
		if _, err := h.Call(ctx, "goda_term#ui#terminal#_switch", p, bufnr); err != nil {
			return err
		}
		// The job went elsewhere: back to the directory of the start.
		line, _ := h.Call(ctx, "goda_term#ui#terminal#_command_line", p.PromptPattern, "$")
		v, err := h.Call(ctx, "goda_term#ui#terminal#_get_cwd", pid, line)
		if err != nil {
			return err
		}
		if c, _ := v.(string); c != "" && c != cwd {
			if err := t.cd(ctx, h, cwd); err != nil {
				return err
			}
		}
	} else {
		if len(p.Command) == 0 {
			return fmt.Errorf("goda-term: the param \"command\" of the ui terminal is not set")
		}
		v, err := h.Call(ctx, "goda_term#ui#terminal#_new", p, cwd)
		if err != nil {
			return err
		}
		var st struct{ Bufnr, Jobid, Pid int }
		if err := sdk.Decode(v, &st); err != nil {
			return err
		}
		t.mu.Lock()
		t.bufnr, t.jobid, t.pid = st.Bufnr, st.Jobid, st.Pid
		bufnr = st.Bufnr
		t.mu.Unlock()
	}
	_, err = h.Call(ctx, "goda_term#ui#_init_variables", a.Options.Name, "terminal", bufnr, cwd)
	return err
}

// current reports whether the current buffer is the UI's.
func (t *instance) current(ctx context.Context, h sdk.Host) (bool, error) {
	bufnr, _, _ := t.state()
	v, err := h.Call(ctx, "bufnr", "%")
	return err == nil && toInt(v) == bufnr, err
}

func (t *instance) GetInput(ctx context.Context, h sdk.Host, a term.Args) (string, error) {
	p, err := params(a)
	if err != nil {
		return "", err
	}
	if ok, err := t.current(ctx, h); err != nil || !ok || p.PromptPattern == "" {
		return "", err
	}
	v, err := h.Call(ctx, "goda_term#ui#terminal#_input", p.PromptPattern)
	s, _ := v.(string)
	return s, err
}

func (t *instance) send(ctx context.Context, h sdk.Host, keys string) error {
	bufnr, jobid, _ := t.state()
	_, err := h.Call(ctx, "goda_term#ui#terminal#_send", bufnr, jobid, keys)
	return err
}

func (t *instance) redraw(ctx context.Context, h sdk.Host) error {
	bufnr, _, _ := t.state()
	_, err := h.Call(ctx, "goda_term#ui#terminal#_redraw", bufnr)
	return err
}

func (t *instance) cd(ctx context.Context, h sdk.Host, dir string) error {
	if !isDir(dir) {
		return nil
	}
	cmd := "\x15cd '" + dir + "'\r"
	if runtime.GOOS == "windows" {
		// /d: on another drive too.
		cmd = "cd /d \"" + dir + "\"\r"
	}
	if err := t.send(ctx, h, cmd); err != nil {
		return err
	}
	if err := t.redraw(ctx, h); err != nil {
		return err
	}
	return h.Cmd(ctx, "let t:goda_term_ui_last_directory = "+vimString(dir))
}

func vimString(s string) string {
	out := []byte{'\''}
	for i := 0; i < len(s); i++ {
		if s[i] == '\'' {
			out = append(out, '\'')
		}
		out = append(out, s[i])
	}
	return string(append(out, '\''))
}

func str(p term.Params, key string) string {
	s, _ := p[key].(string)
	return s
}

// commandLine is the command line of the cursor, if the UI's buffer and
// its prompt pattern are those of the cursor.
func (t *instance) commandLine(ctx context.Context, h sdk.Host, p Params) (string, bool, error) {
	if ok, err := t.current(ctx, h); err != nil || !ok || p.PromptPattern == "" {
		return "", false, err
	}
	v, err := h.Call(ctx, "goda_term#ui#terminal#_command_line", p.PromptPattern)
	s, _ := v.(string)
	return s, err == nil, err
}

func (t *instance) Actions() map[string]term.Action {
	withParams := func(f func(context.Context, sdk.Host, Params, term.Params) error) term.Action {
		return func(ctx context.Context, h sdk.Host, a term.Args, ap term.Params) error {
			p, err := params(a)
			if err != nil {
				return err
			}
			return f(ctx, h, p, ap)
		}
	}
	search := func(flags string) term.Action {
		return withParams(func(ctx context.Context, h sdk.Host, p Params, _ term.Params) error {
			if ok, err := t.current(ctx, h); err != nil || !ok || p.PromptPattern == "" {
				return err
			}
			_, err := h.Call(ctx, "goda_term#ui#_search_prompt", p.PromptPattern, flags)
			return err
		})
	}
	return map[string]term.Action{
		// cd {directory}: the job goes there.
		"cd": func(ctx context.Context, h sdk.Host, _ term.Args, ap term.Params) error {
			if ok, err := t.current(ctx, h); err != nil || !ok {
				return err
			}
			return t.cd(ctx, h, str(ap, "directory"))
		},
		// executeLine: the command line of the cursor, executed.
		"executeLine": withParams(func(ctx context.Context, h sdk.Host, p Params, _ term.Params) error {
			line, ok, err := t.commandLine(ctx, h, p)
			if err != nil || !ok {
				return err
			}
			return t.send(ctx, h, line+"\r")
		}),
		// insert {str}: typed in the job.
		"insert": func(ctx context.Context, h sdk.Host, _ term.Args, ap term.Params) error {
			return t.send(ctx, h, str(ap, "str"))
		},
		"nextPrompt":     search("Wn"),
		"previousPrompt": search("bWn"),
		// pastePrompt: the command line of the cursor, typed in the job.
		"pastePrompt": withParams(func(ctx context.Context, h sdk.Host, p Params, _ term.Params) error {
			line, ok, err := t.commandLine(ctx, h, p)
			if err != nil || !ok {
				return err
			}
			return t.send(ctx, h, line)
		}),
		"redraw": func(ctx context.Context, h sdk.Host, _ term.Args, _ term.Params) error {
			if ok, err := t.current(ctx, h); err != nil || !ok {
				return err
			}
			return t.redraw(ctx, h)
		},
		// send {str}: typed in the job, and executed.
		"send": func(ctx context.Context, h sdk.Host, _ term.Args, ap term.Params) error {
			if err := t.send(ctx, h, str(ap, "str")+"\r"); err != nil {
				return err
			}
			return t.redraw(ctx, h)
		},
		// setCommandLine {str}: typed in the job.
		"setCommandLine": func(ctx context.Context, h sdk.Host, _ term.Args, ap term.Params) error {
			if err := t.send(ctx, h, str(ap, "str")); err != nil {
				return err
			}
			return t.redraw(ctx, h)
		},
	}
}
