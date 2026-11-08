// The binary of the end-to-end tests: goda-term with the UI terminal.
package main

import (
	"github.com/Shougo/goda-term-ui-terminal.vim/ext/ui/terminal"
	"github.com/Shougo/goda-term.vim/term"
)

func main() {
	b := term.New()
	b.AddUI(terminal.New())
	b.Main()
}
