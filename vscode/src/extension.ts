import * as path from 'path';
import { workspace, ExtensionContext, window } from 'vscode';
import {
  LanguageClient,
  LanguageClientOptions,
  ServerOptions,
  TransportKind
} from 'vscode-languageclient/node';

let client: LanguageClient;

export function activate(context: ExtensionContext) {
  // Get the server path from configuration
  const config = workspace.getConfiguration('rubyTiLsp');
  const serverPath = config.get<string>('serverPath', 'ti-lsp');

  window.showInformationMessage(`Ruby-TI LSP: Starting server at ${serverPath}`);

  // Server options
  const serverOptions: ServerOptions = {
    command: serverPath,
    args: [],
    options: {
      env: process.env
    }
  };

  // Client options
  const clientOptions: LanguageClientOptions = {
    // Register the server for Ruby documents
    documentSelector: [{ scheme: 'file', language: 'ruby' }],
    synchronize: {
      // Notify the server about file changes to .rb files
      fileEvents: workspace.createFileSystemWatcher('**/*.rb')
    }
  };

  // Create the language client and start it
  client = new LanguageClient(
    'rubyTiLsp',
    'Ruby-TI Language Server',
    serverOptions,
    clientOptions
  );

  // Start the client (also starts the server)
  client.start().then(() => {
    window.showInformationMessage('Ruby-TI LSP: Server started successfully');
  }).catch((error) => {
    window.showErrorMessage(`Ruby-TI LSP: Failed to start server: ${error.message}`);
    console.error('Ruby-TI LSP error:', error);
  });
}

export function deactivate(): Thenable<void> | undefined {
  if (!client) {
    return undefined;
  }
  return client.stop();
}
