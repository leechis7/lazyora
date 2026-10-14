package main

import (
	"flag"
	"fmt"
	"os"

	"lazyora/internal/ui"
)

func main() {
	demo := flag.Bool("demo", false, "preview the interface without an Oracle server")
	flag.Parse()
	if err := ui.Run(*demo); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
