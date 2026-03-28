package utils

import "strings"

var mdV2Replacer = strings.NewReplacer(
	"_", "\\_",
	"*", "\\*",
	"`", "\\`",
	"[", "\\[",
	"]", "\\]",
	"(", "\\(",
	")", "\\)",
	"~", "\\~",
	">", "\\>",
	"#", "\\#",
	"+", "\\+",
	"-", "\\-",
	"=", "\\=",
	"|", "\\|",
	"{", "\\{",
	"}", "\\}",
	".", "\\.",
	"!", "\\!",
)

func EscapeMarkdownV2(s string) string {
	return mdV2Replacer.Replace(s)
}
