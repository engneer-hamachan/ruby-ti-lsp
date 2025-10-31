package lsp

import (
	"encoding/json"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/tliron/glsp/server"
)

var handler protocol.Handler
var documentContents = make(map[string]string)

func NewServer() *server.Server {
	handler = protocol.Handler{
		Initialize:             initialize,
		TextDocumentDidOpen:    textDocumentDidOpen,
		TextDocumentCompletion: textDocumentCompletion,
		TextDocumentDidChange:  textDocumentDidChange,
		TextDocumentDidSave:    textDocumentDidSave,
		TextDocumentDefinition: textDocumentDefinition,
		TextDocumentCodeLens:   textDocumentCodeLens,
		TextDocumentCodeAction: textDocumentCodeAction,
	}

	server := server.NewServer(&handler, "ruby-ti", false)
	return server
}

func initialize(
	ctx *glsp.Context,
	params *protocol.InitializeParams,
) (any, error) {

	capabilities := handler.CreateServerCapabilities()

	capabilities.CompletionProvider = &protocol.CompletionOptions{
		TriggerCharacters: []string{
			"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m",
			"n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z",
			"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M",
			"N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z",
			"0", "1", "2", "3", "4", "5", "6", "7", "8", "9",
			".", "_",
		},
	}

	capabilities.CodeLensProvider = &protocol.CodeLensOptions{
		ResolveProvider: &[]bool{false}[0],
	}

	capabilities.CodeActionProvider = &protocol.CodeActionOptions{
		CodeActionKinds: []protocol.CodeActionKind{
			protocol.CodeActionKindQuickFix,
		},
	}

	syncKind := protocol.TextDocumentSyncKindFull

	capabilities.TextDocumentSync =
		protocol.TextDocumentSyncOptions{
			OpenClose: &[]bool{true}[0],
			Change:    &syncKind,
			Save:      &protocol.SaveOptions{IncludeText: &[]bool{true}[0]},
		}

	return protocol.InitializeResult{
		Capabilities: capabilities,
		ServerInfo: &protocol.InitializeResultServerInfo{
			Name:    "ruby-ti-lsp",
			Version: &[]string{"beta"}[0],
		},
	}, nil
}

func textDocumentDidOpen(
	ctx *glsp.Context,
	params *protocol.DidOpenTextDocumentParams,
) error {

	documentContents[params.TextDocument.URI] = params.TextDocument.Text

	go publishDiagnostics(ctx, params.TextDocument.URI, params.TextDocument.Text)

	return nil
}

var changeEvent struct {
	Text string `json:"text"`
}

func textDocumentDidChange(
	ctx *glsp.Context,
	params *protocol.DidChangeTextDocumentParams,
) error {

	if len(params.ContentChanges) < 1 {
		return nil
	}

	change := params.ContentChanges[0]

	changeEventBytes, err := json.Marshal(change)
	if err != nil {
		return nil
	}

	if err := json.Unmarshal(changeEventBytes, &changeEvent); err == nil {
		documentContents[params.TextDocument.URI] = changeEvent.Text

		go publishDiagnostics(ctx, params.TextDocument.URI, changeEvent.Text)
	}

	return nil
}

func textDocumentDidSave(
	ctx *glsp.Context,
	params *protocol.DidSaveTextDocumentParams,
) error {

	content, ok := documentContents[params.TextDocument.URI]
	if ok {
		content = *params.Text
	}

	go publishDiagnostics(ctx, params.TextDocument.URI, content)

	// Check if saved file is a builtin config JSON and run make install
	go checkAndRunMakeInstall(params.TextDocument.URI)

	return nil
}

func textDocumentCompletion(
	ctx *glsp.Context,
	params *protocol.CompletionParams,
) (any, error) {

	var items []protocol.CompletionItem

	content, ok := documentContents[params.TextDocument.URI]
	if !ok {
		return nil, nil
	}

	signatures :=
		findComplection(content, params.Position.Line, params.Position.Character)

	for _, sig := range signatures {
		items =
			append(items, protocol.CompletionItem{
				Label:  sig.Method,
				Detail: &sig.Detail,
			})
	}

	return items, nil
}

func textDocumentDefinition(
	ctx *glsp.Context,
	params *protocol.DefinitionParams,
) (any, error) {

	content, ok := documentContents[params.TextDocument.URI]
	if !ok {
		return nil, nil
	}

	location, err := findDefinition(content, params)

	return location, err
}

func publishDiagnostics(
	ctx *glsp.Context,
	uri protocol.DocumentUri,
	content string,
) {

	ctx.Notify(
		protocol.ServerTextDocumentPublishDiagnostics,
		&protocol.PublishDiagnosticsParams{
			URI:         uri,
			Diagnostics: []protocol.Diagnostic{},
		},
	)

	diagnostics := runDiagnostics(content)

	ctx.Notify(
		protocol.ServerTextDocumentPublishDiagnostics,
		&protocol.PublishDiagnosticsParams{
			URI:         uri,
			Diagnostics: diagnostics,
		},
	)
}
