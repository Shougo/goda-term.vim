# goda-term.vim

Terminals and shells in Vim and Neovim on
[goda.vim](https://github.com/Shougo/goda.vim): the successor of
[ddt.vim](https://github.com/Shougo/ddt.vim), with the configuration in Go
instead of TypeScript, and no Deno.

The UIs are in repositories of their own:

- [goda-term-ui-shell.vim](https://github.com/Shougo/goda-term-ui-shell.vim):
  a buffer with a prompt; each command line runs in a pseudo terminal, and
  its output (with its colors) is written in the buffer (ddt-ui-shell).
- [goda-term-ui-terminal.vim](https://github.com/Shougo/goda-term-ui-terminal.vim):
  a terminal job of Vim or Neovim (ddt-ui-terminal).

## Requirements

- Vim 9.1 or Neovim 0.11
- Go 1.26
- [goda.vim](https://github.com/Shougo/goda.vim)

## Usage

The configuration is a Go module that adds the UIs you use (see
[examples/config](examples/config)) in
`~/.vim/goda-term` (Neovim: `stdpath('config') .. '/goda-term'`). Build it:

```vim
call goda_term#build()
```

Then:

```vim
nnoremap [Space]t <Cmd>call goda_term#start(#{
      \   name: t:->get('goda_term_ui_terminal_last_name',
      \                 'terminal-' .. win_getid()),
      \   ui: 'terminal',
      \ })<CR>
```

The sources `term_tab` and `term_shell_history` of
[goda-fi.vim](https://github.com/Shougo/goda-fi.vim) are in `ext/fi`.

See `:help goda-term` for everything else, and `:help goda-term-ddt` for
the differences from ddt.vim.
