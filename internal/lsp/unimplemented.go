package lsp

import (
	"context"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

// unimplementedServer satisfies protocol.Server with stub methods that return
// jsonrpc2.ErrMethodNotFound for requests and nil for notifications. Real
// handlers in [Server] override the methods we implement; everything else
// falls through to these stubs.
//
// This file is mechanical. When go.lsp.dev/protocol adds new Server methods,
// stub them here too — Go's interface check will fail compilation otherwise.
type unimplementedServer struct{}

// --- Notifications (no result, return nil to silently ignore) ---

func (unimplementedServer) WorkDoneProgressCancel(context.Context, *protocol.WorkDoneProgressCancelParams) error {
	return nil
}
func (unimplementedServer) LogTrace(context.Context, *protocol.LogTraceParams) error { return nil }
func (unimplementedServer) SetTrace(context.Context, *protocol.SetTraceParams) error { return nil }
func (unimplementedServer) DidChange(context.Context, *protocol.DidChangeTextDocumentParams) error {
	return nil
}
func (unimplementedServer) DidChangeConfiguration(context.Context, *protocol.DidChangeConfigurationParams) error {
	return nil
}
func (unimplementedServer) DidChangeWatchedFiles(context.Context, *protocol.DidChangeWatchedFilesParams) error {
	return nil
}
func (unimplementedServer) DidChangeWorkspaceFolders(context.Context, *protocol.DidChangeWorkspaceFoldersParams) error {
	return nil
}
func (unimplementedServer) DidClose(context.Context, *protocol.DidCloseTextDocumentParams) error {
	return nil
}
func (unimplementedServer) DidOpen(context.Context, *protocol.DidOpenTextDocumentParams) error {
	return nil
}
func (unimplementedServer) DidSave(context.Context, *protocol.DidSaveTextDocumentParams) error {
	return nil
}
func (unimplementedServer) WillSave(context.Context, *protocol.WillSaveTextDocumentParams) error {
	return nil
}
func (unimplementedServer) DidCreateFiles(context.Context, *protocol.CreateFilesParams) error {
	return nil
}
func (unimplementedServer) DidRenameFiles(context.Context, *protocol.RenameFilesParams) error {
	return nil
}
func (unimplementedServer) DidDeleteFiles(context.Context, *protocol.DeleteFilesParams) error {
	return nil
}
func (unimplementedServer) CodeLensRefresh(context.Context) error       { return nil }
func (unimplementedServer) SemanticTokensRefresh(context.Context) error { return nil }

// --- Requests (return ErrMethodNotFound so the client sees a real LSP error) ---

func (unimplementedServer) CodeAction(context.Context, *protocol.CodeActionParams) ([]protocol.CodeAction, error) {
	return nil, jsonrpc2.ErrMethodNotFound
}
func (unimplementedServer) CodeLens(context.Context, *protocol.CodeLensParams) ([]protocol.CodeLens, error) {
	return nil, jsonrpc2.ErrMethodNotFound
}
func (unimplementedServer) CodeLensResolve(context.Context, *protocol.CodeLens) (*protocol.CodeLens, error) {
	return nil, jsonrpc2.ErrMethodNotFound
}
func (unimplementedServer) ColorPresentation(context.Context, *protocol.ColorPresentationParams) ([]protocol.ColorPresentation, error) {
	return nil, jsonrpc2.ErrMethodNotFound
}
func (unimplementedServer) Completion(context.Context, *protocol.CompletionParams) (*protocol.CompletionList, error) {
	return nil, jsonrpc2.ErrMethodNotFound
}
func (unimplementedServer) CompletionResolve(context.Context, *protocol.CompletionItem) (*protocol.CompletionItem, error) {
	return nil, jsonrpc2.ErrMethodNotFound
}
func (unimplementedServer) Declaration(context.Context, *protocol.DeclarationParams) ([]protocol.Location, error) {
	return nil, jsonrpc2.ErrMethodNotFound
}
func (unimplementedServer) Definition(context.Context, *protocol.DefinitionParams) ([]protocol.Location, error) {
	return nil, jsonrpc2.ErrMethodNotFound
}
func (unimplementedServer) DocumentColor(context.Context, *protocol.DocumentColorParams) ([]protocol.ColorInformation, error) {
	return nil, jsonrpc2.ErrMethodNotFound
}
func (unimplementedServer) DocumentHighlight(context.Context, *protocol.DocumentHighlightParams) ([]protocol.DocumentHighlight, error) {
	return nil, jsonrpc2.ErrMethodNotFound
}
func (unimplementedServer) DocumentLink(context.Context, *protocol.DocumentLinkParams) ([]protocol.DocumentLink, error) {
	return nil, jsonrpc2.ErrMethodNotFound
}
func (unimplementedServer) DocumentLinkResolve(context.Context, *protocol.DocumentLink) (*protocol.DocumentLink, error) {
	return nil, jsonrpc2.ErrMethodNotFound
}
func (unimplementedServer) DocumentSymbol(context.Context, *protocol.DocumentSymbolParams) ([]interface{}, error) {
	return nil, jsonrpc2.ErrMethodNotFound
}
func (unimplementedServer) ExecuteCommand(context.Context, *protocol.ExecuteCommandParams) (interface{}, error) {
	return nil, jsonrpc2.ErrMethodNotFound
}
func (unimplementedServer) FoldingRanges(context.Context, *protocol.FoldingRangeParams) ([]protocol.FoldingRange, error) {
	return nil, jsonrpc2.ErrMethodNotFound
}
func (unimplementedServer) Formatting(context.Context, *protocol.DocumentFormattingParams) ([]protocol.TextEdit, error) {
	return nil, jsonrpc2.ErrMethodNotFound
}
func (unimplementedServer) Hover(context.Context, *protocol.HoverParams) (*protocol.Hover, error) {
	return nil, jsonrpc2.ErrMethodNotFound
}
func (unimplementedServer) Implementation(context.Context, *protocol.ImplementationParams) ([]protocol.Location, error) {
	return nil, jsonrpc2.ErrMethodNotFound
}
func (unimplementedServer) OnTypeFormatting(context.Context, *protocol.DocumentOnTypeFormattingParams) ([]protocol.TextEdit, error) {
	return nil, jsonrpc2.ErrMethodNotFound
}
func (unimplementedServer) PrepareRename(context.Context, *protocol.PrepareRenameParams) (*protocol.Range, error) {
	return nil, jsonrpc2.ErrMethodNotFound
}
func (unimplementedServer) RangeFormatting(context.Context, *protocol.DocumentRangeFormattingParams) ([]protocol.TextEdit, error) {
	return nil, jsonrpc2.ErrMethodNotFound
}
func (unimplementedServer) References(context.Context, *protocol.ReferenceParams) ([]protocol.Location, error) {
	return nil, jsonrpc2.ErrMethodNotFound
}
func (unimplementedServer) Rename(context.Context, *protocol.RenameParams) (*protocol.WorkspaceEdit, error) {
	return nil, jsonrpc2.ErrMethodNotFound
}
func (unimplementedServer) SignatureHelp(context.Context, *protocol.SignatureHelpParams) (*protocol.SignatureHelp, error) {
	return nil, jsonrpc2.ErrMethodNotFound
}
func (unimplementedServer) Symbols(context.Context, *protocol.WorkspaceSymbolParams) ([]protocol.SymbolInformation, error) {
	return nil, jsonrpc2.ErrMethodNotFound
}
func (unimplementedServer) TypeDefinition(context.Context, *protocol.TypeDefinitionParams) ([]protocol.Location, error) {
	return nil, jsonrpc2.ErrMethodNotFound
}
func (unimplementedServer) WillSaveWaitUntil(context.Context, *protocol.WillSaveTextDocumentParams) ([]protocol.TextEdit, error) {
	return nil, jsonrpc2.ErrMethodNotFound
}
func (unimplementedServer) ShowDocument(context.Context, *protocol.ShowDocumentParams) (*protocol.ShowDocumentResult, error) {
	return nil, jsonrpc2.ErrMethodNotFound
}
func (unimplementedServer) WillCreateFiles(context.Context, *protocol.CreateFilesParams) (*protocol.WorkspaceEdit, error) {
	return nil, jsonrpc2.ErrMethodNotFound
}
func (unimplementedServer) WillRenameFiles(context.Context, *protocol.RenameFilesParams) (*protocol.WorkspaceEdit, error) {
	return nil, jsonrpc2.ErrMethodNotFound
}
func (unimplementedServer) WillDeleteFiles(context.Context, *protocol.DeleteFilesParams) (*protocol.WorkspaceEdit, error) {
	return nil, jsonrpc2.ErrMethodNotFound
}
func (unimplementedServer) PrepareCallHierarchy(context.Context, *protocol.CallHierarchyPrepareParams) ([]protocol.CallHierarchyItem, error) {
	return nil, jsonrpc2.ErrMethodNotFound
}
func (unimplementedServer) IncomingCalls(context.Context, *protocol.CallHierarchyIncomingCallsParams) ([]protocol.CallHierarchyIncomingCall, error) {
	return nil, jsonrpc2.ErrMethodNotFound
}
func (unimplementedServer) OutgoingCalls(context.Context, *protocol.CallHierarchyOutgoingCallsParams) ([]protocol.CallHierarchyOutgoingCall, error) {
	return nil, jsonrpc2.ErrMethodNotFound
}
func (unimplementedServer) SemanticTokensFull(context.Context, *protocol.SemanticTokensParams) (*protocol.SemanticTokens, error) {
	return nil, jsonrpc2.ErrMethodNotFound
}
func (unimplementedServer) SemanticTokensFullDelta(context.Context, *protocol.SemanticTokensDeltaParams) (interface{}, error) {
	return nil, jsonrpc2.ErrMethodNotFound
}
func (unimplementedServer) SemanticTokensRange(context.Context, *protocol.SemanticTokensRangeParams) (*protocol.SemanticTokens, error) {
	return nil, jsonrpc2.ErrMethodNotFound
}
func (unimplementedServer) LinkedEditingRange(context.Context, *protocol.LinkedEditingRangeParams) (*protocol.LinkedEditingRanges, error) {
	return nil, jsonrpc2.ErrMethodNotFound
}
func (unimplementedServer) Moniker(context.Context, *protocol.MonikerParams) ([]protocol.Moniker, error) {
	return nil, jsonrpc2.ErrMethodNotFound
}

// Request is a generic escape hatch for methods not in the typed interface.
func (unimplementedServer) Request(context.Context, string, interface{}) (interface{}, error) {
	return nil, jsonrpc2.ErrMethodNotFound
}
