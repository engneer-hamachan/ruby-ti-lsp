.PHONY: install

install:
	go build -o ti-lsp main.go
	mkdir -p ./bin
	mv ./ti-lsp ./bin/ti-lsp
