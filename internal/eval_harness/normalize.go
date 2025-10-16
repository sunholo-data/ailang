package eval_harness

import (
	"fmt"
	"regexp"
	"strings"
)

// RepairLog tracks transformations applied by normalizeProgram
type RepairLog struct {
	Wrapped       bool     // True if bare expression was wrapped in module scaffold
	AddedModule   bool     // True if module declaration was added
	AddedImports  []string // List of imports that were injected
	CallFixes     int      // Number of bare function calls that were fixed with parens
	AddedMainFunc bool     // True if main function was synthesized
}

// normalizeProgram ensures AI-generated code has proper AILANG structure
// This fixes common issues where models emit:
// - Bare expressions (no module/main)
// - Missing imports (std/io)
// - Bare function calls like "print 5" instead of "print(5)"
func normalizeProgram(src string, caps []string) (string, RepairLog) {
	log := RepairLog{}

	// Fix bare function calls FIRST (before wrapping): "print 5 % 3" → "print(5 % 3)"
	// Only fix print/println to avoid breaking valid syntax
	fixed, n := fixBarePrintCalls(src)
	if n > 0 {
		src = fixed
		log.CallFixes = n
	}

	// Detect if this is a bare expression or incomplete program
	hasModule := strings.Contains(src, "module ")
	hasMain := strings.Contains(src, "func main(")

	// If no module structure at all, wrap as complete module
	if !hasModule || !hasMain {
		src = wrapAsModule(src, caps, hasModule)
		log.Wrapped = true
		if !hasModule {
			log.AddedModule = true
		}
		if !hasMain {
			log.AddedMainFunc = true
		}
	}

	// Ensure std/io import if code uses print/println/readLine
	if needsIO(src) && !hasIOImport(src) {
		src = ensureIOImport(src)
		log.AddedImports = append(log.AddedImports, "std/io")
	}

	return src, log
}

// wrapAsModule wraps code in a complete module structure
func wrapAsModule(src string, caps []string, hasModule bool) string {
	// Determine if main should have IO effects
	hasIO := len(caps) == 0 || containsString(caps, "IO")
	effectSig := ""
	if hasIO {
		effectSig = " ! {IO}"
	}

	var sb strings.Builder

	// Add module declaration if missing
	if !hasModule {
		sb.WriteString("module benchmark/solution\n\n")
	}

	// Add imports (namespace imports not yet supported, use whole module)
	if hasIO && needsIO(src) {
		sb.WriteString("import std/io\n")
	}
	sb.WriteString("\n")

	// Check if source already has function definitions
	hasFunc := regexp.MustCompile(`(?m)^\s*(?:export\s+)?func\s+\w+`).MatchString(src)

	if hasFunc {
		// Source has functions - just add main if missing
		if !strings.Contains(src, "func main(") {
			// Assume source has helper functions, just need to add main
			sb.WriteString(src)
			sb.WriteString("\n\nexport func main() -> ()")
			sb.WriteString(effectSig)
			sb.WriteString(" {\n")
			sb.WriteString("  () // TODO: Call your functions here\n")
			sb.WriteString("}\n")
		} else {
			// Has functions including main
			sb.WriteString(src)
		}
	} else {
		// Source is just an expression - wrap in main with println
		sb.WriteString("export func main() -> ()")
		sb.WriteString(effectSig)
		sb.WriteString(" {\n")
		sb.WriteString("  println(show(\n")
		sb.WriteString("    (")
		// Indent the expression
		indented := strings.ReplaceAll(strings.TrimSpace(src), "\n", "\n      ")
		sb.WriteString(indented)
		sb.WriteString(")\n")
		sb.WriteString("  ))\n")
		sb.WriteString("}\n")
	}

	return sb.String()
}

// needsIO checks if code uses IO functions
func needsIO(src string) bool {
	ioFuncs := []string{"print(", "println(", "readLine(", "print ", "println ", "readLine "}
	for _, fn := range ioFuncs {
		if strings.Contains(src, fn) {
			return true
		}
	}
	return false
}

// hasIOImport checks if std/io is already imported
func hasIOImport(src string) bool {
	return regexp.MustCompile(`import\s+std/io`).MatchString(src)
}

// ensureIOImport adds std/io import after module declaration
func ensureIOImport(src string) string {
	// Find module line
	moduleRe := regexp.MustCompile(`(?m)^module\s+\S+\s*$`)
	loc := moduleRe.FindStringIndex(src)
	if loc == nil {
		// No module line, add import at top
		return "import std/io\n" + src
	}

	// Insert after module line
	insertPos := loc[1]
	return src[:insertPos] + "\nimport std/io" + src[insertPos:]
}

// fixBarePrintCalls converts "print expr" to "print(expr)"
// Only fixes safe patterns to avoid breaking valid code
func fixBarePrintCalls(src string) (string, int) {
	count := 0

	// Pattern: start of line, optional whitespace, print/println, space, then non-paren content
	// This matches: "print 5 % 3" or "println x + y"
	// But NOT: "print(5)" or "println(x)"
	pattern := regexp.MustCompile(`(?m)^(\s*)(print|println)(\s+)([^(\s][^\n]*)$`)

	result := pattern.ReplaceAllStringFunc(src, func(match string) string {
		parts := pattern.FindStringSubmatch(match)
		if len(parts) != 5 {
			return match
		}
		indent := parts[1]
		funcName := parts[2]
		// parts[3] is the space
		arg := parts[4]

		count++
		return fmt.Sprintf("%s%s(%s)", indent, funcName, arg)
	})

	return result, count
}

// containsString checks if a slice contains a string
func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
