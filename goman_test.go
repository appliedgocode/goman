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
		want    []string
		wantErr bool
	}{
		{"github",
			"github.com/user/repo",
			[]string{
				"https://github.com/user/repo/blob/main/",
				"https://github.com/user/repo/blob/trunk/",
				"https://github.com/user/repo/blob/master/",
				"https://github.com/user/repo/",
			},
			false},
		{"githubcmd1",
			"github.com/user/repo/cmd/cmdname",
			[]string{
				"https://github.com/user/repo/cmd/cmdname/blob/main/",
				"https://github.com/user/repo/cmd/cmdname/blob/trunk/",
				"https://github.com/user/repo/cmd/cmdname/blob/master/",
				"https://github.com/user/repo/cmd/cmdname/",
			},
			false},
		{"gitlab",
			"gitlab.com/user/repo",
			[]string{
				"https://gitlab.com/user/repo/-/blob/main/",
				"https://gitlab.com/user/repo/-/blob/trunk/",
				"https://gitlab.com/user/repo/-/blob/master/",
				"https://gitlab.com/user/repo/",
			},
			false},
		// TODO: all of the aboce with v2 repos
		{"vanity",
			"npf.io/gorram",
			[]string{
				"https://npf.io/gorram/", // TODO
			},
			false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := possibleReadmeURLs(tt.src, "") // TODO - also test version strings
			if len(got) != len(tt.want) {
				t.Errorf("getRawReadmeURL() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("getRawReadmeURL() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

// TODO: find a reasonable test of findLocalReadme()

func Test_sources(t *testing.T) {
	type args struct {
		src string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{"base", args{"github.com/user/repo"}, []string{"github.com/user/repo"}},
		{"basev2", args{"github.com/user/repo/v2"},
			[]string{
				"github.com/user/repo/v2",
				"github.com/user/repo",
			},
		},
		{"cmd", args{"github.com/user/repo/cmd/name"},
			[]string{
				"github.com/user/repo/cmd/name",
				"github.com/user/repo",
			},
		},
		{"cmdv2", args{"github.com/user/repo/v2/cmd/name"},
			[]string{
				"github.com/user/repo/v2/cmd/name",
				"github.com/user/repo/cmd/name",
				"github.com/user/repo/v2",
				"github.com/user/repo",
			},
		},
		{"caddy", args{"github.com/mholt/caddy/caddy"},
			[]string{
				"github.com/mholt/caddy/caddy",
				"github.com/mholt/caddy",
			},
		},
		{"caddyv2", args{"github.com/mholt/caddy/caddy/v2"},
			[]string{
				"github.com/mholt/caddy/caddy/v2",
				"github.com/mholt/caddy/caddy",
				"github.com/mholt/caddy/v2",
				"github.com/mholt/caddy",
			},
		},
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
