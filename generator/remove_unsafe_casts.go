package main

import (
	"strings"
)

// RemoveUnsafeCasts attempts to replace unsafe.Pointer casts with direct static casts
// where possible, making the generated code cleaner and avoiding unsafe imports.
func RemoveUnsafeCasts(src []byte) []byte {
	code := string(src)

	// Pattern: (*(*TargetType)(unsafe.Pointer(&variable))) -> TargetType(variable)
	// This handles cases like (*(*H)(unsafe.Pointer(&key))) -> H(key)

	// Find and replace unsafe pointer patterns
	result := code

	// Pattern 1: (*(*Type)(unsafe.Pointer(&var))) -> Type(var)
	// This is safe when Type and var have the same underlying type
	for {
		// Look for the pattern: (*(*
		start := strings.Index(result, "(*(*")
		if start == -1 {
			break
		}

		// Find the matching closing parentheses
		pos := start + 4 // skip "(*(*"

		// Extract the target type
		typeEnd := strings.Index(result[pos:], ")")
		if typeEnd == -1 {
			break
		}
		targetType := result[pos : pos+typeEnd]
		pos += typeEnd + 1 // skip type and ")"

		// Check for "(unsafe.Pointer(&"
		unsafePattern := "(unsafe.Pointer(&"
		if !strings.HasPrefix(result[pos:], unsafePattern) {
			// Not our pattern, skip
			result = result[:start] + result[start+1:] // remove one char and continue
			continue
		}
		pos += len(unsafePattern)

		// Extract the variable name
		varEnd := strings.Index(result[pos:], "))")
		if varEnd == -1 {
			break
		}
		varName := result[pos : pos+varEnd]
		endPos := pos + varEnd + 2 // position after "))"

		// Find the end of the entire expression - look for the final closing parenthesis
		finalPos := endPos
		if finalPos < len(result) && result[finalPos] == ')' {
			finalPos++ // skip the final closing parenthesis of the outer cast
		}

		// Replace the entire unsafe cast with a direct cast
		newPattern := targetType + "(" + varName + ")"

		result = result[:start] + newPattern + result[finalPos:]
	}

	// Pattern 2: Remove unused unsafe import if no more unsafe references
	if !strings.Contains(result, "unsafe.") {
		// Remove the unsafe import line
		lines := strings.Split(result, "\n")
		var filteredLines []string
		inImportBlock := false
		emptyImportBlock := true

		for _, line := range lines {
			trimmed := strings.TrimSpace(line)

			// Detect import block
			if strings.HasPrefix(trimmed, "import (") {
				inImportBlock = true
				emptyImportBlock = true
				filteredLines = append(filteredLines, line)
				continue
			}
			if inImportBlock && trimmed == ")" {
				inImportBlock = false
				// Only add the closing paren if the import block is not empty
				if !emptyImportBlock {
					filteredLines = append(filteredLines, line)
				} else {
					// Remove the entire empty import block
					filteredLines = filteredLines[:len(filteredLines)-1] // remove "import ("
				}
				continue
			}

			// Skip unsafe import line
			if inImportBlock && (trimmed == `"unsafe"` || strings.Contains(trimmed, `"unsafe"`)) {
				continue
			}

			// Mark import block as non-empty if we have other imports
			if inImportBlock && trimmed != "" && !strings.Contains(trimmed, "unsafe") {
				emptyImportBlock = false
			}

			// Also handle single-line unsafe import
			if trimmed == `import "unsafe"` {
				continue
			}

			filteredLines = append(filteredLines, line)
		}
		result = strings.Join(filteredLines, "\n")
	}

	// Use sanitizeCode to format and clean up the result
	sanitized, err := sanitizeCode([]byte(result))
	if err != nil {
		// If sanitization fails, return the unsanitized result
		return []byte(result)
	}

	return sanitized
}
