" goda-term.vim - terminals and shells in Vim/Neovim on goda.vim, the
" successor of ddt.vim.
"
" The Go side (the user's own build of a configuration, see the README) has
" the UIs; this file starts it and passes on what is asked. The UIs have
" their own Vim script (autoload/goda_term/ui/*.vim).

" The version of this Vim script that the binary goes with (ScriptVersion of
" its Go side): goda.vim refuses a binary of another one, and says to build
" it again. Bumped with it when they no longer work together.
let s:version = #{version: 1, module: 'github.com/Shougo/goda-term.vim'}

" Patches of the options made at run time, as ddt#custom#patch_*(). They
" live here, not in the binary: they survive a restart of it, and they can
" be made before it is started. Every request carries them.
let s:global_patch = {}
let s:local_patch = {}

" ---------------------------------------------------------------------------
" Setup

" The binary is machine specific, so it is not built into the configuration
" directory (which belongs in your dotfiles) but into a cache directory.
function! s:cache_dir() abort
  if has('nvim')
    let base = stdpath('cache')
  elseif !empty($XDG_CACHE_HOME)
    let base = $XDG_CACHE_HOME . '/vim'
  elseif has('win32') && !empty($LOCALAPPDATA)
    let base = $LOCALAPPDATA . '/vim'
  else
    let base = expand('~/.cache/vim')
  endif
  return base . '/goda-term'
endfunction

function! s:default_bin() abort
  if exists('g:goda_term#binary')
    return expand(g:goda_term#binary)
  endif
  return s:cache_dir() . '/goda-term' . (has('win32') ? '.exe' : '')
endfunction

function! s:config_dir() abort
  if exists('g:goda_term#config_dir')
    return expand(g:goda_term#config_dir)
  endif
  return (has('nvim') ? stdpath('config') : expand('~/.vim')) . '/goda-term'
endfunction

" Build the user's configuration (a Go module in g:goda_term#config_dir),
" after "go mod tidy". The running binary is stopped, and the new one starts
" with the next request. When the build fails, the previous binary keeps
" running.
function! goda_term#build() abort
  let dir = s:config_dir()
  if !isdirectory(dir)
    throw 'goda-term: no config directory: ' . dir
  endif
  let bin = s:default_bin()
  call mkdir(fnamemodify(bin, ':h'), 'p')
  let tidy = system('go -C ' . shellescape(dir) . ' mod tidy')
  let tidy_failed = v:shell_error
  let tmp = bin . '.new' . (has('win32') ? '.exe' : '')
  let out = system('go -C ' . shellescape(dir) . ' build -o ' . shellescape(tmp) . ' .')
  if v:shell_error
    throw 'goda-term: go build failed:' . "\n" . out
          \ . (tidy_failed ? "\n" . 'go mod tidy failed too:' . "\n" . tidy : '')
  endif
  if s:own_binary()
    call goda#stop('term')
  endif
  if rename(tmp, bin) != 0
    throw 'goda-term: cannot replace ' . bin
  endif
  call goda#register('term', function('s:start'), s:version)
  echomsg 'goda-term: built ' . bin
endfunction

" Is the plugin served by the binary of goda-term (not by another plugin's
" binary that contains it)?
function! s:own_binary() abort
  let plugins = goda#plugins()
  return has_key(plugins, 'term')
        \ && plugins.term ==# fnamemodify(s:default_bin(), ':p')
endfunction

" Start the binary: goda.vim calls it when a request needs the plugin "term"
" and no binary serves it.
function! s:start() abort
  let bin = s:default_bin()
  if !executable(bin)
    throw 'goda-term: ' . bin . ' does not exist yet; build it with: call goda_term#build()'
  endif
  let plugins = goda#start(bin)
  if index(plugins, 'term') < 0
    if !empty(plugins)
      call goda#stop(plugins[0])
    endif
    throw 'goda-term: ' . bin . ' does not contain the "term" plugin'
  endif
  " A binary built from an invalid configuration starts anyway and says why.
  let status = goda#request('term', 'status')
  if type(status) == v:t_string && status !=# ''
    call goda#stop('term')
    throw 'goda-term: fix the configuration in ' . s:config_dir()
          \ . ' and run goda_term#build(): ' . status
  endif
endfunction

call goda#register('term', function('s:start'), s:version)

function! s:print_error(msg) abort
  echohl ErrorMsg
  for line in split(a:msg, "\n")
    echomsg line
  endfor
  echohl None
endfunction

" ---------------------------------------------------------------------------
" API (that of ddt.vim)

" Show the session of these options (merged over the configuration and the
" patches): a new one, or the one of the name again.
function! goda_term#start(options = {}) abort
  try
    let context = #{cwd: getcwd(), bufNr: bufnr('%'), winId: win_getid()}
    call goda#request('term', 'start', a:options, context, s:patches())
  catch
    call s:print_error(v:exception)
  endtry
endfunction

" Run the UI action {action} of the session {name}.
function! goda_term#ui_action(name, action, params = {}) abort
  if a:name ==# ''
    return
  endif
  if getcmdwintype() !=# ''
    return
  endif
  try
    call goda#request('term', 'uiAction', a:name, a:action, a:params)
  catch
    call s:print_error(v:exception)
  endtry
endfunction

" The command line being typed in the session {name}.
function! goda_term#get_input(name) abort
  if a:name ==# '' || !goda#is_running('term')
    return ''
  endif
  return goda#request('term', 'getInput', a:name)
endfunction

" The names of the sessions.
function! goda_term#sessions() abort
  return goda#is_running('term') ? goda#request('term', 'sessions') : []
endfunction

" ---------------------------------------------------------------------------
" Patches: goda_term#custom#*() keep them here.

function! goda_term#_patch_global(patch, set = v:false) abort
  let s:global_patch = a:set ? deepcopy(a:patch) : s:merge(s:global_patch, a:patch)
endfunction

function! goda_term#_patch_local(name, patch, set = v:false) abort
  let s:local_patch[a:name] = a:set ? deepcopy(a:patch)
        \ : s:merge(get(s:local_patch, a:name, {}), a:patch)
endfunction

function! goda_term#_get_global() abort
  return deepcopy(s:global_patch)
endfunction

function! goda_term#_get_local() abort
  return deepcopy(s:local_patch)
endfunction

function! goda_term#_patches() abort
  return s:patches()
endfunction

function! s:patches() abort
  return #{global: s:global_patch, local: s:local_patch}
endfunction

" {dst} with {src} over it; the params of a UI are merged param by param.
function! s:merge(dst, src) abort
  let out = deepcopy(a:dst)
  for [key, value] in items(a:src)
    if key ==# 'uiParams' && type(value) == v:t_dict
      let params = get(out, key, {})
      for [ui, p] in items(value)
        let params[ui] = extend(get(params, ui, {}), p)
      endfor
      let out[key] = params
    else
      let out[key] = deepcopy(value)
    endif
  endfor
  return out
endfunction
