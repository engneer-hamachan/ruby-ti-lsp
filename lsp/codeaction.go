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

// SetEnvContent sets the embedded .env content
func SetEnvContent(content string) {
	envContent = content
}

// TiClassConfig represents the JSON structure for ti builtin config
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
	Type            []string `json:"type"`
	IsConditional   bool     `json:"is_conditional,omitempty"`
	IsDestructive   bool     `json:"is_destructive,omitempty"`
}

// ErrorInfo contains parsed error information from ti diagnostics
type ErrorInfo struct {
	Line         uint32
	ErrorType    string // "class" or "method"
	ClassName    string
	MethodName   string
	MethodType   string // "instance" or "class"
	ErrorMessage string
}

func textDocumentCodeAction(
	ctx *glsp.Context,
	params *protocol.CodeActionParams,
) (any, error) {
	var codeActions []protocol.CodeAction

	// Parse diagnostics to find errors
	for _, diagnostic := range params.Context.Diagnostics {
		errorInfo := parseErrorMessage(diagnostic.Message, diagnostic.Range.Start.Line)
		if errorInfo == nil {
			continue
		}

		// Create appropriate code action based on error type
		if errorInfo.ErrorType == "class" {
			action := createClassCodeAction(errorInfo, params.TextDocument.URI, diagnostic)
			if action != nil {
				codeActions = append(codeActions, *action)
			}
		} else if errorInfo.ErrorType == "method" {
			action := createMethodCodeAction(errorInfo, params.TextDocument.URI, diagnostic)
			if action != nil {
				codeActions = append(codeActions, *action)
			}
		}
	}

	return codeActions, nil
}

// parseErrorMessage parses ti error messages to extract error information
func parseErrorMessage(message string, line uint32) *ErrorInfo {
	// Pattern: class 'Hoge' is not defined
	classPattern := regexp.MustCompile(`class '([^']+)' is not defined`)
	if matches := classPattern.FindStringSubmatch(message); len(matches) > 1 {
		return &ErrorInfo{
			Line:         line,
			ErrorType:    "class",
			ClassName:    matches[1],
			ErrorMessage: message,
		}
	}

	// Pattern: instance method 'xxx' is not defined for Test
	instanceMethodPattern := regexp.MustCompile(`instance method '([^']+)' is not defined for ([^\s]+)`)
	if matches := instanceMethodPattern.FindStringSubmatch(message); len(matches) > 2 {
		return &ErrorInfo{
			Line:         line,
			ErrorType:    "method",
			MethodName:   matches[1],
			ClassName:    matches[2],
			MethodType:   "instance",
			ErrorMessage: message,
		}
	}

	// Pattern: class method 'xxx' is not defined for Test
	classMethodPattern := regexp.MustCompile(`class method '([^']+)' is not defined for ([^\s]+)`)
	if matches := classMethodPattern.FindStringSubmatch(message); len(matches) > 2 {
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

// createClassCodeAction creates a code action to generate a new class JSON file
func createClassCodeAction(
	errorInfo *ErrorInfo,
	uri protocol.DocumentUri,
	diagnostic protocol.Diagnostic,
) *protocol.CodeAction {
	title := fmt.Sprintf("Create class definition for '%s'", errorInfo.ClassName)

	// Find ruby-ti builtin_config directory
	configDir := findBuiltinConfigDir()
	if configDir == "" {
		return nil
	}

	filePath := filepath.Join(configDir, strings.ToLower(errorInfo.ClassName)+".json")

	// Create new class config with proper structure
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

	// Create workspace edit to write the file
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

// createMethodCodeAction creates a code action to add a method to existing class JSON
func createMethodCodeAction(
	errorInfo *ErrorInfo,
	uri protocol.DocumentUri,
	diagnostic protocol.Diagnostic,
) *protocol.CodeAction {
	title := fmt.Sprintf("Add method '%s' to class '%s'", errorInfo.MethodName, errorInfo.ClassName)

	// Find ruby-ti builtin_config directory
	configDir := findBuiltinConfigDir()
	if configDir == "" {
		return nil
	}

	filePath := filepath.Join(configDir, strings.ToLower(errorInfo.ClassName)+".json")

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil
	}

	// Read existing config
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}

	var classConfig TiClassConfig
	if err := json.Unmarshal(data, &classConfig); err != nil {
		return nil
	}

	// Add new method with proper structure
	newMethod := TiMethod{
		Name: errorInfo.MethodName,
		Arguments: []TiArgument{
			{Type: []string{"Untyped"}},
		},
		ReturnType: TiReturnType{
			Type: []string{"Untyped"},
		},
	}

	// Add to appropriate method list based on method type
	if errorInfo.MethodType == "class" {
		classConfig.ClassMethods = append(classConfig.ClassMethods, newMethod)
	} else {
		classConfig.InstanceMethods = append(classConfig.InstanceMethods, newMethod)
	}

	// Marshal back to JSON
	jsonData, err := json.MarshalIndent(classConfig, "", "  ")
	if err != nil {
		return nil
	}

	// Write file directly
	if err := os.WriteFile(filePath, []byte(string(jsonData)+"\n"), 0644); err != nil {
		return nil
	}

	// Run make install in ruby-ti directory
	rubyTiPath := getRubyTiPath()
	if rubyTiPath != "" {
		cmd := exec.Command("make", "install")
		cmd.Dir = rubyTiPath
		cmd.Run() // Ignore errors for now
	}

	// Create workspace edit to open the file
	fileUri := protocol.DocumentUri("file://" + filePath)
	command := &protocol.Command{
		Title:     "Open file",
		Command:   "vscode.open",
		Arguments: []any{fileUri},
	}

	kind := protocol.CodeActionKindQuickFix

	return &protocol.CodeAction{
		Title:       title,
		Kind:        &kind,
		Diagnostics: []protocol.Diagnostic{diagnostic},
		Command:     command,
	}
}

// getRubyTiPath reads RUBY_TI_PATH from embedded .env file
func getRubyTiPath() string {
	lines := strings.Split(envContent, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Parse RUBY_TI_PATH=value
		if strings.HasPrefix(line, "RUBY_TI_PATH=") {
			path := strings.TrimPrefix(line, "RUBY_TI_PATH=")
			path = strings.TrimSpace(path)
			// Remove quotes if present
			path = strings.Trim(path, "\"'")
			if path != "" {
				return path
			}
		}
	}
	return ""
}

// findBuiltinConfigDir attempts to find the ruby-ti builtin_config directory
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
