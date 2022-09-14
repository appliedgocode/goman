# goman - the missing man page for Go binaries

> Note: Find the latest version here: [appliedgocode/goman](https://github.com/appliedgocode/goman). 

![goman logo](goman.png)

Almost all Go binaries come without any man page, even when properly installed through means like Homebrew.

`goman` substitutes the missing man page by the README file from the Go binary's sources.

`goman` first grabs the source path from the binary. Then it tries to locate the README file locally via the GOPATH. If this fails, it tries to fetch the README file from the binary's public repository. 

For that last option, `goman` makes a couple of assumptions about the location, but at least with github and gitlab, those assumptions should be valid.

- - -

**Featured on [episode #51](https://changelog.com/gotime/51) of the Go Time podcast! 
(In the "Free Software Friday" section, at 1:02:35)**

- - -

## Demo

![goman demo](goman.gif)


## Usage

    goman <go binary file>

    goman <go binary file> | less -R

(`-R` tells `less` to render ANSI color codes.)


## Installation 

### Binaries

#### Homebrew

```sh
brew tap appliedgo/tools
brew install goman
```

#### Manual download
On the release pages, open the latest release and download the binary that matches your OS and architecture.

### From the source

You need a Go toolchain installed.

    go install github.com/appliedgocode/goman@latest

This downloads and installs `goman` to `$(go env GOPATH)/bin` (or `$(go env GOBIN)` if set). 

## Shell Integration

`goman` can blend in with the standard `man` command. 

Bash example (to be placed in `~/.bashrc`):

```bash
man() { 
    if ! /usr/bin/man $1; then 
        goman $1 | less -R; 
    fi; 
}
```

Fish example:

```fish
function man
    /usr/bin/man $argv; or goman $argv[1] | less -R
end
```


## Credits and Licenses

All code that is written by myself is governed by a 3-clause BSD license, see [LICENSE.txt](https://github.com/christophberger/goman/blob/master/LICENSE.txt).

The `which` package is part of the [which command](https://github.com/bfontaine/which) that is licensed under the MIT license; see [LICENSE.which.txt](https://github.com/christophberger/goman/blob/master/LICENSE.which.txt).

The code that extracts the source code path from a go binary is a part of the [`gorebuild` tool](https://github.com/FiloSottile/gorebuild) that is published under the MIT license; See [LICENSE.dwarf.go.txt](https://github.com/christophberger/goman/blob/master/LICENSE.dwarf.go.txt).

The Markdown renderer is [a fork](https://github.com/ec1oud/blackfriday) of [blackfriday](https://github.com/russross/blackfriday) with extra code for rendering Markdown to plain text with ANSI color codes. See [LICENSE.blackfriday.txt](https://github.com/christophberger/goman/blob/master/LICENSE.blackfriday.txt) and the copyright notice in [ec1oud/blackfriday/ansi.go](https://github.com/christophberger/goman/blob/master/vendor/github.com/ec1oud/blackfriday/ansi.go).


## Limitations

In its current state, `goman` is little more than a proof of concept. Bugs certainly do exist, as well as functional shortcomings due to oversimplified design, such as:

* `goman` assumes that the README file contains either Markdown text or plain text. I know of at least one README.md that contains HTML. `goman` does not treat such cases in any special way.

* If a binary originates from a command subdirectory of a project, chances are that this subdirectory contains no extra README file. `goman` then tries to find the README file in the parent directories.

* Some binaries contain an absolute path to their source code, and `goman` assumes that the GOPATH used at compile time is the part from the root to the first directory named `/src/`. If the GOPATH itself contains a `/src/` directory (e.g., "export GOPATH=/home/user/src/go"), `goman` fails extracting the relative source code path.

* `goman`'s output may wrap character-wise instead of word-wise.

* Path redirection to canonical paths (like, e.g. from "https://npf.io/gorram" to https://github.com/natefinch/gorram) are not handled right now.


## See also

[mdcat](https://github.com/ec1oud/mdcat) - a `cat` tool for Markdown

[mandown](https://github.com/driusan/mandown) - write *real* man pages in Markdown

[mango](https://github.com/slyrz/mango) - generate man pages from your source code

[gorebuild](https://github.com/FiloSottile/gorebuild) - rebuild Go binaries from source

[binstale](https://github.com/shurcooL/binstale) - check if your go binaries are outdated

[bin](https://github.com/rjeczalik/bin) and gobin - update all your Go binaries


## Changelog

### v0.2.3

Large update to implement a simpler and more reliable method of getting the readme path for binaries built with Module support.

TODOs:
- Support vanity module paths
- Determine semver tag, in order to fetch the readme version that matches with the compiled version

### v0.2.2 (2021-04-23)

- Add goreleaser.yml
- Various fixes
### v0.2.1 (2017-07-04)

Fix stripping prefix from absolute path on Windows (PR #5)

### v0.2.0 (2017-07-03)

Add support for the PE file format (Windows). (Implements issue #3)

### v0.1.3 (2017-06-28)

Change search strategy for README file to cover all possible cases. (Fixes issue #2)

### v0.1.2 (2017-06-27)

Fix slice panic if URL path is shorter than "github.com" (issue #1)
