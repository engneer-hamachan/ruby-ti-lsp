package lsp

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

// runDiagnostics executes ti command and returns diagnostics
func runDiagnostics(content string) []protocol.Diagnostic {
	tmpFile, err := os.CreateTemp("", "ruby-ti-lsp-*.rb")
	if err != nil {
		return []protocol.Diagnostic{}
	}

	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.WriteString(content); err != nil {
		return []protocol.Diagnostic{}
	}

	tmpFile.Close()

	ctx, cancel :=
		context.WithTimeout(context.Background(), 1000*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ti", tmpFile.Name())

	// ti command outputs errors to stdout
	output, _ := cmd.Output()

	return parseErrorsFromTiOutput(string(output))
}

// parseErrorsFromTiOutput parses ti command output and extracts error lines
// Error lines don't have prefixes (@, %, $) but contain :::
func parseErrorsFromTiOutput(output string) []protocol.Diagnostic {
	var diagnostics []protocol.Diagnostic
	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()

		if len(line) == 0 {
			continue
		}

		// Skip lines with prefixes (@, %, $)
		if line[0] == '@' || line[0] == '%' || line[0] == '$' {
			continue
		}

		// Error lines contain :::
		if !strings.Contains(line, ":::") {
			continue
		}

		parts := strings.SplitN(line, ":::", 3)
		if len(parts) < 3 {
			continue
		}

		rowStr := parts[1]
		message := parts[2]

		row, err := strconv.Atoi(rowStr)
		if err != nil {
			continue
		}

		// ti uses 1-based line numbers, LSP uses 0-based
		if row > 0 {
			row--
		}

		diagnostic := protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      uint32(row),
					Character: 0,
				},
				End: protocol.Position{
					Line:      uint32(row),
					Character: 1000, // End of line
				},
			},
			Severity: &[]protocol.DiagnosticSeverity{
				protocol.DiagnosticSeverityError,
			}[0],
			Source: &[]string{"ruby-ti"}[0],
			Message: message,
		}

		diagnostics = append(diagnostics, diagnostic)
	}

	return diagnostics
}

// parseErrorStrings converts ti error strings to LSP diagnostics
// Error format: filename:::row:::message
func parseErrorStrings(errors []string) []protocol.Diagnostic {
	var diagnostics []protocol.Diagnostic

	for _, errStr := range errors {
		parts := strings.SplitN(errStr, ":::", 3)
		if len(parts) < 3 {
			continue
		}

		rowStr := parts[1]
		message := parts[2]

		row, err := strconv.Atoi(rowStr)
		if err != nil {
			continue
		}

		// ti uses 1-based line numbers, LSP uses 0-based
		if row > 0 {
			row--
		}

		diagnostic := protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      uint32(row),
					Character: 0,
				},
				End: protocol.Position{
					Line:      uint32(row),
					Character: 1000, // End of line
				},
			},
			Severity: &[]protocol.DiagnosticSeverity{
				protocol.DiagnosticSeverityError,
			}[0],
			Source: &[]string{"ruby-ti"}[0],
			Message: message,
		}

		diagnostics = append(diagnostics, diagnostic)
	}

	return diagnostics
}
