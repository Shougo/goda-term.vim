" The Vim script of the UI "terminal": a terminal job of Vim or Neovim.

" Start the job of {params} in a new buffer of the current window: {bufnr,
" jobid, pid}.
function! goda_term#ui#terminal#_new(params, cwd) abort
  call goda_term#ui#_split(a:params)
  let opts = extend(deepcopy(a:params.extraTermOptions), #{cwd: a:cwd})
  if has('nvim')
    " :terminal would replace the current buffer.
    enew
    let jobid = jobstart(a:params.command, extend(opts, #{term: v:true}))
    let pid = jobpid(jobid)
  else
    let buf = term_start(a:params.command, extend(opts, #{curwin: v:true, term_kill: 'kill'}))
    let jobid = -1
    let pid = buf > 0 ? job_info(term_getjob(buf)).process : -1
  endif
  let bufnr = bufnr('%')
  if has('nvim')
    if a:params.startInsert
      startinsert
    else
      stopinsert
    endif
  else
    " Vim redraws a terminal in Terminal-Job mode.
    startinsert
  endif
  call s:init_options(bufnr)
  return #{bufnr: bufnr, jobid: jobid, pid: pid}
endfunction

" Show the buffer {bufnr} of the session again.
function! goda_term#ui#terminal#_switch(params, bufnr) abort
  call goda_term#ui#_split(a:params)
  execute 'buffer' a:bufnr
endfunction

function! s:init_options(bufnr) abort
  let winid = get(win_findbuf(a:bufnr), 0, -1)
  for [name, value] in items(#{list: 0, foldenable: 0, number: 0,
        \ relativenumber: 0, spell: 0, wrap: 0, colorcolumn: '',
        \ foldcolumn: 0, signcolumn: 'no'})
    call setwinvar(winid, '&' . name, value)
  endfor
  if exists('+statuscolumn')
    call setwinvar(winid, '&statuscolumn', '')
  endif
  if exists('+smoothscroll')
    " Neovim freezes in a terminal with 'smoothscroll'.
    call setwinvar(winid, '&smoothscroll', 0)
  endif
  call setbufvar(a:bufnr, '&bufhidden', 'hide')
  call setbufvar(a:bufnr, '&swapfile', 0)
  " Twice: after the ftplugin of Vim too.
  call setbufvar(a:bufnr, '&filetype', 'goda-term-terminal')
  call setbufvar(a:bufnr, '&filetype', 'goda-term-terminal')
endfunction

" The current directory of the job {pid}, or that of the command line.
function! goda_term#ui#terminal#_get_cwd(pid, cmdline) abort
  let cwd = printf('/proc/%d/cwd', a:pid)
  if isdirectory(cwd)
    return resolve(cwd)
  elseif executable('lsof')
    return matchstr(system('lsof -a -d cwd -p ' . a:pid), '\f\+\ze\n$')
  endif
  return expand(matchstr(a:cmdline, '\W\%(cd\s\+\)\?\zs\%(\S\|\\\s\)\+$'))
endfunction

" The command line: line {lnum} without the prompt.
function! goda_term#ui#terminal#_command_line(pattern, lnum = '.') abort
  return substitute(getline(a:lnum), a:pattern, '', '')
endfunction

" What is typed before the cursor in the command line.
function! goda_term#ui#terminal#_input(pattern) abort
  let line = goda_term#ui#terminal#_command_line(a:pattern)
  return line[: col('.') - (mode() ==# 'n' ? 2 : 3)]
endfunction

" Send {keys} to the job.
function! goda_term#ui#terminal#_send(bufnr, jobid, keys) abort
  if has('nvim')
    call chansend(a:jobid, a:keys)
  else
    call term_sendkeys(a:bufnr, a:keys)
    redraw
  endif
endfunction

" Redraw the terminal: Vim does it in Terminal-Job mode, a moment later
" (many redraws in a row are one).
let s:timers = {}
function! goda_term#ui#terminal#_redraw(bufnr) abort
  if has('nvim')
    redraw
    return
  endif
  if has_key(s:timers, a:bufnr)
    call timer_stop(s:timers[a:bufnr])
  endif
  let s:timers[a:bufnr] = timer_start(75, {-> s:redraw(a:bufnr)})
endfunction

function! s:redraw(bufnr) abort
  silent! unlet s:timers[a:bufnr]
  let ids = win_findbuf(a:bufnr)
  if empty(ids) || mode() !=# 'n'
    return
  endif
  let winid = win_getid()
  call win_gotoid(ids[0])
  redraw
  silent! normal! A
  sleep 50m
  call feedkeys("\<C-\>\<C-n>", 'n')
  call win_gotoid(winid)
endfunction
