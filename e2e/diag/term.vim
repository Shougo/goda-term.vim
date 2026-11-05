" A terminal job of Neovim without goda-term (issue #1): for each command,
" what jobstart({term: v:true}) gives, written as JSON lines to $DIAG_OUT.
" Run with nvim --headless -S, or sourced by ui.py (a UI attached).
let s:cmds = has('win32')
      \ ? [['cmd.exe'], ['cmd.exe', '/d', '/k'], ['cmd.exe', '/c', 'echo hello'],
      \    ['powershell.exe', '-NoLogo', '-NoExit']]
      \ : [['sh'], ['sh', '-i'], ['sh', '-c', 'echo hello']]
let s:mode = $DIAG_MODE !=# '' ? $DIAG_MODE : 'headless'

function! s:on_exit(st, id, code, event) abort
  let a:st.exit = {'code': a:code, 'event': a:event,
        \ 'ms': float2nr(reltimefloat(reltime(a:st.start)) * 1000)}
endfunction

function! s:on_stdout(st, id, data, event) abort
  if len(a:st.stdout) < 2000
    let a:st.stdout .= join(a:data, "\n")
  endif
endfunction

function! s:lines() abort
  return filter(getline(1, '$'), 'v:val !=# ""')
endfunction

function! s:run(cmd) abort
  enew!
  let st = {'mode': s:mode, 'nvim': luaeval('tostring(vim.version())'),
        \ 'uis': len(nvim_list_uis()), 'cmd': a:cmd, 'stdout': '', 'exit': v:null,
        \ 'start': reltime()}
  try
    let st.id = jobstart(a:cmd, {'term': v:true,
          \ 'on_exit': function('s:on_exit', [st]),
          \ 'on_stdout': function('s:on_stdout', [st])})
  catch
    let st.id = -100
    let st.error = v:exception
  endtry
  let st.terminal_job_id = get(b:, 'terminal_job_id', -1)
  if st.id > 0
    try
      let st.pid = jobpid(st.id)
    catch
      let st.pid = v:exception
    endtry
    call wait(1500, {-> st.exit isnot v:null})
    let st.jobwait_1500ms = jobwait([st.id], 0)[0]
    let st.lines_1500ms = s:lines()
    if st.exit is v:null
      " Alive: does it run what it is sent?
      try
        call chansend(st.id, "echo hello\r")
      catch
        let st.chansend = v:exception
      endtry
      call wait(2000, {-> match(s:lines(), 'hello') >= 0 || st.exit isnot v:null})
    endif
    let st.jobwait_end = jobwait([st.id], 0)[0]
    let st.lines_end = s:lines()[: 20]
    " The exit by itself (null: alive after 3.5 seconds, then stopped).
    let st.exited = st.exit
    if st.exit is v:null
      call jobstop(st.id)
      call wait(1000, {-> st.exit isnot v:null})
    endif
  endif
  let st.exited = get(st, 'exited', st.exit)
  unlet st.start st.exit
  return st
endfunction

let s:out = []
for s:cmd in s:cmds
  call add(s:out, json_encode(s:run(s:cmd)))
endfor
call add(s:out, json_encode({'mode': s:mode, 'nvim': luaeval('tostring(vim.version())'),
      \ 'messages': execute('messages')}))
call writefile(s:out, $DIAG_OUT, 'a')
