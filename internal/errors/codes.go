// Package errors provides centralized error code definitions for AILANG.
// All error codes follow a consistent taxonomy for AI-friendly error reporting.
package errors

// Error code constants organized by phase.
// Each constant represents a specific error condition with structured reporting.
const (
	// ============================================================================
	// Parser Errors (PAR###)
	// ============================================================================

	// PAR001 indicates an unexpected token was encountered during parsing
	PAR001 = "PAR001"

	// PAR002 indicates a missing closing delimiter (paren, bracket, brace)
	PAR002 = "PAR002"

	// PAR003 indicates invalid function declaration syntax
	PAR003 = "PAR003"

	// PAR004 indicates invalid module declaration syntax
	PAR004 = "PAR004"

	// PAR005 indicates invalid import statement syntax
	PAR005 = "PAR005"

	// PAR006 indicates invalid test block syntax
	PAR006 = "PAR006"

	// PAR007 indicates invalid property block syntax
	PAR007 = "PAR007"

	// PAR008 indicates invalid pattern match syntax
	PAR008 = "PAR008"

	// PAR009 indicates invalid type annotation syntax
	PAR009 = "PAR009"

	// PAR010 indicates unexpected token after func signature (likely missing {)
	PAR010 = "PAR010"

	// PAR011 indicates missing ) in parameter list
	PAR011 = "PAR011"

	// PAR012 indicates reserved keyword used as identifier
	PAR012 = "PAR012"

	// PAR013 indicates unexpected token before tests/properties (common missing })
	PAR013 = "PAR013"

	// PAR999 indicates internal parser error/panic
	PAR999 = "PAR999"

	// ============================================================================
	// Module System Errors (MOD###)
	// ============================================================================

	// MOD001 indicates module name doesn't match file path
	MOD001 = "MOD001"

	// MOD002 indicates multiple module declarations in single file
	MOD002 = "MOD002"

	// MOD003 indicates unsupported re-export attempt
	MOD003 = "MOD003"

	// MOD004 indicates duplicate export in module
	MOD004 = "MOD004"

	// MOD005 indicates invalid module path format
	MOD005 = "MOD005"

	// MOD006-MOD009 reserved for future use

	// MOD010 indicates module/path mismatch (e.g., module foo/bar but file is foo/baz.ail)
	MOD010 = "MOD010"

	// MOD_EXPORT_PRIVATE indicates attempted export of underscore-prefixed name
	MOD_EXPORT_PRIVATE = "MOD_EXPORT_PRIVATE"

	// MOD011 indicates multiple module declarations per file
	MOD011 = "MOD011"

	// MOD012 indicates implicit module (file without module declaration)
	MOD012 = "MOD012"

	// ============================================================================
	// Loader Errors (LDR###)
	// ============================================================================

	// LDR001 indicates module file not found
	LDR001 = "LDR001"

	// LDR002 indicates circular module dependency detected
	LDR002 = "LDR002"

	// LDR003 indicates duplicate module definition
	LDR003 = "LDR003"

	// LDR004 indicates import of non-existent export
	LDR004 = "LDR004"

	// LDR005 indicates ambiguous import (multiple modules export same name)
	LDR005 = "LDR005"

	// ============================================================================
	// Import Errors (IMP###)
	// ============================================================================

	// IMP001 indicates invalid import syntax
	IMP001 = "IMP001"

	// IMP002 indicates import of non-existent module
	IMP002 = "IMP002"

	// IMP003 indicates cyclic import dependency
	IMP003 = "IMP003"

	// IMP004 indicates invalid selective import syntax
	IMP004 = "IMP004"

	// IMP005 indicates importing from non-module file
	IMP005 = "IMP005"

	// IMP006-IMP009 reserved for future use

	// IMP010 indicates unknown import form (reject aliasing for MVP)
	IMP010 = "IMP010"

	// IMP011 indicates unknown symbol in selective import
	IMP011 = "IMP011"

	// IMP012 indicates duplicate binding introduced by import
	IMP012 = "IMP012"

	// ============================================================================
	// Desugaring Errors (DSG###)
	// ============================================================================

	// DSG001 indicates invalid desugaring transformation
	DSG001 = "DSG001"

	// DSG002 indicates alpha-renaming conflict
	DSG002 = "DSG002"

	// DSG003 indicates recursive function without proper binding
	DSG003 = "DSG003"

	// DSG004-DSG009 reserved for future use

	// DSG010 indicates pure func calls effectful things (warning placeholder)
	DSG010 = "DSG010"

	// ============================================================================
	// Type Checking Errors (TC###) - Already defined in json_encoder.go
	// ============================================================================
	// TC001-TC007 defined in json_encoder.go

	// TC008 indicates recursive type without base case
	TC008 = "TC008"

	// TC009 indicates effect constraint violation
	TC009 = "TC009"

	// TC010 indicates missing type class instance
	TC010 = "TC010"

	// ============================================================================
	// Elaboration Errors (ELB###) - Already defined in json_encoder.go
	// ============================================================================
	// ELB001-ELB004 defined in json_encoder.go

	// ELB005 indicates invalid Core AST structure after elaboration
	ELB005 = "ELB005"

	// ELB006 indicates failed ANF normalization
	ELB006 = "ELB006"

	// ============================================================================
	// Linking Errors (LNK###) - Already defined in json_encoder.go
	// ============================================================================
	// LNK001-LNK004 defined in json_encoder.go

	// LNK005 indicates version mismatch in linked modules
	LNK005 = "LNK005"

	// ============================================================================
	// Evaluation Errors (EVA###)
	// ============================================================================

	// EVA001 indicates unbound variable at runtime
	EVA001 = "EVA001"

	// EVA002 indicates pattern match failure at runtime
	EVA002 = "EVA002"

	// EVA003 indicates type assertion failed
	EVA003 = "EVA003"

	// EVA004 indicates effect capability not provided
	EVA004 = "EVA004"

	// EVA005 indicates infinite recursion detected
	EVA005 = "EVA005"

	// ============================================================================
	// Runtime Errors (RT###) - Already defined in json_encoder.go
	// ============================================================================
	// RT001-RT006 defined in json_encoder.go

	// RT007 indicates out of memory
	RT007 = "RT007"

	// RT008 indicates timeout exceeded
	RT008 = "RT008"
)

// ErrorInfo provides structured information about an error code
type ErrorInfo struct {
	Code        string
	Phase       string
	Category    string
	Description string
}

// ErrorRegistry maps error codes to their information
var ErrorRegistry = map[string]ErrorInfo{
	// Parser errors
	PAR001: {PAR001, "parser", "syntax", "Unexpected token"},
	PAR002: {PAR002, "parser", "syntax", "Missing closing delimiter"},
	PAR003: {PAR003, "parser", "syntax", "Invalid function declaration"},
	PAR004: {PAR004, "parser", "syntax", "Invalid module declaration"},
	PAR005: {PAR005, "parser", "syntax", "Invalid import statement"},
	PAR006: {PAR006, "parser", "syntax", "Invalid test block"},
	PAR007: {PAR007, "parser", "syntax", "Invalid property block"},
	PAR008: {PAR008, "parser", "syntax", "Invalid pattern match"},
	PAR009: {PAR009, "parser", "syntax", "Invalid type annotation"},
	PAR010: {PAR010, "parser", "syntax", "Unexpected token after func signature"},
	PAR011: {PAR011, "parser", "syntax", "Missing ) in parameter list"},
	PAR012: {PAR012, "parser", "syntax", "Reserved keyword as identifier"},
	PAR013: {PAR013, "parser", "syntax", "Unexpected token before tests/properties"},

	// Module errors
	MOD001: {MOD001, "module", "structure", "Module name/path mismatch"},
	MOD002: {MOD002, "module", "structure", "Multiple modules per file"},
	MOD003: {MOD003, "module", "feature", "Re-export not supported"},
	MOD004: {MOD004, "module", "namespace", "Duplicate export"},
	MOD005: {MOD005, "module", "syntax", "Invalid module path"},
	MOD010:            {MOD010, "module", "validation", "Module/path mismatch"},
	MOD011:            {MOD011, "module", "structure", "Multiple module declarations"},
	MOD012:            {MOD012, "module", "structure", "Implicit module warning"},
	MOD_EXPORT_PRIVATE: {MOD_EXPORT_PRIVATE, "module", "validation", "Export of private (underscore) name"},

	// Loader errors
	LDR001: {LDR001, "loader", "resolution", "Module not found"},
	LDR002: {LDR002, "loader", "dependency", "Circular dependency"},
	LDR003: {LDR003, "loader", "namespace", "Duplicate module"},
	LDR004: {LDR004, "loader", "resolution", "Import not exported"},
	LDR005: {LDR005, "loader", "resolution", "Ambiguous import"},

	// Import errors
	IMP001: {IMP001, "import", "syntax", "Invalid import syntax"},
	IMP002: {IMP002, "import", "resolution", "Module not found"},
	IMP003: {IMP003, "import", "dependency", "Cyclic import"},
	IMP004: {IMP004, "import", "syntax", "Invalid selective import"},
	IMP005: {IMP005, "import", "validation", "Non-module import"},
	IMP010: {IMP010, "import", "feature", "Unknown import form"},
	IMP011: {IMP011, "import", "resolution", "Unknown symbol in import"},
	IMP012: {IMP012, "import", "namespace", "Duplicate binding from import"},

	// Desugar errors
	DSG001: {DSG001, "desugar", "transform", "Invalid desugaring"},
	DSG002: {DSG002, "desugar", "scope", "Alpha-renaming conflict"},
	DSG003: {DSG003, "desugar", "recursion", "Invalid recursive binding"},
	DSG010: {DSG010, "desugar", "purity", "Pure func calls effectful (warning)"},

	// Type checking errors
	TC001: {TC001, "typecheck", "type", "Type mismatch"},
	TC002: {TC002, "typecheck", "scope", "Unbound variable"},
	TC003: {TC003, "typecheck", "constraint", "Constraint solving failed"},
	TC004: {TC004, "typecheck", "unification", "Occurs check failed"},
	TC005: {TC005, "typecheck", "kind", "Kind mismatch"},
	TC006: {TC006, "typecheck", "annotation", "Missing type annotation"},
	TC007: {TC007, "typecheck", "defaulting", "Defaulting ambiguity"},
	TC008: {TC008, "typecheck", "recursion", "Non-terminating type"},
	TC009: {TC009, "typecheck", "effect", "Effect constraint violated"},
	TC010: {TC010, "typecheck", "instance", "Missing type class instance"},

	// Elaboration errors
	ELB001: {ELB001, "elaborate", "structure", "Invalid AST structure"},
	ELB002: {ELB002, "elaborate", "dictionary", "Dictionary resolution failed"},
	ELB003: {ELB003, "elaborate", "transform", "ANF transformation error"},
	ELB004: {ELB004, "elaborate", "pattern", "Non-exhaustive pattern"},
	ELB005: {ELB005, "elaborate", "validation", "Invalid Core AST"},
	ELB006: {ELB006, "elaborate", "normalize", "ANF normalization failed"},

	// Linking errors
	LNK001: {LNK001, "link", "instance", "Missing dictionary instance"},
	LNK002: {LNK002, "link", "instance", "Ambiguous instance"},
	LNK003: {LNK003, "link", "module", "Module not found"},
	LNK004: {LNK004, "link", "dependency", "Circular dependency"},
	LNK005: {LNK005, "link", "version", "Version mismatch"},

	// Evaluation errors
	EVA001: {EVA001, "eval", "scope", "Unbound variable"},
	EVA002: {EVA002, "eval", "pattern", "Pattern match failure"},
	EVA003: {EVA003, "eval", "type", "Type assertion failed"},
	EVA004: {EVA004, "eval", "effect", "Missing capability"},
	EVA005: {EVA005, "eval", "recursion", "Infinite recursion"},

	// Runtime errors
	RT001: {RT001, "runtime", "arithmetic", "Division by zero"},
	RT002: {RT002, "runtime", "pattern", "Pattern match failure"},
	RT003: {RT003, "runtime", "bounds", "Index out of bounds"},
	RT004: {RT004, "runtime", "null", "Null pointer"},
	RT005: {RT005, "runtime", "stack", "Stack overflow"},
	RT006: {RT006, "runtime", "type", "Type assertion failed"},
	RT007: {RT007, "runtime", "memory", "Out of memory"},
	RT008: {RT008, "runtime", "timeout", "Timeout exceeded"},
}

// GetErrorInfo returns information about an error code
func GetErrorInfo(code string) (ErrorInfo, bool) {
	info, exists := ErrorRegistry[code]
	return info, exists
}

// IsParserError checks if the error code is a parser error
func IsParserError(code string) bool {
	info, exists := GetErrorInfo(code)
	return exists && info.Phase == "parser"
}

// IsModuleError checks if the error code is a module error
func IsModuleError(code string) bool {
	info, exists := GetErrorInfo(code)
	return exists && info.Phase == "module"
}

// IsLoaderError checks if the error code is a loader error
func IsLoaderError(code string) bool {
	info, exists := GetErrorInfo(code)
	return exists && info.Phase == "loader"
}

// IsTypeError checks if the error code is a type checking error
func IsTypeError(code string) bool {
	info, exists := GetErrorInfo(code)
	return exists && info.Phase == "typecheck"
}

// IsRuntimeError checks if the error code is a runtime error
func IsRuntimeError(code string) bool {
	info, exists := GetErrorInfo(code)
	return exists && (info.Phase == "runtime" || info.Phase == "eval")
}

// IsImportError checks if the error code is an import error
func IsImportError(code string) bool {
	info, exists := GetErrorInfo(code)
	return exists && info.Phase == "import"
}

// IsDesugarError checks if the error code is a desugaring error
func IsDesugarError(code string) bool {
	info, exists := GetErrorInfo(code)
	return exists && info.Phase == "desugar"
}

// IsElaborationError checks if the error code is an elaboration error
func IsElaborationError(code string) bool {
	info, exists := GetErrorInfo(code)
	return exists && info.Phase == "elaborate"
}

// IsLinkError checks if the error code is a linking error
func IsLinkError(code string) bool {
	info, exists := GetErrorInfo(code)
	return exists && info.Phase == "link"
}
