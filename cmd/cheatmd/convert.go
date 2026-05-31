package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cheatmd-dev/cheatmd/pkg/convert"
)

var convertCmd = &cobra.Command{
	Use:   "convert [format] [input]",
	Short: "Convert cheatsheets from other formats (navi, tldr, cheat) to CheatMD",
	Long: `Convert cheatsheets from other popular formats into CheatMD executable markdown format.

Supported formats:
  - navi: Converts .cheat files (replaces <var> with $var, parses tags, variables, and extends/imports)
  - tldr: Converts TLDR markdown pages (replaces {{var}} with $var, creates interactive prompts)
  - cheat: Converts cheat/cheat plain-text cheatsheets (parses frontmatter and comments)

The input can be a single file or a directory. If a directory is provided, it will recursively find and convert all matching files.

Examples:
  cheatmd convert navi git.cheat -o git.md
  cheatmd convert tldr ~/tldr/pages/common/tar.md -o tar.md
  cheatmd convert cheat ~/cheats/personal/ -o ~/my-cheats/`,
	Args: cobra.ExactArgs(2),
	RunE: runConvert,
}

func init() {
	convertCmd.Flags().StringP("output", "o", ".", "Output file or directory path (defaults to current directory)")
}

func runConvert(cmd *cobra.Command, args []string) error {
	format := strings.ToLower(args[0])
	inputPath := args[1]
	outputPath, _ := cmd.Flags().GetString("output")

	if format != "navi" && format != "tldr" && format != "cheat" {
		return fmt.Errorf("invalid format %q: must be one of: navi, tldr, cheat", format)
	}

	inputAbs, err := filepath.Abs(inputPath)
	if err != nil {
		return fmt.Errorf("failed to resolve input path: %w", err)
	}

	info, err := os.Stat(inputAbs)
	if err != nil {
		return fmt.Errorf("input path error: %w", err)
	}

	if info.IsDir() {
		// navi conversion is special-cased: @extends crosses file boundaries
		// so we need to parse every .cheat file into a shared index before
		// emitting any of them. The other formats are still per-file.
		if format == "navi" {
			return convert.ConvertNaviDirectory(inputAbs, outputPath)
		}
		return convert.ConvertDirectory(format, inputAbs, outputPath)
	}
	return convert.ConvertFile(format, inputAbs, outputPath)
}
