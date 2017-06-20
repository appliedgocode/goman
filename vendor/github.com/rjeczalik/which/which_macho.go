package which

import "debug/macho"

type machotbl struct {
	*macho.File
	typ *PlatformType
}

func newmacho(path string) (tabler, error) {
	f, err := macho.Open(path)
	if err != nil {
		return nil, err
	}
	tbl := machotbl{f, nil}
	switch tbl.Cpu {
	case macho.Cpu386:
		tbl.typ = PlatformDarwin386
	case macho.CpuAmd64:
		tbl.typ = PlatformDarwinAMD64
	}
	return tbl, nil
}

func (tbl machotbl) Close() error {
	return tbl.File.Close()
}

func (tbl machotbl) Pcln() ([]byte, error) {
	pcln := tbl.Section("__gopclntab")
	if pcln == nil {
		return nil, ErrNotGoExec
	}
	return pcln.Data()
}

func (tbl machotbl) Sym() ([]byte, error) {
	sym := tbl.Section("__gosymtab")
	if sym == nil {
		return nil, ErrNotGoExec
	}
	return sym.Data()
}

func (tbl machotbl) Text() (uint64, error) {
	text := tbl.Section("__text")
	if text == nil {
		return 0, ErrNotGoExec
	}
	return text.Addr, nil
}

func (tbl machotbl) Type() *PlatformType {
	return tbl.typ
}
