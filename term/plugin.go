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
)

// PluginName is the name of the plugin: goda#request('term', ...).
const PluginName = "term"

// Builder collects the UIs and the configuration.
type Builder struct {
	uis    map[string]UI
	global Options
	local  map[string]Options
	errs   []string
}

// New returns an empty Builder.
func New() *Builder {
	return &Builder{uis: map[string]UI{}, local: map[string]Options{}}
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
	b.global = b.patch(b.global, o)
	return b
}

// PatchLocal patches the options of the session name.
func (b *Builder) PatchLocal(name string, o Options) *Builder {
	b.local[name] = b.patch(b.local[name], o)
	return b
}

func (b *Builder) patch(dst, src Options) Options {
	d, err := toMap(dst)
	if err == nil {
		var s optionMap
		if s, err = toMap(src); err == nil {
			var o Options
			if o, err = fromMap(merge(d, s)); err == nil {
				if dst.Name == "" && src.Name == "" {
					o.Name = ""
				}
				return o
			}
		}
	}
	b.errs = append(b.errs, err.Error())
	return dst
}

// Plugin is the plugin of the configuration, or the errors in it.
func (b *Builder) Plugin() (*Plugin, error) {
	if len(b.errs) > 0 {
		return nil, fmt.Errorf("goda-term: %s", strings.Join(b.errs, "; "))
	}
	p := &Plugin{uis: b.uis, sessions: map[string]*session{}}
	var err error
	if p.global, err = toMap(b.global); err != nil {
		return nil, err
	}
	p.local = map[string]optionMap{}
	for name, o := range b.local {
		if p.local[name], err = toMap(o); err != nil {
			return nil, err
		}
	}
	return p, nil
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
	global optionMap
	local  map[string]optionMap

	mu       sync.Mutex
	sessions map[string]*session
}

type session struct {
	ui       string
	instance Instance
	args     Args
}

func (p *Plugin) Name() string { return PluginName }

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

// patches are those of goda_term#custom#*(), which every request carries.
type patches struct {
	Global optionMap            `json:"global"`
	Local  map[string]optionMap `json:"local"`
}

// resolve is the options of a start with the options given.
func (p *Plugin) resolve(user optionMap, pt patches) (Options, error) {
	name, _ := user["name"].(string)
	if name == "" {
		name, _ = pt.Global["name"].(string)
	}
	if name == "" {
		name, _ = p.global["name"].(string)
	}
	if name == "" {
		name = "default"
	}
	m := merge(p.global, p.local[name])
	m = merge(m, pt.Global)
	m = merge(m, pt.Local[name])
	m = merge(m, user)
	m["name"] = name
	return fromMap(m)
}

// start(options, context, patches) shows the session of the options.
func (p *Plugin) start(ctx context.Context, h sdk.Host, params []any) (any, error) {
	var user optionMap
	var c Context
	var pt patches
	if err := decodeParams(params, &user, &c, &pt); err != nil {
		return nil, err
	}
	o, err := p.resolve(user, pt)
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
	if err := decodeParams(params, &name, &action, &ap); err != nil {
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
	if err := decodeParams(params, &name); err != nil {
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

// options(name, patches) are the options a start of name would have,
// without those of the start (goda_term#custom#get_current()).
func (p *Plugin) options(_ context.Context, _ sdk.Host, params []any) (any, error) {
	var name string
	var pt patches
	if err := decodeParams(params, &name, &pt); err != nil {
		return nil, err
	}
	if _, args, err := p.session(name); err == nil {
		return toMap(args.Options)
	}
	o, err := p.resolve(optionMap{"name": name}, pt)
	if err != nil {
		return nil, err
	}
	return toMap(o)
}

// decodeParams decodes the params given into dsts, in order; those not
// given stay as they are.
func decodeParams(params []any, dsts ...any) error {
	for i, d := range dsts {
		if i >= len(params) || params[i] == nil {
			continue
		}
		if err := sdk.Decode(params[i], d); err != nil {
			return fmt.Errorf("goda-term: param %d: %w", i+1, err)
		}
	}
	return nil
}
