# CheatMD

[![CI](https://github.com/cheatmd-dev/cheatmd/actions/workflows/ci.yml/badge.svg)](https://github.com/cheatmd-dev/cheatmd/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/cheatmd-dev/cheatmd)](https://goreportcard.com/report/github.com/cheatmd-dev/cheatmd)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Executable Markdown cheatsheets. Write readable docs, run interactive commands.

![demo](assets/demo.gif)

## Install

```bash
go install github.com/cheatmd-dev/cheatmd/cmd/cheatmd@latest
```

## Quick Start

```bash
cheatmd                    # Browse current directory
cheatmd ~/cheats           # Browse specific directory
cheatmd -q "docker"        # Start with search query
cheatmd --history          # Re-run from execution history
cheatmd convert tldr tar.md -o ~/cheats/tar.md # convert a tldr cheat
cheatmd packs list         # List installable cheat packs from the registry
cheatmd packs install      # Pick and install cheat packs (interactive)
```

### First run

The first time you run `cheatmd` (before a config file exists) on an
interactive terminal, it offers to set you up: it writes a config to
`~/.config/cheatmd/cheatmd.yaml` and lets you pick **starter cheat packs** to
install from the [registry](https://github.com/cheatmd-dev/registry) into
`~/.local/share/cheatmd/cheats`. Pick packs from the checklist with `space`, confirm
with `enter`. Point `registry_url` at a private/self-hosted `registry.yaml` to
customize the offered packs. Setup is skipped in headless/piped invocations.

You can install packs any time with the `packs` command — it doesn't depend on
the first-run prompt:

```bash
cheatmd packs list                 # show available packs (and which are installed)
cheatmd packs install              # interactive picker
cheatmd packs install git docker   # install specific packs by name
```

## Features

### Write cheats as Markdown

Any heading with a code block is a cheat. Variables like `$container` become
interactive prompts powered by shell commands:

````markdown
## Docker: exec into container

```sh title:"Execute shell in container"
docker exec -it $container /bin/sh
```
<!-- cheat
var container = docker ps --format "{{.Names}}" --- --header "Select container"
-->
````

### Fuzzy search with frecency

Type to filter across titles, commands, descriptions, and tags. Cheats you
run often and recently float to the top.

### Variables from shell output

Variable values come from shell command output - 0 lines gives a text prompt,
1 line pre-fills, 2+ lines give a filterable picker.

### Reusable modules

Export a variable definition once, import it in any cheat:

```text
<!-- cheat
export docker_container
var container = docker ps --format "{{.Names}}"
-->
```

### Chains

Multi-step workflows that advance one step per launch:

```text
/chain release     <- search chains in the picker
```

### Tags

Cheats are tagged automatically from folder paths, YAML front matter, footer
blocks, inline `#hashtags`, and heading text - all searchable from the picker.

### Shell integration

Embed CheatMD directly into your shell prompt, tmux, or Zellij:

```bash
eval "$(cheatmd widget bash)"   # Ctrl+G opens the selector
```

### Linting

Validate DSL syntax, imports, chains, and undeclared variables:

```bash
cheatmd --lint ~/cheats
```

Language-aware: shell builtins like `$HOME` and PowerShell `$true` won't
trigger false positives.

### Dump

Export cheat metadata as JSON or CSV for indexing and tooling:

```bash
cheatmd dump ~/cheats --json
```

### Convert existing cheatsheets

Bring existing collections into CheatMD:

```bash
cheatmd convert navi ~/navi-cheats -o ~/cheats
cheatmd convert tldr ~/tldr/pages/common/tar.md -o ~/cheats/tar.md
cheatmd convert cheat ~/cheat/cheatsheets -o ~/cheats
```

Supported inputs are `navi`, `tldr`, and `cheat/cheatsheets`.

### Headless Mode

Run CheatMD without a TUI to integrate with other tools (like VS Code or Obsidian) using a JSON-RPC interface over standard I/O:

```bash
cheatmd --headless -q "docker exec"
```

## Editor Extensions

- **[VS Code](../cheatmd.ext/cheatmd-vscode)** - syntax highlighting, lint diagnostics, completion, and CodeLens execution.
- **[Neovim](../cheatmd.ext/cheatmd-neovim)** - syntax highlighting, async diagnostics, completion, and `:CheatMDRun`.
- **[Obsidian](../cheatmd.ext/cheatmd-obsidian)** - inline run buttons, lint status, execution results, and variable autocomplete.

## TUI Keys

| Key | Action |
|-----|--------|
| `Ctrl-H` | Execution history |
| `Ctrl-T` | Substitute search (env + shell history) |
| `Ctrl-Y` | Markdown preview |
| `Ctrl-O` | Open source file in editor |
| `Tab` | Path completion / copy selection |

## Documentation

Full documentation lives in the **[Wiki](../../wiki)**:

- **[Getting Started](../../wiki/Getting-Started)** - first steps with CheatMD
- **[Writing Cheats](../../wiki/Writing-Cheats)** - heading structure, code blocks, metadata
- **[Variables](../../wiki/Variables)** - prompt, shell, and literal forms
- **[Selector Options](../../wiki/Selector-Options)** - `--header`, `--column`, `--map`
- **[Conditionals](../../wiki/Conditionals)** - `if` / `fi` branching
- **[Modules](../../wiki/Modules)** - `export` / `import`
- **[Chains](../../wiki/Chains)** - multi-step workflows
- **[Tags](../../wiki/Tags)** - five tag sources
- **[Configuration](../../wiki/Configuration)** - `cheatmd.yaml` reference
- **[Shell Integration](../../wiki/Shell-Integration)** - widget, tmux, zellij
- **[Linting](../../wiki/Linting)** - syntax and reference validation
- **[Dump](../../wiki/Dump)** - metadata export
- **[Convert](../../wiki/Convert)** - import navi, tldr, and cheat/cheatsheets collections
- **[Headless Mode](../../wiki/Headless-Mode)** - JSON-RPC programmatic interface
- **[Recipes](../../wiki/Recipes)** - copy-pasteable patterns

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[MIT](LICENSE)
