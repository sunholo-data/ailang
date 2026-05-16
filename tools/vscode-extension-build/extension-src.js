// AILANG VS Code extension entry point.
// Bundles `vscode-languageclient` to spawn `ailang lsp --stdio` and wire
// the AILANG language server into VS Code on activation.
//
// This is the SOURCE. The committed `extension.js` checked in under
// cmd/ailang/editor_assets/vscode/ is the esbuild-bundled output of
// THIS file plus its node_modules. Rebuild via tools/vscode-extension-build/bundle.sh.

const { LanguageClient, TransportKind } = require('vscode-languageclient/node');

/** @type {LanguageClient | undefined} */
let client;

function activate(/* context */) {
  /** @type {import('vscode-languageclient/node').ServerOptions} */
  const serverOptions = {
    command: 'ailang',
    args: ['lsp', '--stdio'],
    transport: TransportKind.stdio,
  };

  /** @type {import('vscode-languageclient/node').LanguageClientOptions} */
  const clientOptions = {
    documentSelector: [{ scheme: 'file', language: 'ailang' }],
    synchronize: {
      // No explicit file watcher — the LSP only acts on opened documents.
      // (Workspace-wide scanning is deferred per M-AILANG-LSP-WORKSPACE-SCAN.)
    },
  };

  client = new LanguageClient(
    'ailang',
    'AILANG Language Server',
    serverOptions,
    clientOptions
  );

  // start() returns a promise; we don't await it (VS Code activation must
  // resolve quickly). Errors surface in the Output panel under
  // "AILANG Language Server".
  client.start();
}

function deactivate() {
  if (!client) {
    return undefined;
  }
  return client.stop();
}

module.exports = { activate, deactivate };
