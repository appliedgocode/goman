# goman - the missing man page for Go binaries

Almost all Go binaries come without any man page, even when properly installed through means like Homebrew.

`goman` substitutes the missing man page by the README file from the Go binary's sources.

`goman` first grabs the source path from the binary. Then it tries to locate the README file locally via the GOPATH. If this fails, it tries to fetch the README file from the binary's public repository. 

For that last option, `goman` makes a couple of assumptions about the location, but at least with github and gitlab, those assumptions should be valid. 


## Usage

    goman <go binary file>


## Installation 

    go get -u github.com/christophberger/goman


## License

See `LICENSE*.txt`.


## See also

[mdcat](https://github.com/ec1oud/mdcat) - a `cat` tool for Markdown

[mandown](https://github.com/driusan/mandown) - write *real* man pages in Markdown

