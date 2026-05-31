package ui

import "regexp"

var (
	ansiRegex    = regexp.MustCompile("[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))")
	controlRegex = regexp.MustCompile("[\r\x07\x08\x7F\x0B\x0C]")
)

// StripANSI removes terminal escape sequences and dangerous control characters
// (like \r, BEL, Backspace, DEL) while preserving \n and \t.
// This prevents maliciously crafted cheat fields from rewriting the terminal
// screen or executing control sequences when rendered.
func StripANSI(str string) string {
	s := ansiRegex.ReplaceAllString(str, "")
	return controlRegex.ReplaceAllString(s, "")
}
