package main

import (
"strings"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/cheatmd-dev/cheatmd/internal/packmanifest"
	"github.com/cheatmd-dev/cheatmd/internal/ui"
	"github.com/cheatmd-dev/cheatmd/pkg/config"
	"github.com/cheatmd-dev/cheatmd/pkg/registry"
)

var packsCmd = &cobra.Command{
	Use:	"packs",
	Short:	"Browse and install cheat packs from the registry",
	Long: `Browse and install community cheat packs from the registry.

The registry is a YAML manifest of installable cheat repositories, configured
via the registry_url setting. Installed packs land in the cheats directory
(see DefaultCheatsDir / the "path" setting) under <cheats-dir>/<pack-name>/.`,
}

var packsListCmd = &cobra.Command{
	Use:	"list",
	Short:	"List available cheat packs",
	Args:	cobra.NoArgs,
	RunE:	runPacksList,
}

var packsInstallCmd = &cobra.Command{
	Use:	"install [name...]",
	Short:	"Install cheat packs (interactive picker when no names are given)",
	RunE:	runPacksInstall,
}

var packsUpdateCmd = &cobra.Command{
	Use:	"update [name...]",
	Short:	"Update installed cheat packs",
	RunE:	runPacksUpdate,
}

var packsRemoveCmd = &cobra.Command{
	Use:	"remove [name...]",
	Short:	"Remove installed cheat packs",
	Args:	cobra.MinimumNArgs(1),
	RunE:	runPacksRemove,
}

func init() {
	packsCmd.AddCommand(packsListCmd)
	packsCmd.AddCommand(packsInstallCmd)
	packsCmd.AddCommand(packsUpdateCmd)
	packsCmd.AddCommand(packsRemoveCmd)
}

func runPacksList(cmd *cobra.Command, args []string) error {
	reg, err := registry.Fetch(cmd.Context(), config.Get().RegistryURL)
	if err != nil {
		return fmt.Errorf("fetch registry: %w", err)
	}

	// A corrupt/unreadable manifest shouldn't break listing; treat as empty.
	manifest, err := packmanifest.Load(config.CheatsInstallDir())
	if err != nil {
		manifest = &packmanifest.Manifest{}
	}

	out := cmd.OutOrStdout()
	w := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
	for _, p := range reg.Packs {
		tag := "community"
		if p.Official {
			tag = "official"
		}

		status := " "
		date := ""
		if entry := manifest.Get(p.Name); entry != nil {
			status = "✓"
			date = entry.InstalledAt.Format("2006-01-02")
		}

		fmt.Fprintf(w, "%s %s\t%s\t%s\t%s\n", status, p.Name, tag, date, p.Repo)
	}
	w.Flush()
	return nil
}

func runPacksModify(cmd *cobra.Command, args []string, verb string) error {
	chosen, err := fetchAndChoosePacks(cmd, args)
	if err != nil {
		return err
	}
	if len(chosen) == 0 {
		return nil
	}

	out := cmd.ErrOrStderr()
	dest, installed := installPacks(cmd.Context(), out, chosen)

	// Capitalize verb for output
	capitalizedVerb := strings.ToUpper(verb[:1]) + verb[1:]

	fmt.Fprintf(out, "%s %d pack(s) in %s\n", capitalizedVerb, installed, dest)
	if installed < len(chosen) {
		// "install" -> "install", "update" -> "update"
		action := verb
		if verb == "updated" {
			action = "update"
		} else if verb == "installed" {
			action = "install"
		}
		return fmt.Errorf("%d of %d pack(s) failed to %s", len(chosen)-installed, len(chosen), action)
	}
	return nil
}

func runPacksInstall(cmd *cobra.Command, args []string) error {
	return runPacksModify(cmd, args, "installed")
}

func runPacksUpdate(cmd *cobra.Command, args []string) error {
	return runPacksModify(cmd, args, "updated")
}

func runPacksRemove(cmd *cobra.Command, args []string) error {
	dest := config.CheatsInstallDir()
	manifest, err := packmanifest.Load(dest)
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	removed := 0
	for _, name := range args {
		if !manifest.Has(name) {
			fmt.Fprintf(out, "Pack %q is not installed.\n", name)
			continue
		}

		target := filepath.Join(dest, name)
		if err := os.RemoveAll(target); err != nil {
			fmt.Fprintf(out, "Failed to remove %q: %v\n", name, err)
			continue
		}

		manifest.Remove(name)
		removed++
		fmt.Fprintf(out, "Removed pack %q\n", name)
	}

	if removed > 0 {
		if err := packmanifest.Save(dest, manifest); err != nil {
			return fmt.Errorf("save manifest: %w", err)
		}
	}
	return nil
}

// fetchAndChoosePacks handles the common setup of fetching the registry
// and showing the picker (or parsing args) for both install and update.
func fetchAndChoosePacks(cmd *cobra.Command, args []string) ([]registry.Pack, error) {
	reg, err := registry.Fetch(cmd.Context(), config.Get().RegistryURL)
	if err != nil {
		return nil, fmt.Errorf("fetch registry: %w", err)
	}

	chosen, err := choosePacks(reg, args)
	if err != nil {
		return nil, err
	}

	if len(chosen) == 0 {
		fmt.Fprintln(cmd.ErrOrStderr(), "No packs selected.")
	}
	return chosen, nil
}

// choosePacks resolves which packs to install: the named packs when names are
// given, otherwise the user's selection from the interactive picker.
func choosePacks(reg *registry.Registry, names []string) ([]registry.Pack, error) {
	if len(names) > 0 {
		chosen, err := reg.Select(names)
		if err != nil {
			return nil, fmt.Errorf("%w (try `cheatmd packs list`)", err)
		}
		return chosen, nil
	}

	if !isInteractive() {
		return nil, fmt.Errorf("no pack names given and not an interactive terminal; pass names, e.g. `cheatmd packs install git docker`")
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
		return nil, fmt.Errorf("pack selection: %w", err)
	}
	return chosen, nil
}
