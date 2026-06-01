# 02 · Variables

The three `var` forms, plus environment pre-fill, resolution order, and line
continuation. Inside `<!-- cheat -->` blocks the DSL always uses `$name`.

## Prompt-only

`var name` asks the user to type a value.

```sh title:"Type a value with no default"
echo "you typed: $note"
```
<!-- cheat
var note
-->

## Prompt-only with a header label

```sh title:"Prompt with a custom header"
echo "host=$host"
```
<!-- cheat
var host --- --header "Enter the target hostname"
-->

## Shell-sourced picker (2+ lines of output)

Two or more output lines become a filterable selection list.

```sh title:"Pick from a list"
echo "picked $fruit"
```
<!-- cheat
var fruit = printf 'apple\nbanana\ncherry\n' --- --header "Pick a fruit"
-->

## Shell-sourced editable default (1 line of output)

A single output line pre-fills the input; the user can accept or edit it.

```sh title:"One line pre-fills the input"
echo "timeout=$timeout"
```
<!-- cheat
var timeout = echo "10" --- --header "Timeout (seconds)"
-->

## Shell-sourced empty output falls back to a text prompt

Zero output lines degrade gracefully to a manual text entry.

```sh title:"Empty output -> manual text entry"
echo "value=$freeform"
```
<!-- cheat
var freeform = printf '' --- --header "Nothing to pick — type it"
-->

## Literal value (`:=`, no shell)

The value is used directly and may reference earlier-resolved variables.

```sh title:"Use fixed literal values"
echo "GET $endpoint"
```
<!-- cheat
var version = echo "v1" --- --header "Version"
var endpoint := https://api.example.com/$version/status
-->

## Pre-fill from the environment

A variable whose name matches an environment variable pre-fills from it.

```sh title:"$HOME pre-fills from your environment"
ls $HOME
```
<!-- cheat
var HOME
-->

## Resolution order (a later var uses an earlier one)

Variables resolve top-to-bottom; later shell commands can use earlier values.

```sh title:"namespace is derived from the chosen context"
echo "context=$context namespace=$namespace"
```
<!-- cheat
var context = printf 'minikube\ndocker-desktop\n' --- --header "Context"
var namespace = echo "ns-for-$context" --- --header "Namespace"
-->

## Line continuation with a trailing backslash

```sh title:"A long var definition split across lines"
echo "service=$service"
```
<!-- cheat
var service = printf 'web\napi\nworker\n' \
    --- --header "Pick a service" \
        --delimiter " "
-->
