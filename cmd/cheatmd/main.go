package main

import (
	"os"
)

var version = "1.0.0-rc.1"

func main() {
	rootCmd.Version = version
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
