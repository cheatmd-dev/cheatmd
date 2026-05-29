package resolver

import (
	"os/exec"
	"strings"

	"github.com/gubarz/cheatmd/pkg/config"
)

// GetDisplayColumn extracts the display column from a line based on a delimiter.
func GetDisplayColumn(line, delimiter string, column int) string {
	if delimiter == "" || column == 0 {
		return line
	}
	parts := strings.Split(line, delimiter)
	if column > 0 && column <= len(parts) {
		return strings.TrimSpace(parts[column-1])
	}
	return line
}

// ParseShellArgs parses a string into arguments, respecting quotes.
func ParseShellArgs(s string) []string {
	var args []string
	var current strings.Builder
	var inQuote bool
	var quoteChar byte

	for i := 0; i < len(s); i++ {
		c := s[i]

		if inQuote {
			if c == quoteChar {
				inQuote = false
			} else {
				current.WriteByte(c)
			}
		} else {
			switch c {
			case '"', '\'':
				inQuote = true
				quoteChar = c
			case ' ', '\t':
				if current.Len() > 0 {
					args = append(args, current.String())
					current.Reset()
				}
			default:
				current.WriteByte(c)
			}
		}
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}

// ApplyMapTransform transforms the selected value based on delimiter, select column, and map command options.
func ApplyMapTransform(value string, opts SelectOptions) string {
	// Apply select-column extraction first
	if opts.SelectColumn > 0 && opts.Delimiter != "" {
		parts := strings.Split(value, opts.Delimiter)
		if opts.SelectColumn <= len(parts) {
			value = strings.TrimSpace(parts[opts.SelectColumn-1])
		}
	}

	// Then apply map command if present
	if opts.MapCmd == "" {
		return value
	}

	// Run the map command with the value as stdin
	cmd := exec.Command(config.GetShell(), "-c", opts.MapCmd)
	cmd.Stdin = strings.NewReader(value)
	out, err := cmd.Output()
	if err != nil {
		return value // fallback to original on error
	}
	return strings.TrimSpace(string(out))
}
