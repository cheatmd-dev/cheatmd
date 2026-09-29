// Package shellgen generates shell scripts for bash and zsh bindings and widgets
// (such as the ctrl+g interactive widget) that integrate cheatmd natively into
// the user's shell environment.
package shellgen

import (
	"fmt"
	"strings"

	"github.com/cheatmd-dev/cheatmd/pkg/config"
)

// BashWidget returns a bash script that binds the configured key to an
// interactive cheatmd widget, replacing the current command line with the
// selected command.
func BashWidget() string {
	return bashWidget("cheatmd")
}

func bashWidget(command string) string {
	keyWidget := config.Get().KeyWidget
	return fmt.Sprintf(`#!/usr/bin/env bash

_cheatmd_widget() {
   local -r input="${READLINE_LINE}"

   local output
   if [ -z "${input}" ]; then
      output="$(%s --print)" || return
   else
      output="$(%s --print --match "$input")" || return
   fi

   if [ -n "$output" ]; then
      READLINE_LINE="$output"
      READLINE_POINT=${#READLINE_LINE}
   fi
}

if [ ${BASH_VERSION:0:1} -lt 4 ]; then
   echo "cheatmd widget requires bash 4+" >&2
else
   bind -x '"%s": _cheatmd_widget'
fi
`, command, command, keyWidget)
}

// ZshWidget returns a zsh script that binds the configured key to an
// interactive cheatmd widget, replacing the current command line with the
// selected command.
func ZshWidget() string {
	return zshWidget("cheatmd")
}

func zshWidget(command string) string {
	keyWidget := config.Get().KeyWidget
	// Convert bash-style keybinding to zsh format (e.g., \C-g -> ^g)
	zshKey := convertToZshKey(keyWidget)
	return fmt.Sprintf(`#!/usr/bin/env zsh

_cheatmd_widget() {
   local input="$BUFFER"

   local output
   if [ -z "$input" ]; then
      output="$(%s --print)" || return
   else
      output="$(%s --print --match "$input")" || return
   fi

   if [ -n "$output" ]; then
      BUFFER="$output"
      CURSOR=${#BUFFER}
   fi

   zle reset-prompt
}

zle -N _cheatmd_widget
bindkey '%s' _cheatmd_widget
`, command, command, zshKey)
}

// FishWidget returns a fish script that binds the configured key to an
// interactive cheatmd widget, replacing the current command line with the
// selected command.
func FishWidget() string {
	return fishWidget("cheatmd")
}

func fishWidget(command string) string {
	keyWidget := config.Get().KeyWidget
	// Convert bash-style keybinding to fish format (e.g., \C-g -> \cg)
	fishKey := convertToFishKey(keyWidget)
	return fmt.Sprintf(`function _cheatmd_widget
   set -l input (commandline)
   set -l output
   set -l cmd_status 0

   if test -z "$input"
      set output (%s --print)
      set cmd_status $status
   else
      set output (%s --print --match "$input")
      set cmd_status $status
   end

   if test $cmd_status -ne 0
      return
   end

   if test -n "$output"
      commandline -r "$output"
      commandline -f end-of-line
   end

   commandline -f repaint
end

bind %s _cheatmd_widget
`, command, command, fishKey)
}

// Widget generates an integration bound to a particular executable. Quote its
// path for the chosen shell so spaces and apostrophes remain part of the path.
func Widget(shell, executable string) (string, error) {
	quoted := "'" + strings.ReplaceAll(executable, "'", "'\"'\"'") + "'"
	switch shell {
	case "bash":
		return bashWidget(quoted), nil
	case "zsh":
		return zshWidget(quoted), nil
	case "fish":
		quoted = "'" + strings.NewReplacer("\\", "\\\\", "'", "\\'").Replace(executable) + "'"
		return fishWidget(quoted), nil
	default:
		return "", fmt.Errorf("unsupported shell: %s (supported: bash, zsh, fish)", shell)
	}
}

// convertToZshKey converts a bash-style keybinding to zsh format
// e.g., \C-g -> ^g, \C-x -> ^x
func convertToZshKey(key string) string {
	if strings.HasPrefix(key, "\\C-") {
		return "^" + strings.ToLower(key[3:])
	}
	// Already in zsh format or other format
	return key
}

// convertToFishKey converts a bash-style keybinding to fish format
// e.g., \C-g -> \cg, \C-x -> \cx
func convertToFishKey(key string) string {
	if strings.HasPrefix(key, "\\C-") {
		return "\\c" + strings.ToLower(key[3:])
	}
	// Already in fish format or other format
	return key
}
