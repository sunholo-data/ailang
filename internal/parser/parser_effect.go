package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/lexer"
)

// parseEffectAnnotation parses effect annotations: ! {IO, FS, Net}
// Also supports budget syntax: ! {IO @limit=5, FS @limit=2}
// And minimum syntax: ! {IO @min=1} or ! {IO @min=1 @limit=5}
// Validates effect names and detects duplicates
func (p *Parser) parseEffectAnnotation() []ast.EffectAnnotation {
	// Known canonical effect names
	knownEffects := map[string]bool{
		"IO":          true,
		"FS":          true,
		"Net":         true,
		"Clock":       true,
		"Rand":        true,
		"DB":          true,
		"Trace":       true,
		"Async":       true,
		"Env":         true, // Environment variable access (v0.4.0+)
		"Debug":       true, // Structured tracing/assertions (v0.4.10+, ghost effect)
		"AI":          true, // General-purpose AI oracle (v0.5.1+)
		"SharedMem":   true, // Shared memory cache (v0.5.11+, M-DX15)
		"SharedIndex": true, // Similarity index for semantic retrieval (v0.5.11+, M-DX16)
	}

	// We're at the BANG token
	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	effects := []ast.EffectAnnotation{}
	seen := make(map[string]bool)

	// Parse comma-separated effect names with optional budgets
	for !p.peekTokenIs(lexer.RBRACE) && !p.peekTokenIs(lexer.EOF) {
		p.nextToken()

		if !p.curTokenIs(lexer.IDENT) {
			p.report("PAR_EFF004_INVALID",
				"effect name must be an identifier",
				"Use one of: IO, FS, Net, Clock, Rand, DB, Trace, Async, Env, Debug, AI, SharedMem, SharedIndex")
			continue
		}

		effectName := p.curToken.Literal
		effectLine := p.curToken.Line
		effectCol := p.curToken.Column
		effectFile := p.curToken.File

		// Check if this is an effect row variable (lowercase identifier like 'e', 'eff')
		// Row variables enable effect polymorphism: func mapE[a, b, e](f: (a) -> b ! {e}, ...) -> [b] ! {e}
		// BUT: if the lowercase name case-insensitively matches a known effect (e.g., "io" -> "IO"),
		// it's a typo, not a row variable.
		isRowVar := len(effectName) > 0 && effectName[0] >= 'a' && effectName[0] <= 'z'
		if isRowVar {
			// Check if this is actually a typo for a known effect
			for k := range knownEffects {
				if strings.EqualFold(k, effectName) {
					isRowVar = false
					break
				}
			}
		}

		// Check for unknown effects (skip check for row variables)
		if !isRowVar && !knownEffects[effectName] {
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
		}

		// Parse optional annotations: @min=N and/or @limit=N (M-DX25 M4)
		var budget *int
		var min *int
		for p.peekTokenIs(lexer.AT) {
			p.nextToken() // consume @

			// Expect "limit" or "min"
			if !p.expectPeek(lexer.IDENT) {
				p.report("PAR_EFF005_BUDGET",
					"expected 'limit' or 'min' after '@'",
					"Use @limit=N or @min=N")
				break
			}

			annotation := p.curToken.Literal
			if annotation != "limit" && annotation != "min" {
				p.report("PAR_EFF005_BUDGET",
					fmt.Sprintf("unknown annotation '@%s'", annotation),
					"Use @limit=N to set maximum or @min=N to set minimum")
				// Skip to next comma or closing brace
				for !p.peekTokenIs(lexer.COMMA) && !p.peekTokenIs(lexer.RBRACE) && !p.peekTokenIs(lexer.EOF) && !p.peekTokenIs(lexer.AT) {
					p.nextToken()
				}
				continue
			}

			// Expect =
			if !p.expectPeek(lexer.ASSIGN) {
				p.report("PAR_EFF005_BUDGET",
					fmt.Sprintf("expected '=' after '%s'", annotation),
					fmt.Sprintf("Use @%s=N", annotation))
				continue
			}

			// Expect integer value (possibly negative)
			isNegative := false
			if p.peekTokenIs(lexer.MINUS) {
				p.nextToken() // consume -
				isNegative = true
			}

			if !p.expectPeek(lexer.INT) {
				p.report("PAR_EFF005_BUDGET",
					fmt.Sprintf("@%s value must be an integer", annotation),
					fmt.Sprintf("Use a positive integer like @%s=5", annotation))
				continue
			}

			val, err := strconv.Atoi(p.curToken.Literal)
			if err != nil {
				p.report("PAR_EFF005_BUDGET",
					fmt.Sprintf("invalid %s value '%s'", annotation, p.curToken.Literal),
					fmt.Sprintf("Use a positive integer like @%s=5", annotation))
				continue
			}

			if isNegative {
				val = -val
			}
			if val < 0 {
				p.report("PAR_EFF006_NEGATIVE",
					fmt.Sprintf("@%s cannot be negative: %d", annotation, val),
					"Use a positive integer (0 is allowed)")
				continue
			}

			// Store in appropriate field
			if annotation == "limit" {
				budget = &val
			} else { // annotation == "min"
				min = &val
			}
		}

		effects = append(effects, ast.EffectAnnotation{
			Name:     effectName,
			IsRowVar: isRowVar,
			Budget:   budget,
			Min:      min,
			Pos:      ast.Pos{Line: effectLine, Column: effectCol, File: effectFile},
		})

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
