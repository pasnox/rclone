package vfstest

import (
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/rclone/rclone/vfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFileModTime tests mod times on files
func TestFileModTime(t *testing.T) {
	run.skipIfNoFUSE(t)

	run.createFile(t, "file", "123")

	mtime := time.Date(2012, time.November, 18, 17, 32, 31, 0, time.UTC)
	err := run.os.Chtimes(run.path("file"), mtime, mtime)
	require.NoError(t, err)

	info, err := run.os.Stat(run.path("file"))
	require.NoError(t, err)

	// avoid errors because of timezone differences
	assert.Equal(t, info.ModTime().Unix(), mtime.Unix())

	run.rm(t, "file")
}

// run.os.Create without opening for write too
func osCreate(name string) (vfs.OsFiler, error) {
	return run.os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
}

// run.os.Create with append
func osAppend(name string) (vfs.OsFiler, error) {
	return run.os.OpenFile(name, os.O_WRONLY|os.O_APPEND, 0666)
}

// TestFileModTimeWithOpenWriters tests mod time on open files
func TestFileModTimeWithOpenWriters(t *testing.T) {
	run.skipIfNoFUSE(t)
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on Windows")
	}

	mtime := time.Date(2012, time.November, 18, 17, 32, 31, 0, time.UTC)
	filepath := run.path("cp-archive-test")

	f, err := osCreate(filepath)
	require.NoError(t, err)

	_, err = f.Write([]byte{104, 105})
	require.NoError(t, err)

	err = run.os.Chtimes(filepath, mtime, mtime)
	require.NoError(t, err)

	err = f.Close()
	require.NoError(t, err)

	run.waitForWriters()

	info, err := run.os.Stat(filepath)
	require.NoError(t, err)

	// avoid errors because of timezone differences
	assert.Equal(t, info.ModTime().Unix(), mtime.Unix())

	run.rm(t, "cp-archive-test")
}

func TestSymlinks(t *testing.T) {
	run.skipIfNoFUSE(t)

	// if runtime.GOOS == "windows" {
	// 	t.Skip("Skipping test on Windows")
	// }

	run.mkdir(t, "dir1")
	run.mkdir(t, "dir1/sub1dir1")
	run.createFile(t, "dir1/file1", "potato")

	run.mkdir(t, "dir2")
	run.mkdir(t, "dir2/sub1dir2")
	run.createFile(t, "dir2/file1", "chicken")

	run.checkDir(t, "dir1/|dir1/sub1dir1/|dir1/file1 6|dir2/|dir2/sub1dir2/|dir2/file1 7")

	// Link to a file
	run.relativeSymlink(t, "dir1/file1", "dir1file1_link")

	run.checkDir(t, "dir1/|dir1/sub1dir1/|dir1/file1 6|dir2/|dir2/sub1dir2/|dir2/file1 7|dir1file1_link.rclonelink 10")

	dir1file1_link, err := run.os.Stat(run.path("dir1file1_link"))
	require.NoError(t, err)

	assert.Equal(t, dir1file1_link.Name(), "dir1file1_link")
	assert.Equal(t, dir1file1_link.IsDir(), false)

	assert.Equal(t, run.readFile(t, "dir1file1_link"), "potato")

	err = writeFile(run.path("dir1file1_link"), []byte("carrot"), 0600)
	require.NoError(t, err)

	assert.Equal(t, run.readFile(t, "dir1file1_link"), "carrot")
	assert.Equal(t, run.readFile(t, "dir1/file1"), "carrot")

	err = run.os.Rename(run.path("dir1file1_link"), run.path("dir1file1_link")+"_bla")
	require.NoError(t, err)

	run.checkDir(t, "dir1/|dir1/sub1dir1/|dir1/file1 6|dir2/|dir2/sub1dir2/|dir2/file1 7|dir1file1_link_bla.rclonelink 10")

	assert.Equal(t, run.readlink(t, "dir1file1_link_bla"), "dir1/file1")

	run.rm(t, "dir1file1_link_bla")

	run.checkDir(t, "dir1/|dir1/sub1dir1/|dir1/file1 6|dir2/|dir2/sub1dir2/|dir2/file1 7")

	// Link to a dir
	run.relativeSymlink(t, "dir1", "dir1_link")

	run.checkDir(t, "dir1/|dir1/sub1dir1/|dir1/file1 6|dir2/|dir2/sub1dir2/|dir2/file1 7|dir1_link.rclonelink 4")

	dir1_link, err := run.os.Stat(run.path("dir1_link"))
	require.NoError(t, err)

	assert.Equal(t, dir1_link.Name(), "dir1_link")
	assert.Equal(t, dir1_link.IsDir(), true)

	_, err = run.os.OpenFile(run.path("dir1_link"), os.O_WRONLY, 0600)
	require.Error(t, err)

	dirLinksEntries := make(dirMap)
	run.readLocal(t, dirLinksEntries, "dir1_link")

	assert.Equal(t, len(dirLinksEntries), 2)

	dir1Entries := make(dirMap)
	run.readLocal(t, dir1Entries, "dir1")

	assert.Equal(t, len(dir1Entries), 2)

	run.rm(t, "dir1_link") // run.rmdir works fine as well

	run.checkDir(t, "dir1/|dir1/sub1dir1/|dir1/file1 6|dir2/|dir2/sub1dir2/|dir2/file1 7")
}
