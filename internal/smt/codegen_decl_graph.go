package smt

import "strings"

type pendingDecl struct {
	sortName string
	decl     string
}

func registerRecordAlias(aliasName string, fieldSorts map[string]string) {
	fieldNames := SortedFieldNamesStr(fieldSorts)
	activeRecordTypes[aliasName] = &RecordTypeInfo{
		SortName:   aliasName,
		CtorName:   RecordConstructorName(aliasName),
		FieldNames: fieldNames,
		FieldSorts: fieldSorts,
	}
	activeFieldSetToSort[strings.Join(fieldNames, ",")] = aliasName
}
