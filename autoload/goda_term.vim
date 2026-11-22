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

let s:goda = goda#configured#new('term', s:version)

" Build the user's configuration (a Go module in g:goda_term#config_dir) with
" goda#build(): with the plugins installed (g:goda#workspace). The running
" binary is stopped, and the new one starts with the next request. When the
" build fails, the previous binary keeps running.
function! goda_term#build() abort
  call s:goda.build()
  call s:goda.register()
endfunction

call s:goda.register()

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
