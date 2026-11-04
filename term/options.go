package term

import (
	"encoding/json"
	"fmt"
	"maps"
)

// Options are those of ddt.vim: of the configuration (Builder.PatchGlobal,
// PatchLocal), of goda_term#custom#patch_global() and patch_local(), and of
// goda_term#start(), merged in that order. A UI's params are merged param by
// param (a map of a param, aliases say, is replaced whole).
type Options struct {
	// Name is the session: a start with the name of another shows it again.
	// Default "default".
	Name string `json:"name,omitempty"`
	// UI is the name of the UI ("shell", "terminal").
	UI string `json:"ui,omitempty"`
	// UIParams are the params of each UI, by its name.
	UIParams map[string]Params `json:"uiParams,omitempty"`
	// NvimServer is the server of Neovim for $EDITOR (not used yet, as in
	// ddt.vim).
	NvimServer string `json:"nvimServer,omitempty"`
	// Debug is for the UIs.
	Debug bool `json:"debug,omitempty"`
}

// Params are the params of a UI: its Params decodes them (DecodeParams).
type Params map[string]any

// optionMap is Options as Vim sends them.
type optionMap = map[string]any

func toMap(o Options) (optionMap, error) {
	b, err := json.Marshal(o)
	if err != nil {
		return nil, err
	}
	m := optionMap{}
	return m, json.Unmarshal(b, &m)
}

// merge is the options of dst with those of src over them.
func merge(dst, src optionMap) optionMap {
	out := maps.Clone(dst)
	if out == nil {
		out = optionMap{}
	}
	for k, v := range src {
		if k == "uiParams" {
			out[k] = mergeParams(asMap(out[k]), asMap(v))
			continue
		}
		out[k] = v
	}
	return out
}

func mergeParams(dst, src map[string]any) map[string]any {
	out := maps.Clone(dst)
	if out == nil {
		out = map[string]any{}
	}
	for ui, p := range src {
		m := maps.Clone(asMap(out[ui]))
		if m == nil {
			m = map[string]any{}
		}
		maps.Copy(m, asMap(p))
		out[ui] = m
	}
	return out
}

func asMap(v any) map[string]any {
	switch m := v.(type) {
	case map[string]any:
		return m
	case Params:
		return m
	}
	return nil
}

func fromMap(m optionMap) (Options, error) {
	var o Options
	b, err := json.Marshal(m)
	if err != nil {
		return o, err
	}
	if err := json.Unmarshal(b, &o); err != nil {
		return o, fmt.Errorf("invalid options: %w", err)
	}
	if o.Name == "" {
		o.Name = "default"
	}
	return o, nil
}

// DecodeParams is defaults with params over it: the params of a UI, as its
// struct (json tags are the names of the params).
func DecodeParams[T any](params Params, defaults T) (T, error) {
	b, err := json.Marshal(params)
	if err != nil {
		return defaults, err
	}
	p := defaults
	if err := json.Unmarshal(b, &p); err != nil {
		return defaults, fmt.Errorf("invalid params: %w", err)
	}
	return p, nil
}
