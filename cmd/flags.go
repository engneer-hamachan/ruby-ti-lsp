package cmd

import "flag"

var IsStrictMode bool

func ParseFlags() {
	flag.BoolVar(&IsStrictMode, "strict", false, "Enable strict mode for ti diagnostics")
	flag.Parse()
}
