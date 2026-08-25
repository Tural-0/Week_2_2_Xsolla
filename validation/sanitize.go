package validation

import "strings"

func String(value string) string {
	return strings.TrimSpace(value)
}

func Email(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
