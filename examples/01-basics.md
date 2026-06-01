# 01 · Basics

The minimum viable cheat is a heading plus a fenced code block. No metadata
required, so they only appear when require_cheat_block: false.

## Hello world

```sh title:"Print a greeting"
echo "hello from cheatmd"
```

## Heading levels are all equal

CheatMD treats `#`, `##`, `###` … the same for cheat detection.

### Third-level heading still works

```sh title:"List the current directory"
ls -la
```

## The title attribute is the description

The `title:"…"` after the fence language shows as the picker description.

```sh title:"Print the current date"
date
```

## A cheat with no title is fine too

```sh
echo "no title attribute here"
```

## Fence language is a linter hint only

The language (` ```powershell `) drives syntax-aware linting; execution always
uses your configured shell. (This one just echoes, so it is safe to run.)

```powershell title:"Fence lang affects linting, not execution"
echo "pretend this is PowerShell"
```
