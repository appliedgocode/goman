package main

import (
	"os"
	"testing"
)

func Test_getMainPathDwarf(t *testing.T) {
	type args struct {
		file string
	}
	tests := []struct {
		name string
		args args
		want struct {
			path, ver string
		}
		wantErr bool
	}{
		{"ELF", args{"testdata/goman_linux"}, struct{ path, ver string }{path: pwd(), ver: ""}, false},
		{"Mach-O", args{"testdata/goman_macos"}, struct{ path, ver string }{path: pwd(), ver: ""}, false},
		{"PE", args{"testdata/goman.exe"}, struct{ path, ver string }{path: pwd(), ver: ""}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, ver, err := getMainPathDwarf(tt.args.file)
			if (err != nil) != tt.wantErr {
				t.Errorf("getMainPathDwarf() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if path != tt.want.path {
				t.Errorf("getMainPathDwarf() path = %v, want %v", path, tt.want.path)
			}
			if ver != tt.want.ver {
				t.Errorf("getMainPathDwarf() ver = %v, want %v", ver, tt.want.ver)
			}
		})
	}
}

// Attempt to get the working dir, ignore any error intentionally
func pwd() string {
	wd, _ := os.Getwd()
	return wd
}
