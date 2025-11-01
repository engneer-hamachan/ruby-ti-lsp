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

func makeTypeDetail(typeName string) string {
	typeDescriptions := map[string]string{
		// Basic types
		"Int":    "Integer type - represents whole numbers",
		"Float":  "Float type - represents decimal numbers",
		"String": "String type - represents text",
		"Bool":   "Boolean type - true or false",
		"Nil":    "Nil type - represents absence of value",
		"Symbol": "Symbol type - immutable identifier",

		// Container types
		"Array":       "Array type - collection of elements",
		"Hash":        "Hash type - key-value mapping",
		"IntArray":    "Array of integers - specialized array containing only Int elements",
		"FloatArray":  "Array of floats - specialized array containing only Float elements",
		"StringArray": "Array of strings - specialized array containing only String elements",

		// Default types (used when type cannot be inferred)
		"DefaultInt":     "Default integer - fallback type when Int inference fails",
		"DefaultFloat":   "Default float - fallback type when Float inference fails",
		"DefaultString":  "Default string - fallback type when String inference fails",
		"DefaultBlock":   "Default block - fallback type when Block inference fails",
		"DefaultUntyped": "Default untyped - fallback when no type can be inferred",

		// Block and functional types
		"Block":            "Block type - represents a code block/lambda",
		"BlockResultArray": "Array of block results - array containing elements returned by block execution",
		"KeyValueArray":    "Array of key-value pairs - array created from hash entries",
		"KeyArray":         "Array of keys - array containing hash keys",

		// Special type system types
		"Untyped":             "Untyped - represents values without type constraints",
		"Unify":               "Unify variants - combines multiple type variants into one unified type",
		"OptionalUnify":       "Optional unify - unifies variants and adds Nil as possible type",
		"SelfConvertArray":    "Self converted to array - converts receiver object into array of its variants",
		"SelfArgument":        "Self argument - returns Nil, single arg, or array based on argument count",
		"UnifiedSelfArgument": "Unified self argument - unified version of self argument type",
		"Flatten":             "Flatten - flattens nested structures into single level",

		// Union and special
		"Number":  "Number type - union of Int and Float",
		"Union":   "Union type - represents multiple possible types",
		"Self":    "Self type - refers to the receiver objects type",
		"Range":   "Range type - represents a range of values",
		"Keyword": "Keyword argument - named parameter in method call",

		// Test/other
		"IntInt": "Int or Int union - used for testing union types",
	}

	if desc, ok := typeDescriptions[typeName]; ok {
		return desc
	}
	return typeName
}

func findJsonTypeCompletion(content string, line uint32, character uint32) []Sig {
	if !isInTypeArray(content, line, character) {
		return []Sig{}
	}

	types := getAllTypes()
	var signatures []Sig
	for _, typeName := range types {
		if typeName == "IntInt" {
			continue
		}

		signatures = append(signatures, Sig{
			Method: typeName,
			Detail: makeTypeDetail(typeName),
		})
	}

	return signatures
}
