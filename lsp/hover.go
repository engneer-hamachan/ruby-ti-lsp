package lsp

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

func findHover(
	content string,
	params *protocol.HoverParams,
) (*protocol.Hover, error) {

	codeLines := strings.Split(content, "\n")
	if int(params.Position.Line) >= len(codeLines) {
		return nil, nil
	}

	currentLine := codeLines[params.Position.Line]

	targetCode :=
		extractTargetCode(currentLine, int(params.Position.Character))
	if targetCode == "" {
		return nil, nil
	}

	codeLines[params.Position.Line] = targetCode
	modifiedContent := strings.Join(codeLines, "\n")

	tmpFile, err := os.CreateTemp("", "ruby-ti-lsp-*.rb")
	if err != nil {
		return nil, nil
	}

	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.WriteString(modifiedContent); err != nil {
		return nil, nil
	}
	tmpFile.Close()

	hoverInfo := getTiOutForHover(tmpFile.Name(), int(params.Position.Line)+1)

	if hoverInfo == "" {
		return nil, nil
	}

	hover := &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: hoverInfo,
		},
	}

	return hover, nil
}

// getTiOutForHover gets hover information by running ti --hover --row=<row>
func getTiOutForHover(filename string, row int) string {

	ctx, cancel :=
		context.WithTimeout(context.Background(), 1000*time.Millisecond)
	defer cancel()

	cmd :=
		exec.CommandContext(ctx, "ti", filename, "--hover", fmt.Sprintf("--row=%d", row))

	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	return parseHoverOutput(output)
}

// parseHoverOutput parses ti --hover output and formats it as markdown
func parseHoverOutput(cmdOutput []byte) string {
	content := string(cmdOutput)
	lines := strings.Split(content, "\n")

	var markdownBuilder strings.Builder

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse lines starting with % (methodName:::signature:::documentation)
		if sigLine, ok := strings.CutPrefix(line, "%"); ok {
			parts := strings.SplitN(sigLine, ":::", 3)
			if len(parts) < 2 {
				continue
			}

			signature := parts[1]
			documentation := ""
			if len(parts) >= 3 {
				documentation = parts[2]
			}

			// Format as markdown with signature in code block
			markdownBuilder.WriteString("```ruby\n")
			markdownBuilder.WriteString(signature)
			markdownBuilder.WriteString("\n```\n")

			// Add documentation if available
			if documentation != "" {
				markdownBuilder.WriteString("\n---\n\n")
				markdownBuilder.WriteString(documentation)
			}

			// Only show first match
			break
		}
	}

	return strings.TrimSpace(markdownBuilder.String())
}
