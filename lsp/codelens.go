package lsp

import (
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type DefineInfo struct {
	FileName  string
	Row       int
	Signature string
}

func parseDefineInfo(line string) (*DefineInfo, error) {
	if !strings.HasPrefix(line, "@") {
		return nil, nil
	}

	line = strings.TrimPrefix(line, "@")

	parts := strings.SplitN(line, ":::", 3)
	if len(parts) != 3 {
		return nil, nil
	}

	row, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, err
	}

	return &DefineInfo{
		FileName:  parts[0],
		Row:       row,
		Signature: parts[2],
	}, nil
}

func getDefineInfos(content string) ([]DefineInfo, error) {
	tmpFile, err := os.CreateTemp("", "ruby-ti-lsp-*.rb")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.WriteString(content); err != nil {
		return nil, err
	}

	if err := tmpFile.Sync(); err != nil {
		return nil, err
	}

	ctx, cancel :=
		context.WithTimeout(context.Background(), 1000*time.Millisecond)

	defer cancel()

	cmd := exec.CommandContext(ctx, "ti", tmpFile.Name(), "-i")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return []DefineInfo{}, nil
	}

	lines := strings.Split(string(output), "\n")
	var infos []DefineInfo

	for _, line := range lines {
		info, err := parseDefineInfo(line)
		if err != nil {
			continue
		}
		if info != nil {
			infos = append(infos, *info)
		}
	}

	return infos, nil
}

func findCodeLens(content string) ([]protocol.CodeLens, error) {
	infos, err := getDefineInfos(content)
	if err != nil {
		return nil, err
	}

	var lenses []protocol.CodeLens

	for _, info := range infos {
		line := uint32(info.Row - 1)

		lens := protocol.CodeLens{
			Range: protocol.Range{
				Start: protocol.Position{Line: line, Character: 0},
				End:   protocol.Position{Line: line, Character: 0},
			},
			Command: &protocol.Command{
				Title:   info.Signature,
				Command: "ruby-ti.showSignature",
			},
		}

		lenses = append(lenses, lens)
	}

	return lenses, nil
}

func textDocumentCodeLens(
	ctx *glsp.Context,
	params *protocol.CodeLensParams,
) ([]protocol.CodeLens, error) {

	content, ok := documentContents[params.TextDocument.URI]
	if !ok {
		return nil, nil
	}

	lenses, err := findCodeLens(content)
	if err != nil {
		return nil, nil
	}

	return lenses, nil
}
