package main

import (
	"ruby-ti-lsp/lsp"
)

func main() {
	server := lsp.NewServer()
	server.RunStdio()
}
