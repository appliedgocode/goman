package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/bfontaine/which/which"
	"github.com/ec1oud/blackfriday"
	"github.com/pkg/errors"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage:")
		fmt.Println()
		fmt.Println("goman <name of Go binary>")
		fmt.Println()
		fmt.Println("goman is man for Go binaries. It attempts to fetch the README file of a Go binary's project and displays it in the terminal, if found.")
		return
	}

	exec := os.Args[1]

	// Determine the path of `exec`
	path, err := getPath(exec)
	if err != nil {
		log.Println("Cannot determine the path of", exec)
		log.Fatalln(errors.WithStack(err))
	}

	// Extract the source path from the binary
	src, err := getMainPath(path)
	// src, err := which.Import(path)
	if err != nil {
		log.Println("Cannot determine source path of", path)
		log.Fatalln(errors.WithStack(err))
	}

	src = stripPath(src)

	// Find the README
	readme, isMarkdown, err := findReadme(src)
	if err != nil {
		log.Println("No README found for", exec, "at", src)
		log.Fatalln(errors.WithStack(err))
	}

	if isMarkdown {
		readme = mdToAnsi(readme)
	}

	fmt.Println(string(readme))

}

// getPath receives the name of an executable and determines its path
// based on $PATH, $GOPATH, or the current directory (in this order).
func getPath(name string) (string, error) {

	// Try $PATH first.
	s := which.One(name) // $ which <name>
	if s != "" {
		return s, nil
	}

	// Next, try $GOPATH/bin
	paths := gopath()
	for i := 0; s == "" && i < len(paths); i++ {
		s = which.OneWithPath(name, paths[i]+filepath.Join("bin"))
	}
	if s != "" {
		return s, nil
	}

	// Finally, try the current directory.
	wd, err := os.Getwd()
	if err != nil {
		return "", errors.Wrap(err, "Unable to determine current directory")
	}
	s = which.OneWithPath(name, wd)
	if s == "" {
		return "", errors.New(name + " not found in any of " + os.Getenv("PATH") + ":" + strings.Join(paths, ":"))
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

// gopath returns a list of paths as defined by the GOPATH environment
// variable, or the default gopath if $GOPATH is empty.
func gopath() []string {
	gp := os.Getenv("GOPATH")
	if gp == "" {
		return []string{defaultGopath()}
	}
	return strings.Split(gp, pathssep())
}

// defaultGopath returns the default Go path, depending on the operating system.
func defaultGopath() string {
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

// strip path strips the GOPATH prefix from the raw source code path
// as returned by getMainPath.
func stripPath(path string) string {
	if !filepath.IsAbs(path) {
		return path
	}
	n := strings.Index(path, "/src/")
	if n > -1 {
		return path[n+5:]
	}
	return path
}

// findReadme attempts to find the file README.md either locally
// or in the remote repository of the executable.
func findReadme(src string) (readme []byte, isMarkdown bool, err error) {
	readme, isMarkdown, err = findLocalReadme(src)
	if err == nil {
		return readme, isMarkdown, nil
	}
	readme, isMarkdown, err = findRemoteReadme(src)
	if err != nil {
		return nil, false, errors.Wrap(err, "Did not find a readme locally nor in the remote repository")
	}
	return readme, isMarkdown, nil
}

// findLocalReadme is a helper function for findReadme. It searches the README file
// locally in $GOPATH/src/<src>.
func findLocalReadme(src string) (readme []byte, isMarkdown bool, err error) {
	found := false
	for _, gp := range gopath() {

		// Create the path to the README file and open the file.
		// If this fails, try the next GOPATH entry.
		fp := filepath.Join(gp, "src", src, "README.md") // TODO: Take other filenames into account
		rf, err := os.Open(fp)
		if err != nil {
			err = errors.Wrap(err, "README not found at location "+fp)
			_ = rf.Close()
			continue
		}

		// Allocate the slice for reading the file.
		fi, err := rf.Stat()
		if err != nil {
			return nil, false, errors.Wrap(err, "Cannot determine file stats")
		}
		readme = make([]byte, fi.Size())

		// Read the whole file.
		n, err := rf.Read(readme)
		if err != nil {
			return readme[:n], false, errors.Wrap(err, "Error reading from file "+fp)
		}
		_ = rf.Close()
		found = true
		break
	}
	if !found {
		return nil, false, errors.Errorf("No README found for %s in any of %s\n", src, gopath())
	}
	return readme, true, err
}

// findRemoteReadme is a helper function for findReadme. It attempts to locate the README in the remote repository identified by http(s)://<src>/blob/master/README.m
func findRemoteReadme(src string) ([]byte, bool, error) {
	// TODO:
	return []byte(""), false, errors.New("Not implemented")
}

func mdToAnsi(readme []byte) []byte {

	// The code in this function was copied from github.com/ec1oud/mdcat
	extensions := 0
	extensions |= blackfriday.EXTENSION_NO_INTRA_EMPHASIS
	extensions |= blackfriday.EXTENSION_TABLES
	extensions |= blackfriday.EXTENSION_FENCED_CODE
	extensions |= blackfriday.EXTENSION_AUTOLINK
	extensions |= blackfriday.EXTENSION_STRIKETHROUGH
	extensions |= blackfriday.EXTENSION_SPACE_HEADERS

	ansiFlags := 0
	ansiFlags |= blackfriday.ANSI_USE_SMARTYPANTS
	ansiFlags |= blackfriday.ANSI_SMARTYPANTS_FRACTIONS
	ansiFlags |= blackfriday.ANSI_SMARTYPANTS_LATEX_DASHES

	renderer := blackfriday.AnsiRenderer(80, ansiFlags) // TODO get terminal width

	return blackfriday.Markdown(readme, renderer, extensions)
}
