// The binary of the end-to-end tests: the UIs of goda-term.vim.
package main

import (
	"github.com/Shougo/goda-term.vim/ext/ui/shell"
	"github.com/Shougo/goda-term.vim/ext/ui/terminal"
	"github.com/Shougo/goda-term.vim/term"
)

func main() {
	b := term.New()
	b.AddUI(shell.New())
	b.AddUI(terminal.New())
	b.Main()
}
