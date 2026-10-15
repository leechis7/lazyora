package main

import (
	"flag"
	"fmt"
	"os"

	"lazyora/internal/appinfo"
	"lazyora/internal/ui"
)

func main() {
	demo := flag.Bool("demo", false, "preview the interface without an Oracle server")
	version := flag.Bool("version", false, "print the application version and exit")
	flag.Parse()
	if *version {
		fmt.Printf("%s v%s\n", appinfo.Name, appinfo.Version)
		return
	}
	if err := ui.Run(*demo); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
