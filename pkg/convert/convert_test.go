package convert

import (
	"strings"
	"testing"
)

func TestConvertNavi(t *testing.T) {
	input := `% git, checkout
# Switch to a branch
@ git_utils
git checkout <branch>

$ branch: git branch --format="%(refname:short)" --- --header "Select branch"
`

	converted, err := ConvertNavi(input, "git.cheat")
	if err != nil {
		t.Fatalf("ConvertNavi failed: %v", err)
	}

	if !strings.Contains(converted, "## Switch to a branch") {
		t.Errorf("Expected heading '## Switch to a branch', got: %s", converted)
	}

	if !strings.Contains(converted, "#git #checkout") {
		t.Errorf("Expected localized hashtags, got: %s", converted)
	}

	if !strings.Contains(converted, "git checkout $branch") {
		t.Errorf("Expected placeholder replacement in command, got: %s", converted)
	}

	if !strings.Contains(converted, "import git") {
		t.Errorf("Expected import git in cheat block, got: %s", converted)
	}

	if !strings.Contains(converted, "import git_utils") {
		t.Errorf("Expected import git_utils in cheat block, got: %s", converted)
	}

	if !strings.Contains(converted, "export git") {
		t.Errorf("Expected export git in global block, got: %s", converted)
	}

	if !strings.Contains(converted, "var branch = git branch --format=\"%(refname:short)\" --- --header \"Select branch\"") {
		t.Errorf("Expected var definition in global block, got: %s", converted)
	}
}

func TestConvertNaviSelectorAdaptation(t *testing.T) {
	input := `% test, selector
# Retrieve items with various options
git log <item>

$ item: git log --oneline --- --headers 2 --column 3 --header "Select item"
`

	converted, err := ConvertNavi(input, "test.cheat")
	if err != nil {
		t.Fatalf("ConvertNavi failed: %v", err)
	}

	expectedVar := `var item = git log --oneline | tail -n +3 --- --select-column 3 --header "Select item" --delimiter "\t"`
	if !strings.Contains(converted, expectedVar) {
		t.Errorf("Expected var definition:\n%s\ngot:\n%s", expectedVar, converted)
	}
}

func TestConvertTldr(t *testing.T) {
	input := `# tar

> Archiving utility.
> More information: <https://www.gnu.org/software/tar/>.

- Create a gzipped archive:
` + "```" + `sh
tar -czvf {{path/to/archive.tar.gz}} {{path/to/directory}}
` + "```" + `
`

	converted, err := ConvertTldr(input, "tar.md")
	if err != nil {
		t.Fatalf("ConvertTldr failed: %v", err)
	}

	if !strings.Contains(converted, "## Create a gzipped archive") {
		t.Errorf("Expected heading '## Create a gzipped archive', got: %s", converted)
	}

	if !strings.Contains(converted, "tar -czvf $path_to_archive_tar_gz $path_to_directory") {
		t.Errorf("Expected command placeholders to be replaced, got: %s", converted)
	}

	if !strings.Contains(converted, "var path_to_archive_tar_gz = --- --header \"path/to/archive.tar.gz\"") {
		t.Errorf("Expected var definition for first placeholder, got: %s", converted)
	}

	if !strings.Contains(converted, "var path_to_directory = --- --header \"path/to/directory\"") {
		t.Errorf("Expected var definition for second placeholder, got: %s", converted)
	}
}

func TestConvertCheat(t *testing.T) {
	input := `---
syntax: sh
tags: [ tar, archive ]
---
# To create a gzipped archive:
tar -czvf <archive.tar.gz> {{path/to/directory}}
`

	converted, err := ConvertCheat(input, "tar")
	if err != nil {
		t.Fatalf("ConvertCheat failed: %v", err)
	}

	if !strings.Contains(converted, "## To create a gzipped archive") {
		t.Errorf("Expected heading, got: %s", converted)
	}

	if !strings.Contains(converted, "#tar #archive") {
		t.Errorf("Expected localized hashtags, got: %s", converted)
	}

	if strings.Contains(converted, "Tags: #tar #archive") {
		t.Errorf("Did not expect redundant repeating tags line in body, got: %s", converted)
	}

	if !strings.Contains(converted, "tar -czvf $archive_tar_gz $path_to_directory") {
		t.Errorf("Expected placeholder replacement, got: %s", converted)
	}

	if !strings.Contains(converted, "var archive_tar_gz = --- --header \"archive.tar.gz\"") {
		t.Errorf("Expected var definition for navi placeholder, got: %s", converted)
	}

	if !strings.Contains(converted, "var path_to_directory = --- --header \"path/to/directory\"") {
		t.Errorf("Expected var definition for tldr placeholder, got: %s", converted)
	}
}
