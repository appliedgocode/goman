package which

import (
	"debug/pe"
	"fmt"
)

type petbl struct {
	*pe.File
	typ  *PlatformType
	base uint64
}

func newpe(path string) (tabler, error) {
	f, err := pe.Open(path)
	if err != nil {
		return nil, err
	}
	tbl := petbl{f, nil, 0}
	switch oh := tbl.OptionalHeader.(type) {
	case *pe.OptionalHeader32:
		tbl.base = uint64(oh.ImageBase)
		tbl.typ = PlatformWindows386
	case *pe.OptionalHeader64:
		tbl.base = oh.ImageBase
		tbl.typ = PlatformWindowsAMD64
	default:
		tbl.Close()
		return nil, ErrNotGoExec
	}
	return tbl, nil
}

func (tbl petbl) Close() error {
	return tbl.File.Close()
}

func (tbl petbl) Pcln() ([]byte, error) {
	return loadPETable(tbl.File, "pclntab", "epclntab")
}

func (tbl petbl) Sym() ([]byte, error) {
	return loadPETable(tbl.File, "symtab", "esymtab")
}

func (tbl petbl) Text() (uint64, error) {
	text := tbl.Section(".text")
	if text == nil {
		return 0, ErrNotGoExec
	}
	return tbl.base + uint64(text.VirtualAddress), nil
}

func (tbl petbl) Type() *PlatformType {
	return tbl.typ
}

// findPESymbol was stolen from $GOROOT/src/cmd/addr2line/main.go:181
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

// loadPETable was stolen from $GOROOT/src/cmd/addr2line/main.go:197
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
