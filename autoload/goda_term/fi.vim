" The Vim script of the goda-fi.vim sources of goda-term.vim (ext/fi):
" term_tab and term_shell_history.

" The tab pages: [{word, cwd, tabNr}], with the directory of their last UI.
function! goda_term#fi#_tabs() abort
  let tabs = []
  for tabnr in range(1, tabpagenr('$'))
    let cwd = gettabvar(tabnr, 'goda_term_ui_last_directory', '')
    let exists = cwd !=# ''
    if !exists
      let cwd = getcwd(0, tabnr)
    endif
    let word = exists ? cwd
          \ : cwd ==# '' ? '[no goda-term tab] tab ' . tabnr : '[no goda-term] ' . cwd
    call add(tabs, #{word: word, cwd: cwd ==# '' ? getcwd() : cwd, tabNr: tabnr})
  endfor
  return tabs
endfunction

" Close the tab pages {tabnrs} (not the current one).
function! goda_term#fi#_tab_delete(tabnrs) abort
  for tabnr in reverse(sort(filter(copy(a:tabnrs), {_, n -> n > 0 && n != tabpagenr()}), 'n'))
    if !empty(tabpagebuflist(tabnr))
      execute 'silent! tabclose' tabnr
    endif
  endfor
endfunction

" Go to the tab page {tabnr}, and its last UI.
function! goda_term#fi#_tab_switch(tabnr) abort
  if a:tabnr <= 0 || empty(tabpagebuflist(a:tabnr))
    return
  endif
  execute 'tabnext' a:tabnr
  let bufnr = get(t:, 'goda_term_ui_last_bufnr', -1)
  if bufnr > 0
    let g:goda_term_ui_last_winid = win_getid()
    if bufnr('%') == bufnr
      call goda_term#ui#do_action('redraw')
    endif
  endif
endfunction

" A new directory for {path} (asked with {prompt}): '' when cancelled.
function! s:ask_dir(prompt, cwd) abort
  let dir = input(a:prompt, a:cwd, 'dir')
  redraw
  if dir ==# ''
    return ''
  endif
  if filereadable(dir)
    echohl ErrorMsg | echomsg dir . ' is not a directory.' | echohl None
    return ''
  endif
  if !isdirectory(dir)
    if confirm(dir . ' is not a directory.  Create?', "&Yes\n&No\n&Cancel") != 1
      return ''
    endif
    call mkdir(dir, 'p')
  endif
  return dir
endfunction

" Change the directory of the tab page {tabnr} and of its last UI.
function! goda_term#fi#_tab_edit(tabnr) abort
  if a:tabnr <= 0 || empty(tabpagebuflist(a:tabnr))
    return
  endif
  let cwd = gettabvar(a:tabnr, 'goda_term_ui_last_directory', getcwd(0, a:tabnr))
  let dir = s:ask_dir('New goda-term cwd: ', cwd)
  if dir ==# '' || dir ==# cwd
    return
  endif
  execute 'tabnext' a:tabnr
  let bufnr = get(t:, 'goda_term_ui_last_bufnr', -1)
  if bufnr > 0
    execute 'buffer' bufnr
    call goda_term#ui#do_action('cd', #{directory: dir})
  endif
  execute 'noautocmd tcd' fnameescape(dir)
endfunction

" A new tab page after {tabnr}, in a directory asked for (from {cwd}).
function! goda_term#fi#_tab_new(tabnr, cwd) abort
  let dir = s:ask_dir('New cwd: ', a:cwd ==# '' ? getcwd() : a:cwd)
  if dir ==# ''
    return
  endif
  execute 'tabnext' a:tabnr
  tabnew
  execute 'tcd' fnameescape(dir)
endfunction

" Run the UI action {action} of the session of the current buffer, with a
" command line {str} (edited first with {edit}).
function! goda_term#fi#_command_line(action, str, edit) abort
  let str = a:str
  if a:edit
    let str = input('New Commandline: ', str, 'shellcmdline')
    redraw
    if str ==# ''
      return
    endif
  endif
  call goda_term#ui#do_action(a:action, #{str: str})
endfunction
