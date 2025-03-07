package build

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/windmilleng/tilt/internal/testutils/tempdir"
	"github.com/windmilleng/tilt/pkg/model"
)

func TestFilesToPathMappings(t *testing.T) {
	f := tempdir.NewTempDirFixture(t)
	defer f.TearDown()

	paths := []string{
		"sync1/fileA",
		"sync1/child/fileB",
		"sync2/fileC",
	}
	f.TouchFiles(paths)

	absPaths := make([]string, len(paths))
	for i, p := range paths {
		absPaths[i] = f.JoinPath(p)
	}
	// Add a file that doesn't exist on local -- but we still expect it to successfully
	// map to a ContainerPath.
	absPaths = append(absPaths, filepath.Join(f.Path(), "sync2/file_deleted"))

	syncs := []model.Sync{
		model.Sync{
			LocalPath:     f.JoinPath("sync1"),
			ContainerPath: "/dest1",
		},
		model.Sync{
			LocalPath:     f.JoinPath("sync2"),
			ContainerPath: "/nested/dest2",
		},
	}
	actual, err := FilesToPathMappings(absPaths, syncs)
	if err != nil {
		f.T().Fatal(err)
	}

	expected := []PathMapping{
		PathMapping{
			LocalPath:     filepath.Join(f.Path(), "sync1/fileA"),
			ContainerPath: "/dest1/fileA",
		},
		PathMapping{
			LocalPath:     filepath.Join(f.Path(), "sync1/child/fileB"),
			ContainerPath: "/dest1/child/fileB",
		},
		PathMapping{
			LocalPath:     filepath.Join(f.Path(), "sync2/fileC"),
			ContainerPath: "/nested/dest2/fileC",
		},
		PathMapping{
			LocalPath:     filepath.Join(f.Path(), "sync2/file_deleted"),
			ContainerPath: "/nested/dest2/file_deleted",
		},
	}

	assert.ElementsMatch(t, expected, actual)
}

func TestFileToDirectoryPathMapping(t *testing.T) {
	f := tempdir.NewTempDirFixture(t)
	defer f.TearDown()

	paths := []string{
		"sync1/fileA",
	}
	f.TouchFiles(paths)

	absPaths := make([]string, len(paths))
	for i, p := range paths {
		absPaths[i] = f.JoinPath(p)
	}

	syncs := []model.Sync{
		model.Sync{
			LocalPath:     f.JoinPath("sync1", "fileA"),
			ContainerPath: "/dest1/",
		},
	}

	actual, err := FilesToPathMappings(absPaths, syncs)
	if err != nil {
		f.T().Fatal(err)
	}

	expected := []PathMapping{
		PathMapping{
			LocalPath:     filepath.Join(f.Path(), "sync1/fileA"),
			ContainerPath: "/dest1/fileA",
		},
	}

	assert.ElementsMatch(t, expected, actual)
}

func TestFileNotInSyncYieldsNoMapping(t *testing.T) {
	f := tempdir.NewTempDirFixture(t)
	defer f.TearDown()

	files := []string{f.JoinPath("not/synced/fileA")}

	syncs := []model.Sync{
		model.Sync{
			LocalPath:     f.JoinPath("sync1"),
			ContainerPath: "/dest1",
		},
	}

	actual, err := FilesToPathMappings(files, syncs)
	if err != nil {
		f.T().Fatal(err)
	}
	assert.Empty(t, actual, "expected no path mapping returned for a file not matching any syncs")
}
