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
	"bufio"
	"flag"
	"fmt"
	"go/build"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/bfontaine/which/which"
	"github.com/ec1oud/blackfriday"
	"github.com/pkg/errors"
)

var (
	names      = []string{"README.md", "README", "README.txt", "README.markdown"}
	remoteOnly *bool
)

func main() {

	log.SetFlags(0)

	verbose := flag.Bool("v", false, "Verbose error output")
	remoteOnly = flag.Bool("r", false, "Skip local search (as the local file may be outdated)")
	flag.Parse()

	if len(flag.Args()) != 1 {
		usage()
		return
	}
	switch flag.Args()[0] {
	// easier than defining four flags and checking for the string "help" also:
	case "-h", "-help", "--help", "-?", "help":
		readme, _, err := findReadme("github.com/christophberger/goman")
		if err != nil {
			usage()
			return
		}
		fmt.Println(string(mdToAnsi(readme)))
		return
	}

	exec := flag.Args()[0]

	// Determine the location of `exec`
	path, err := getExecPath(exec)
	if err != nil {
		log.Println(exec + ": command not found")
		if *verbose {
			log.Println(errors.WithStack(err))
		}
		return
	}

	// Extract the source path from the binary
	src, err := getMainPath(path)
	if err != nil {
		log.Println("No source path in", path, "-", exec, "is perhaps no Go binary")
		if *verbose {
			log.Println(errors.WithStack(err))
		}
		return
	}

	// Find the README
	readme, source, err := findReadme(src)
	if err != nil {
		log.Println("No README found for", exec, "at", src)
		if *verbose {
			log.Println(errors.WithStack(err))
		}
		return
	}

	readme = mdToAnsi(readme)

	fmt.Printf("%s\n\n(Source: %s)\n\n", string(readme), source)

}

func usage() {
	fmt.Println(`Usage:

goman <name of Go binary>

goman is man for Go binaries. It attempts to fetch the README file of a Go binary's project and displays it in the terminal, if found.
`)
}

// getExecPath receives the name of an executable and determines its path
// based on $PATH, $GOPATH, or the current directory (in this order).
func getExecPath(name string) (string, error) {

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
		return []string{build.Default.GOPATH}
	}

	return strings.Split(gp, pathssep())
}

// findReadme attempts to find the file README.md either locally
// or in the remote repository of the executable.
func findReadme(src string) (readme []byte, source string, err error) {

	if !*remoteOnly {
		readme, source, err = findLocalReadme(src)
		if err == nil {
			return readme, source, nil
		}
	}

	readme, source, err = findRemoteReadme(src)
	if err != nil {
		return nil, "", errors.Wrap(err, "Did not find a readme locally nor in the remote repository")
	}

	return readme, source, nil
}

// findLocalReadme is a helper function for findReadme. It searches the README file
// locally in $GOPATH/src/<src>.
func findLocalReadme(src string) (readme []byte, fp string, err error) {

	found := false
	var e error

	// If the source path contains no /cmd/, sources contains only src.
	// Otherwise it contains the unchanged src and src with /cmd/... stripped.
	// The point is, most subcommands have no own README file, and therefore
	// we must also search in the main project.
	sources := []string{src}
	if containsCmdPath(src) {
		sources = append(sources, removeCmdPath(src))
	}

allLoops:
	// We have to search across all gopath elements, across all README file names,
	// and for sources containing /cmd/ also the main project.
	for _, gp := range gopath() {
		for _, name := range names {
			for _, source := range sources {

				// Create the path to the README file and open the file.
				// If this fails, try the next GOPATH entry.
				fp = filepath.Join(gp, "src", source, name)
				rf, err := os.Open(fp)
				if err != nil {
					e = errors.Wrap(err, "README not found at location "+fp)
					_ = rf.Close()
					continue
				}

				// Allocate the slice for reading the file.
				fi, err := rf.Stat()
				if err != nil {
					return nil, "", errors.Wrap(err, "Cannot determine file stats")
				}
				readme = make([]byte, fi.Size())

				// Read the whole file.
				n, err := rf.Read(readme)
				if err != nil {
					return readme[:n], "", errors.Wrap(err, "Error reading from file "+fp)
				}
				_ = rf.Close()
				found = true
				break allLoops
			}
		}
	}

	if !found {
		return nil, "", errors.Wrapf(e, "No README found for %s in any of %s\n", src, gopath())
	}

	return readme, fp, err
}

// findRemoteReadme is a helper function for findReadme. It attempts to locate the README in the remote repository at either of: -
// - http(s)://host.com/<user>/<project>/blob/master/<readme name>
// - http(s)://host.com/<user>/<project>/blob/master/cmd/<cmdname>/<readme name>

func findRemoteReadme(src string) (readme []byte, url string, err error) {

	var e error
	sources := []string{src}

	if containsCmdPath(src) {
		sources = append(sources, removeCmdPath(src))
	}

	for _, source := range sources {
		url, err = getRawReadmeURL(source)
		for _, name := range names {
			readme, err = httpGetReadme(url + name)
			if err == nil {
				return readme, url, nil
			}
			e = errors.Wrap(err, "")
		}
	}

	return nil, "", errors.Wrap(e, "Failed to retrieve README")
}

func httpGetReadme(url string) ([]byte, error) {

	var client = &http.Client{
		Timeout: time.Second * 10,
	}

	response, err := client.Get(url)
	if err != nil {
		return nil, errors.Wrap(err, "Failed downloading the README from "+url)
	}

	if response.StatusCode != 200 {
		return nil, errors.New("HTTP GET returned " + response.Status + " for URL " + url)
	}

	r := bufio.NewReader(response.Body)
	readme, err := r.ReadBytes(0)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "Error reading README from HTTP response")
	}

	return readme, nil
}

func containsCmdPath(s string) bool {
	return strings.Contains(s, "/cmd/")
}
func removeCmdPath(s string) string {
	return s[:strings.Index(s, "/cmd/")]
}

// getRawReadmeURL receives the relative path to the project and returns
// the URL to the raw README.md file (WITHOUT the file name itself, but
// WITH a trailing slash).
// Currently it knows how to do this for github.com and gitlab.com only.
func getRawReadmeURL(src string) (string, error) {

	prefix := "https://"
	gh := "github.com/"
	gl := "gitlab.com/"
	ghraw := "raw.githubusercontent.com/"

	// https://raw.githubusercontent.com/ec1oud/mdcat/master/README.md
	// https://gitlab.com/SporeDB/sporedb/raw/master/README.md

	src = filepath.ToSlash(src)

	if src[:len(gh)] == gh {
		return prefix + ghraw + src[len(gh):] + "/master/", nil
	}

	if src[:len(gl)] == gl {
		return prefix + src + "/raw/master/", nil
	}

	return "", errors.New("No supported host found in source path: " + src)
}

func mdToAnsi(readme []byte) []byte {

	// The code in this function was copied from github.com/ec1oud/mdcat. See LICENSE.mdcat.txt
	extensions := 0
	extensions |= blackfriday.EXTENSION_NO_INTRA_EMPHASIS
	extensions |= blackfriday.EXTENSION_TABLES
	extensions |= blackfriday.EXTENSION_FENCED_CODE
	extensions |= blackfriday.EXTENSION_AUTOLINK
	extensions |= blackfriday.EXTENSION_STRIKETHROUGH
	extensions |= blackfriday.EXTENSION_SPACE_HEADERS

	ansiFlags := 0

	renderer := blackfriday.AnsiRenderer(80, ansiFlags) // TODO get terminal width

	return blackfriday.Markdown(readme, renderer, extensions)
}
