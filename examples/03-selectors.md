# 03 · Selector Options

Options appear after `---` on a `var` line and control how the picker displays
shell output and what value is returned.

## `--header`

```sh title:"Header text above the picker"
echo "region=$region"
```
<!-- cheat
var region = printf 'us-east-1\neu-west-1\nap-south-1\n' --- --header "Choose a region"
-->

## `--delimiter` and `--column` (display only)

`--column` changes what the user *sees*; the whole original line is returned.

```sh title:"Show column 1, return the full line"
echo "selected: $row"
```
<!-- cheat
var row = printf 'web 10.0.0.1 running\napi 10.0.0.2 running\n' --- --delimiter " " --column 1 --header "Service (column 1 shown)"
-->

## `--select-column` (display one column, return another)

The user sees the description; the variable receives the key.

```sh title:"Show description, return the key"
echo "auth=$auth"
```
<!-- cheat
var auth = printf 'key\tUse an SSH key\npassword\tUse a password\n' --- --delimiter '\t' --column 2 --select-column 1 --header "Auth method"
-->

## `--map` (transform the selected value)

Pipe the selection through a command via stdin.

```sh title:"Lower-case the picked value"
echo "letter=$letter"
```
<!-- cheat
var letter = printf 'A\nB\nC\n' --- --map "tr '[:upper:]' '[:lower:]'" --header "Pick (returned lower-case)"
-->

## `--multi` (select several, joined)

Space toggles checkboxes; selections join on `--delimiter` (comma by default).

```sh title:"Open several ports"
echo "ports=$ports"
```
<!-- cheat
var ports = printf '80\n443\n8080\n' --- --multi --delimiter , --header "Select ports (Space toggles)"
-->

## All options combined

Display the name, return the IP, then map it.

```sh title:"Name shown, /24 of the IP returned"
echo "target=$target"
```
<!-- cheat
var target = printf 'web 10.0.0.1\napi 10.0.0.2\n' \
    --- --delimiter " " \
        --column 1 \
        --select-column 2 \
        --map "cut -d. -f1-3" \
        --header "Pick host (name shown, IP/24 returned)"
-->
