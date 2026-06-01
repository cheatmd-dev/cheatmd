package convert

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ConvertNaviDirectory walks every .cheat file in inputDir, parses them all
// into a shared NaviIndex (so @extends references can resolve across files),
// and writes one converted markdown file per source under outputDir.
func ConvertNaviDirectory(inputDir, outputDir string) error {
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

	results := ConvertNaviTree(sources)
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
func collectNaviSources(inputDir string) ([]NaviSource, []string, error) {
	var sources []NaviSource
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
		sources = append(sources, NaviSource{Path: path, Content: string(data)})
		rels = append(rels, rel)
		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("error walking directory: %w", err)
	}
	return sources, rels, nil
}

// ConvertFile converts a single file of the given format.
func ConvertFile(format, inputPath, outputPath string) error {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", inputPath, err)
	}

	var converted string
	switch format {
	case "navi":
		converted, err = ConvertNavi(string(data), inputPath)
	case "tldr":
		converted, err = ConvertTldr(string(data), inputPath)
	case "cheat":
		converted, err = ConvertCheat(string(data), inputPath)
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

// ConvertDirectory scans the input directory for legacy cheat sheet files
// matching the specified format (e.g., "navi") and converts them into standard
// cheatmd Markdown files in the output directory.
func ConvertDirectory(format, inputDir, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
	}

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

		if !shouldConvertFile(format, d.Name()) {
			return nil
		}

		return convertAndWriteFile(format, inputDir, outputDir, path)
	})

	if err != nil {
		return fmt.Errorf("error walking directory: %w", err)
	}

	return nil
}

func shouldConvertFile(format, filename string) bool {
	switch format {
	case "navi":
		return strings.HasSuffix(strings.ToLower(filename), ".cheat")
	case "tldr":
		return strings.HasSuffix(strings.ToLower(filename), ".md")
	case "cheat":
		return !strings.Contains(filename, ".") && !strings.HasPrefix(filename, "_")
	default:
		return false
	}
}

func convertAndWriteFile(format, inputDir, outputDir, path string) error {
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
		converted, err = ConvertNavi(string(data), path)
	case "tldr":
		converted, err = ConvertTldr(string(data), path)
	case "cheat":
		converted, err = ConvertCheat(string(data), path)
	}

	if err != nil {
		return fmt.Errorf("failed to convert file %s: %w", path, err)
	}

	parentDir := filepath.Dir(targetFile)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", parentDir, err)
	}

	if err := os.WriteFile(targetFile, []byte(converted), 0644); err != nil {
		return fmt.Errorf("failed to write converted file %s: %w", targetFile, err)
	}

	fmt.Printf("✓ Converted %s (%s) -> %s\n", rel, format, targetFile)
	return nil
}
