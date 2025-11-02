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
	typeDetails := map[string]string{
		// Basic types
		"Int":    "Integer type",
		"Float":  "Float type",
		"String": "String type",
		"Bool":   "Boolean type",
		"Nil":    "Nil type",
		"Symbol": "Symbol type",

		// Container types
		"Array":       "Array type",
		"Hash":        "Hash type",
		"IntArray":    "Array of integers",
		"FloatArray":  "Array of floats",
		"StringArray": "Array of strings",

		// Default types (types with default values)
		"DefaultInt":     "Integer with default value",
		"DefaultFloat":   "Float with default value",
		"DefaultString":  "String with default value",
		"DefaultBlock":   "Block with default value",
		"DefaultUntyped": "Untyped with default value",

		// Block and functional types
		"Block":            "Block type",
		"BlockResultArray": "Array of block results",
		"KeyValueArray":    "Array of key-value pairs",
		"KeyArray":         "Array of hash keys",

		// Special type system types
		"Untyped":             "Untyped",
		"Unify":               "Unify variants",
		"OptionalUnify":       "Optional unify",
		"SelfConvertArray":    "Self converted to array",
		"SelfArgument":        "Self argument",
		"UnifiedSelfArgument": "Unified self argument",
		"Flatten":             "Flatten",

		// Union and special
		"Number":  "Number type (Int or Float)",
		"Union":   "Union type",
		"Self":    "Self type",
		"Range":   "Range type",
		"Keyword": "Keyword argument",

		// Test/other
		"IntInt": "Int or Int union",
	}

	if detail, ok := typeDetails[typeName]; ok {
		return detail
	}
	return typeName
}

func makeTypeDocumentation(typeName string) string {
	typeDocs := map[string]string{
		// Basic types
		"Int":    "Represents whole numbers.\n\nExamples: 42, -10, 0",
		"Float":  "Represents decimal numbers.\n\nExamples: 3.14, -0.5, 2.0",
		"String": "Represents text values.\n\nExamples: \"hello\", 'world'",
		"Bool":   "Represents boolean values.\n\nExamples: true, false",
		"Nil":    "Represents absence of value.\n\nExample: nil",
		"Symbol": "Represents immutable identifiers.\n\nExamples: :name, :status, :active",

		// Container types
		"Array":       "Ordered collection of elements.\n\nExamples:\n[1, 2, 3]\n['a', 'b', 'c']",
		"Hash":        "Key-value pairs mapping.\n\nExample:\n{name: 'Alice', age: 30}",
		"IntArray":    "Array containing only integers.\n\nExample:\n[1, 2, 3, 4, 5]",
		"FloatArray":  "Array containing only floats.\n\nExample:\n[1.5, 2.7, 3.14]",
		"StringArray": "Array containing only strings.\n\nExample:\n['foo', 'bar', 'baz']",

		// Default types (types with default values)
		"DefaultInt":     "Parameter with default integer value.\n\nExample:\ndef foo(x: 1)\n  # x has default value 1\nend",
		"DefaultFloat":   "Parameter with default float value.\n\nExample:\ndef foo(x: 1.5)\n  # x has default value 1.5\nend",
		"DefaultString":  "Parameter with default string value.\n\nExample:\ndef foo(name: \"default\")\n  # name has default value \"default\"\nend",
		"DefaultBlock":   "Parameter with default block value.\n\nExample:\ndef foo(block: ->{puts 1})\n  # block has default lambda\nend",
		"DefaultUntyped": "Parameter with default value of any type.\n\nExample:\ndef foo(x: some_value)\n  # x has default value\nend",

		// Block and functional types
		"Block":            "Code block or lambda.\n\nExamples:\n{|x| x * 2}\n->(x) { x + 1 }",
		"BlockResultArray": "Array of elements returned by block execution.\n\nExample:\n[1,2,3].map{|x| x*2}\n#=> [2,4,6]",
		"KeyValueArray":    "Array created from hash entries.\n\nExample:\n{a: 1, b: 2}.to_a\n#=> [[:a, 1], [:b, 2]]",
		"KeyArray":         "Array containing hash keys.\n\nExample:\n{a: 1, b: 2}.keys\n#=> [:a, :b]",

		// Special type system types
		"Untyped":             "Value without type constraints.\n\nAllows any operation without type checking.",
		"Unify":               "Merges type variants into single unified type.\n\nCombines Union<Int, String> variants into unified type.",
		"OptionalUnify":       "Unified type that may also be nil.\n\nUnifies variants and adds Nil as a possible type.",
		"SelfConvertArray":    "Converts receiver to array of its type variants.\n\nReceiver object is converted into array containing its variants.",
		"SelfArgument":        "Variable return type based on argument count.\n\n0 args: returns Nil\n1 arg: returns the value\n2+ args: returns Array",
		"UnifiedSelfArgument": "Unified version of SelfArgument type.\n\nCombined type from self argument variants.",
		"Flatten":             "Flattens nested array structures.\n\nExample:\n[[1,2],[3,4]].flatten\n#=> [1,2,3,4]",

		// Union and special
		"Number":  "Union of Int and Float types.\n\nExamples:\n42 (Int)\n3.14 (Float)",
		"Union":   "Represents multiple possible types.\n\nExample:\nString | Int | Nil",
		"Self":    "Refers to the receiver object's own type.\n\nReturns the type of the object itself.",
		"Range":   "Represents a sequence of values.\n\nExamples:\n1..10\n'a'..'z'",
		"Keyword": "Named parameter in method definition.\n\nExample:\ndef foo(name:, age:)\n  # name and age are keyword arguments\nend",

		// Test/other
		"IntInt": "Union type used for testing.\n\nInt | Int",
	}

	if doc, ok := typeDocs[typeName]; ok {
		return doc
	}
	return ""
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
			Method:        typeName,
			Detail:        makeTypeDetail(typeName),
			Documentation: makeTypeDocumentation(typeName),
		})
	}

	return signatures
}
