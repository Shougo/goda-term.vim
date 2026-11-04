" The end-to-end scenario: the UIs in a real Vim or Neovim. Failures go to
" $GODA_TERM_RESULT; it says "ok" at the end.

set nocompatible
set encoding=utf-8
let s:errors = []
let s:result = $GODA_TERM_RESULT

execute 'set runtimepath^=' . fnameescape($GODA_VIM_DIR)
execute 'set runtimepath^=' . fnameescape($GODA_TERM_DIR)
let g:goda_term#binary = $GODA_TERM_BINARY

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
    if has('nvim')
      sleep 20m
    else
      call term_wait(a:bufnr, 20)
    endif
  endfor
  call add(s:errors, 'timed out: ' . a:cond . ' ' . string(s:lines('')))
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

function! s:terminal() abort
  if has('win32')
    let [command, prompt] = [['cmd.exe'], '\f\+>']
  else
    let [command, prompt] = [['sh'], '% ']
  endif
  call goda_term#custom#patch_global(#{uiParams: #{terminal: #{
        \ command: command, promptPattern: prompt,
        \ extraTermOptions: #{env: #{PS1: '% '}},
        \ }}})
  call goda_term#start(#{name: 'term', ui: 'terminal'})
  let s:buf = bufnr('%')
  call s:check('filetype', &filetype, 'goda-term-terminal')
  call s:check('variables', [b:goda_term_name, t:goda_term_ui_terminal_last_name,
        \ t:goda_term_ui_last_bufnr], ['term', 'term', s:buf])
  call s:check('sessions', goda_term#sessions(), ['term'])
  call s:wait('!empty(s:lines(''^' . escape(prompt, '\') . '''))')

  call goda_term#ui#do_action('send', #{str: 'echo hello'})
  call s:wait('len(s:lines(''^hello$'')) == 1')

  " cd: the job is there.
  let dir = resolve(fnamemodify(tempname(), ':p'))
  call mkdir(dir, 'p')
  call goda_term#ui#do_action('cd', #{directory: dir})
  call s:check('last directory', t:goda_term_ui_last_directory, dir)
  call goda_term#ui#do_action('send', #{str: has('win32') ? 'cd' : 'pwd'})
  call s:wait('!empty(s:lines(''^' . escape(dir, '\.~[]*') . '$''))')

  " The prompts: before the cursor, and after it.
  if !has('win32')
    call s:refresh()
    call cursor(search('^hello$', 'w'), 1)
    call goda_term#ui#do_action('previousPrompt')
    call s:check('previousPrompt', [getline('.'), col('.')], ['% echo hello', 3])
    call goda_term#ui#do_action('nextPrompt')
    call s:check('nextPrompt', getline('.') =~# '^% cd ', v:true)
    call goda_term#ui#do_action('previousPrompt')
    call cursor(0, col('$') - 1)
    call s:check('get_input', goda_term#ui#get_input(), 'echo hello')
    " executeLine: that of the cursor again.
    call goda_term#ui#do_action('executeLine')
    call s:wait('len(s:lines(''^hello$'')) >= 2')
  endif

  " Shown again from another window: the same buffer and job.
  wincmd n
  call goda_term#start(#{name: 'term', ui: 'terminal'})
  call s:check('again', bufnr('%'), s:buf)

  " An unknown action says so.
  call s:errmsgs()
  call goda_term#ui#do_action('unknown')
  call s:check('unknown action', s:errmsgs() =~# 'no such action', v:true)
  " Without a command.
  call goda_term#custom#patch_local('nocmd', #{uiParams: #{terminal: #{command: []}}})
  call goda_term#start(#{name: 'nocmd', ui: 'terminal'})
  call s:check('no command', s:errmsgs() =~# '"command"', v:true)
  call s:check('get_current', goda_term#custom#get_current('term').ui, 'terminal')
endfunction

try
  call s:terminal()
catch
  call add(s:errors, v:throwpoint . ': ' . v:exception)
endtry
call writefile(s:errors + (empty(s:errors) ? ['ok'] : []) + goda#log(), s:result)
call goda#stop()
qall!
