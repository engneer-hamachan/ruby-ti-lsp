package lsp

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func removeTaiilDot(content string, line uint32, character uint32) string {
	lines := strings.Split(content, "\n")
	if int(line) < len(lines) {
		currentLine := lines[line]

		if int(character) <= len(currentLine) {
			currentLine = currentLine[:character]
		}

		trimmed := strings.TrimSpace(currentLine)
		if strings.HasSuffix(trimmed, ".") {
			currentLine = strings.TrimSuffix(currentLine, ".")
		}

		lines[line] = currentLine
		content = strings.Join(lines, "\n")
	}

	return content
}

func getSignatures(cmdOutput []byte) []Sig {
	methodSet := make(map[string]bool)
	scanner := bufio.NewScanner(strings.NewReader(string(cmdOutput)))

	var responseSignatures []Sig

	for scanner.Scan() {
		line := scanner.Text()

		// Check for signature lines (prefixed with %)
		if sigLine, ok := strings.CutPrefix(line, "%"); ok {
			parts := strings.SplitN(sigLine, ":::", 2)
			if len(parts) < 2 {
				continue
			}
			methodName := parts[0]
			detail := parts[1]

			if !methodSet[detail] {
				methodSet[detail] = true
				responseSignatures = append(responseSignatures, Sig{
					Method: methodName,
					Detail: detail,
				})
			}
		}
	}

	return responseSignatures
}

func findComplection(content string, line uint32, character uint32) []Sig {
	content = removeTaiilDot(content, line, character)

	tmpFile, err := os.CreateTemp("", "ruby-ti-lsp-*.rb")
	if err != nil {
		return []Sig{}
	}

	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.WriteString(content); err != nil {
		return []Sig{}
	}

	tmpFile.Close()

	ctx, cancel :=
		context.WithTimeout(context.Background(), 1000*time.Millisecond)

	defer cancel()

	cmd :=
		exec.CommandContext(
			ctx,
			"ti",
			tmpFile.Name(),
			"--suggest",
			fmt.Sprintf("--row=%d", line+1),
		)

	output, err := cmd.Output()
	if err != nil {
		return []Sig{}
	}

	return getSignatures(output)
}
