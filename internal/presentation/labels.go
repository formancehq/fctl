package presentation

import (
	"slices"
	"strings"
	"unicode"
)

func fieldLabel(key string) string {
	if strings.ContainsFunc(key, func(char rune) bool {
		return unicode.IsControl(char) || unicode.Is(unicode.Cf, char)
	}) {
		return escape(key)
	}
	chars := []rune(key)
	var words strings.Builder
	for i, char := range chars {
		if char == '_' || char == '-' {
			char = ' '
		} else if wordBoundary(chars, i) {
			if _, err := words.WriteRune(' '); err != nil {
				return escape(key)
			}
		}
		if _, err := words.WriteRune(char); err != nil {
			return escape(key)
		}
	}
	labels := strings.Fields(words.String())
	for i, word := range labels {
		labels[i] = labelWord(word)
	}
	return strings.Join(labels, " ")
}

func labelWord(word string) string {
	upper := strings.ToUpper(word)
	if upper == "IDS" {
		return "IDs"
	}
	if slices.Contains([]string{"ID", "URL", "URI", "API", "HTTP", "JSON", "JWT", "OIDC", "UUID"}, upper) {
		return upper
	}
	letters := []rune(word)
	letters[0] = unicode.ToUpper(letters[0])
	return string(letters)
}

func wordBoundary(chars []rune, i int) bool {
	if i == 0 || !unicode.IsUpper(chars[i]) {
		return false
	}
	if unicode.IsLower(chars[i-1]) || unicode.IsDigit(chars[i-1]) {
		return true
	}
	// Preserve plural acronyms such as IDs instead of splitting them as I Ds.
	if i+1 < len(chars) && chars[i+1] == 's' && (i+2 == len(chars) || !unicode.IsLower(chars[i+2])) {
		return false
	}
	return unicode.IsUpper(chars[i-1]) && i+1 < len(chars) && unicode.IsLower(chars[i+1])
}
