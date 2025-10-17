# Ruby-TI LSP

Language Server Protocol (LSP) implementation for Ruby-TI static type analyzer.

## Overview

Ruby-TI LSP provides IDE features for Ruby code using the Ruby-TI type analyzer:

- **Code Completion**: Auto-complete method suggestions based on type inference
- **Go to Definition**: Jump to method and class definitions
- **Type-aware Navigation**: Navigate using inheritance hierarchy

## Requirements

- Go 1.24.5 or later
- [Ruby-TI](https://github.com/your-repo/ruby-ti) installed and available in PATH as `ti` command

## Installation

```bash
make install
```

This will build the `ti-lsp` binary and install it to `./bin/ti-lsp`.

## Usage

### Command Line

Run the LSP server:

```bash
./bin/ti-lsp
```

The server communicates via stdio and follows the LSP specification.

### Editor Integration

#### Neovim

Add the following to your Neovim configuration:

```lua
vim.lsp.start({
  name = 'ruby-ti-lsp',
  cmd = {'/path/to/ruby-ti-lsp/bin/ti-lsp'},
  root_dir = vim.fs.dirname(vim.fs.find({'Gemfile', '.git'}, { upward = true })[1]),
})
```

#### VSCode

Create a VSCode extension or use a generic LSP client with:

```json
{
  "command": "/path/to/ruby-ti-lsp/bin/ti-lsp",
  "args": []
}
```

## Features

### Code Completion

Type a method name or use `.` after an object to get method suggestions:

```ruby
str = "hello"
str.  # Shows String methods: upcase, downcase, length, etc.
```

### Go to Definition

Place cursor on a method name and use "Go to Definition" to jump to where it's defined:

```ruby
def greet(name)
  puts "Hello, #{name}"
end

greet("World")  # Ctrl+Click or gd on 'greet' jumps to definition
```

## Architecture

Ruby-TI LSP communicates with the `ti` command to perform type analysis:

- **Completion**: Uses `ti <file> -a <line>` to get method suggestions
- **Definition**: Uses `ti <file> --define <line>` to get definition locations

The LSP server creates temporary files for real-time analysis and parses the output from the `ti` command.

## Development

### Project Structure

```
ruby-ti-lsp/
├── main.go           # Entry point
├── lsp/
│   ├── server.go     # LSP server initialization
│   ├── complection.go # Code completion logic
│   ├── definition.go  # Go to definition logic
│   └── types.go       # Type definitions
├── Makefile
└── README.md
```

### Building

```bash
go build -o ti-lsp main.go
```

## License

This project uses the following third-party libraries:

- [glsp](https://github.com/tliron/glsp) - Licensed under Apache License 2.0
- [commonlog](https://github.com/tliron/commonlog) - Licensed under Apache License 2.0
- [kutil](https://github.com/tliron/kutil) - Licensed under Apache License 2.0

See the respective repositories for full license texts.

### Apache License 2.0 Notice

This software includes components licensed under the Apache License 2.0:

```
Copyright (c) The respective authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```
