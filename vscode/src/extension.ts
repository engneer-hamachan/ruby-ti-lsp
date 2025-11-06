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
  const config = workspace.getConfiguration('rubyTiLsp');
  const serverPath = config.get<string>('serverPath', 'ti-lsp');

  window.showInformationMessage(`Ruby-TI LSP: Starting server at ${serverPath}`);

  const serverOptions: ServerOptions = {
    command: serverPath,
    args: [],
    options: {
      env: process.env
    }
  };

  const clientOptions: LanguageClientOptions = {
    documentSelector: [
      { scheme: 'file', language: 'ruby' },
      { scheme: 'file', language: 'json' }
    ],
    synchronize: {
      fileEvents: workspace.createFileSystemWatcher('**/*.{rb,json}')
    }
  };

  client = new LanguageClient(
    'rubyTiLsp',
    'Ruby-TI Language Server',
    serverOptions,
    clientOptions
  );

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
