package main

import (
	"testing"
)

func Test_getRawReadmeURL(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		want    string
		wantErr bool
	}{
		{"github", "github.com/user/repo", "https://raw.githubusercontent.com/user/repo/master/", false},
		{"githubcmd1", "github.com/user/repo/cmd/cmdname", "https://raw.githubusercontent.com/user/repo/cmd/cmdname/master/", false},
		{"gitlab", "gitlab.com/user/repo", "https://gitlab.com/user/repo/raw/master/", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getRawReadmeURL(tt.src)
			if (err != nil) != tt.wantErr {
				t.Errorf("getRawReadmeURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
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
		{"goman", args{"github.com/christophberger/goman"}, false},
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
