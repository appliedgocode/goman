package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	whichcmd "github.com/bfontaine/which/which"
	"github.com/pkg/errors"
	"github.com/rjeczalik/which"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage:")
		fmt.Println()
		fmt.Println("goman <name of Go binary>")
		fmt.Println()
		fmt.Println("goman is man for Go binaries. It attempts to fetch the README file of a Go binary's project and displays it in the terminal, if found")
		return
	}

	exec := os.Args[1]

	// Determine the path of `exec`
	path, err := getPath(exec)
	if err != nil {
		log.Println("Cannot determine the path of", exec)
		log.Fatal(errors.Cause(err))
	}

	// Extract the source path from the binary
	src, err := which.Import(path)
	if err != nil {
		log.Println("Cannot determine source path of", path)
		log.Fatal(errors.Cause(err))
	}

	fmt.Println(src)

}

// getPath receives the name of an executable and determines its path
// based on $PATH, $GOPATH, or the current directory (in this order).
func getPath(name string) (string, error) {

	// Try $PATH first.
	s := whichcmd.One(name)
	if s != "" {
		return s, nil
	}

	// Next, try $GOPATH/bin
	path := getGOPATH()
	for i := 0; s == "" && i < len(path); i++ {
		s = whichcmd.OneWithPath(name, path[i]+filepath.Join("bin"))
	}
	if s != "" {
		return s, nil
	}

	// Finally, try the current directory.
	wd, err := os.Getwd()
	if err != nil {
		return "", errors.Wrap(err, "Unable to determine current directory")
	}
	s = whichcmd.OneWithPath(name, wd)
	if s == "" {
		return "", errors.New(name + " not found in any of " + os.Getenv("PATH") + ":" + strings.Join(path, ":"))
	}

	return s, nil
}

// pathssep returns the separator between the paths of $PATH or %PATH%.
func pathssep() string {
	sep := ":"
	if runtime.GOOS == "windows" {
		sep = ";"
	}
	return sep
}

// getGOPATH returns a list of paths representing the
func getGOPATH() []string {
	gp := os.Getenv("GOPATH")
	if gp == "" {
		return []string{defaultGOPATH()}
	}
	return strings.Split(gp, pathssep())
}

func defaultGOPATH() string {
	// https://stackoverflow.com/a/32650077
	env := "HOME"
	if runtime.GOOS == "windows" {
		env = "USERPROFILE"
	} else if runtime.GOOS == "plan9" {
		env = "home"
	}
	if home := os.Getenv(env); home != "" {
		def := filepath.Join(home, "go")
		if filepath.Clean(def) == filepath.Clean(runtime.GOROOT()) {
			// Don't set the default GOPATH to GOROOT,
			// as that will trigger warnings from the go tool.
			return ""
		}
		return def
	}
	return ""
}
