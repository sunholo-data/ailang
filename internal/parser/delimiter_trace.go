package parser

import (
	"fmt"
	"os"

	"github.com/sunholo/ailang/internal/lexer"
)

// delimiterTrace provides runtime delimiter tracking for debugging parser issues
// Enable with: DEBUG_DELIMITERS=1 ailang run file.ail

type delimiterContext string

const (
	delimCtxMatch    delimiterContext = "match"
	delimCtxBlock    delimiterContext = "block"
	delimCtxFunction delimiterContext = "function"
	delimCtxLambda   delimiterContext = "lambda"
	delimCtxRecord   delimiterContext = "record"
	delimCtxList     delimiterContext = "list"
	delimCtxCase     delimiterContext = "case"
)

type delimiterFrame struct {
	context delimiterContext
	line    int
	col     int
	depth   int
}

type delimiterTracer struct {
	enabled bool
	stack   []delimiterFrame
}

var globalDelimiterTracer = &delimiterTracer{
	enabled: os.Getenv("DEBUG_DELIMITERS") == "1",
	stack:   []delimiterFrame{},
}

// Push a delimiter onto the stack
func (d *delimiterTracer) push(ctx delimiterContext, line, col int) {
	if !d.enabled {
		return
	}

	depth := len(d.stack)
	frame := delimiterFrame{
		context: ctx,
		line:    line,
		col:     col,
		depth:   depth,
	}
	d.stack = append(d.stack, frame)

	indent := ""
	for i := 0; i < depth; i++ {
		indent += "  "
	}

	fmt.Fprintf(os.Stderr, "[DELIM_OPEN %s] %s%s { at %d:%d (depth=%d)\n",
		ctx, indent, ctx, line, col, depth)
}

// Pop a delimiter from the stack
func (d *delimiterTracer) pop(ctx delimiterContext, line, col int) {
	if !d.enabled {
		return
	}

	if len(d.stack) == 0 {
		fmt.Fprintf(os.Stderr, "[DELIM_ERROR] Attempted to close %s } at %d:%d but stack is empty!\n",
			ctx, line, col)
		return
	}

	frame := d.stack[len(d.stack)-1]
	d.stack = d.stack[:len(d.stack)-1]

	depth := frame.depth
	indent := ""
	for i := 0; i < depth; i++ {
		indent += "  "
	}

	if frame.context != ctx {
		fmt.Fprintf(os.Stderr, "[DELIM_MISMATCH] %s%s } at %d:%d closes %s { from %d:%d (expected %s)\n",
			indent, ctx, line, col, frame.context, frame.line, frame.col, ctx)
	} else {
		fmt.Fprintf(os.Stderr, "[DELIM_CLOSE %s] %s%s } at %d:%d (opened at %d:%d, depth=%d)\n",
			ctx, indent, ctx, line, col, frame.line, frame.col, depth)
	}
}

// Show current stack state
func (d *delimiterTracer) showStack() {
	if !d.enabled || len(d.stack) == 0 {
		return
	}

	fmt.Fprintf(os.Stderr, "[DELIM_STACK] Current depth: %d\n", len(d.stack))
	for i := len(d.stack) - 1; i >= 0; i-- {
		frame := d.stack[i]
		indent := ""
		for j := 0; j < frame.depth; j++ {
			indent += "  "
		}
		fmt.Fprintf(os.Stderr, "[DELIM_STACK]   %s%s { from %d:%d\n",
			indent, frame.context, frame.line, frame.col)
	}
}

// Helper methods for Parser to call

func (p *Parser) traceDelimiterOpen(ctx delimiterContext) {
	pos := p.curPos()
	globalDelimiterTracer.push(ctx, pos.Line, pos.Column)
}

func (p *Parser) traceDelimiterClose(ctx delimiterContext) {
	pos := p.curPos()
	globalDelimiterTracer.pop(ctx, pos.Line, pos.Column)
}

func (p *Parser) traceDelimiterStack() {
	globalDelimiterTracer.showStack()
}

// Helper to trace when we consume a delimiter token
func (p *Parser) traceDelimiterToken(tokenType lexer.TokenType, action string) {
	if !globalDelimiterTracer.enabled {
		return
	}

	if tokenType == lexer.LBRACE || tokenType == lexer.RBRACE {
		pos := p.curPos()
		symbol := "{"
		if tokenType == lexer.RBRACE {
			symbol = "}"
		}
		fmt.Fprintf(os.Stderr, "[DELIM_TOKEN] %s %s at %d:%d (cur=%s, peek=%s)\n",
			action, symbol, pos.Line, pos.Column,
			p.curToken.Type, p.peekToken.Type)
	}
}
