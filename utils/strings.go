package utils

import "strings"

func ContainsArabicNumerals(s string) bool {
	arabicNumerals := []rune{'٠', '١', '٢', '٣', '٤', '٥', '٦', '٧', '٨', '٩'}
	for _, char := range s {
		for _, num := range arabicNumerals {
			if char == num {
				return true
			}
		}
	}
	return false
}

func NormalizeNumerals(s string) string {
	arabicToEnglish := map[rune]rune{
		'٠': '0', '١': '1', '٢': '2', '٣': '3', '٤': '4',
		'٥': '5', '٦': '6', '٧': '7', '٨': '8', '٩': '9',
	}
	var result strings.Builder
	for _, r := range s {
		if val, ok := arabicToEnglish[r]; ok {
			result.WriteRune(val)
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}
