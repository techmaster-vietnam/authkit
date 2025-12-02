package utils

import "strings"

// PascalToSnakeCase converts PascalCase to snake_case
// Ví dụ: "Mobile" -> "mobile", "FullName" -> "full_name"
func PascalToSnakeCase(pascal string) string {
	if pascal == "" {
		return pascal
	}

	var result strings.Builder
	for i, r := range pascal {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteByte('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}
