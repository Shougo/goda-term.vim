package term

import "github.com/Shougo/goda.vim/sdk/custom"

// Options are those of ddt.vim: of the configuration (Builder.PatchGlobal,
// PatchLocal) and of goda_term#start(), merged in that order. A UI's params are merged param by
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
type optionMap = custom.Map

// newCustom is the options of the configuration: a UI's params are merged
// param by param (goda.vim's sdk/custom).
func newCustom() *custom.Patches[Options] {
	return custom.New[Options](custom.Params("uiParams"))
}

// DecodeParams is defaults with params over it: the params of a UI, as its
// struct (json tags are the names of the params).
func DecodeParams[T any](params Params, defaults T) (T, error) {
	return custom.DecodeParams(params, defaults)
}
