package resolver

import (
	"reflect"
	"testing"

	"github.com/gubarz/cheatmd/pkg/parser"
)

func TestBuildMatchPattern(t *testing.T) {
	tests := []struct {
		name         string
		cmd          string
		wantVarNames []string
	}{
		{
			name:         "simple var",
			cmd:          "echo $name",
			wantVarNames: []string{"name"},
		},
		{
			name:         "var with double quotes",
			cmd:          `curl "$url"`,
			wantVarNames: []string{"url"},
		},
		{
			name:         "var with single quotes",
			cmd:          `ssh '$user'@host`,
			wantVarNames: []string{"user"},
		},
		{
			name:         "multiple vars",
			cmd:          "tool run --port $port $host",
			wantVarNames: []string{"port", "host"},
		},
		{
			name:         "no vars",
			cmd:          "echo hello world",
			wantVarNames: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pattern, varNames := buildMatchPattern(tt.cmd)

			if pattern == nil && len(tt.wantVarNames) > 0 {
				t.Fatalf("buildMatchPattern() returned nil pattern, expected vars %v", tt.wantVarNames)
			}

			if len(varNames) != len(tt.wantVarNames) {
				t.Fatalf("buildMatchPattern() varNames = %v, want %v", varNames, tt.wantVarNames)
			}

			for i := range varNames {
				if varNames[i] != tt.wantVarNames[i] {
					t.Errorf("buildMatchPattern() varNames[%d] = %q, want %q", i, varNames[i], tt.wantVarNames[i])
				}
			}
		})
	}
}

func TestParseShellArgs(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "double quoted delimiter",
			input: `--delimiter "\t" --column 2`,
			want:  []string{"--delimiter", `\t`, "--column", "2"},
		},
		{
			name:  "single quoted",
			input: `--delimiter '\t' --column 1`,
			want:  []string{"--delimiter", `\t`, "--column", "1"},
		},
		{
			name:  "double quoted with space",
			input: `--header "Pick a host" --column 1`,
			want:  []string{"--header", "Pick a host", "--column", "1"},
		},
		{
			name:  "no args",
			input: "",
			want:  nil,
		},
		{
			name:  "extra whitespace",
			input: `  --delimiter   ","  `,
			want:  []string{"--delimiter", ","},
		},
		{
			name:  "map command",
			input: `--map "awk '{print $1}'"`,
			want:  []string{"--map", "awk '{print $1}'"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseShellArgs(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseShellArgs(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseSelectorOpts(t *testing.T) {
	tests := []struct {
		name string
		args string
		want SelectOptions
	}{
		{
			name: "delimiter and column",
			args: `--delimiter "\t" --column 2`,
			want: SelectOptions{Delimiter: `\t`, Column: 2},
		},
		{
			name: "all options",
			args: `--delimiter "," --column 2 --select-column 1 --map "cut -d: -f1"`,
			want: SelectOptions{Delimiter: ",", Column: 2, SelectColumn: 1, MapCmd: "cut -d: -f1"},
		},
		{
			name: "header is ignored in SelectOptions",
			args: `--header "Pick one" --delimiter ":"`,
			want: SelectOptions{Delimiter: ":"},
		},
		{
			name: "empty args",
			args: "",
			want: SelectOptions{},
		},
		{
			name: "select-column only",
			args: `--select-column 3`,
			want: SelectOptions{SelectColumn: 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseSelectorOpts(tt.args)
			if got != tt.want {
				t.Errorf("ParseSelectorOpts(%q) = %+v, want %+v", tt.args, got, tt.want)
			}
		})
	}
}

func TestGetDisplayColumn(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		delimiter string
		column    int
		want      string
	}{
		{
			name:      "tab delimited column 2",
			line:      "192.168.1.1\twebserver\tlinux",
			delimiter: "\t",
			column:    2,
			want:      "webserver",
		},
		{
			name:      "comma delimited column 1",
			line:      "primary,blue,active",
			delimiter: ",",
			column:    1,
			want:      "primary",
		},
		{
			name:      "column out of range returns full line",
			line:      "one\ttwo",
			delimiter: "\t",
			column:    5,
			want:      "one\ttwo",
		},
		{
			name:      "column 0 returns full line",
			line:      "one\ttwo\tthree",
			delimiter: "\t",
			column:    0,
			want:      "one\ttwo\tthree",
		},
		{
			name:      "empty delimiter returns full line",
			line:      "one\ttwo",
			delimiter: "",
			column:    2,
			want:      "one\ttwo",
		},
		{
			name:      "trims whitespace",
			line:      "one\t  two  \tthree",
			delimiter: "\t",
			column:    2,
			want:      "two",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetDisplayColumn(tt.line, tt.delimiter, tt.column)
			if got != tt.want {
				t.Errorf("GetDisplayColumn(%q, %q, %d) = %q, want %q", tt.line, tt.delimiter, tt.column, got, tt.want)
			}
		})
	}
}

func TestApplyMapTransform_SelectColumn(t *testing.T) {
	tests := []struct {
		name  string
		value string
		opts  SelectOptions
		want  string
	}{
		{
			name:  "extract column 1 from tab-delimited",
			value: "192.168.1.1\twebserver\tlinux",
			opts:  SelectOptions{Delimiter: "\t", SelectColumn: 1},
			want:  "192.168.1.1",
		},
		{
			name:  "extract column 2 from comma-delimited",
			value: "admin,secret,active",
			opts:  SelectOptions{Delimiter: ",", SelectColumn: 2},
			want:  "secret",
		},
		{
			name:  "column out of range returns original",
			value: "one\ttwo",
			opts:  SelectOptions{Delimiter: "\t", SelectColumn: 10},
			want:  "one\ttwo",
		},
		{
			name:  "no select-column returns original",
			value: "one\ttwo\tthree",
			opts:  SelectOptions{Delimiter: "\t", SelectColumn: 0},
			want:  "one\ttwo\tthree",
		},
		{
			name:  "no delimiter returns original",
			value: "one\ttwo",
			opts:  SelectOptions{Delimiter: "", SelectColumn: 1},
			want:  "one\ttwo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ApplyMapTransform(tt.value, tt.opts)
			if got != tt.want {
				t.Errorf("ApplyMapTransform(%q, %+v) = %q, want %q", tt.value, tt.opts, got, tt.want)
			}
		})
	}
}

func TestEndToEnd_DelimiterColumnPipeline(t *testing.T) {
	shellOutput := "192.168.1.1,webserver\n10.0.0.1,db"
	selectorArgs := `--delimiter "," --column 2 --select-column 1`

	lines := parser.SplitLines(shellOutput)
	if len(lines) != 2 {
		t.Fatalf("splitLines() = %d lines, want 2", len(lines))
	}

	opts := ParseSelectorOpts(selectorArgs)
	if opts.Delimiter != "," {
		t.Errorf("opts.Delimiter = %q, want %q", opts.Delimiter, ",")
	}
	if opts.Column != 2 {
		t.Errorf("opts.Column = %d, want 2", opts.Column)
	}
	if opts.SelectColumn != 1 {
		t.Errorf("opts.SelectColumn = %d, want 1", opts.SelectColumn)
	}

	display0 := GetDisplayColumn(lines[0], opts.Delimiter, opts.Column)
	display1 := GetDisplayColumn(lines[1], opts.Delimiter, opts.Column)
	if display0 != "webserver" {
		t.Errorf("display[0] = %q, want %q", display0, "webserver")
	}
	if display1 != "db" {
		t.Errorf("display[1] = %q, want %q", display1, "db")
	}

	selected := ApplyMapTransform(lines[0], opts)
	if selected != "192.168.1.1" {
		t.Errorf("ApplyMapTransform() = %q, want %q", selected, "192.168.1.1")
	}

	selected2 := ApplyMapTransform(lines[1], opts)
	if selected2 != "10.0.0.1" {
		t.Errorf("ApplyMapTransform() = %q, want %q", selected2, "10.0.0.1")
	}
}
