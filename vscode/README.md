# Ruby-TI LSP for Visual Studio Code

VSCode extension for Ruby-TI Language Server Protocol implementation.

## Features

- **Code Completion**: Intelligent Ruby code suggestions powered by Ruby-TI type analyzer
- **Go to Definition**: Navigate to method definitions across your codebase
- **Type-aware Analysis**: Static type analysis for Ruby code

## Requirements

- **ti command**: The Ruby-TI type analyzer must be installed and available in your PATH
- **VSCode**: Version 1.75.0 or later

## Installation

### From Source

1. Clone the ruby-ti-lsp repository
2. Navigate to the `vscode` directory:
   ```bash
   cd vscode
   ```

3. Install dependencies:
   ```bash
   npm install
   ```

4. Compile the extension:
   ```bash
   npm run compile
   ```

5. Package the extension:
   ```bash
   npm run package
   ```

6. Install the generated `.vsix` file in VSCode:
   - Open VSCode
   - Press `Ctrl+Shift+P` (or `Cmd+Shift+P` on macOS)
   - Type "Extensions: Install from VSIX"
   - Select the generated `ruby-ti-lsp-*.vsix` file

## Configuration

Configure the extension in your VSCode settings:

```json
{
  "rubyTiLsp.serverPath": "ti-lsp",
  "rubyTiLsp.trace.server": "off"
}
```

### Settings

- `rubyTiLsp.serverPath`: Path to the ti-lsp server executable (default: "ti-lsp")
- `rubyTiLsp.trace.server`: Trace communication between VSCode and the language server
  - `off`: No tracing
  - `messages`: Trace messages
  - `verbose`: Verbose tracing

## Usage

Once installed and configured, the extension will automatically activate when you open Ruby files (`.rb`).

### Features

1. **Auto-completion**: Type Ruby code and see intelligent suggestions based on type analysis
2. **Go to Definition**: Right-click on a method and select "Go to Definition" or press `F12`

## Building the LSP Server

The ti-lsp server must be built separately. From the root of the ruby-ti-lsp repository:

```bash
make install
```

This creates the `ti-lsp` binary at `./bin/ti-lsp`. Make sure this is in your PATH or configure `rubyTiLsp.serverPath` to point to it.

## Development

To work on the extension:

1. Open the `vscode` directory in VSCode
2. Press `F5` to launch a new VSCode window with the extension loaded
3. Make changes to `src/extension.ts`
4. Reload the extension window to see changes

## Troubleshooting

### Extension not activating

- Check that the `ti-lsp` server is installed and in your PATH
- Verify the `rubyTiLsp.serverPath` setting points to the correct executable

### No completions or definitions

- Ensure the `ti` command is installed and working
- Check VSCode's Output panel (select "Ruby-TI Language Server" from the dropdown) for error messages

## License

MIT
