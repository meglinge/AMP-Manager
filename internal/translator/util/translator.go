// Package util provides JSON manipulation utilities for translator operations.
package util

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// Walk recursively traverses a JSON structure to find all occurrences of a specific field.
// It builds paths to each occurrence and adds them to the provided paths slice.
func Walk(value gjson.Result, path, field string, paths *[]string) {
	switch value.Type {
	case gjson.JSON:
		// For JSON objects and arrays, iterate through each child
		value.ForEach(func(key, val gjson.Result) bool {
			var childPath string
			// Escape special characters for gjson/sjson path syntax
			var keyReplacer = strings.NewReplacer(".", "\\.", "*", "\\*", "?", "\\?")
			safeKey := keyReplacer.Replace(key.String())

			if path == "" {
				childPath = safeKey
			} else {
				childPath = path + "." + safeKey
			}
			if key.String() == field {
				*paths = append(*paths, childPath)
			}
			Walk(val, childPath, field, paths)
			return true
		})
	case gjson.String, gjson.Number, gjson.True, gjson.False, gjson.Null:
		// Terminal types - no further traversal needed
	}
}

// RenameKey renames a key in a JSON string by moving its value to a new key path
// and then deleting the old key path.
func RenameKey(jsonStr, oldKeyPath, newKeyPath string) (string, error) {
	value := gjson.Get(jsonStr, oldKeyPath)

	if !value.Exists() {
		return "", fmt.Errorf("old key '%s' does not exist", oldKeyPath)
	}

	interimJson, err := sjson.SetRaw(jsonStr, newKeyPath, value.Raw)
	if err != nil {
		return "", fmt.Errorf("failed to set new key '%s': %w", newKeyPath, err)
	}

	finalJson, err := sjson.Delete(interimJson, oldKeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to delete old key '%s': %w", oldKeyPath, err)
	}

	return finalJson, nil
}

// DeleteKey removes all occurrences of a key from a JSON string.
func DeleteKey(jsonStr, keyName string) string {
	paths := make([]string, 0)
	Walk(gjson.Parse(jsonStr), "", keyName, &paths)
	for _, p := range paths {
		jsonStr, _ = sjson.Delete(jsonStr, p)
	}
	return jsonStr
}

// FixJSON converts non-standard JSON that uses single quotes for strings into
// RFC 8259-compliant JSON by converting those single-quoted strings to
// double-quoted strings with proper escaping.
//
// Examples:
//
//	{'a': 1, 'b': '2'}      => {"a": 1, "b": "2"}
//	{"t": 'He said "hi"'} => {"t": "He said \"hi\""}
func FixJSON(input string) string {
	var out bytes.Buffer

	inDouble := false
	inSingle := false
	escaped := false // applies within the current string state

	// Helper to write a rune, escaping double quotes when inside a converted
	// single-quoted string (which becomes a double-quoted string in output).
	writeConverted := func(r rune) {
		if r == '"' {
			out.WriteByte('\\')
			out.WriteByte('"')
			return
		}
		out.WriteRune(r)
	}

	runes := []rune(input)
	for i := 0; i < len(runes); i++ {
		r := runes[i]

		if inDouble {
			out.WriteRune(r)
			if escaped {
				escaped = false
				continue
			}
			if r == '\\' {
				escaped = true
				continue
			}
			if r == '"' {
				inDouble = false
			}
			continue
		}

		if inSingle {
			if escaped {
				escaped = false
				switch r {
				case 'n', 'r', 't', 'b', 'f', '/', '"':
					out.WriteByte('\\')
					out.WriteRune(r)
				case '\\':
					out.WriteByte('\\')
					out.WriteByte('\\')
				case '\'':
					out.WriteRune('\'')
				case 'u':
					out.WriteByte('\\')
					out.WriteByte('u')
					for k := 0; k < 4 && i+1 < len(runes); k++ {
						peek := runes[i+1]
						if (peek >= '0' && peek <= '9') || (peek >= 'a' && peek <= 'f') || (peek >= 'A' && peek <= 'F') {
							out.WriteRune(peek)
							i++
						} else {
							break
						}
					}
				default:
					out.WriteByte('\\')
					out.WriteRune(r)
				}
				continue
			}

			if r == '\\' {
				escaped = true
				continue
			}
			if r == '\'' {
				out.WriteByte('"')
				inSingle = false
				continue
			}
			writeConverted(r)
			continue
		}

		// Outside any string
		if r == '"' {
			inDouble = true
			out.WriteRune(r)
			continue
		}
		if r == '\'' {
			inSingle = true
			out.WriteByte('"')
			continue
		}
		out.WriteRune(r)
	}

	if inSingle {
		out.WriteByte('"')
	}

	return out.String()
}
