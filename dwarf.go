/*
This file is based on a copy of https://github.com/FiloSottile/gorebuild/blob/master/dwarf.go.
The file is (c) by Filippo Valsorda under the MIT license. See LICENSE.dwarf.go.txt.
*/

package main

import (
	"debug/elf"
	"debug/gosym"
	"debug/macho"
	"debug/pe"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

func getTableElf(f *os.File) (textStart uint64, symtab, pclntab []byte, err error) {
	obj, err := elf.NewFile(f)
	if err != nil {
		return 0, nil, nil, errors.Wrap(err, "file is not an ELF binary")
	}
	if sect := obj.Section(".text"); sect == nil {
		return 0, nil, nil, errors.New("empty .text")
	} else {
		textStart = sect.Addr
	}
	if sect := obj.Section(".gosymtab"); sect != nil {
		if symtab, err = sect.Data(); err != nil {
			return 0, nil, nil, errors.Wrap(err, "error reading .gosymtab")
		}
	}
	if sect := obj.Section(".gopclntab"); sect != nil {
		if pclntab, err = sect.Data(); err != nil {
			return 0, nil, nil, errors.Wrap(err, "error reading .gopclntab")
		}
	} else {
		return 0, nil, nil, errors.New("empty .gopclntab")
	}

	return textStart, symtab, pclntab, nil
}

func getTableMachO(f *os.File) (textStart uint64, symtab, pclntab []byte, err error) {
	obj, err := macho.NewFile(f)
	if err != nil {
		return 0, nil, nil, errors.Wrap(err, "file has neither elf nor macho format")
	}

	if sect := obj.Section("__text"); sect == nil {
		return 0, nil, nil, errors.New("empty __text")
	} else {
		textStart = sect.Addr
	}
	if sect := obj.Section("__gosymtab"); sect != nil {
		if symtab, err = sect.Data(); err != nil {
			return 0, nil, nil, errors.Wrap(err, "error reading __gosymtab")
		}
	}
	if sect := obj.Section("__gopclntab"); sect != nil {
		if pclntab, err = sect.Data(); err != nil {
			return 0, nil, nil, errors.Wrap(err, "error reading __gopclntab")
		}
	} else {
		return 0, nil, nil, errors.New("empty __gopclntab")
	}
	return textStart, symtab, pclntab, nil

}

// Borrowed from https://golang.org/src/cmd/internal/objfile/pe.go
// With hat tip to the Delve authors https://github.com/derekparker/delve/blob/master/pkg/proc/bininfo.go#L427
func getTablePe(f *os.File) (textStart uint64, symtab, pclntab []byte, err error) {

	obj, err := pe.NewFile(f)
	if err != nil {
		return 0, nil, nil, errors.Wrap(err, "file is not a PE binary")
	}
	var imageBase uint64
	switch oh := obj.OptionalHeader.(type) {
	case *pe.OptionalHeader32:
		imageBase = uint64(oh.ImageBase)
	case *pe.OptionalHeader64:
		imageBase = oh.ImageBase
	default:
		return 0, nil, nil, fmt.Errorf("pe file format not recognized")
	}
	if sect := obj.Section(".text"); sect != nil {
		textStart = imageBase + uint64(sect.VirtualAddress)
	}
	if pclntab, err = loadPETable(obj, "runtime.pclntab", "runtime.epclntab"); err != nil {
		// We didn't find the symbols, so look for the names used in 1.3 and earlier.
		// TODO: Remove code looking for the old symbols when we no longer care about 1.3.
		if pclntab, err = loadPETable(obj, "pclntab", "epclntab"); err != nil {
			return 0, nil, nil, errors.Wrap(err, "(e)pclntab not found")
		}
	}
	if symtab, err = loadPETable(obj, "runtime.symtab", "runtime.esymtab"); err != nil {
		// Same as above.
		if symtab, err = loadPETable(obj, "symtab", "esymtab"); err != nil {
			return 0, nil, nil, errors.Wrap(err, "(e)symtab not found")
		}
	}
	return textStart, symtab, pclntab, nil
}

func findPESymbol(f *pe.File, name string) (*pe.Symbol, error) {
	for _, s := range f.Symbols {
		if s.Name != name {
			continue
		}
		if s.SectionNumber <= 0 {
			return nil, fmt.Errorf("symbol %s: invalid section number %d", name, s.SectionNumber)
		}
		if len(f.Sections) < int(s.SectionNumber) {
			return nil, fmt.Errorf("symbol %s: section number %d is larger than max %d", name, s.SectionNumber, len(f.Sections))
		}
		return s, nil
	}
	return nil, fmt.Errorf("no %s symbol found", name)
}

func loadPETable(f *pe.File, sname, ename string) ([]byte, error) {
	ssym, err := findPESymbol(f, sname)
	if err != nil {
		return nil, err
	}
	esym, err := findPESymbol(f, ename)
	if err != nil {
		return nil, err
	}
	if ssym.SectionNumber != esym.SectionNumber {
		return nil, fmt.Errorf("%s and %s symbols must be in the same section", sname, ename)
	}
	sect := f.Sections[ssym.SectionNumber-1]
	data, err := sect.Data()
	if err != nil {
		return nil, err
	}
	return data[ssym.Value:esym.Value], nil
}

func getTable(file string) (*gosym.Table, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, errors.Wrap(err, "Error opening file "+file)
	}

	textStart, symtab, pclntab, err := getTableElf(f)
	if err != nil {
		if textStart, symtab, pclntab, err = getTableMachO(f); err != nil {
			if textStart, symtab, pclntab, err = getTablePe(f); err != nil {
				return nil, errors.Wrap(err, "file format is neither of ELF, Mach-O, or PE")
			}
		}
	}

	pcln := gosym.NewLineTable(pclntab, textStart)
	t, err := gosym.NewTable(symtab, pcln)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create symbol table")
	}
	return t, nil
}

func getMainPathDwarf(file string) (string, string, error) {
	table, err := getTable(file)
	if err != nil {
		return "", "", errors.Wrap(err, "main path not found (getTable)")
	}
	gosymFunc := table.LookupFunc("main.main")
	if gosymFunc == nil {
		return "", "", errors.Wrap(err, "main path not found (LookupFunc)")
	}
	path, _, _ := table.PCToLine(gosymFunc.Entry)
	return stripPath(filepath.Dir(path)), "", nil
}

// strip path strips the GOPATH prefix from the raw source code path
// as returned by getMainPath.
// If the path is absolute, stripPath first assumes a GOPATH prefix and
// searches for the first occurrence of "/src/". It returns the part
// after "/src/".
// If the absolute path contains no "/src/", stripPath searches for "/pkg/mod/",
// which is where Go modules are stored.
// If the absolute path contains neither "/src/" nor "/pkg/mod/",
// stripPath returns the full path.
// If the path is relative, stripPath does not touch the path at all.
func stripPath(path string) string {
	path = filepath.ToSlash(path)
	if !filepath.IsAbs(path) {
		return path
	}
	n := strings.Index(path, "/src/")
	if n != -1 {
		return path[n+5:]
	}
	n = strings.Index(path, "/pkg/mod/")
	if n != -1 {
		return path[n+9:]
	}
	return path
}
