.PHONY: install install-vsix

install:
	go build -o ti-lsp main.go
	mkdir -p ./bin
	mv ./ti-lsp ./bin/ti-lsp

install-vsix:
	cd vscode && npm run compile && npm run package
	code --install-extension vscode/ruby-ti-lsp-*.vsix
