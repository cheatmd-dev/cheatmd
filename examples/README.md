# CheatMD Examples — Feature Coverage Corpus

These files exercise **every** CheatMD feature, one area per file. They serve
two purposes:

1. **Living documentation** — a runnable tour of the syntax.
2. **Manual-test fixtures** — point the binary here to smoke-test a release.

Every command is side-effect free (only `echo`/`printf`/`true`/`ls`/`date`),
so it is safe to run any cheat with `--exec`.

- `cheatmd examples` — browse the whole corpus
- `cheatmd --lint examples` — must be clean
- `cheatmd dump examples` — list every cheat

(This README has no fenced code blocks on purpose, so it is not itself parsed
as a cheat.)

| File | Covers |
|------|--------|
| `01-basics.md` | minimal cheats, `title:`, heading levels, fence languages |
| `02-variables.md` | all three `var` forms, env pre-fill, resolution order, line continuation |
| `03-selectors.md` | `--header` `--delimiter` `--column` `--select-column` `--map` `--multi`, combined |
| `04-conditionals.md` | `==`, `!=`, all-conditional vars, **nested** `if`/`fi` |
| `05-modules.md` | `export` / `import` reusable definitions |
| `06-chains.md` | ordered multi-step `chain` workflow |
| `07-tags.md` | front matter, footer hashtags, inline `#tag`, heading hints |
| `08-variable-syntax.md` | `$name` vs `<name>` command syntax (`var_syntax: dollar`/`angle`/`both`) |

See the [wiki](https://github.com/cheatmd-dev/cheatmd/wiki) for the full
reference on each feature.
