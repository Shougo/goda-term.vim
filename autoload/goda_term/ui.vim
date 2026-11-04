" The UI API, as ddt#ui#*(). The session is that of the current buffer,
" unless {name} is given.

function! goda_term#ui#do_action(action, params = {}, name = get(b:, 'goda_term_name', '')) abort
  call goda_term#ui_action(a:name, a:action, a:params)
endfunction

function! goda_term#ui#get_input(name = get(b:, 'goda_term_name', '')) abort
  return goda_term#get_input(a:name)
endfunction

" Delete the current buffer forcibly, close the current tab, and go back to
" the window of the last UI (an editor started by $EDITOR in a session).
function! goda_term#ui#kill_editor() abort
  if !exists('g:goda_term_ui_last_winid')
    return
  endif
  let tabpagenr = tabpagenr()
  call goda_term#ui#buffer_delete(bufnr('%'))
  call win_gotoid(g:goda_term_ui_last_winid)
  execute 'tabclose' tabpagenr
  if !has('nvim') && &l:buftype =~# 'terminal'
    " Vim redraws a terminal in Terminal-Job mode.
    silent! normal! A
  endif
endfunction

function! goda_term#ui#buffer_delete(bufnr) abort
  if a:bufnr < 0
    return
  endif
  let winid = get(win_findbuf(a:bufnr), 0, -1)
  if winid > 0
    let winid_save = win_getid()
    call win_gotoid(winid)
    silent enew
    execute 'silent! bdelete!' a:bufnr
    call win_gotoid(winid_save)
  else
    execute 'silent! bdelete!' a:bufnr
  endif
endfunction

" Set the variables of the session {name} of the UI {ui} in the buffer
" {bufnr} (the current window).
function! goda_term#ui#_init_variables(name, ui, bufnr, cwd) abort
  call setbufvar(a:bufnr, 'goda_term_name', a:name)
  let t:goda_term_name = a:name
  let t:goda_term_ui_last_bufnr = a:bufnr
  let t:goda_term_ui_last_directory = a:cwd
  call settabvar(tabpagenr(), 'goda_term_ui_' . a:ui . '_last_name', a:name)
  let g:goda_term_ui_last_winid = win_getid()
endfunction

" After a start: $EDITOR for the jobs of the sessions.
function! goda_term#ui#_started(nvim_server) abort
  call s:set_editor(a:nvim_server)
  redraw
endfunction

" After a UI action.
function! goda_term#ui#_acted() abort
  let g:goda_term_ui_last_winid = win_getid()
  redraw
endfunction

function! s:set_editor(nvim_server) abort
  let editor = ''
  if exists('g:loaded_guise')
    " vim-guise sets $EDITOR itself.
  elseif v:progname ==# 'nvim' && executable('nvr') && v:servername !=# ''
    " neovim-remote (Neovim has no --remote-tab-wait-silent yet).
    let editor = printf('nvr --servername=%s --remote-tab-wait-silent', v:servername)
  elseif executable(v:progpath) && has('clientserver') && v:servername !=# ''
    let editor = printf('%s --servername=%s --remote-tab-wait-silent',
          \ v:progpath, v:servername)
  elseif executable(v:progpath)
    let editor = v:progpath
  endif
  if editor !=# ''
    let $EDITOR = editor
    let $GIT_EDITOR = editor
  endif
endfunction
