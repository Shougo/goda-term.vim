module github.com/Shougo/goda-term.vim/e2e

go 1.26.0

require (
	github.com/Shougo/goda-term-ui-terminal.vim v0.1.0
	github.com/Shougo/goda-term.vim v0.1.3-0.20261107042316-3e34a0add5f3
)

require github.com/Shougo/goda.vim v0.7.2 // indirect

// This goda-term.vim.
replace github.com/Shougo/goda-term.vim => ../
