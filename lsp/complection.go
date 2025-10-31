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

func getAllTypes() []string {
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ti", "--all-type")
	output, err := cmd.Output()
	if err != nil {
		return []string{}
	}

	var types []string
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			types = append(types, line)
		}
	}

	return types
}

func isInTypeArray(content string, line uint32, character uint32) bool {
	lines := strings.Split(content, "\n")
	if int(line) >= len(lines) {
		return false
	}

	currentLine := lines[line]
	if int(character) > len(currentLine) {
		return false
	}

	beforeCursor := currentLine[:character]

	quoteCount := strings.Count(beforeCursor, "\"")
	if quoteCount%2 == 0 {
		return false
	}

	for i := int(line); i >= 0 && i >= int(line)-10; i-- {
		checkLine := strings.TrimSpace(lines[i])
		if strings.Contains(checkLine, "\"type\"") || strings.Contains(checkLine, `"type":`) {
			for j := i; j <= int(line); j++ {
				l := strings.TrimSpace(lines[j])
				if strings.HasPrefix(l, "]") && j < int(line) {
					return false
				}
			}
			return true
		}
		if strings.HasPrefix(checkLine, "}") || strings.HasPrefix(checkLine, "{") {
			return false
		}
	}

	return false
}

func findJsonTypeCompletion(content string, line uint32, character uint32) []Sig {
	if !isInTypeArray(content, line, character) {
		return []Sig{}
	}

	types := getAllTypes()
	var signatures []Sig
	for _, typeName := range types {
		signatures = append(signatures, Sig{
			Method: typeName,
			Detail: "Type",
		})
	}

	return signatures
}
