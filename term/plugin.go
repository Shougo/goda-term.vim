// Package term is goda-term.vim, the successor of ddt.vim: a configuration
// (main.go) adds the UIs and patches the options, and serves the plugin.
//
//	b := term.New()
//	b.AddUI(terminal.New())
//	b.PatchGlobal(term.Options{UIParams: map[string]term.Params{
//		"terminal": {"command": []string{"zsh"}},
//	}})
//	b.Main()
package term

import (
	"context"
	"errors"
	"fmt"
	"log"
	"slices"
	"strings"
	"sync"

	"github.com/Shougo/goda.vim/host"
	"github.com/Shougo/goda.vim/sdk"
	"github.com/Shougo/goda.vim/sdk/custom"
)

// PluginName is the name of the plugin: goda#request('term', ...).
const PluginName = "term"

// Builder collects the UIs and the configuration.
type Builder struct {
	uis    map[string]UI
	custom *custom.Patches[Options]
	errs   []string
}

// New returns an empty Builder.
func New() *Builder {
	return &Builder{uis: map[string]UI{}, custom: newCustom()}
}

// AddUI adds a UI under its Name.
func (b *Builder) AddUI(u UI) *Builder {
	if _, dup := b.uis[u.Name()]; dup {
		b.errs = append(b.errs, fmt.Sprintf("ui %q added twice", u.Name()))
	}
	b.uis[u.Name()] = u
	return b
}

// PatchGlobal patches the options of every session.
func (b *Builder) PatchGlobal(o Options) *Builder {
	if err := b.custom.PatchGlobal(o); err != nil {
		b.errs = append(b.errs, err.Error())
	}
	return b
}

// PatchLocal patches the options of the session name.
func (b *Builder) PatchLocal(name string, o Options) *Builder {
	if err := b.custom.PatchLocal(name, o); err != nil {
		b.errs = append(b.errs, err.Error())
	}
	return b
}

// Plugin is the plugin of the configuration, or the errors in it.
func (b *Builder) Plugin() (*Plugin, error) {
	if len(b.errs) > 0 {
		return nil, fmt.Errorf("goda-term: %s", strings.Join(b.errs, "; "))
	}
	return &Plugin{uis: b.uis, custom: b.custom, sessions: map[string]*session{}}, nil
}

// Main serves the plugin, with the other plugins given, as the whole
// program: the last line of main.go. A configuration error does not end
// it: goda_term#start() shows it, and the other plugins work.
func (b *Builder) Main(others ...sdk.Plugin) {
	host.Main(append([]sdk.Plugin{b.MainPlugin()}, others...)...)
}

// MainPlugin is the plugin, for the Main of another binary: with an invalid
// configuration, one that tells it.
func (b *Builder) MainPlugin() sdk.Plugin {
	p, err := b.Plugin()
	if err != nil {
		log.Print(err)
		return brokenPlugin{err}
	}
	return p
}

type brokenPlugin struct{ err error }

func (brokenPlugin) Name() string { return PluginName }

// ScriptVersion: a broken configuration says why, not that the binary is to
// be built again.
func (brokenPlugin) ScriptVersion() int { return ScriptVersion }

func (p brokenPlugin) Handlers() map[string]sdk.Handler {
	hs := map[string]sdk.Handler{}
	for name := range (&Plugin{}).Handlers() {
		hs[name] = func(context.Context, sdk.Host, []any) (any, error) { return nil, p.err }
	}
	hs["status"] = func(context.Context, sdk.Host, []any) (any, error) { return p.err.Error(), nil }
	return hs
}

// Plugin serves the sessions.
type Plugin struct {
	uis    map[string]UI
	custom *custom.Patches[Options]

	mu       sync.Mutex
	sessions map[string]*session
}

type session struct {
	ui       string
	instance Instance
	args     Args
}

func (p *Plugin) Name() string { return PluginName }

// ScriptVersion is the version of the Vim script (autoload/goda_term.vim) this
// binary goes with (sdk.ScriptVersioner): goda.vim refuses a binary of
// another one and says to build it again. Bump it with s:version there when
// the Vim script and the binary no longer work together.
const ScriptVersion = 2

func (p *Plugin) ScriptVersion() int { return ScriptVersion }

func (p *Plugin) Handlers() map[string]sdk.Handler {
	return map[string]sdk.Handler{
		"status":   func(context.Context, sdk.Host, []any) (any, error) { return "", nil },
		"start":    p.start,
		"uiAction": p.uiAction,
		"getInput": p.getInput,
		"sessions": p.sessionNames,
		"options":  p.options,
	}
}

// resolve is the options of a start with the options given.
func (p *Plugin) resolve(user optionMap) (Options, error) {
	o, _, err := p.custom.Resolve(user, custom.Request{})
	return o, err
}

// start(options, context) shows the session of the options.
func (p *Plugin) start(ctx context.Context, h sdk.Host, params []any) (any, error) {
	var user optionMap
	var c Context
	if err := sdk.DecodeArgs(params, &user, sdk.Optional(&c)); err != nil {
		return nil, err
	}
	o, err := p.resolve(user)
	if err != nil {
		return nil, err
	}
	if o.UI == "" {
		return nil, errors.New("goda-term: no ui (the option \"ui\")")
	}
	u, ok := p.uis[o.UI]
	if !ok {
		return nil, fmt.Errorf("goda-term: no such ui: %q", o.UI)
	}
	p.mu.Lock()
	s := p.sessions[o.Name]
	if s == nil || s.ui != o.UI {
		// Another UI of the name: a new one.
		s = &session{ui: o.UI, instance: u.New()}
		p.sessions[o.Name] = s
	}
	s.args = Args{Options: o, Params: o.UIParams[o.UI], Context: c}
	args := s.args
	p.mu.Unlock()
	if args.Params == nil {
		args.Params = Params{}
	}
	// $EDITOR first: the job of a terminal has the environment of its
	// start ("git commit" in it edits in this Vim), as in ddt.vim.
	if _, err := h.Call(ctx, "goda_term#ui#_set_editor", o.NvimServer); err != nil {
		return nil, err
	}
	if err := s.instance.Redraw(ctx, h, args); err != nil {
		return nil, err
	}
	_, err = h.Cmd(ctx, "redraw"), nil
	return nil, err
}

func (p *Plugin) session(name string) (*session, Args, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	s := p.sessions[name]
	if s == nil {
		return nil, Args{}, fmt.Errorf("goda-term: no such session: %q", name)
	}
	return s, s.args, nil
}

// uiAction(name, action, params) runs the UI action.
func (p *Plugin) uiAction(ctx context.Context, h sdk.Host, params []any) (any, error) {
	var name, action string
	var ap Params
	if err := sdk.DecodeArgs(params, &name, &action, sdk.Optional(&ap)); err != nil {
		return nil, err
	}
	s, args, err := p.session(name)
	if err != nil {
		return nil, err
	}
	act, ok := s.instance.Actions()[action]
	if !ok {
		names := make([]string, 0)
		for n := range s.instance.Actions() {
			names = append(names, n)
		}
		slices.Sort(names)
		return nil, fmt.Errorf("goda-term: no such action of the ui %q: %q (%s)",
			s.ui, action, strings.Join(names, ", "))
	}
	if ap == nil {
		ap = Params{}
	}
	if args.Params == nil {
		args.Params = Params{}
	}
	if err := act(ctx, h, args, ap); err != nil {
		return nil, err
	}
	_, err = h.Call(ctx, "goda_term#ui#_acted")
	return nil, err
}

// getInput(name) is the command line of the session.
func (p *Plugin) getInput(ctx context.Context, h sdk.Host, params []any) (any, error) {
	var name string
	if err := sdk.DecodeArgs(params, &name); err != nil {
		return nil, err
	}
	s, args, err := p.session(name)
	if err != nil {
		return "", nil
	}
	if args.Params == nil {
		args.Params = Params{}
	}
	return s.instance.GetInput(ctx, h, args)
}

func (p *Plugin) sessionNames(context.Context, sdk.Host, []any) (any, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	names := make([]string, 0, len(p.sessions))
	for n := range p.sessions {
		names = append(names, n)
	}
	slices.Sort(names)
	return names, nil
}

// options(name) are the options of the session name, or those a start of
// name would have, without options of its own (goda_term#get_current()).
func (p *Plugin) options(_ context.Context, _ sdk.Host, params []any) (any, error) {
	var name string
	if err := sdk.DecodeArgs(params, &name); err != nil {
		return nil, err
	}
	if _, args, err := p.session(name); err == nil {
		return custom.ToMap(args.Options)
	}
	o, err := p.resolve(optionMap{"name": name})
	if err != nil {
		return nil, err
	}
	return custom.ToMap(o)
}
