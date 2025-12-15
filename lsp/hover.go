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

	tmpFile, err := os.CreateTemp("", "ruby-ti-lsp-*.rb")
	if err != nil {
		return nil, nil
	}

	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.WriteString(content); err != nil {
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

func getTiOutForHover(filename string, row int) string {
	ctx, cancel :=
		context.WithTimeout(context.Background(), 1000*time.Millisecond)

	defer cancel()

	cmd :=
		exec.CommandContext(
			ctx,
			"ti",
			filename,
			"--hover",
			fmt.Sprintf("--row=%d", row),
		)

	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	return parseHoverOutput(output)
}

func parseHoverOutput(cmdOutput []byte) string {
	content := string(cmdOutput)
	lines := strings.Split(content, "\n")

	var markdownBuilder strings.Builder

	line := strings.TrimSpace(lines[0])

	if sigLine, ok := strings.CutPrefix(line, "%"); ok {
		parts := strings.SplitN(sigLine, ":::", 3)
		if len(parts) < 2 {
			return ""
		}

		signature := parts[1]
		documentation := ""
		if len(parts) >= 3 {
			documentation = parts[2]
		}

		markdownBuilder.WriteString("```ruby\n")
		markdownBuilder.WriteString(signature)
		markdownBuilder.WriteString("\n```\n")

		if documentation != "" {
			markdownBuilder.WriteString("\n---\n\n")
			markdownBuilder.WriteString(documentation)
		}
	}
	return strings.TrimSpace(markdownBuilder.String())
}
