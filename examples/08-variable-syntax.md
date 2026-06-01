# 08 · Variable Syntax (`$name` vs `<name>`)

The `var_syntax` config setting controls which forms CheatMD recognizes in
**command text**: `dollar` (default, `$name`), `angle` (`<name>`), or `both`.
Inside `<!-- cheat -->` blocks the DSL **always** uses `$name`, regardless of
this setting, so your metadata stays portable.

> The angle-bracket variables below only resolve when `var_syntax: angle` or
> `var_syntax: both`. Under the default `dollar` they render literally.

## Angle-bracket variables (requires `var_syntax: angle` or `both`)

The command uses `<user>` and `<host>`; the DSL declares them with `$`.

```sh title:"SSH to a host using <angle> variables"
echo "ssh <user>@<host>"
```
<!-- cheat
var user = echo "$USER" --- --header "Username"
var host --- --header "Hostname"
-->

## Mixed `$` and `<>` in one command (requires `var_syntax: both`)

```sh title:"Mix both variable forms"
echo "curl <url> -H \"Authorization: Bearer $token\""
```
<!-- cheat
var url --- --header "URL"
var token --- --header "Token"
-->

## Declaration-free (requires `var_syntax: both` + `allow_undeclared_vars: true`)

With both settings enabled, undeclared `$` and `<>` references are prompted
automatically — no `<!-- cheat -->` block is needed at all.

```sh title:"No metadata block; every reference is prompted"
echo "scp <file> $user@<host>:/tmp/"
```
