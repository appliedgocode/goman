// cmd/gowhich shows the import path of Go executables
//
// cmd/gowhich takes one argument, which is either program name or abosolute or
// relative path to an executable; when a program name is provided, it's looked up
// up in the $PATH.
//
// cmd/gowhich looks for a main.main symbol in the given executable and tries
// to guess the import name from its source files path.
//
// cmd/gowhich does not work on Go executables from $GOTOOLDIR.
//
// Example usage
//
//   ~ $ gowhich godoc
//   code.google.com/p/go.tools/cmd/godoc
package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/rjeczalik/which"
)

func die(v interface{}) {
	fmt.Fprintln(os.Stderr, v)
	os.Exit(1)
}

const usage = `NAME:
	gowhich - shows the import path of Go executables

USAGE:
	gowhich name|path

EXAMPLES:
	gowhich godoc
	gowhich ~/bin/godoc`

func ishelp(s string) bool {
	return s == "-h" || s == "-help" || s == "help" || s == "--help" || s == "/?"
}

func main() {
	if len(os.Args) != 2 {
		die(usage)
	}
	if ishelp(os.Args[1]) {
		fmt.Println(usage)
		return
	}
	path, err := exec.LookPath(os.Args[1])
	if err != nil {
		die(err)
	}
	pkg, err := which.Import(path)
	if err != nil {
		die(err)
	}
	fmt.Println(pkg)
}
