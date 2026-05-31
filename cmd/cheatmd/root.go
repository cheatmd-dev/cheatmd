package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/cheatmd-dev/cheatmd/internal/headless"
	"github.com/cheatmd-dev/cheatmd/internal/ui"
	"github.com/cheatmd-dev/cheatmd/pkg/config"
	"github.com/cheatmd-dev/cheatmd/pkg/executor"
	"github.com/cheatmd-dev/cheatmd/pkg/parser"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:		"cheatmd [path]",
	Short:		"Executable Markdown Cheatsheets",
	SilenceUsage:	true,
	Long: `Command cheatsheet tool that uses real Markdown files.

Browse your cheatsheets interactively, select commands,
fill in variables, and execute or copy the result.`,
	Args:	cobra.MaximumNArgs(1),
	RunE:	runCheats,
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.AddCommand(widgetCmd)
	rootCmd.AddCommand(chainCmd)
	rootCmd.AddCommand(dumpCmd)
	rootCmd.AddCommand(composeCmd)
	rootCmd.AddCommand(convertCmd)
	rootCmd.AddCommand(packsCmd)
	rootCmd.AddCommand(initCmd)
	chainCmd.AddCommand(chainResetCmd)

	rootCmd.PersistentFlags().StringP("query", "q", "", "Initial search query")
	rootCmd.PersistentFlags().Bool("headless", false, "Run in headless interactive JSON-RPC mode")
	rootCmd.PersistentFlags().StringP("match", "m", "", "Match a full command pattern and auto-fill its variables")
	rootCmd.PersistentFlags().BoolP("print", "p", false, "Print command (default)")
	rootCmd.PersistentFlags().BoolP("copy", "c", false, "Copy command")
	rootCmd.PersistentFlags().BoolP("exec", "e", false, "Execute command")
	rootCmd.PersistentFlags().BoolP("auto", "a", false, "Auto-select if query matches exactly one result")
	rootCmd.PersistentFlags().BoolP("benchmark", "b", false, "Benchmark load time and exit")
	rootCmd.PersistentFlags().BoolP("history", "H", false, "Open the execution history picker")
	rootCmd.PersistentFlags().BoolP("lint", "l", false, "Lint cheats and exit")
	rootCmd.PersistentFlags().BoolP("strict", "s", false, "Treat lint warnings as errors")
}

func initConfig() {
	if err := config.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
	}
}

func runCheats(cmd *cobra.Command, args []string) error {
	absPath, err := resolveCheatPath(args)
	if err != nil {
		return err
	}

	configureExecutionMode(cmd)

	if lintFlag, _ := cmd.Flags().GetBool("lint"); lintFlag {
		return runLint(cmd, absPath)
	}

	benchmark, _ := cmd.Flags().GetBool("benchmark")
	start := time.Now()

	index, err := loadAndParseCheats(absPath)
	if err != nil {
		return err
	}

	warnDuplicateExports(cmd, index)

	if len(index.Cheats) == 0 {
		return fmt.Errorf("no cheats found in %s. Run `cheatmd init` to install starter packs", absPath)
	}

	exec := executor.NewExecutor(index)

	if benchmark {
		return runBenchmark(cmd, index, start)
	}

	return executeHeadlessOrTUI(cmd, index, exec)
}

func resolveCheatPath(args []string) (string, error) {
	path := "."
	if len(args) > 0 {
		path = args[0]
	} else if config.Get().Path != "." {
		path = config.Get().Path
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("error resolving path: %w", err)
	}

	if _, err := os.Stat(absPath); err != nil {
		return "", fmt.Errorf("path error: %w", err)
	}
	return absPath, nil
}

func configureExecutionMode(cmd *cobra.Command) {
	if p, _ := cmd.Flags().GetBool("print"); p {
		config.Get().Output = "print"
	} else if c, _ := cmd.Flags().GetBool("copy"); c {
		config.Get().Output = "copy"
	} else if e, _ := cmd.Flags().GetBool("exec"); e {
		config.Get().Output = "exec"
	}

	if auto, _ := cmd.Flags().GetBool("auto"); auto {
		config.Get().AutoSelect = true
	}
}

func loadAndParseCheats(absPath string) (*parser.CheatIndex, error) {
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("path error: %w", err)
	}

	p := parser.NewParser()
	if info.IsDir() {
		return p.ParseDirectory(absPath)
	}
	return p.ParseSingleFile(absPath)
}

func warnDuplicateExports(cmd *cobra.Command, index *parser.CheatIndex) {
	if len(index.Duplicates) > 0 {
		fmt.Fprintln(cmd.ErrOrStderr(), "Warning: duplicate exports found:")
		for _, dup := range index.Duplicates {
			fmt.Fprintf(cmd.ErrOrStderr(), "  export %q defined in:\n    - %s\n    - %s\n", dup.Name, dup.File1, dup.File2)
		}
		fmt.Fprintln(cmd.ErrOrStderr())
	}
}

func runBenchmark(cmd *cobra.Command, index *parser.CheatIndex, start time.Time) error {
	elapsed := time.Since(start)
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Fprintf(cmd.OutOrStdout(), "Loaded %d cheats in %v\n", len(index.Cheats), elapsed)
	fmt.Fprintf(cmd.OutOrStdout(), "Memory: Alloc=%dMB, TotalAlloc=%dMB, Sys=%dMB, HeapObjects=%d\n",
		m.Alloc/1024/1024, m.TotalAlloc/1024/1024, m.Sys/1024/1024, m.HeapObjects)
	return nil
}

func executeHeadlessOrTUI(cmd *cobra.Command, index *parser.CheatIndex, exec *executor.Executor) error {
	query, _ := cmd.Flags().GetString("query")
	match, _ := cmd.Flags().GetString("match")

	if headlessFlag, _ := cmd.Flags().GetBool("headless"); headlessFlag {
		return headless.Run(index, exec, query, match)
	}

	var finalCmd string
	var err error
	if historyFlag, _ := cmd.Flags().GetBool("history"); historyFlag {
		finalCmd, err = ui.RunHistory(index, exec)
	} else {
		finalCmd, err = ui.Run(index, exec, query, match)
	}

	if err != nil {
		return err
	}
	if finalCmd == "" {
		return nil
	}

	if preHook := config.Get().PreHook; preHook != "" {
		finalCmd = preHook + finalCmd
	}
	if postHook := config.Get().PostHook; postHook != "" {
		finalCmd = finalCmd + postHook
	}

	switch config.Get().Output {
	case "exec":
		fmt.Fprint(cmd.ErrOrStderr(), finalCmd)
		return exec.OutputWithMode(finalCmd, executor.OutputExec)
	case "copy":
		return exec.OutputWithMode(finalCmd, executor.OutputCopy)
	default:	// print
		fmt.Fprint(cmd.OutOrStdout(), finalCmd)
		return nil
	}
}
