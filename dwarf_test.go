package main

import (
	"os"
	"testing"
)

func Test_getMainPath(t *testing.T) {
	type args struct {
		file string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{"ELF", args{"testdata/goman_linux"}, pwd(), false},
		{"Mach-O", args{"testdata/goman_macos"}, pwd(), false},
		{"PE", args{"testdata/goman.exe"}, pwd(), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getMainPath(tt.args.file)
			if (err != nil) != tt.wantErr {
				t.Errorf("getMainPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getMainPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Attempt to get the working dir, ignore any error intentionally
func pwd() string {
	wd, _ := os.Getwd()
	return wd
}