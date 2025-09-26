" Vim syntax file
" Language: AILANG
" Maintainer: AILANG Team
" Latest Revision: 2025

if exists("b:current_syntax")
  finish
endif

" Keywords
syn keyword ailangKeyword let var func pure import module export type
syn keyword ailangKeyword interface typeclass instance property forall exists
syn keyword ailangKeyword parallel spawn select yield await
syn keyword ailangConditional if then else match with
syn keyword ailangRepeat loop break continue
syn keyword ailangException try catch finally throw
syn keyword ailangOperator and or not is as in

" Built-in types
syn keyword ailangType int float bool string char unit never
syn keyword ailangType Result Option Channel Effect Session

" Built-in functions
syn keyword ailangBuiltin print readFile writeFile httpGet httpPost
syn keyword ailangBuiltin length map filter fold reduce sort reverse
syn keyword ailangBuiltin head tail cons append

" Comments
syn match ailangComment "--.*$"

" Strings
syn region ailangString start='"' end='"' contains=ailangEscape
syn region ailangString start="'" end="'" contains=ailangEscape
syn match ailangEscape contained "\\."

" Quasiquotes
syn region ailangSQL start="sql\"\"\"" end="\"\"\""
syn region ailangHTML start="html\"\"\"" end="\"\"\""
syn region ailangShell start="shell\"\"\"" end="\"\"\""
syn region ailangRegex start="regex/" end="/[gimsu]*"
syn region ailangJSON start="json{" end="}"
syn region ailangURL start="url\"" end="\""

" Numbers
syn match ailangNumber "\<\d\+\>"
syn match ailangNumber "\<\d\+\.\d\+\>"
syn match ailangNumber "\<0x[0-9a-fA-F]\+\>"

" Operators
syn match ailangOperator "[+\-*/%]"
syn match ailangOperator "[<>=!]="
syn match ailangOperator "[<>]"
syn match ailangOperator "&&\|||"
syn match ailangOperator "->\|=>\|<-"
syn match ailangOperator "|>"
syn match ailangOperator "<<"
syn match ailangOperator "?"
syn match ailangOperator "!"
syn match ailangOperator "\.\.\."

" Type names (capitalized identifiers)
syn match ailangTypeName "\<[A-Z][A-Za-z0-9]*\>"

" Function names
syn match ailangFunction "\<[a-z][A-Za-z0-9]*\>\s*("

" Highlighting
hi def link ailangKeyword Keyword
hi def link ailangConditional Conditional
hi def link ailangRepeat Repeat
hi def link ailangException Exception
hi def link ailangOperator Operator
hi def link ailangType Type
hi def link ailangTypeName Type
hi def link ailangBuiltin Function
hi def link ailangFunction Function
hi def link ailangComment Comment
hi def link ailangString String
hi def link ailangEscape SpecialChar
hi def link ailangSQL String
hi def link ailangHTML String
hi def link ailangShell String
hi def link ailangRegex String
hi def link ailangJSON String
hi def link ailangURL String
hi def link ailangNumber Number

let b:current_syntax = "ailang"