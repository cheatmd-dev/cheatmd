# 04 · Conditionals

`if $var == value` … `fi` branches variable definitions on a value resolved
earlier in the same block. Only `==` and `!=` are supported.

## Branch on a choice (`==`)

```sh title:"SSH flags depend on the auth method"
echo "ssh $ssh_flags user@host"
```
<!-- cheat
var auth = printf 'key\npassword\n' --- --header "Auth method"
if $auth == key
var ssh_flags := -o PreferredAuthentications=publickey
fi
if $auth == password
var ssh_flags := -o PreferredAuthentications=password
fi
-->

## Negative condition (`!=`) and an all-conditional variable

`flags` is defined *only* inside a conditional. If the condition is false it is
silently set to empty — no dangling prompt.

```sh title:"Add -i unless the mode is force"
echo "rm $flags somefile"
```
<!-- cheat
var mode = printf 'safe\nforce\n' --- --header "Mode"
if $mode != force
var flags := -i
fi
-->

## Nested conditionals

Nested `if`/`fi` is supported — the region prompt only appears for `prod`, and
the confirmation is gated on a second, nested condition.

```sh title:"Region is prompted only for prod"
echo "env=$env region=$region confirm=$confirm"
```
<!-- cheat
var env = printf 'dev\nprod\n' --- --header "Environment"
var confirm := no
if $env == prod
var region = printf 'us-east-1\neu-west-1\n' --- --header "Region"
if $region == us-east-1
var confirm := yes-primary
fi
fi
-->
