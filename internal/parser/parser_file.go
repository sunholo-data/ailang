package parser

import (
	"fmt"
	"strings"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/errors"
	"github.com/sunholo/ailang/internal/lexer"
)

// ParseFile parses a complete AILANG source file
func (p *Parser) ParseFile() (file *ast.File) {
	// Add panic recovery to convert panics to parser errors
	defer func() {
		if r := recover(); r != nil {
			// Convert panic to parser error
			var msg string
			if err, ok := r.(error); ok {
				msg = err.Error()
			} else {
				msg = fmt.Sprintf("%v", r)
			}

			p.errors = append(p.errors, NewParserError(
				errors.PAR999, // Generic parser panic code
				p.curPos(),
				p.curToken,
				fmt.Sprintf("parser panic: %s", msg),
				nil,
				"This is an internal parser error. Please report this issue."))

			// Return a minimal valid AST
			if file == nil {
				file = &ast.File{
					Decls:      []ast.Node{},
					Statements: []ast.Node{},
				}
			}
		}
	}()

	file = &ast.File{
		Pos: p.curPos(),
	}

	// Optional module declaration
	if p.curTokenIs(lexer.MODULE) {
		file.Module = p.parseModuleDecl()
		p.nextToken()
	}

	// Import declarations
	for p.curTokenIs(lexer.IMPORT) {
		imp := p.parseImportDecl()
		if imp != nil {
			file.Imports = append(file.Imports, imp)
		}
		p.nextToken()
	}

	// Export declarations (standalone export list)
	if p.curTokenIs(lexer.EXPORT) && p.peekTokenIs(lexer.LBRACE) {
		p.parseExportList()
		p.nextToken()
	}

	// Top-level declarations
	for !p.curTokenIs(lexer.EOF) {
		if decl := p.parseTopLevelDecl(); decl != nil {
			// Separate functions from other statements
			if funcDecl, ok := decl.(*ast.FuncDecl); ok {
				file.Funcs = append(file.Funcs, funcDecl)
			} else {
				file.Statements = append(file.Statements, decl)
			}
			// Keep in Decls for backward compatibility
			file.Decls = append(file.Decls, decl)
		}
		if !p.curTokenIs(lexer.EOF) {
			p.nextToken()
		}
	}

	return file
}

// parseModuleDecl parses a module declaration
func (p *Parser) parseModuleDecl() *ast.ModuleDecl {
	startPos := p.curPos()
	p.expectPeek(lexer.IDENT)

	// Build module path (e.g., "foo/bar")
	path := p.curToken.Literal
	for p.peekTokenIs(lexer.SLASH) {
		p.nextToken() // consume slash
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}
		path += "/" + p.curToken.Literal
	}

	endPos := p.curPos()
	return &ast.ModuleDecl{
		Path: path,
		Pos:  startPos,
		Span: ast.Span{Start: startPos, End: endPos},
	}
}

// parseImportDecl parses an import declaration
func (p *Parser) parseImportDecl() *ast.ImportDecl {
	startPos := p.curPos()
	imp := &ast.ImportDecl{
		Pos: startPos,
	}

	p.nextToken() // consume 'import'

	// Detect JavaScript/ES6 pattern: "import X from 'Y'"
	if p.curTokenIs(lexer.IDENT) && p.peekTokenIs(lexer.IDENT) {
		// Check if next identifier is "from" (common JS pattern)
		if p.peekToken.Literal == "from" {
			p.errors = append(p.errors, NewSuggestionError(
				"IMP012_UNSUPPORTED_NAMESPACE",
				p.curPos(),
				p.curToken,
				"namespace imports not yet supported (JavaScript-style 'import X from Y' syntax)",
				[]string{
					"import std/net (httpRequest)     -- For HTTP requests",
					"import std/json (encode, decode) -- For JSON parsing",
					"import std/io (println)          -- For I/O operations",
				},
				"https://ailang.sunholo.com/docs/guides/module_execution",
			))
			return nil
		}
	}

	// Parse import path - can be string or path segments: ./relative, ../parent, std/io
	if p.curTokenIs(lexer.STRING) {
		imp.Path = p.curToken.Literal
	} else {
		// Build path from segments: segment ("/" segment)*
		// segment = IDENT | "." | ".."
		path := ""

		// Handle leading dots for relative paths
		if p.curTokenIs(lexer.DOT) {
			path = "."
			// Check for ./ or ../
			if p.peekTokenIs(lexer.DOT) {
				p.nextToken()
				path = ".."
			}
			if p.peekTokenIs(lexer.SLASH) {
				p.nextToken() // consume slash
				path += "/"
				p.nextToken() // move to next segment
			}
		}

		// Parse path segments
		if p.curTokenIs(lexer.IDENT) {
			if path != "" && !strings.HasSuffix(path, "/") {
				path += "/"
			}
			path += p.curToken.Literal

			for p.peekTokenIs(lexer.SLASH) {
				p.nextToken() // consume slash
				p.nextToken() // move to next segment

				if p.curTokenIs(lexer.IDENT) {
					path += "/" + p.curToken.Literal
				} else if p.curTokenIs(lexer.DOT) {
					// Handle .. in middle of path
					if p.peekTokenIs(lexer.DOT) {
						p.nextToken()
						path += "/.."
					} else {
						path += "/."
					}
				} else {
					p.errors = append(p.errors, NewParserError(errors.IMP010, p.curPos(), p.curToken,
						"expected path segment after /",
						[]lexer.TokenType{lexer.IDENT},
						"Add path segment or remove trailing /"))
					return nil
				}
			}
		} else if path == "" {
			// No valid path found
			p.errors = append(p.errors, NewParserError(errors.IMP001, p.curPos(), p.curToken,
				"expected import path",
				[]lexer.TokenType{lexer.STRING, lexer.IDENT, lexer.DOT},
				"Provide a valid import path"))
			return nil
		}

		imp.Path = path
	}

	// Check for module aliasing: import std/list as List
	if p.peekTokenIs(lexer.AS) {
		p.nextToken() // consume 'as'
		p.nextToken() // move to alias identifier

		if !p.curTokenIs(lexer.IDENT) {
			p.errors = append(p.errors, NewParserError(errors.IMP001, p.curPos(), p.curToken,
				"expected module alias identifier after 'as'",
				[]lexer.TokenType{lexer.IDENT},
				"Provide a module alias: import std/list as List"))
			return nil
		}
		imp.ModuleAlias = p.curToken.Literal
	}

	// Check for selective imports: import module (symbol1, symbol2)
	// or: import module as Alias (symbol1, symbol2)
	if p.peekTokenIs(lexer.LPAREN) {
		p.nextToken() // consume (
		p.nextToken() // move to first symbol

		// Initialize SymbolAliases map if needed
		imp.SymbolAliases = make(map[string]string)

		for !p.curTokenIs(lexer.RPAREN) && !p.curTokenIs(lexer.EOF) {
			if p.curTokenIs(lexer.IDENT) {
				symbolName := p.curToken.Literal
				imp.Symbols = append(imp.Symbols, symbolName)

				// Check for symbol aliasing: symbol as alias
				if p.peekTokenIs(lexer.AS) {
					p.nextToken() // consume 'as'
					p.nextToken() // move to alias identifier

					if !p.curTokenIs(lexer.IDENT) {
						p.errors = append(p.errors, NewParserError(errors.IMP001, p.curPos(), p.curToken,
							"expected symbol alias identifier after 'as'",
							[]lexer.TokenType{lexer.IDENT},
							"Provide a symbol alias: import std/string (length as stringLength)"))
						return nil
					}
					imp.SymbolAliases[symbolName] = p.curToken.Literal
				}
			}

			if p.peekTokenIs(lexer.COMMA) {
				p.nextToken() // consume comma
				p.nextToken() // move to next symbol
			} else {
				break
			}
		}

		if !p.expectPeek(lexer.RPAREN) {
			return nil
		}
	} else if imp.ModuleAlias == "" {
		// No module alias and no selective imports - require one or the other
		p.errors = append(p.errors, NewParserError("IMP012_UNSUPPORTED_NAMESPACE", p.curPos(), p.curToken,
			"namespace imports not yet supported",
			[]lexer.TokenType{lexer.LPAREN, lexer.AS},
			"Use selective import: import module/path (symbol1, symbol2)\nOr module alias: import std/list as List"))
		return nil
	}
	// If ModuleAlias is set but no selective imports, that's valid (creates qualified namespace)

	endPos := p.curPos()
	imp.Span = ast.Span{Start: startPos, End: endPos}
	return imp
}

// parseExportList parses a standalone export list: export { name1, name2 }
func (p *Parser) parseExportList() []string {
	var exports []string

	if !p.expectPeek(lexer.LBRACE) {
		return exports
	}
	p.nextToken() // move to first export

	for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
		if p.curTokenIs(lexer.IDENT) {
			exports = append(exports, p.curToken.Literal)
		}

		if p.peekTokenIs(lexer.COMMA) {
			p.nextToken() // consume comma
			p.nextToken() // move to next export
		} else {
			break
		}
	}

	if !p.expectPeek(lexer.RBRACE) {
		return exports
	}

	// Store exports in File's metadata (we'll need to extend the File struct later)
	return exports
}
