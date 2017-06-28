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

	"golang.org/x/crypto/ssh/terminal"

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

allLoops:
	// We have to search across all gopath elements, across all README file names,
	// and also up the directory tree (in case the command is a subproject)
	for _, gp := range gopath() {
		for _, name := range names {
			for _, source := range sources(src) {

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
	// TODO: implement resolveVanityImport
	// src, err = resolveVanityImports(src)
	// if err != nil {
	// 	return nil, "", errors.Wrap(err, "error resolving vanity import for "+src)
	// }
	for _, source := range sources(src) {
		url = getReadmeURL(source)
		for _, name := range names {
			readme, e = httpGetReadme(url + name)
			if e == nil {
				return readme, url, nil
			}
			err = errors.Wrap(e, "") // collect all errors from the loop
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

// sourcePathIterator returns a function that yields the full `src` string on the
// first call, and the `src` string stripped of the last directory on every
// subsequent call, until reaching the root directory of the project.
// Then any subsequent call yields an empty string.
//
// Example:
//
// src = github.com/user/project/subdir/subdir
// 1st invocation yields src
// 2nd invocation yields github.com/user/project/subdir
// 3rd invocation yields github.com/user/project
// 4th invocation yields ""
func sources(src string) []string {
	strings.Trim(src, "/")
	dirs := strings.Split(src, "/")
	paths := make([]string, 0, len(dirs))

	for len(dirs) > 0 {
		paths = append(paths, strings.Join(dirs, "/"))
		dirs = dirs[:len(dirs)-1]
	}
	return paths
}

// Resolve any redirects to get the correct URL to the remote repository.
// func resolveVanityImports(src string) (string, error) {
// 	var client = &http.Client{
// 		Timeout: time.Second * 10,
// 	}

// 	response, err := client.Get("https://" + src + "?go-get=1")
// 	if err != nil {
// 		return "", errors.Wrap(err, "failed retrieving header data of URL "+src)
// 	}

// 	// TODO:
// 	// extract meta tag "go-import" from response body
// 	// <meta name="go-import" content="import-prefix vcs repo-root">
// 	// get the `repo-root` part
// 	// strip the https:// prefix
// }

// getReadmeURL receives the relative path to the project and returns
// the URL to the raw README.md file (WITHOUT the file name itself, but
// WITH a trailing slash).
// Currently it knows how to do this for github.com and gitlab.com only.
// For all other sites, it returns `https://<src>/`.
//
// Examples:
//
// From github.com/ec1oud/mdcat to:
// https://raw.githubusercontent.com/ec1oud/mdcat/master/
//
// From gitlab.com/SporeDB/sporedb to:
// https://gitlab.com/SporeDB/sporedb/raw/master/

func getReadmeURL(src string) string {

	prefix := "https://"
	gh := "github.com/"
	gl := "gitlab.com/"
	ghraw := "raw.githubusercontent.com/"

	src = strings.Trim(filepath.ToSlash(src), "/")

	if len(src) >= len(prefix)+len(gh) && src[:len(gh)] == gh {
		return prefix + ghraw + src[len(gh):] + "/master/"
	}

	if len(src) >= len(prefix)+len(gl) && src[:len(gl)] == gl {
		return prefix + src + "/raw/master/"
	}

	return prefix + src + "/"
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

	// Get the current terminal width, or 80 if the width cannot be determined
	w, _, err := terminal.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		w = 80
	}
	renderer := blackfriday.AnsiRenderer(w, ansiFlags)

	return blackfriday.Markdown(readme, renderer, extensions)
}
