package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// defaultConfigTemplate is the starter config written by WriteDefaultConfig
// during first-run setup. Keeping it embedded keeps generated configs in sync
// with the documented example.
//
//go:embed default_config.yaml
var defaultConfigTemplate []byte

// ============================================================================
// Configuration Types
// ============================================================================

// Config holds the application configuration
type Config struct {
	Path                string `mapstructure:"path"`
	RegistryURL         string `mapstructure:"registry_url"`
	Output              string `mapstructure:"output"`
	Shell               string `mapstructure:"shell"`
	Editor              string `mapstructure:"editor"`
	PreHook             string `mapstructure:"pre_hook"`
	PostHook            string `mapstructure:"post_hook"`
	RequireCheatBlock   bool   `mapstructure:"require_cheat_block"`
	AllowUndeclaredVars bool   `mapstructure:"allow_undeclared_vars"`
	VarSyntax           string `mapstructure:"var_syntax"`
	AutoSelect          bool   `mapstructure:"auto_select"`
	AutoContinue        bool   `mapstructure:"auto_continue"`

	// Keybindings
	KeyWidget     string `mapstructure:"key_widget"`
	KeyOpen       string `mapstructure:"key_open"`
	KeySubstitute string `mapstructure:"key_substitute"`
	KeyPreview    string `mapstructure:"key_preview"`
	KeyHistory    string `mapstructure:"key_history"`

	// Execution history
	HistoryFile string `mapstructure:"history_file"`
	HistoryMax  int    `mapstructure:"history_max"`

	// Substitute search
	SubstituteSources []string `mapstructure:"substitute_sources"`

	// Display options
	ShowFolder    bool `mapstructure:"show_folder"`
	ShowFile      bool `mapstructure:"show_file"`
	PreviewHeight int  `mapstructure:"preview_height"`

	// Colors
	Colors ColorConfig

	// Columns
	Columns ColumnConfig
}

// ColorConfig holds all color settings
type ColorConfig struct {
	Header   string `mapstructure:"color_header"`
	Command  string `mapstructure:"color_command"`
	Desc     string `mapstructure:"color_desc"`
	Path     string `mapstructure:"color_path"`
	Border   string `mapstructure:"color_border"`
	Cursor   string `mapstructure:"color_cursor"`
	Selected string `mapstructure:"color_selected"`
	Dim      string `mapstructure:"color_dim"`
}

// ColumnConfig holds column width settings
type ColumnConfig struct {
	Gap     int `mapstructure:"column_gap"`
	Header  int `mapstructure:"column_header"`
	Desc    int `mapstructure:"column_desc"`
	Command int `mapstructure:"column_command"`
}

// ============================================================================
// Default Values
// ============================================================================

// Defaults for configuration
// DefaultRegistryURL is the canonical cheat-pack registry manifest. Override
// via the registry_url config key for private/self-hosted registries.
const DefaultRegistryURL = "https://raw.githubusercontent.com/cheatmd-dev/registry/main/registry.yaml"

var defaults = struct {
	path                string
	registryURL         string
	output              string
	shell               string
	editor              string
	preHook             string
	postHook            string
	requireCheatBlock   bool
	allowUndeclaredVars bool
	varSyntax           string
	autoSelect          bool
	autoContinue        bool
	keyWidget           string
	keyOpen             string
	keySubstitute       string
	keyPreview          string
	keyHistory          string
	historyFile         string
	historyMax          int
	substituteSources   []string
	showFolder          bool
	showFile            bool
	previewHeight       int
	colors              ColorConfig
	columns             ColumnConfig
}{
	path:                ".",
	registryURL:         DefaultRegistryURL,
	output:              "print",
	shell:               "", // Set dynamically
	editor:              "", // Empty means use system default (xdg-open/open/start)
	preHook:             "",
	postHook:            "",
	requireCheatBlock:   false,
	allowUndeclaredVars: false,
	varSyntax:           "dollar",
	autoSelect:          false,
	autoContinue:        false,
	keyWidget:           "\\C-g",  // Ctrl+G for shell widgets
	keyOpen:             "ctrl+o", // Ctrl+O in TUI
	keySubstitute:       "ctrl+t", // Ctrl+T opens substitute search during var resolution
	keyPreview:          "ctrl+y", // Ctrl+Y opens markdown preview of current cheat's file
	keyHistory:          "ctrl+h", // Ctrl+H opens execution history
	historyFile:         "",       // Empty -> $XDG_DATA_HOME/cheatmd/history.jsonl
	historyMax:          1000,
	substituteSources:   []string{"env", "history"},
	showFolder:          true,
	showFile:            true,
	previewHeight:       6,
	colors: ColorConfig{
		Header:   "36",  // Cyan
		Command:  "32",  // Green
		Desc:     "246", // Light gray
		Path:     "33",  // Yellow
		Border:   "242", // Gray
		Cursor:   "212", // Pink
		Selected: "236", // Dark bg
		Dim:      "245", // Medium gray
	},
	columns: ColumnConfig{
		Gap:     4,
		Header:  40,
		Desc:    40,
		Command: 60,
	},
}

// ============================================================================
// Global Config
// ============================================================================

// cfg is the global config instance
var cfg Config

// Init initializes configuration with viper
func Init() error {
	setDefaults()
	configureViper()

	var configErr error
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			configErr = fmt.Errorf("config file error: %w", err)
		}
	}

	if err := viper.Unmarshal(&cfg); err != nil {
		return err
	}

	return configErr
}

// setDefaults sets all default values in viper
func setDefaults() {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}

	viper.SetDefault("path", defaults.path)
	viper.SetDefault("registry_url", defaults.registryURL)
	viper.SetDefault("output", defaults.output)
	viper.SetDefault("shell", shell)
	viper.SetDefault("editor", defaults.editor)
	viper.SetDefault("pre_hook", defaults.preHook)
	viper.SetDefault("post_hook", defaults.postHook)
	viper.SetDefault("require_cheat_block", defaults.requireCheatBlock)
	viper.SetDefault("allow_undeclared_vars", defaults.allowUndeclaredVars)
	viper.SetDefault("var_syntax", defaults.varSyntax)
	viper.SetDefault("auto_select", defaults.autoSelect)
	viper.SetDefault("auto_continue", defaults.autoContinue)

	// Keybindings
	viper.SetDefault("key_widget", defaults.keyWidget)
	viper.SetDefault("key_open", defaults.keyOpen)
	viper.SetDefault("key_substitute", defaults.keySubstitute)
	viper.SetDefault("key_preview", defaults.keyPreview)
	viper.SetDefault("key_history", defaults.keyHistory)

	// Execution history
	viper.SetDefault("history_file", defaults.historyFile)
	viper.SetDefault("history_max", defaults.historyMax)

	// Substitute search
	viper.SetDefault("substitute_sources", defaults.substituteSources)

	// Display options
	viper.SetDefault("show_folder", defaults.showFolder)
	viper.SetDefault("show_file", defaults.showFile)
	viper.SetDefault("preview_height", defaults.previewHeight)

	// Colors
	viper.SetDefault("color_header", defaults.colors.Header)
	viper.SetDefault("color_command", defaults.colors.Command)
	viper.SetDefault("color_desc", defaults.colors.Desc)
	viper.SetDefault("color_path", defaults.colors.Path)
	viper.SetDefault("color_border", defaults.colors.Border)
	viper.SetDefault("color_cursor", defaults.colors.Cursor)
	viper.SetDefault("color_selected", defaults.colors.Selected)
	viper.SetDefault("color_dim", defaults.colors.Dim)

	// Columns
	viper.SetDefault("column_gap", defaults.columns.Gap)
	viper.SetDefault("column_header", defaults.columns.Header)
	viper.SetDefault("column_desc", defaults.columns.Desc)
	viper.SetDefault("column_command", defaults.columns.Command)
}

// configureViper sets up viper configuration sources
func configureViper() {
	viper.SetConfigName("cheatmd")
	viper.SetConfigType("yaml")

	if home, err := os.UserHomeDir(); err == nil {
		viper.AddConfigPath(filepath.Join(home, ".config", "cheatmd"))
		viper.AddConfigPath(home)
	}
	viper.AddConfigPath(".")

	viper.SetEnvPrefix("CHEATMD")
	viper.AutomaticEnv()
}

// ============================================================================
// Getters - Core Settings
// ============================================================================

// GetPath returns the cheat path with tilde expansion
func GetPath() string {
	return expandTilde(viper.GetString("path"))
}

// GetRegistryURL returns the cheat-pack registry manifest URL.
func GetRegistryURL() string {
	return viper.GetString("registry_url")
}

// GetOutput returns the output mode
func GetOutput() string {
	return viper.GetString("output")
}

// GetShell returns the configured shell
func GetShell() string {
	return viper.GetString("shell")
}

// GetPreHook returns the pre-execution hook
func GetPreHook() string {
	return viper.GetString("pre_hook")
}

// GetPostHook returns the post-execution hook
func GetPostHook() string {
	return viper.GetString("post_hook")
}

// GetEditor returns the configured editor command (empty = system default)
func GetEditor() string {
	return viper.GetString("editor")
}

// GetAllowUndeclaredVars returns whether variables referenced in a cheat's
// command but not declared in any <!-- cheat --> block should be prompted at
// runtime. When false (default), undeclared variables are silently skipped.
//
// Reads from the cached struct populated at Init() because this getter is
// called in hot paths (per-variable, per-render).
func GetAllowUndeclaredVars() bool {
	return cfg.AllowUndeclaredVars
}

// GetVarSyntax returns the configured variable syntax mode.
// Valid values: "dollar" (default), "angle", "both".
//
// Reads from the cached struct populated at Init() because this getter is
// called in hot paths (per-variable, per-render).
func GetVarSyntax() string {
	v := cfg.VarSyntax
	switch v {
	case "dollar", "angle", "both":
		return v
	default:
		return "dollar"
	}
}

// VarSyntaxAllowsDollar reports whether $name is recognized as a variable.
func VarSyntaxAllowsDollar() bool {
	s := GetVarSyntax()
	return s == "dollar" || s == "both"
}

// VarSyntaxAllowsAngle reports whether <name> is recognized as a variable.
func VarSyntaxAllowsAngle() bool {
	s := GetVarSyntax()
	return s == "angle" || s == "both"
}

// GetRequireCheatBlock returns whether to require cheat blocks
func GetRequireCheatBlock() bool {
	return viper.GetBool("require_cheat_block")
}

// GetAutoSelect returns whether to auto-select single matches
func GetAutoSelect() bool {
	return viper.GetBool("auto_select")
}

// GetAutoContinue returns whether to auto-continue when vars are prefilled from environment
func GetAutoContinue() bool {
	return viper.GetBool("auto_continue")
}

// ============================================================================
// Getters - Keybindings
// ============================================================================

// GetKeyWidget returns the keybinding for shell widget activation (e.g., "\C-g" for Ctrl+G)
func GetKeyWidget() string {
	return viper.GetString("key_widget")
}

// GetKeyOpen returns the keybinding for opening markdown in editor (e.g., "ctrl+o")
func GetKeyOpen() string {
	return viper.GetString("key_open")
}

// GetKeySubstitute returns the keybinding for opening the substitute search
// during variable resolution (e.g., "ctrl+t").
func GetKeySubstitute() string {
	return viper.GetString("key_substitute")
}

// GetKeyPreview returns the keybinding for opening the markdown preview of
// the current cheat's source file (e.g., "ctrl+y").
func GetKeyPreview() string {
	return viper.GetString("key_preview")
}

// GetKeyHistory returns the keybinding for opening the execution history
// overlay (e.g., "ctrl+h").
func GetKeyHistory() string {
	return viper.GetString("key_history")
}

// GetHistoryFile returns the override path for the history file, or "" for
// the default ($XDG_DATA_HOME/cheatmd/history.jsonl).
func GetHistoryFile() string {
	return viper.GetString("history_file")
}

// GetHistoryMax returns the cap on history entries shown in the picker.
// Zero or negative means unlimited.
func GetHistoryMax() int {
	return viper.GetInt("history_max")
}

// GetSubstituteSources returns the enabled sources for substitute search.
// Valid entries: "env", "history". Empty disables the feature.
func GetSubstituteSources() []string {
	return viper.GetStringSlice("substitute_sources")
}

// ============================================================================
// Getters - Display Options
// ============================================================================

// GetShowFolder returns whether to show folder in title/list
func GetShowFolder() bool {
	return viper.GetBool("show_folder")
}

// GetShowFile returns whether to show file in title/list
func GetShowFile() bool {
	return viper.GetBool("show_file")
}

// GetPreviewHeight returns the preview section height in lines
func GetPreviewHeight() int {
	return viper.GetInt("preview_height")
}

// ============================================================================
// Getters - Colors
// ============================================================================

// GetColorHeader returns the header color code
func GetColorHeader() string {
	return viper.GetString("color_header")
}

// GetColorCommand returns the command color code
func GetColorCommand() string {
	return viper.GetString("color_command")
}

// GetColorDesc returns the description color code
func GetColorDesc() string {
	return viper.GetString("color_desc")
}

// GetColorPath returns the path color code
func GetColorPath() string {
	return viper.GetString("color_path")
}

// GetColorBorder returns the border color code
func GetColorBorder() string {
	return viper.GetString("color_border")
}

// GetColorCursor returns the cursor color code
func GetColorCursor() string {
	return viper.GetString("color_cursor")
}

// GetColorSelected returns the selected background color code
func GetColorSelected() string {
	return viper.GetString("color_selected")
}

// GetColorDim returns the dimmed text color code
func GetColorDim() string {
	return viper.GetString("color_dim")
}

// ============================================================================
// Getters - Columns
// ============================================================================

// GetColumnGap returns column spacing
func GetColumnGap() int {
	return viper.GetInt("column_gap")
}

// GetColumnHeader returns header column width
func GetColumnHeader() int {
	return viper.GetInt("column_header")
}

// GetColumnDesc returns description column width
func GetColumnDesc() int {
	return viper.GetInt("column_desc")
}

// GetColumnCommand returns command column width
func GetColumnCommand() int {
	return viper.GetInt("column_command")
}

// ============================================================================
// Setters
// ============================================================================

// SetOutput sets the output mode at runtime
func SetOutput(mode string) {
	viper.Set("output", mode)
	cfg.Output = mode
}

// SetAutoSelect sets auto-select mode at runtime
func SetAutoSelect(enabled bool) {
	viper.Set("auto_select", enabled)
	cfg.AutoSelect = enabled
}

// ============================================================================
// First-run setup
// ============================================================================

// DefaultConfigPath returns the path where WriteDefaultConfig writes the
// starter config: ~/.config/cheatmd/cheatmd.yaml.
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "cheatmd.yaml"
	}
	return filepath.Join(home, ".config", "cheatmd", "cheatmd.yaml")
}

// CheatsInstallDir returns the directory cheat packs should be installed into,
// and where installed-pack detection looks. It is the configured cheats path
// when the user has set one to a real directory; otherwise it falls back to
// DefaultCheatsDir(). This keeps "where packs land" aligned with "where cheats
// are browsed" (GetPath) instead of always using the XDG default.
//
// The default path is "." (browse the current directory); we treat that as
// "unset" for install purposes so packs never land in an arbitrary cwd.
func CheatsInstallDir() string {
	if p := GetPath(); p != "" && p != "." {
		return p
	}
	return DefaultCheatsDir()
}

// DefaultCheatsDir returns the directory where first-run setup installs
// starter cheat packs. Cheats are user data, so they live under
// $XDG_DATA_HOME/cheatmd/cheats (falling back to
// ~/.local/share/cheatmd/cheats), matching where history is stored.
func DefaultCheatsDir() string {
	if xdg := strings.TrimSpace(os.Getenv("XDG_DATA_HOME")); xdg != "" {
		return filepath.Join(xdg, "cheatmd", "cheats")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "cheats"
	}
	return filepath.Join(home, ".local", "share", "cheatmd", "cheats")
}

// WriteDefaultConfig writes the embedded starter config to path, creating
// parent directories as needed. It refuses to overwrite an existing file.
//
// The template's "path:" line is rewritten to the resolved DefaultCheatsDir()
// so the config always points at the same directory first-run setup installs
// into, even when $XDG_DATA_HOME relocates it.
func WriteDefaultConfig(path string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("config already exists at %s", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	content := withCheatsPath(string(defaultConfigTemplate), DefaultCheatsDir())
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// withCheatsPath replaces the first top-level "path:" entry in the template
// with the given cheats directory, leaving comments and other lines intact. If
// no such line exists (e.g. a hand-edited template), the template is returned
// unchanged.
func withCheatsPath(template, cheatsDir string) string {
	lines := strings.Split(template, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "path:") {
			lines[i] = "path: " + cheatsDir
			break
		}
	}
	return strings.Join(lines, "\n")
}

// ============================================================================
// Helpers
// ============================================================================

// expandTilde expands ~ to the user's home directory
func expandTilde(path string) string {
	if len(path) == 0 || path[0] != '~' {
		return path
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, path[1:])
	}
	return path
}
