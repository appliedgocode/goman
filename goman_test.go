package main

import (
	"os/exec"
	"reflect"
	"strings"
	"testing"
)

func Test_getReadmeURL(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		want    string
		wantErr bool
	}{
		{"github", "github.com/user/repo", "https://raw.githubusercontent.com/user/repo/master/", false},
		{"githubcmd1", "github.com/user/repo/cmd/cmdname", "https://raw.githubusercontent.com/user/repo/cmd/cmdname/master/", false},
		{"gitlab", "gitlab.com/user/repo", "https://gitlab.com/user/repo/raw/master/", false},
		{"vanity", "npf.io/gorram", "https://npf.io/gorram/", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getReadmeURL(tt.src)
			if got != tt.want {
				t.Errorf("getRawReadmeURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

// This is a very cheap test that only checks if the function
// returns no error.
// Hence the tests for README content and resulting file path
// are inactive.

func Test_findLocalReadme(t *testing.T) {
	type args struct {
		src string
	}
	tests := []struct {
		name string
		args args
		//	wantReadme []byte
		//	wantFp     string
		wantErr bool
	}{
		{"./testdata/goman_macos", args{"github.com/appliedgocode/goman"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// gotReadme, gotFp, err := findLocalReadme(tt.args.src)
			_, _, err := findLocalReadme(tt.args.src)
			if (err != nil) != tt.wantErr {
				t.Errorf("findLocalReadme() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// if !reflect.DeepEqual(gotReadme, tt.wantReadme) {
			// 	t.Errorf("findLocalReadme() gotReadme = %v, want %v", gotReadme, tt.wantReadme)
			// }
			// if gotFp != tt.wantFp {
			// 	t.Errorf("findLocalReadme() gotFp = %v, want %v", gotFp, tt.wantFp)
			// }
		})
	}
}

func Test_sources(t *testing.T) {
	type args struct {
		src string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{"base", args{"github.com/user/repo"}, []string{"github.com/user/repo", "github.com/user", "github.com"}},
		{"cmd", args{"github.com/user/repo/cmd/name"}, []string{"github.com/user/repo/cmd/name", "github.com/user/repo/cmd", "github.com/user/repo", "github.com/user", "github.com"}},
		{"caddy", args{"github.com/mholt/caddy/caddy"}, []string{"github.com/mholt/caddy/caddy", "github.com/mholt/caddy", "github.com/mholt", "github.com"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sources(tt.args.src); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("sources() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_gopath(t *testing.T) {

	// Checking against an alternate method of getting the GOPATH
	cmd := exec.Command("go", "env", "GOPATH")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Skipf("gopath() - cannot read GOPATH via `go env GOPATH`: %s", err)
	}

	tests := []struct {
		name string
		want []string
	}{
		{"gopath", strings.Split(strings.Trim(string(out), " \r\n"), ";")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := gopath(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("gopath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_stripModVersion(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"withVersion", args{"path/with/version@v1.0.0"}, "path/with/version"},
		{"withoutVersion", args{"path/without/version"}, "path/without/version"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stripModVersion(tt.args.path); got != tt.want {
				t.Errorf("stripModVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}
