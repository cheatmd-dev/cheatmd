package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/gubarz/cheatmd/pkg/convert"
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
			return convertNaviDirectory(inputAbs, outputPath)
		}
		return convertDirectory(format, inputAbs, outputPath)
	}
	return convertFile(format, inputAbs, outputPath)
}

// convertNaviDirectory walks every .cheat file in inputDir, parses them all
// into a shared NaviIndex (so @extends references can resolve across files),
// and writes one converted markdown file per source under outputDir.
func convertNaviDirectory(inputDir, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
	}

	sources, rels, err := collectNaviSources(inputDir)
	if err != nil {
		return err
	}
	if len(sources) == 0 {
		return nil
	}

	results := convert.ConvertNaviTree(sources)
	for i, res := range results {
		rel := rels[i]
		relBase := strings.TrimSuffix(rel, filepath.Ext(rel))
		targetFile := filepath.Join(outputDir, relBase+".md")
		if err := os.MkdirAll(filepath.Dir(targetFile), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", targetFile, err)
		}
		if err := os.WriteFile(targetFile, []byte(res.Content), 0644); err != nil {
			return fmt.Errorf("failed to write converted file %s: %w", targetFile, err)
		}
		fmt.Printf("✓ Converted %s (navi) -> %s\n", rel, targetFile)
	}
	return nil
}

// collectNaviSources walks inputDir, returning every .cheat file's content
// alongside its relative path so the writer can preserve directory structure.
func collectNaviSources(inputDir string) ([]convert.NaviSource, []string, error) {
	var sources []convert.NaviSource
	var rels []string

	err := filepath.WalkDir(inputDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") && d.Name() != "." {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".cheat") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}
		rel, err := filepath.Rel(inputDir, path)
		if err != nil {
			return err
		}
		sources = append(sources, convert.NaviSource{Path: path, Content: string(data)})
		rels = append(rels, rel)
		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("error walking directory: %w", err)
	}
	return sources, rels, nil
}

func convertFile(format, inputPath, outputPath string) error {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", inputPath, err)
	}

	var converted string
	switch format {
	case "navi":
		converted, err = convert.ConvertNavi(string(data), inputPath)
	case "tldr":
		converted, err = convert.ConvertTldr(string(data), inputPath)
	case "cheat":
		converted, err = convert.ConvertCheat(string(data), inputPath)
	}

	if err != nil {
		return fmt.Errorf("conversion failed: %w", err)
	}

	targetFile := outputPath
	outInfo, err := os.Stat(outputPath)
	if err == nil && outInfo.IsDir() {
		// Output is an existing directory, save as <basename>.md
		base := filepath.Base(inputPath)
		base = strings.TrimSuffix(base, filepath.Ext(base))
		targetFile = filepath.Join(outputPath, base+".md")
	} else if err != nil && (strings.HasSuffix(outputPath, "/") || strings.HasSuffix(outputPath, "\\")) {
		// Output doesn't exist but looks like a directory path
		if err := os.MkdirAll(outputPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", outputPath, err)
		}
		base := filepath.Base(inputPath)
		base = strings.TrimSuffix(base, filepath.Ext(base))
		targetFile = filepath.Join(outputPath, base+".md")
	} else {
		// Output is a file path, ensure directory exists
		dir := filepath.Dir(outputPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	if err := os.WriteFile(targetFile, []byte(converted), 0644); err != nil {
		return fmt.Errorf("failed to write output to %s: %w", targetFile, err)
	}

	fmt.Printf("✓ Converted %s (%s) -> %s\n", filepath.Base(inputPath), format, targetFile)
	return nil
}

func convertDirectory(format, inputDir, outputDir string) error {
	// Create output dir if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
	}

	err := filepath.WalkDir(inputDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			// Skip hidden dirs
			if strings.HasPrefix(d.Name(), ".") && d.Name() != "." {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if the file matches our format's expected extension/criteria
		shouldConvert := false
		switch format {
		case "navi":
			shouldConvert = strings.HasSuffix(strings.ToLower(d.Name()), ".cheat")
		case "tldr":
			shouldConvert = strings.HasSuffix(strings.ToLower(d.Name()), ".md")
		case "cheat":
			// cheat/cheat community files typically have no extension and aren't hidden
			shouldConvert = !strings.Contains(d.Name(), ".") && !strings.HasPrefix(d.Name(), "_")
		}

		if !shouldConvert {
			return nil
		}

		// Calculate relative path to maintain structure
		rel, err := filepath.Rel(inputDir, path)
		if err != nil {
			return err
		}

		relBase := strings.TrimSuffix(rel, filepath.Ext(rel))
		targetFile := filepath.Join(outputDir, relBase+".md")

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}

		var converted string
		switch format {
		case "navi":
			converted, err = convert.ConvertNavi(string(data), path)
		case "tldr":
			converted, err = convert.ConvertTldr(string(data), path)
		case "cheat":
			converted, err = convert.ConvertCheat(string(data), path)
		}

		if err != nil {
			return fmt.Errorf("failed to convert file %s: %w", path, err)
		}

		// Ensure parent directory exists for the output file
		parentDir := filepath.Dir(targetFile)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", parentDir, err)
		}

		if err := os.WriteFile(targetFile, []byte(converted), 0644); err != nil {
			return fmt.Errorf("failed to write converted file %s: %w", targetFile, err)
		}

		fmt.Printf("✓ Converted %s (%s) -> %s\n", rel, format, targetFile)
		return nil
	})

	if err != nil {
		return fmt.Errorf("error walking directory: %w", err)
	}

	return nil
}
