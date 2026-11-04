// Command goda-term is a configuration of goda-term.vim: put it in
// ~/.vim/goda-term (Neovim: stdpath('config') .. '/goda-term') and build it
// with goda_term#build(). It is translated from the ddt.ts of a ddt.vim
// user.
package main

import (
	"runtime"

	"github.com/Shougo/goda-term.vim/ext/ui/terminal"
	"github.com/Shougo/goda-term.vim/term"
)

func main() {
	prompt := `\w*% \?`
	if runtime.GOOS == "windows" {
		prompt = `\f\+>`
	}
	b := term.New()
	b.AddUI(terminal.New())
	b.PatchGlobal(term.Options{
		UIParams: map[string]term.Params{
			"terminal": {
				"command":       []string{"zsh"},
				"promptPattern": prompt,
			},
		},
	})
	b.Main()
}
