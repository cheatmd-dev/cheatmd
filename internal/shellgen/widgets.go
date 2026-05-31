package shellgen

import (
	"fmt"
	"strings"

	"github.com/cheatmd-dev/cheatmd/pkg/config"
)

func BashWidget() string {
	keyWidget := config.Get().KeyWidget
	return fmt.Sprintf(`#!/usr/bin/env bash

_cheatmd_widget() {
   local -r input="${READLINE_LINE}"

   local output
   if [ -z "${input}" ]; then
      output="$(cheatmd --print)" || return
   else
      output="$(cheatmd --print --match "$input")" || return
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
`, keyWidget)
}

func ZshWidget() string {
	keyWidget := config.Get().KeyWidget
	// Convert bash-style keybinding to zsh format (e.g., \C-g -> ^g)
	zshKey := convertToZshKey(keyWidget)
	return fmt.Sprintf(`#!/usr/bin/env zsh

_cheatmd_widget() {
   local input="$BUFFER"

   local output
   if [ -z "$input" ]; then
      output="$(cheatmd --print)" || return
   else
      output="$(cheatmd --print --match "$input")" || return
   fi

   if [ -n "$output" ]; then
      BUFFER="$output"
      CURSOR=${#BUFFER}
   fi

   zle reset-prompt
}

zle -N _cheatmd_widget
bindkey '%s' _cheatmd_widget
`, zshKey)
}

func FishWidget() string {
	keyWidget := config.Get().KeyWidget
	// Convert bash-style keybinding to fish format (e.g., \C-g -> \cg)
	fishKey := convertToFishKey(keyWidget)
	return fmt.Sprintf(`function _cheatmd_widget
   set -l input (commandline)
   set -l output
   set -l cmd_status 0

   if test -z "$input"
      set output (cheatmd --print)
      set cmd_status $status
   else
      set output (cheatmd --print --match "$input")
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
`, fishKey)
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
