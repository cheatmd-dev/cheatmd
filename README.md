# CheatMD

[![CI](https://github.com/cheatmd-dev/cheatmd/actions/workflows/ci.yml/badge.svg)](https://github.com/cheatmd-dev/cheatmd/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/cheatmd-dev/cheatmd)](https://goreportcard.com/report/github.com/cheatmd-dev/cheatmd)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Executable Markdown cheatsheets. Write readable docs, run interactive commands.

![demo](assets/demo.gif)

## Install

```bash
go install github.com/cheatmd-dev/cheatmd/cmd/cheatmd@v1.0.0-rc.2
```

Requires Go 1.26+.

## Quick Start

```bash
cheatmd                    # Browse current directory
cheatmd ~/cheats           # Browse a cheats directory
cheatmd -q "docker"        # Start with a search query
cheatmd init               # Initialize default config
cheatmd --lint ~/cheats    # Check cheats for syntax/reference issues
cheatmd packs install      # Install starter cheat packs
```

Any Markdown heading with a code block can become a cheat:

````markdown
## Docker: exec into container

```sh title:"Execute shell in container"
docker exec -it $container /bin/sh
```
<!-- cheat
var container = docker ps --format "{{.Names}}" --- --header "Select container"
-->
````

## What It Does

- Fuzzy-search cheats by title, command, tags, and description
- Prompt for variables using shell output, editable defaults, or pickers
- Reuse variable definitions with modules
- Run ordered multi-step workflows with chains
- Lint cheat syntax, imports, chains, and variables
- Export parsed metadata as JSON or CSV
- Convert existing `navi`, `tldr`, and `cheat/cheatsheets` collections
- Run headless over JSON-RPC for editor integrations

## Editor Extensions

- [VS Code](https://github.com/cheatmd-dev/cheatmd-vscode) - syntax highlighting, diagnostics, completion, and CodeLens execution
- [Neovim](https://github.com/cheatmd-dev/cheatmd-neovim) - syntax highlighting, async diagnostics, completion, and `:CheatMDRun`
- [Obsidian](https://github.com/cheatmd-dev/cheatmd-obsidian) - inline run buttons, lint status, execution results, and variable autocomplete

## Documentation

Full documentation lives in the [Wiki](https://github.com/cheatmd-dev/cheatmd/wiki):

- [Getting Started](https://github.com/cheatmd-dev/cheatmd/wiki/Getting-Started)
- [Writing Cheats](https://github.com/cheatmd-dev/cheatmd/wiki/Writing-Cheats)
- [Variables](https://github.com/cheatmd-dev/cheatmd/wiki/Variables)
- [Selector Options](https://github.com/cheatmd-dev/cheatmd/wiki/Selector-Options)
- [Configuration](https://github.com/cheatmd-dev/cheatmd/wiki/Configuration)
- [Convert](https://github.com/cheatmd-dev/cheatmd/wiki/Convert)
- [Headless Mode](https://github.com/cheatmd-dev/cheatmd/wiki/Headless-Mode)
- [Recipes](https://github.com/cheatmd-dev/cheatmd/wiki/Recipes)

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## Support

If CheatMD saves you time, you can support the project on [Ko-fi](https://ko-fi.com/gubarz).

## License

[MIT](LICENSE)
