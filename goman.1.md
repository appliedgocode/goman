% GOMAN(1) Goman User Manual
% Christoph Berger
% 2024-12-14

# NAME

goman - find a Go binary's README and display it in the terminal like a man page

# SYNOSPIS

goman &lt;path to Go binary file>

goman &lt;path to Go binary file> | less -R


# DESCRIPTION

goman inspects the provided Go binary file to find the originating repository. It then searches the repository for a README file and displays its content in the terminal. 

# OPTIONS

-r
: Skip local search (as the local file may be outdated)
-v
: Verbose error output


# EXAMPLES

goman goman

goman hugo | less

goman docker | less -R

# LIMITATIONS

Some Go binaries are built with `ldflags` that shave some information from the binary, or are post-processed with `upx` to minimize their size. In such cases, goman may not be able to find the path to the repository.

# SEE ALSO

man(1)
