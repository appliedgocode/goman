package main

import (
	"bufio"
	"bytes"
	"fmt"
	"go/build"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/bfontaine/which/which"
	"github.com/ec1oud/blackfriday"
	"github.com/pkg/errors"
)

func main() {
	if len(os.Args) != 2 {
		usage()
		return
	}
	switch os.Args[1] {
	case "-h", "-help", "--help", "-?", "help":
		readme, err := findReadme("github.com/christophberger/goman")
		if err != nil {
			usage()
			return
		}
		fmt.Println(string(mdToAnsi(readme)))
		return
	}

	exec := os.Args[1]

	// Determine the location of `exec`
	path, err := getExecPath(exec)
	if err != nil {
		log.Println("Cannot determine the path of", exec)
		log.Fatalln(errors.WithStack(err))
	}

	// Extract the source path from the binary
	src, err := getMainPath(path)
	if err != nil {
		log.Println("Cannot determine source path of", path)
		log.Fatalln(errors.WithStack(err))
	}

	// Find the README
	readme, err := findReadme(src)
	if err != nil {
		log.Println("No README found for", exec, "at", src)
		log.Fatalln(errors.WithStack(err))
	}

	readme = mdToAnsi(readme)

	fmt.Println(string(readme))

}

func usage() {
	fmt.Println("Usage:")
	fmt.Println()
	fmt.Println("goman <name of Go binary>")
	fmt.Println()
	fmt.Println("goman is man for Go binaries. It attempts to fetch the README file of a Go binary's project and displays it in the terminal, if found.")
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
func findReadme(src string) (readme []byte, err error) {
	readme, err = findLocalReadme(src)
	if err == nil {
		return readme, nil
	}
	readme, err = findRemoteReadme(src)
	if err != nil {
		return nil, errors.Wrap(err, "Did not find a readme locally nor in the remote repository")
	}
	return readme, nil
}

// findLocalReadme is a helper function for findReadme. It searches the README file
// locally in $GOPATH/src/<src>.
func findLocalReadme(src string) (readme []byte, err error) {
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
			return nil, errors.Wrap(err, "Cannot determine file stats")
		}
		readme = make([]byte, fi.Size())

		// Read the whole file.
		n, err := rf.Read(readme)
		if err != nil {
			return readme[:n], errors.Wrap(err, "Error reading from file "+fp)
		}
		_ = rf.Close()
		found = true
		break
	}
	if !found {
		return nil, errors.Errorf("No README found for %s in any of %s\n", src, gopath())
	}
	return readme, err
}

// findRemoteReadme is a helper function for findReadme. It attempts to locate the README in the remote repository at either of: -
// - http(s)://host.com/<user>/<project>/blob/master/README.md
// - http(s)://host.com/<user>/<project>/blob/master/cmd/<cmdname>/README.md

func findRemoteReadme(src string) ([]byte, error) {
	prefix := "https://"
	blobmaster := "/blob/master/"
	uri := ""

	src = filepath.ToSlash(src)

	if strings.Contains(src, "/cmd/") {
		cidx := strings.Index(src, "/cmd/")

		// Insert /blob/master/ before /cmd/
		src = src[:cidx] + blobmaster + src[cidx:]

		uri = url.PathEscape(prefix + src + "/README.md")
		readme, err := httpGetReadme(uri)
		if err == nil {
			return readme, nil
		}
	}
	// This path is invoked if either no /cmd/ directory is in the URL,
	// or if no README was found in the /cmd/<cmdname>/ location.
	uri = url.PathEscape(prefix + src + blobmaster + "README.md")
	readme, err := httpGetReadme(uri)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to retrieve README")
	}

	return readme, nil
}

func httpGetReadme(url string) ([]byte, error) {
	var netClient = &http.Client{
		Timeout: time.Second * 10,
	}
	response, err := netClient.Get(url)
	if err != nil {
		return nil, errors.Wrap(err, "Failed downloading the README from "+url)
	}

	r := bufio.NewReader(response.Body)
	readme, err := r.ReadBytes(0)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "Error reading README from HTTP response")
	}
	if bytes.Compare(readme, []byte("Not Found")) == 0 {
		return nil, errors.New("README not found at " + url)
	}
	return readme, nil
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
	ansiFlags |= blackfriday.ANSI_USE_SMARTYPANTS
	ansiFlags |= blackfriday.ANSI_SMARTYPANTS_FRACTIONS
	ansiFlags |= blackfriday.ANSI_SMARTYPANTS_LATEX_DASHES

	renderer := blackfriday.AnsiRenderer(80, ansiFlags) // TODO get terminal width

	return blackfriday.Markdown(readme, renderer, extensions)
}
