// cmd/gofile shows the platform string of Go executables
//
// cmd/gofile takes one argument, which is either program name or abosolute or
// relative path to an executable; when a program name is provided, it's looked up
// up in the $PATH.
//
// Example usage
//
//   ~ $ gowhich godoc
//   darwin_amd64
package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/rjeczalik/which"
)

func die(v interface{}) {
	fmt.Println(os.Stderr, v)
	os.Exit(1)
}

const usage = `NAME:
	gofile - shows the platform string of Go executables

USAGE:
	gofile name|path

EXAMPLES:
	gofile godoc
	gofile ~/bin/godoc`

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
	ex, err := which.NewExec(path)
	if err != nil {
		die(err)
	}
	fmt.Println(ex.Type)
}
