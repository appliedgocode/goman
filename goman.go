// (C) 2017 Christoph Berger <mail@christophberger.com>. Some rights reserved.
// Distributed under a 3-clause BSD license; see LICENSE.txt.

package main

import (
	"bufio"
	"debug/buildinfo"
	"fmt"
	"go/build"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/ec1oud/blackfriday"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh/terminal"
)

var (
	names = []string{"README.md", "README", "README.txt", "readme.md", "readme", "readme.txt", "README.MD", "README.TXT"}
)

func run(exec string) {

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
	src, ver, err := getMainPathAndVersion(path)
	if err != nil {
		log.Println("No source path in", path, "-", exec, "is perhaps no Go binary")
		if *verbose {
			log.Println(errors.WithStack(err))
		}
		return
	}

	// Find the README
	readme, source, err := findReadme(src, ver)
	if err != nil {
		log.Println("No README found for", exec, "in", src)
		if *verbose {
			log.Println(errors.WithStack(err))
		}
		return
	}

	readme = mdToAnsi(readme)

	fmt.Printf("%s\n\n(Source: %s)\n\n", string(readme), source)

}

// getMainPathAndVersion fetches the main maodule path from the binary>'s
// build info, or, if the binary is from the pre-module era,
// from the respective info in the symbol table.
func getMainPathAndVersion(file string) (string, string, error) {
	bi, err := buildinfo.ReadFile(file)
	if err != nil {
		return getMainPathDwarf(file)
	}
	version := bi.Main.Version
	if version == "(devel)" {
		version = ""
	}
	return bi.Path, version, nil
}

// getExecPath receives the name of an executable and determines its path
// based on $PATH, $GOPATH, or the current directory (in this order).
func getExecPath(name string) (string, error) {

	// Try $PATH first.
	s, err := exec.LookPath(name)
	if err == nil {
		return s, nil
	}

	// Next, try $GOPATH/bin
	paths := gopath()
	for i := 0; s == "" && i < len(paths); i++ {
		s, err = exec.LookPath(filepath.Join(paths[i], name))
	}
	if err == nil {
		return s, nil
	}

	// Finally, try the current directory.
	wd, err := os.Getwd()
	if err != nil {
		return "", errors.Wrap(err, "Unable to determine current directory")
	}
	s, err = exec.LookPath(filepath.Join(wd, name))
	if err != nil {
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
		cmd := exec.Command("go", "env", "GOPATH")
		gpb, err := cmd.CombinedOutput()
		if err != nil {
			return []string{build.Default.GOPATH}
		}
		// CombinedOutput seems to include the newline char - remove it
		gp = strings.TrimRight(string(gpb), "\n")
	}

	return strings.Split(gp, pathssep())
}

// findReadme attempts to find the file README.md either locally
// or in the remote repository of the executable.
func findReadme(src, ver string) (readme []byte, source string, err error) {

	src = stripModVersion(src)
	srcs := sources(src)

	if !*remoteOnly {
		readme, source, err = findLocalReadme(srcs)
		if err == nil {
			return readme, source, nil
		}
	}

	readme, source, err = findRemoteReadme(srcs, ver)
	if err != nil {
		return nil, "", errors.Wrap(err, "Did not find a readme locally nor in the remote repository")
	}

	return readme, source, nil
}

// findLocalReadme is a helper function for findReadme. It searches the README file
// locally in $GOPATH/src/<src> or $GOPATH/pkg/mod/<src>.
// If the path is absolute, this means it neither contains /src/ nor /pkg/mod/.
// In this case, findLocalReadme uses the full path.
func findLocalReadme(sources []string) (readme []byte, fp string, err error) {

	found := false
	var e error

allLoops:
	// We have to search across all gopath elements, across all README file names,
	// and also up the directory tree (in case the command is a subproject)
	for _, gp := range gopath() {
		for _, name := range names {
			for _, source := range sources {

				var rf *os.File
				found := false

				// Create the path to the README file and open the file.
				// If this fails, try the next GOPATH entry.
				for _, prefix := range []string{"src", filepath.Join("pkg", "mod")} {
					fp = filepath.Join(gp, prefix, source, name)
					rf, err := os.Open(fp)
					if err == nil {
						found = true
						break
					}
					e = errors.Wrap(err, "README not found at location "+fp)
					_ = rf.Close()
				}

				if !found {
					continue
				}

				// Allocate the slice for reading the file.
				fi, err := rf.Stat()
				if err != nil {
					return nil, "", errors.Wrapf(err, "Cannot determine file stats for %s", fp)
				}
				readme = make([]byte, fi.Size())

				// Read the whole file.
				n, err := rf.Read(readme)
				if err != nil {
					return readme[:n], "", errors.Wrap(err, "error reading from file "+fp)
				}
				_ = rf.Close()
				break allLoops
			}
		}
	}

	if !found {
		return nil, "", errors.Wrapf(e, "no README found for in any of %v\n", sources)
	}

	return readme, fp, err
}

// stripModVersion strips a version suffix from a path.
// Go get may cache repositories locally under $GOPATH/pkg/mod/, appending a
// version string to the repository path. Before reaching out to the remote repository,
// this version string must be stripped from the path.
func stripModVersion(path string) string {
	i := strings.Index(path, "@")
	if i > 0 {
		return path[:i]
	}
	return path
}

// findRemoteReadme is a helper function for findReadme. It attempts to locate the README in the remote repository at either of: -
// - http(s)://host.com/<user>/<project>/blob/main/<readme name>
// - http(s)://host.com/<user>/<project>/blob/main/cmd/<cmdname>/<readme name>
func findRemoteReadme(sources []string, ver string) (readme []byte, url string, err error) {

	var e error
	// TODO: implement resolveVanityImport
	// src, err = resolveVanityImports(src)
	// if err != nil {
	// 	return nil, "", errors.Wrap(err, "error resolving vanity import for "+src)
	// }

	for _, source := range sources {
		urls := possibleReadmeURLs(source, ver)
		for _, name := range names {
			for _, url := range urls {
				// TODO run all HTTP calls concurrently
				readme, e = httpGetReadme(url + name)
				if e == nil {
					return readme, url, nil
				}
				err = errors.Wrap(e, "") // collect all errors from the loop
			}
		}
	}

	return nil, "", errors.Wrap(e, "failed to retrieve README")
}

func httpGetReadme(url string) ([]byte, error) {
	var client = &http.Client{
		Timeout: time.Second * 10,
	}

	response, err := client.Get(url)
	if err != nil {
		return nil, errors.Wrap(err, "failed downloading the README from "+url)
	}

	if response.StatusCode != 200 {
		return nil, errors.New("HTTP GET returned " + response.Status + " for URL " + url)
	}

	r := bufio.NewReader(response.Body)
	readme, err := r.ReadBytes(0)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "error reading README from HTTP response")
	}

	return readme, nil
}

// source returns a slice containing `src` and all paths
// when walking the directory tree up to the root path.
//
// Example 1:
//
// src = github.com/user/project/.../subdir/v2
// Then the slice contains:
// github.com/user/project/.../subdir/v2
// github.com/user/project/.../subdir
// github.com/user/project
//
// Example 2:
// src = github.com/user/project/v2/.../subdir
// Then the slice contains:
// github.com/user/project/v2/.../subdir
// github.com/user/project/.../subdir
// github.com/user/project/v2
// github.com/user/project
func sources(src string) (srcs []string) {
	src = strings.Trim(src, "/")
	dirs := strings.Split(src, "/")

	versionRe := regexp.MustCompile(`(.*)(/v\d+)(/.*|$)`) // detect v2, v3,...
	isVersioned, preVersion, version, postVersion := func() (bool, string, string, string) {
		match := versionRe.FindStringSubmatch(src)
		if match != nil {
			// path with version info found. Remove version string.
			// TODO: instead of this workaround, find the readme at the proper git branch
			return true, match[1], match[2], match[3]
		}
		return false, "", "", ""
	}()

	lenProjPath := 3
	if isVersioned && (len(postVersion) > 0 || // github.com/org/repo/v2/subdir
		len(dirs) == 4) { // github.com/org/repo/v2 (no subdir)
		// version string occurs after project path but before subdir path (if any)
		lenProjPath = 4
	}
	hasSubdirs := len(dirs) > lenProjPath

	upper := 3
	if len(dirs) < 3 {
		upper = len(dirs)
	}
	pathNoSubdirsNoVersion := strings.Join(dirs[0:upper], "/")
	pathNoVersion := preVersion + postVersion

	// The sequence of possible README paths is crucial for fast retrieval
	// with as few HTTP requests as possible. The most likely locations should come first.

	switch {
	case !isVersioned && !hasSubdirs:
		srcs = append(srcs, src)
	case isVersioned && !hasSubdirs:
		srcs = append(srcs, src, pathNoVersion)
	case !isVersioned && hasSubdirs:
		srcs = append(srcs, src, pathNoSubdirsNoVersion)
	case isVersioned && hasSubdirs:
		srcs = append(srcs, src, pathNoVersion, pathNoSubdirsNoVersion+version, pathNoSubdirsNoVersion)
	}
	return srcs
}

// TODO:
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

// possibleReadmeURLs receives the relative path to the project and returns
// the URL to the raw README.md file (WITHOUT the file name itself, but
// WITH a trailing slash).
// Currently it knows how to do this for github.com and gitlab.com only.
// For all other sites, it returns `https://<src>/`.
//
// Examples:
//
// From github.com/ec1oud/mdcat to:
// https://github.com/ec1oud/mdcat/blob/<branch>/
//
// From gitlab.com/SporeDB/sporedb to:
// https://gitlab.com/SporeDB/sporedb/-/blob/<branch>/
//
// From git.sr.ht/~ghost08/photon to:
// https://git.sr.ht/~ghost08/photon/tree/<branch>/item/
func possibleReadmeURLs(src, ver string) []string {

	prefix := "https://"
	gh := "github.com/"
	gl := "gitlab.com/"
	sh := "https://git.sr.ht/"
	// TODO: add source hut - https://git.sr.ht/~<user>/<project>/tree/<branch>/item/README.md
	branches := []string{"main", "trunk", "master"}

	urls := []string{}

	src = strings.Trim(filepath.ToSlash(src), "/")

	for _, branch := range branches {

		// Process github paths
		if len(src) >= len(gh) && src[:len(gh)] == gh {
			urls = append(urls, fmt.Sprintf("%sraw.githubusercontent.com/%s/%s/", prefix, src[len(gl):], branch))
		}

		// Process gitlab paths
		if len(src) >= len(gl) && src[:len(gl)] == gl {
			// TODO -> raw URL includes the commit hash
			urls = append(urls, fmt.Sprintf("%s%s/-/raw/%s/", prefix, src, branch))
		}

		// Process sourcehut paths
		if len(src) >= len(sh) && src[:len(sh)] == sh {
			urls = append(urls, fmt.Sprintf("%s%s/blob/%s/", prefix, src, branch))
		}
	}

	urls = append(urls, fmt.Sprintf("%s%s/", prefix, src))
	return urls
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
