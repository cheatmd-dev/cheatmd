# 05 · Modules

`export` publishes a reusable variable definition; `import` pulls it into other
cheats. Exports and imports resolve across every file in the cheats directory,
so the consumers below could live in any file.

## Module definitions

Bare `<!-- cheat -->` blocks can define exports without a runnable command.

<!-- cheat
export git_branch
var branch = printf 'main\ndevelop\nfeature-x\n' --- --header "Select branch"
-->

<!-- cheat
export container
var container = printf 'web\napi\nworker\n' --- --header "Select container"
-->

## Consumer: checkout a branch

```sh title:"Uses the imported branch picker"
echo "git checkout $branch"
```
<!-- cheat
import git_branch
-->

## Consumer: exec into a container

```sh title:"Uses the imported container picker"
echo "docker exec -it $container /bin/sh"
```
<!-- cheat
import container
-->

## Consumer: import several modules at once

```sh title:"Both modules imported into one cheat"
echo "deploy $branch to $container"
```
<!-- cheat
import git_branch
import container
-->
