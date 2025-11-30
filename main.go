package main

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"ruby-ti-lsp/lsp"
)

func main() {
	if _, err := exec.LookPath("ti"); err != nil {
		fmt.Println("Error: 'ti' command not found in PATH")
		fmt.Println("Ruby-TI LSP requires the 'ti' command to be installed")
		fmt.Println("Please install Ruby-TI: https://github.com/engneer-hamachan/ruby-ti")
		os.Exit(1)
	}

	server := lsp.NewServer()
	server.RunStdio()
}
