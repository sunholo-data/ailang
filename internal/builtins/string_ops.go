package builtins

// String operation builtins — split across focused files for AI-maintainability:
//
//   - string_ops_parse.go     : Parse/split builtins (_stringToInt, _stringToFloat, _str_split)
//   - string_ops_search.go    : Prefix/suffix/char search (_str_chars, _str_startsWith, _str_endsWith, _str_startsWithIC)
//   - string_ops_transform.go : Transform builtins (_string_reverse, _str_join, _str_replace, _str_replaceMany,
//                                                   _str_foldSlices, _str_mapSlicesJoin, _str_words, _str_splitAny)
//
// Registration is performed in string.go init().
