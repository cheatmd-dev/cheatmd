package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/cheatmd-dev/cheatmd/internal/ui"
	"github.com/cheatmd-dev/cheatmd/pkg/config"
	"github.com/cheatmd-dev/cheatmd/pkg/installer"
	"github.com/cheatmd-dev/cheatmd/pkg/packmanifest"
	"github.com/cheatmd-dev/cheatmd/pkg/registry"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize CheatMD config and install starter cheats",
	RunE: func(cmd *cobra.Command, args []string) error {
		out := cmd.OutOrStdout()

		// 1. Create config
		path := config.DefaultConfigPath()
		if err := config.WriteDefaultConfig(path); err != nil {
			if !strings.Contains(err.Error(), "already exists") {
				return fmt.Errorf("could not write config: %w", err)
			}
			fmt.Fprintf(out, "Config already exists at %s\n", path)
		} else {
			fmt.Fprintf(out, "Wrote config to %s\n", path)
		}

		// 2. Ask to install starter cheats
		if !isInteractive() {
			fmt.Fprintln(out, "Non-interactive environment detected. Skipping starter cheat prompt.")
			return nil
		}

		if !promptYesNo("Install cheat packs from the registry?") {
			return nil
		}

		dest := installStarterPacks(cmd, out)
		if dest != "" {
			fmt.Fprintln(out, "\nCheatMD is ready! Run `cheatmd` to explore your new cheats.")
		}
		return nil
	},
}

// installStarterPacks runs the registry picker and installs the chosen packs,
// returning the install directory (empty when nothing was installed). All
// failures are non-fatal: they print a note and leave the user with a config
// but no cheats.
func installStarterPacks(cmd *cobra.Command, out io.Writer) string {
	reg, err := registry.Fetch(cmd.Context(), config.GetRegistryURL())
	if err != nil {
		fmt.Fprintf(out, "Could not fetch the cheat registry: %v\n", err)
		fmt.Fprintln(out, "Skipping official cheat packs — you can add packs later with `cheatmd packs install`.")
		return ""
	}

	manifest, _ := packmanifest.Load(config.CheatsInstallDir())
	installedPacks := make(map[string]bool)
	if manifest != nil {
		for _, p := range manifest.Packs {
			installedPacks[p.Name] = true
		}
	}

	chosen, err := ui.RunPackPicker(reg.Packs, installedPacks)
	if err != nil {
		fmt.Fprintf(out, "Pack selection failed: %v\n", err)
		return ""
	}
	if len(chosen) == 0 {
		fmt.Fprintln(out, "No packs selected.")
		return ""
	}

	dest, installed := installPacks(cmd.Context(), out, chosen)
	if installed == 0 {
		return ""
	}
	fmt.Fprintf(out, "Installed %d pack(s) into %s\n", installed, dest)
	return dest
}

// installPacks installs each pack into the cheats directory, printing progress
// to out, and records successful installs in the pack manifest. It returns the
// destination directory and the number successfully installed. Shared by
// first-run setup and the `packs install` command.
func installPacks(ctx context.Context, out io.Writer, packs []registry.Pack) (dest string, installed int) {
	dest = config.CheatsInstallDir()

	manifest, err := packmanifest.Load(dest)
	if err != nil {
		// A corrupt manifest shouldn't block installs; start fresh but warn.
		fmt.Fprintf(out, "Warning: could not read pack manifest: %v\n", err)
		manifest = &packmanifest.Manifest{}
	}

	for _, pack := range packs {
		fmt.Fprintf(out, "Installing %s...\n", pack.Name)
		if err := installer.Install(ctx, pack, dest); err != nil {
			fmt.Fprintf(out, "  failed: %v\n", err)
			continue
		}
		manifest.Upsert(packmanifest.Entry{
			Name:        pack.Name,
			Repo:        pack.Repo,
			InstalledAt: time.Now(),
		})
		installed++
	}

	if installed > 0 {
		if err := packmanifest.Save(dest, manifest); err != nil {
			fmt.Fprintf(out, "Warning: could not write pack manifest: %v\n", err)
		}
	}
	return dest, installed
}

// isInteractive reports whether both stdin and stdout are terminals.
func isInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

// promptYesNo asks a yes/no question on stderr and reads the answer from stdin.
// Defaults to yes on an empty response.
func promptYesNo(question string) bool {
	fmt.Fprintf(os.Stderr, "%s [Y/n] ", question)
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(scanner.Text())) {
	case "", "y", "yes":
		return true
	default:
		return false
	}
}
