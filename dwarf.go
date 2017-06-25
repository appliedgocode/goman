/*
This file is a copy of https://github.com/FiloSottile/gorebuild/blob/master/dwarf.go.
The file is (c) by Filippo Valsorda under the MIT license. See LICENSE.dwarf.go.txt.
*/

package main

import (
	"debug/elf"
	"debug/gosym"
	"debug/macho"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

func getTable(file string) (*gosym.Table, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}

	var textStart uint64
	var symtab, pclntab []byte

	obj, err := elf.NewFile(f)
	if err == nil {
		if sect := obj.Section(".text"); sect == nil {
			return nil, errors.New("empty .text")
		} else {
			textStart = sect.Addr
		}
		if sect := obj.Section(".gosymtab"); sect != nil {
			if symtab, err = sect.Data(); err != nil {
				return nil, err
			}
		}
		if sect := obj.Section(".gopclntab"); sect != nil {
			if pclntab, err = sect.Data(); err != nil {
				return nil, err
			}
		} else {
			return nil, errors.New("empty .gopclntab")
		}

	} else {
		obj, err := macho.NewFile(f)
		if err != nil {
			return nil, err
		}

		if sect := obj.Section("__text"); sect == nil {
			return nil, errors.New("empty __text")
		} else {
			textStart = sect.Addr
		}
		if sect := obj.Section("__gosymtab"); sect != nil {
			if symtab, err = sect.Data(); err != nil {
				return nil, err
			}
		}
		if sect := obj.Section("__gopclntab"); sect != nil {
			if pclntab, err = sect.Data(); err != nil {
				return nil, err
			}
		} else {
			return nil, errors.New("empty __gopclntab")
		}
	}

	pcln := gosym.NewLineTable(pclntab, textStart)
	return gosym.NewTable(symtab, pcln)
}

func getMainPath(file string) (string, error) {
	table, err := getTable(file)
	if err != nil {
		return "", errors.Wrap(err, "Main path not found")
	}
	path, _, _ := table.PCToLine(table.LookupFunc("main.main").Entry)
	return stripPath(filepath.Dir(path))
}

// strip path strips the GOPATH prefix from the raw source code path
// as returned by getMainPath.
// If the path is absolute, stripPath assumes a GOPATH prefix and
// searches for the first occurrence of "/src/". It returns the part
// after "/src/".
// If the path is relative, stripPath does not touch the path at all.
func stripPath(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return path, nil
	}
	n := strings.Index(path, "/src/")
	if n == -1 {
		return "", errors.New("Path is absolute but contains no '/src/' dir: " + path)
	}
	return path[n+5:], nil
}
