# 06 · Chains

A chain is an ordered, multi-step workflow. Each step is its own cheat tagged
`chain <name> <step>`. Search `/chain` in the picker; selecting the chain runs
the next pending step and exits. The next launch resumes at the following step.
After the last step the chain resets to step 1.

Reset progress with `cheatmd chain reset` (all) or `cheatmd chain reset release`.

## Release · choose version

```sh title:"Step 1 — pick the version"
echo "version=$version"
```
<!-- cheat
chain release 1
var version --- --header "Release version"
-->

## Release · build

```sh title:"Step 2 — build the artifact"
echo "building $version"
```
<!-- cheat
chain release 2
var version --- --header "Release version"
-->

## Release · publish

```sh title:"Step 3 — publish the artifact"
echo "publishing $version"
```
<!-- cheat
chain release 3
var version --- --header "Release version"
-->
