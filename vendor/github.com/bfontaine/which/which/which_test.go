package which

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/bfontaine/vanish/vanish"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func touch(t *testing.T, dir, name string, exe bool) {
	var mode os.FileMode

	if exe {
		mode = 0744
	} else {
		mode = 0644
	}

	require.Nil(t, ioutil.WriteFile(filepath.Join(dir, name), []byte("foo"), mode))
}

func testEnv(t *testing.T, fn func(d1, d2 string)) {
	vanish.Env(func() {
		vanish.Dir(func(dir1 string) {
			touch(t, dir1, "a-exe", true)
			touch(t, dir1, "b-exe", true)
			touch(t, dir1, "c", false)
			touch(t, dir1, "e", false)
			touch(t, dir1, "both", true)

			vanish.Dir(func(dir2 string) {
				touch(t, dir2, "a", false)
				touch(t, dir2, "b-exe", false)
				touch(t, dir2, "e", true)
				touch(t, dir2, "both", true)

				os.Setenv("PATH", fmt.Sprintf("%s:%s", dir1, dir2))

				fn(dir1, dir2)
			})
		})
	})
}

func TestOneExists(t *testing.T) {
	testEnv(t, func(d1, d2 string) {
		assert.Equal(t, d1+"/a-exe", One("a-exe"))
		assert.Equal(t, d2+"/e", One("e"))
	})
}

func TestOneDoesntExists(t *testing.T) {
	testEnv(t, func(d1, d2 string) {
		assert.Equal(t, "", One("c"))
		assert.Equal(t, "", One("z"))
	})
}

func TestAllOneExists(t *testing.T) {
	testEnv(t, func(d1, d2 string) {
		paths := All("a-exe")
		assert.Equal(t, 1, len(paths))

		assert.Equal(t, d1+"/a-exe", paths[0])
	})
}

func TestAllExist(t *testing.T) {
	testEnv(t, func(d1, d2 string) {
		paths := All("both")
		assert.Equal(t, 2, len(paths))

		assert.Equal(t, d1+"/both", paths[0])
		assert.Equal(t, d2+"/both", paths[1])
	})
}

func TestAllNoOneExists(t *testing.T) {
	testEnv(t, func(d1, d2 string) {
		paths := All("nope")
		assert.Equal(t, 0, len(paths))
	})
}

func TestOneWithEmptyPath(t *testing.T) {
	assert.Equal(t, "", OneWithPath("vim", ""))
}

func TestAllWithEmptyPath(t *testing.T) {
	assert.Equal(t, 0, len(AllWithPath("vim", "")))
}

func TestOneExistWithPath(t *testing.T) {
	testEnv(t, func(d1, d2 string) {
		path := OneWithPath("both", d2)
		assert.Equal(t, d2+"/both", path)
	})
}
