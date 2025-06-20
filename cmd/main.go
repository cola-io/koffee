package main

import (
	"fmt"
	"os"

	"cola.io/koffee/cmd/app"
)

func main() {
	if err := app.NewCommand().Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to start mcp server: %v", err)
		os.Exit(1)
	}
}
