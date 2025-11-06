package lsp

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var envContent string

func SetEnvContent(content string) {
	envContent = content
}

type TiClassConfig struct {
	Frame           string     `json:"frame"`
	Class           string     `json:"class"`
	Extends         []string   `json:"extends"`
	InstanceMethods []TiMethod `json:"instance_methods"`
	ClassMethods    []TiMethod `json:"class_methods"`
}

type TiMethod struct {
	Name            string       `json:"name"`
	Arguments       []TiArgument `json:"arguments"`
	BlockParameters []string     `json:"block_parameters,omitempty"`
	ReturnType      TiReturnType `json:"return_type"`
}

type TiArgument struct {
	Type []string `json:"type"`
}

type TiReturnType struct {
	Type          []string `json:"type"`
	IsConditional bool     `json:"is_conditional,omitempty"`
	IsDestructive bool     `json:"is_destructive,omitempty"`
}

type ErrorInfo struct {
	Line         uint32
	ErrorType    string
	ClassName    string
	MethodName   string
	MethodType   string
	ErrorMessage string
}

func textDocumentCodeAction(
	ctx *glsp.Context,
	params *protocol.CodeActionParams,
) (any, error) {

	var codeActions []protocol.CodeAction

	for _, diagnostic := range params.Context.Diagnostics {
		errorInfo :=
			parseErrorMessage(diagnostic.Message, diagnostic.Range.Start.Line)

		if errorInfo == nil {
			continue
		}

		switch errorInfo.ErrorType {
		case "class":
			action :=
				createClassCodeAction(errorInfo, diagnostic)

			if action != nil {
				codeActions = append(codeActions, *action)
			}

		case "method":
			action := createMethodCodeAction(errorInfo, diagnostic)
			if action != nil {
				codeActions = append(codeActions, *action)
			}

			extendsClasses := getExtendsClasses(params.TextDocument.URI, errorInfo.ClassName)
			for _, parentClass := range extendsClasses {
				action := createMethodCodeActionForClass(errorInfo, diagnostic, parentClass)
				if action != nil {
					codeActions = append(codeActions, *action)
				}
			}
		}
	}

	return codeActions, nil
}

func parseErrorMessage(message string, line uint32) *ErrorInfo {
	classPattern :=
		regexp.MustCompile(`class '([^']+)' is not defined`)

	matches := classPattern.FindStringSubmatch(message)

	if len(matches) > 1 {
		return &ErrorInfo{
			Line:         line,
			ErrorType:    "class",
			ClassName:    matches[1],
			ErrorMessage: message,
		}
	}

	instanceMethodPattern :=
		regexp.MustCompile(`instance method '([^']+)' is not defined for ([^\s]+)`)

	matches = instanceMethodPattern.FindStringSubmatch(message)

	if len(matches) > 2 {
		return &ErrorInfo{
			Line:         line,
			ErrorType:    "method",
			MethodName:   matches[1],
			ClassName:    matches[2],
			MethodType:   "instance",
			ErrorMessage: message,
		}
	}

	classMethodPattern :=
		regexp.MustCompile(`class method '([^']+)' is not defined for ([^\s]+)`)

	matches = classMethodPattern.FindStringSubmatch(message)

	if len(matches) > 2 {
		return &ErrorInfo{
			Line:         line,
			ErrorType:    "method",
			MethodName:   matches[1],
			ClassName:    matches[2],
			MethodType:   "class",
			ErrorMessage: message,
		}
	}

	return nil
}

func createClassCodeAction(
	errorInfo *ErrorInfo,
	diagnostic protocol.Diagnostic,
) *protocol.CodeAction {

	title := fmt.Sprintf("Create class definition for '%s'", errorInfo.ClassName)

	configDir := findBuiltinConfigDir()
	if configDir == "" {
		return nil
	}

	filePath :=
		filepath.Join(configDir, strings.ToLower(errorInfo.ClassName)+".json")

	classConfig := TiClassConfig{
		Frame:           "Builtin",
		Class:           errorInfo.ClassName,
		Extends:         []string{},
		InstanceMethods: []TiMethod{},
		ClassMethods: []TiMethod{
			{
				Name:      "new",
				Arguments: []TiArgument{},
				ReturnType: TiReturnType{
					Type: []string{errorInfo.ClassName},
				},
			},
		},
	}

	jsonData, err := json.MarshalIndent(classConfig, "", "  ")
	if err != nil {
		return nil
	}

	changes := make(map[protocol.DocumentUri][]protocol.TextEdit)
	fileUri := protocol.DocumentUri("file://" + filePath)

	changes[fileUri] = []protocol.TextEdit{
		{
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 0},
			},
			NewText: string(jsonData) + "\n",
		},
	}

	edit := protocol.WorkspaceEdit{
		Changes: changes,
	}

	kind := protocol.CodeActionKindQuickFix

	return &protocol.CodeAction{
		Title:       title,
		Kind:        &kind,
		Diagnostics: []protocol.Diagnostic{diagnostic},
		Edit:        &edit,
	}
}

func getDocumentContent(uri protocol.DocumentUri) string {
	content, ok := documentContents[string(uri)]
	if !ok {
		return ""
	}
	return content
}

func getExtendsClasses(uri protocol.DocumentUri, className string) []string {
	documentContent := getDocumentContent(uri)
	if documentContent == "" {
		return []string{}
	}

	tmpFile, err := os.CreateTemp("", "ruby-ti-lsp-*.rb")
	if err != nil {
		return []string{}
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(documentContent); err != nil {
		return []string{}
	}
	tmpFile.Close()

	cmd := exec.Command("ti", tmpFile.Name(), "--extends", "--class="+className)
	output, err := cmd.Output()
	if err != nil {
		return []string{}
	}

	var extendsClasses []string
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && line != className {
			extendsClasses = append(extendsClasses, line)
		}
	}

	return extendsClasses
}

func createMethodCodeActionForClass(
	errorInfo *ErrorInfo,
	diagnostic protocol.Diagnostic,
	targetClass string,
) *protocol.CodeAction {

	var title string

	switch targetClass {
	case errorInfo.ClassName:
		title = fmt.Sprintf(
			"Add method '%s' to class '%s'",
			errorInfo.MethodName,
			targetClass,
		)

	default:
		title = fmt.Sprintf(
			"Add method '%s' to extends class '%s'",
			errorInfo.MethodName,
			targetClass,
		)
	}

	configDir := findBuiltinConfigDir()
	if configDir == "" {
		return nil
	}

	filePath :=
		filepath.Join(configDir, strings.ToLower(targetClass)+".json")

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}

	var classConfig TiClassConfig
	if err := json.Unmarshal(data, &classConfig); err != nil {
		return nil
	}

	newMethod := TiMethod{
		Name: errorInfo.MethodName,
		Arguments: []TiArgument{
			{Type: []string{"Untyped"}},
		},
		ReturnType: TiReturnType{
			Type: []string{"Untyped"},
		},
	}

	if errorInfo.MethodType == "class" {
		classConfig.ClassMethods = append(classConfig.ClassMethods, newMethod)
	} else {
		classConfig.InstanceMethods = append(classConfig.InstanceMethods, newMethod)
	}

	jsonData, err := json.MarshalIndent(classConfig, "", "  ")
	if err != nil {
		return nil
	}

	changes := make(map[protocol.DocumentUri][]protocol.TextEdit)
	fileUri := protocol.DocumentUri("file://" + filePath)

	lines := strings.Split(string(data), "\n")
	lastLine := uint32(len(lines))

	changes[fileUri] = []protocol.TextEdit{
		{
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: lastLine, Character: 0},
			},
			NewText: string(jsonData) + "\n",
		},
	}

	edit := protocol.WorkspaceEdit{
		Changes: changes,
	}

	kind := protocol.CodeActionKindQuickFix

	return &protocol.CodeAction{
		Title:       title,
		Kind:        &kind,
		Diagnostics: []protocol.Diagnostic{diagnostic},
		Edit:        &edit,
	}
}

func createMethodCodeAction(
	errorInfo *ErrorInfo,
	diagnostic protocol.Diagnostic,
) *protocol.CodeAction {
	return createMethodCodeActionForClass(errorInfo, diagnostic, errorInfo.ClassName)
}

func getRubyTiPath() string {
	lines := strings.SplitSeq(envContent, "\n")

	for line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		path, ok := strings.CutPrefix(line, "RUBY_TI_PROJECT_PATH=")
		if ok {
			path = strings.TrimSpace(path)

			path = strings.Trim(path, "\"'")
			if path != "" {
				return path
			}
		}
	}
	return ""
}

func findBuiltinConfigDir() string {
	rubyTiPath := getRubyTiPath()
	if rubyTiPath == "" {
		return ""
	}

	configDir := filepath.Join(rubyTiPath, "builtin", "builtin_config")
	if stat, err := os.Stat(configDir); err == nil && stat.IsDir() {
		return configDir
	}

	return ""
}

func findBuiltinJsonPath(configDir string, className string) string {
	jsonPath := fmt.Sprintf("%s/%s.json", configDir, strings.ToLower(className))
	if _, err := os.Stat(jsonPath); err == nil {
		return jsonPath
	}
	return ""
}

func checkAndRunMakeInstall(uri protocol.DocumentUri) {
	filePath := strings.TrimPrefix(string(uri), "file://")

	if !strings.HasSuffix(filePath, ".json") {
		return
	}

	configDir := findBuiltinConfigDir()
	if configDir == "" {
		return
	}

	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return
	}

	absConfigDir, err := filepath.Abs(configDir)
	if err != nil {
		return
	}

	if !strings.HasPrefix(absFilePath, absConfigDir) {
		return
	}

	rubyTiPath := getRubyTiPath()
	if rubyTiPath == "" {
		return
	}

	cmd := exec.Command("make", "install")
	cmd.Dir = rubyTiPath
	cmd.Run()
}
