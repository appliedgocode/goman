package which

import (
	"path/filepath"
	"testing"
)

const echo = "github.com/rjeczalik/which/testdata/cmd/echo"

var testdata = map[*PlatformType]string{
	PlatformDarwin386:    "testdata/darwin_386/echo",
	PlatformDarwinAMD64:  "testdata/darwin_amd64/echo",
	PlatformFreeBSD386:   "testdata/freebsd_386/echo",
	PlatformFreeBSDAMD64: "testdata/freebsd_amd64/echo",
	PlatformLinux386:     "testdata/linux_386/echo",
	PlatformLinuxAMD64:   "testdata/linux_amd64/echo",
	PlatformWindows386:   "testdata/windows_386/echo.exe",
	PlatformWindowsAMD64: "testdata/windows_amd64/echo.exe",
}

func TestNewExec(t *testing.T) {
	for typ, path := range testdata {
		ex, err := NewExec(filepath.FromSlash(path))
		if err != nil {
			t.Errorf("want err=nil; got %q (typ=%v)", err, typ)
			continue
		}
		if ex.Type != typ {
			t.Errorf("want ex.Type=%v; got %v", typ, ex.Type)
		}
	}
}

func TestImport(t *testing.T) {
	for typ, path := range testdata {
		ex, err := NewExec(filepath.FromSlash(path))
		if err != nil {
			t.Errorf("want err=nil; got %q (typ=%v)", err, typ)
			continue
		}
		imp, err := ex.Import()
		if err != nil {
			t.Errorf("want err=nil; got %q (typ=%v)", err, typ)
			continue
		}
		if imp != echo {
			t.Errorf("want imp=%q; got %q (typ=%v)", echo, imp, typ)
		}
	}
}

func TestType(t *testing.T) {
	for typ, path := range testdata {
		ex, err := NewExec(filepath.FromSlash(path))
		if err != nil {
			t.Errorf("want err=nil; got %q (typ=%v)", err, typ)
			continue
		}
		if ex.Type != typ {
			t.Errorf("want ex.Typ=%v; got %v (typ=%v)", typ, ex.Type, typ)
		}
	}
}
