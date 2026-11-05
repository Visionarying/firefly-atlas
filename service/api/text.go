package api

import "strings"

// firstRuneTitle returns the first visible character of s, uppercased, or "?"
// when s has no usable character. Card icons fall back to a single letter, and
// slicing an empty rune slice would panic at runtime, so every caller goes
// through this helper instead of doing []rune(s)[0:1] directly.
func firstRuneTitle(s string) string {
	for _, r := range strings.TrimSpace(s) {
		return strings.ToUpper(string(r))
	}
	return "?"
}
