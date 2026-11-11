package types

// CoreTypeChecker setters added by M-V1-NOMINAL-TYPE-IDENTITY (kept apart from
// typechecker_core.go, which is at the file-size limit).

// SetConstructorModules sets constructor → defining module of its ADT, the
// module half of SetConstructorTypes (M-V1-NOMINAL-TYPE-IDENTITY).
func (tc *CoreTypeChecker) SetConstructorModules(mods map[string]string) {
	tc.ctorADTModule = mods
}

// RegisterNewtypeADT marks an alias key as a single-constructor ADT wrapping a
// record (M-STREAM-DX/M4): it expands only against a record, never against
// another type constructor (M-V1-NOMINAL-TYPE-IDENTITY).
func (tc *CoreTypeChecker) RegisterNewtypeADT(key string) {
	if tc.newtypes == nil {
		tc.newtypes = make(map[string]bool)
	}
	tc.newtypes[key] = true
}
