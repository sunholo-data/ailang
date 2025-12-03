" Vim syntax file
" Language: AILANG
" Maintainer: AILANG Team
" Latest Revision: 2025

if exists("b:current_syntax")
  finish
endif

" Comments (both -- and // styles)
syn match ailangComment "--.*$" contains=ailangTodo
syn match ailangComment "//.*$" contains=ailangTodo
syn keyword ailangTodo TODO FIXME XXX NOTE contained

" Keywords - Control flow
syn keyword ailangControl if then else match with

" Keywords - Bindings
syn keyword ailangKeyword let letrec in func pure export import module extern as

" Keywords - Types
syn keyword ailangTypeKeyword type class instance forall exists

" Keywords - Testing
syn keyword ailangTesting test tests property properties assert

" Keywords - Concurrency (future)
syn keyword ailangConcurrency spawn parallel select channel send recv timeout

" Boolean literals
syn keyword ailangBoolean true false

" Built-in types
syn keyword ailangType int float bool string char unit

" Effect types
syn keyword ailangEffect IO FS Net Clock Env Rand Debug AI

" Standard library types
syn keyword ailangStdType Option Result List Tuple Array Json Some None Ok Err

" Common prelude functions
syn keyword ailangBuiltin print println show intToFloat floatToInt

" Numbers
syn match ailangNumber "\<0x[0-9a-fA-F]\+\>"
syn match ailangFloat "\<\d\+\.\d\+\([eE][+-]\?\d\+\)\?\>"
syn match ailangNumber "\<\d\+\>"

" Strings
syn region ailangString start=+"+ skip=+\\\\\|\\"+ end=+"+ contains=ailangEscape
syn match ailangChar "'\([^'\\]\|\\[nrt0'\\]\)'"
syn match ailangEscape "\\[nrt0\"'\\]" contained

" Operators
syn match ailangOperator "=>"
syn match ailangOperator "->"
syn match ailangOperator "<-"
syn match ailangOperator "::"
syn match ailangOperator "++"
syn match ailangOperator "&&"
syn match ailangOperator "||"
syn match ailangOperator "=="
syn match ailangOperator "!="
syn match ailangOperator "<="
syn match ailangOperator ">="
syn match ailangOperator "\\"
syn match ailangOperator "!"
syn match ailangOperator "|"
syn match ailangOperator "[+\-*/%<>=]"

" Type names (capitalized)
syn match ailangTypeName "\<[A-Z][a-zA-Z0-9]*\>"

" Function definitions
syn match ailangFuncDef "\<func\s\+\zs[a-z_][a-zA-Z0-9_]*"

" Module paths
syn match ailangModule "\<module\s\+\zs[a-zA-Z][a-zA-Z0-9_/]*"
syn match ailangImport "\<import\s\+\zs[a-zA-Z][a-zA-Z0-9_/]*"

" Highlighting groups
hi def link ailangComment Comment
hi def link ailangTodo Todo
hi def link ailangControl Conditional
hi def link ailangKeyword Keyword
hi def link ailangTypeKeyword Keyword
hi def link ailangTesting Special
hi def link ailangConcurrency Keyword
hi def link ailangBoolean Boolean
hi def link ailangType Type
hi def link ailangEffect Type
hi def link ailangStdType Type
hi def link ailangTypeName Type
hi def link ailangNumber Number
hi def link ailangFloat Float
hi def link ailangString String
hi def link ailangChar Character
hi def link ailangEscape SpecialChar
hi def link ailangOperator Operator
hi def link ailangBuiltin Function
hi def link ailangFuncDef Function
hi def link ailangModule Include
hi def link ailangImport Include

let b:current_syntax = "ailang"
