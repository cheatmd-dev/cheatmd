package main

import (
	"os"
)

var version = "1.0.0-rc.2"

func main() {
	rootCmd.Version = version
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
