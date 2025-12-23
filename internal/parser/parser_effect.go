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
		}

		// Parse optional budget: @limit=N
		var budget *int
		if p.peekTokenIs(lexer.AT) {
			p.nextToken() // consume @

			// Expect "limit"
			if !p.expectPeek(lexer.IDENT) || p.curToken.Literal != "limit" {
				p.report("PAR_EFF005_BUDGET",
					"expected 'limit' after '@'",
					"Use @limit=N to set effect budget")
				// Skip to next comma or closing brace
				for !p.peekTokenIs(lexer.COMMA) && !p.peekTokenIs(lexer.RBRACE) && !p.peekTokenIs(lexer.EOF) {
					p.nextToken()
				}
			} else {
				// Expect =
				if !p.expectPeek(lexer.ASSIGN) {
					p.report("PAR_EFF005_BUDGET",
						"expected '=' after 'limit'",
						"Use @limit=N to set effect budget")
				} else {
					// Expect integer value (possibly negative)
					isNegative := false
					if p.peekTokenIs(lexer.MINUS) {
						p.nextToken() // consume -
						isNegative = true
					}

					if !p.expectPeek(lexer.INT) {
						p.report("PAR_EFF005_BUDGET",
							"budget must be an integer",
							"Use a positive integer like @limit=5")
					} else {
						val, err := strconv.Atoi(p.curToken.Literal)
						if err != nil {
							p.report("PAR_EFF005_BUDGET",
								fmt.Sprintf("invalid budget value '%s'", p.curToken.Literal),
								"Use a positive integer like @limit=5")
						} else {
							if isNegative {
								val = -val
							}
							if val < 0 {
								p.report("PAR_EFF006_NEGATIVE",
									fmt.Sprintf("budget cannot be negative: %d", val),
									"Use a positive integer (0 is allowed for 'no operations')")
							} else {
								budget = &val
							}
						}
					}
				}
			}
		}

		effects = append(effects, ast.EffectAnnotation{
			Name:   effectName,
			Budget: budget,
			Pos:    ast.Pos{Line: effectLine, Column: effectCol, File: effectFile},
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
