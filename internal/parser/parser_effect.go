package parser

import (
	"fmt"
	"strings"

	"github.com/sunholo/ailang/internal/lexer"
)

// parseEffectAnnotation parses effect annotations: ! {IO, FS, Net}
// Validates effect names and detects duplicates
func (p *Parser) parseEffectAnnotation() []string {
	// Known canonical effect names
	knownEffects := map[string]bool{
		"IO":    true,
		"FS":    true,
		"Net":   true,
		"Clock": true,
		"Rand":  true,
		"DB":    true,
		"Trace": true,
		"Async": true,
	}

	// We're at the BANG token
	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	effects := []string{}
	seen := make(map[string]bool)

	// Parse comma-separated effect names
	for !p.peekTokenIs(lexer.RBRACE) && !p.peekTokenIs(lexer.EOF) {
		p.nextToken()

		if !p.curTokenIs(lexer.IDENT) {
			p.report("PAR_EFF004_INVALID",
				"effect name must be an identifier",
				"Use one of: IO, FS, Net, Clock, Rand, DB, Trace, Async")
			continue
		}

		effectName := p.curToken.Literal

		// Check for unknown effects
		if !knownEffects[effectName] {
			// Try to suggest closest match
			suggestion := p.suggestEffect(effectName, knownEffects)
			fix := fmt.Sprintf("Did you mean '%s'?", suggestion)
			p.report("PAR_EFF002_UNKNOWN",
				fmt.Sprintf("unknown effect '%s'", effectName),
				fix)
			// Continue parsing to find more errors
		}

		// Check for duplicates
		if seen[effectName] {
			p.report("PAR_EFF001_DUP",
				fmt.Sprintf("duplicate effect '%s' in annotation", effectName),
				fmt.Sprintf("Remove duplicate '%s'", effectName))
		} else {
			seen[effectName] = true
			effects = append(effects, effectName)
		}

		// Check for comma or closing brace
		if p.peekTokenIs(lexer.RBRACE) {
			break
		}

		if !p.expectPeek(lexer.COMMA) {
			p.reportExpected(lexer.COMMA, "Add ',' between effect names")
			break
		}
	}

	if !p.expectPeek(lexer.RBRACE) {
		p.reportExpected(lexer.RBRACE, "Add '}' to close effect annotation")
	}

	return effects
}

// suggestEffect finds closest matching effect name (simple heuristic)
func (p *Parser) suggestEffect(name string, known map[string]bool) string {
	name = strings.ToLower(name)

	// Check exact match ignoring case
	for k := range known {
		if strings.ToLower(k) == name {
			return k
		}
	}

	// Check prefix match
	for k := range known {
		if strings.HasPrefix(strings.ToLower(k), name) {
			return k
		}
	}

	// Default to IO as most common
	return "IO"
}
