" The end-to-end scenario: goda-term.vim in a real Vim or Neovim, with the UI
" terminal. Failures go to $GODA_TERM_RESULT; it says "ok" at the end.

set nocompatible
set encoding=utf-8
set nomore
syntax on
let s:errors = []
let s:result = $GODA_TERM_RESULT

execute 'set runtimepath^=' . fnameescape($GODA_VIM_DIR)
execute 'set runtimepath^=' . fnameescape($GODA_TERM_DIR)
execute 'set runtimepath^=' . fnameescape($GODA_TERM_UI_DIR)
let g:goda_term#binary = $GODA_TERM_BINARY
" goda-term sets it.
let $EDITOR = ''

function! s:check(name, got, want) abort
  if a:got !=# a:want
    call add(s:errors, printf('%s: got %s, want %s', a:name, string(a:got), string(a:want)))
  endif
endfunction

" Wait for {cond} (an expression) to be true, 5 seconds at most: the job
" writes when it gets to it.
function! s:wait(cond, bufnr = bufnr('%')) abort
  for _ in range(250)
    if eval(a:cond)
      return v:true
    endif
    if has('nvim') || getbufvar(a:bufnr, '&buftype') !=# 'terminal'
      sleep 20m
    else
      call term_wait(a:bufnr, 20)
    endif
  endfor
  let lines = getbufvar(a:bufnr, '&buftype') ==# 'terminal' ? s:lines('') : getbufline(a:bufnr, 1, '$')
  " (Is the job of the terminal running?)
  let job = has('nvim') ? getbufvar(a:bufnr, 'terminal_job_id', -1) : -1
  let status = job > 0 ? ' job ' . job . ' ' . string(jobwait([job], 0)) : ''
  call add(s:errors, 'timed out: ' . a:cond . ' ' . string(lines) . status)
  return v:false
endfunction

" The lines of the job: Vim shows them in the buffer in Terminal-Job mode
" only (its screen has them meanwhile).
function! s:lines(pattern) abort
  if has('nvim')
    let lines = getbufline(s:buf, 1, '$')
  else
    let lines = map(range(1, term_getsize(s:buf)[0]), {_, r -> term_getline(s:buf, r)})
  endif
  return filter(lines, {_, l -> l =~# a:pattern})
endfunction

" Vim: the buffer as the job's screen is.
function! s:refresh() abort
  if !has('nvim') && bufnr('%') == s:buf
    call feedkeys("\<C-\>\<C-n>i\<C-\>\<C-n>", 'xt')
  endif
endfunction

" The error messages since the last call.
function! s:errmsgs() abort
  let msgs = split(execute('messages'), "\n")
  let new = msgs[get(s:, 'nmsgs', 0) :]
  let s:nmsgs = len(msgs)
  return join(new, "\n")
endfunction

" The sessions, the options, the actions and the Vim script of the goda-fi
" sources (the UIs test themselves in their repositories).
function! s:core() abort
  if has('win32')
    " The drive first: 'isfname' of Neovim has no ":" there (\f).
    let [command, prompt, extra] = [['cmd.exe'], '\a:\\[^>]*>', {}]
  else
    let [command, prompt, extra] = [['sh'], '% ', #{env: #{PS1: '% '}}]
  endif
  call goda_term#custom#patch_global(#{uiParams: #{terminal: #{
        \ command: command, promptPattern: prompt, extraTermOptions: extra,
        \ }}})
  call goda_term#start(#{name: 'term', ui: 'terminal'})
  let s:buf = bufnr('%')
  call s:check('filetype', &filetype, 'goda-term-terminal')
  call s:check('variables', [b:goda_term_name, t:goda_term_ui_terminal_last_name,
        \ t:goda_term_ui_last_bufnr, g:goda_term_ui_last_winid], ['term', 'term', s:buf, win_getid()])
  call s:check('sessions', goda_term#sessions(), ['term'])
  call s:check('get_current', goda_term#custom#get_current('term').ui, 'terminal')
  " The jobs have $EDITOR (for "git commit"): it is set before they start.
  if !has('win32')
    call s:check('$EDITOR', $EDITOR !=# '', v:true)
  endif

  " Shown again from another window: the same buffer.
  wincmd n
  call goda_term#start(#{name: 'term', ui: 'terminal'})
  call s:check('again', bufnr('%'), s:buf)

  " An unknown action says so.
  call s:errmsgs()
  call goda_term#ui#do_action('unknown')
  call s:check('unknown action', s:errmsgs() =~# 'no such action', v:true)
  " A local patch: an error of the UI is shown.
  call goda_term#custom#patch_local('nocmd', #{uiParams: #{terminal: #{command: []}}})
  call goda_term#start(#{name: 'nocmd', ui: 'terminal'})
  call s:check('no command', s:errmsgs() =~# '"command"', v:true)

  " The Vim script of the goda-fi sources.
  execute 'buffer' s:buf
  if !has('win32')
    call s:wait('!empty(s:lines(' . string('^' . prompt) . '))')
    call goda_term#fi#_command_line('send', 'echo fromfi', v:false)
    call s:wait('len(s:lines(''^fromfi$'')) == 1')
  endif
  let dir = resolve(fnamemodify(tempname(), ':p'))
  call mkdir(dir, 'p')
  call goda_term#ui#do_action('cd', #{directory: dir})
  call s:check('last directory', t:goda_term_ui_last_directory, dir)
  tabnew
  let tabs = goda_term#fi#_tabs()
  call s:check('tabs', [len(tabs), tabs[0].word, tabs[0].tabNr, tabs[1].word =~# '^\[no goda-term\] '],
        \ [2, dir, 1, 1])
  call goda_term#fi#_tab_switch(1)
  call s:check('tab switch', [tabpagenr(), g:goda_term_ui_last_winid], [1, win_getid()])
  call goda_term#fi#_tab_delete([2, 1])
  call s:check('tab delete', tabpagenr('$'), 1)
endfunction

try
  call s:core()
catch
  call add(s:errors, v:throwpoint . ': ' . v:exception)
endtry
call writefile(s:errors + (empty(s:errors) ? ['ok'] : []) + goda#log(), s:result)
call goda#stop()
qall!
