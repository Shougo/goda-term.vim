package term

import (
	"context"

	"github.com/Shougo/goda.vim/sdk"
)

// UI is a kind of UI ("shell", "terminal"): it makes the UI of each session.
type UI interface {
	Name() string
	// New is the UI of a new session.
	New() Instance
}

// Instance is the UI of a session: the buffer, the job, ...
//
// The editor may call back while a method waits for it (a confirm(), the
// autocommands of a buffer), and another request can come then: an Instance
// does not hold its locks across calls of the editor.
type Instance interface {
	// Redraw shows the UI: goda_term#start().
	Redraw(ctx context.Context, h sdk.Host, a Args) error
	// GetInput is the command line being typed (goda_term#get_input()).
	GetInput(ctx context.Context, h sdk.Host, a Args) (string, error)
	// Actions are the UI actions, by name.
	Actions() map[string]Action
}

// Action is a UI action: goda_term#ui#do_action().
type Action func(ctx context.Context, h sdk.Host, a Args, params Params) error

// Args are what the methods of an Instance get.
type Args struct {
	Options Options
	// Params are the params of the UI (Options.UIParams[Options.UI]).
	Params  Params
	Context Context
}

// Context is where goda_term#start() was called.
type Context struct {
	Cwd   string `json:"cwd"`
	BufNr int    `json:"bufNr"`
	WinID int    `json:"winId"`
}
