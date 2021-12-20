// (C) 2017 Christoph Berger <mail@christophberger.com>. Some rights reserved.
// Distributed under a 3-clause BSD license; see LICENSE.txt.

// goman - the missing man pages for Go binaries
//
// goman replaces missing man pages for Go binaries by locating the README file of the corresponding source code project and rendering this file as plain text with ANSI colors, to be viewed in a terminal, optionally through less -R.
//
// Usage:
//
//     goman <go binary file>
//
// or
//
//     goman <go binary file> | less -R
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
)

func usage() {
	fmt.Print(`Usage:

goman <name of Go binary>

goman is man for Go binaries. It attempts to fetch the README file of a Go binary's project and displays it in the terminal, if found.

`)
	flag.Usage()
	b, _ := debug.ReadBuildInfo()
	fmt.Printf("\nModule info: %s %s\n", b.Main.Path, b.Main.Version)
}

var (
	names      = []string{"README.md", "README", "README.txt", "README.markdown"}
	remoteOnly *bool
	verbose    *bool
)

func main() {

	log.SetFlags(0)

	verbose = flag.Bool("v", false, "Verbose error output")
	remoteOnly = flag.Bool("r", false, "Skip local search (as the local file may be outdated)")
	flag.Parse()

	if len(flag.Args()) != 1 {
		usage()
		return
	}
	switch flag.Args()[0] {
	// easier than defining four flags and checking for the string "help" also:
	case "-h", "-help", "--help", "-?", "help":
		readme, _, err := findReadme("github.com/appliedgocode/goman")
		if err != nil {
			usage()
			return
		}
		fmt.Println(string(mdToAnsi(readme)))
		return
	}

	exec := flag.Args()[0]

	go exitOnSignal()

	run(exec)
}

// In case goman gets stuck somewhere. This should not happen under normal circumstances.
// There are no resources to be cleaned up, so we just exit.
func exitOnSignal() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	sig := <-c
	log.Fatalf("received signal %s", sig)
}
