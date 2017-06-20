package which

import "debug/elf"

type elftbl struct {
	*elf.File
	typ *PlatformType
}

func newelf(path string) (tabler, error) {
	f, err := elf.Open(path)
	if err != nil {
		return nil, err
	}
	tbl := elftbl{f, nil}
	switch [2]bool{tbl.FileHeader.Class == elf.ELFCLASS64, tbl.FileHeader.OSABI == elf.ELFOSABI_FREEBSD} {
	case [2]bool{false, false}:
		tbl.typ = PlatformLinux386
	case [2]bool{true, false}:
		tbl.typ = PlatformLinuxAMD64
	case [2]bool{false, true}:
		tbl.typ = PlatformFreeBSD386
	case [2]bool{true, true}:
		tbl.typ = PlatformFreeBSDAMD64
	}
	return tbl, nil
}

func (tbl elftbl) Close() error {
	return tbl.File.Close()
}

func (tbl elftbl) Pcln() ([]byte, error) {
	pcln := tbl.Section(".gopclntab")
	if pcln == nil {
		return nil, ErrNotGoExec
	}
	return pcln.Data()
}

func (tbl elftbl) Sym() ([]byte, error) {
	sym := tbl.Section(".gosymtab")
	if sym == nil {
		return nil, ErrNotGoExec
	}
	return sym.Data()
}

func (tbl elftbl) Text() (uint64, error) {
	text := tbl.Section(".text")
	if text == nil {
		return 0, ErrNotGoExec
	}
	return text.Addr, nil
}

func (tbl elftbl) Type() *PlatformType {
	return tbl.typ
}
