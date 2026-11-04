" The options patched at run time, as ddt#custom#*(). (The configuration
" itself is Go: main.go, see the README.)

function! goda_term#custom#patch_global(key_or_dict, value = '') abort
  call goda_term#_patch_global(s:dict(a:key_or_dict, a:value))
endfunction

function! goda_term#custom#patch_local(name, key_or_dict, value = '') abort
  call goda_term#_patch_local(a:name, s:dict(a:key_or_dict, a:value))
endfunction

function! goda_term#custom#set_global(dict) abort
  call goda_term#_patch_global(a:dict, v:true)
endfunction

function! goda_term#custom#set_local(name, dict) abort
  call goda_term#_patch_local(a:name, a:dict, v:true)
endfunction

function! goda_term#custom#get_global() abort
  return goda_term#_get_global()
endfunction

function! goda_term#custom#get_local() abort
  return goda_term#_get_local()
endfunction

" The options of the session {name}: those it was started with, or those a
" start of it would have.
function! goda_term#custom#get_current(name = get(b:, 'goda_term_name', '')) abort
  if a:name ==# ''
    return {}
  endif
  return goda#request('term', 'options', a:name, goda_term#_patches())
endfunction

function! s:dict(key_or_dict, value) abort
  return type(a:key_or_dict) == v:t_dict ? a:key_or_dict : {a:key_or_dict : a:value}
endfunction
