//go:build unit

/*
 * @license
 * Copyright 2026 Dynatrace LLC
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package files

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRejectSymlinks(t *testing.T) {

	t.Run("returns no error for nonexistent file on OsFs", func(t *testing.T) {
		fs := afero.NewOsFs()
		err := RejectSymlinkRecursive(fs, filepath.Join(resolvedTempDir(t), "nonexistent.yaml"))
		assert.NoError(t, err)
	})

	t.Run("returns no error for regular file on OsFs", func(t *testing.T) {
		dir := resolvedTempDir(t)
		filePath := filepath.Join(dir, "regular.yaml")
		require.NoError(t, os.WriteFile(filePath, []byte("content"), 0644))

		fs := afero.NewOsFs()
		err := RejectSymlinkRecursive(fs, filePath)
		assert.NoError(t, err)
	})

	t.Run("returns no error for directory on OsFs", func(t *testing.T) {
		fs := afero.NewOsFs()
		err := RejectSymlinkRecursive(fs, resolvedTempDir(t))
		assert.NoError(t, err)
	})

	t.Run("rejects symlink on OsFs", func(t *testing.T) {
		dir := resolvedTempDir(t)
		target := filepath.Join(dir, "target.yaml")
		link := filepath.Join(dir, "link.yaml")

		// link.yaml -> target.yaml
		require.NoError(t, os.WriteFile(target, []byte("secret"), 0644))
		require.NoError(t, os.Symlink(target, link))

		fs := afero.NewOsFs()
		err := RejectSymlinkRecursive(fs, link)
		assert.ErrorAs(t, err, new(*symlinkDetectedError))
	})

	t.Run("rejects symlink pointing outside project on BasePathFs", func(t *testing.T) {
		projectDir := resolvedTempDir(t)

		// project/loot -> /nonexistent/outside/host-secret.txt
		link := filepath.Join(projectDir, "loot")
		require.NoError(t, os.Symlink("/nonexistent/outside/host-secret.txt", link))

		fs := afero.NewBasePathFs(afero.NewOsFs(), projectDir)
		err := RejectSymlinkRecursive(fs, "loot")
		assert.ErrorAs(t, err, new(*symlinkDetectedError))
	})

	t.Run("rejects symlinked parent directory on BasePathFs", func(t *testing.T) {
		projectDir := resolvedTempDir(t)

		// project/secrets -> /nonexistent/outside
		require.NoError(t, os.Symlink("/nonexistent/outside", filepath.Join(projectDir, "secrets")))

		fs := afero.NewBasePathFs(afero.NewOsFs(), projectDir)
		err := RejectSymlinkRecursive(fs, "secrets/secret.txt")
		assert.ErrorAs(t, err, new(*symlinkDetectedError))
	})

	t.Run("rejects nested symlinked parent directory on BasePathFs", func(t *testing.T) {
		projectDir := resolvedTempDir(t)

		// project/a/b -> /nonexistent/outside
		require.NoError(t, os.MkdirAll(filepath.Join(projectDir, "a"), 0755))
		require.NoError(t, os.Symlink("/nonexistent/outside", filepath.Join(projectDir, "a", "b")))

		fs := afero.NewBasePathFs(afero.NewOsFs(), projectDir)
		err := RejectSymlinkRecursive(fs, "a/b/inner/secret.txt")
		assert.ErrorAs(t, err, new(*symlinkDetectedError))
	})

	t.Run("returns no error for regular file on BasePathFs", func(t *testing.T) {
		dir := resolvedTempDir(t)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "file.yaml"), []byte("ok"), 0644))

		fs := afero.NewBasePathFs(afero.NewOsFs(), dir)
		err := RejectSymlinkRecursive(fs, "file.yaml")
		assert.NoError(t, err)
	})

	t.Run("rejects symlink through ReadOnlyFs wrapper", func(t *testing.T) {
		dir := resolvedTempDir(t)
		target := filepath.Join(dir, "target.yaml")
		link := filepath.Join(dir, "link.yaml")

		// link.yaml -> target.yaml
		require.NoError(t, os.WriteFile(target, []byte("data"), 0644))
		require.NoError(t, os.Symlink(target, link))

		fs := afero.NewReadOnlyFs(afero.NewOsFs())
		err := RejectSymlinkRecursive(fs, link)
		assert.ErrorAs(t, err, new(*symlinkDetectedError))
	})

	t.Run("returns no error for regular file through ReadOnlyFs wrapper", func(t *testing.T) {
		dir := resolvedTempDir(t)
		filePath := filepath.Join(dir, "file.yaml")
		require.NoError(t, os.WriteFile(filePath, []byte("ok"), 0644))

		fs := afero.NewReadOnlyFs(afero.NewOsFs())
		err := RejectSymlinkRecursive(fs, filePath)
		assert.NoError(t, err)
	})

	t.Run("returns no error if fs does not support Lstater", func(t *testing.T) {
		dir := resolvedTempDir(t)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "file.yaml"), []byte("data"), 0644))

		// afero.RegexpFs does not implement afero.Lstater
		fs := afero.NewRegexpFs(afero.NewOsFs(), nil)
		err := RejectSymlinkRecursive(fs, filepath.Join(dir, "file.yaml"))
		assert.NoError(t, err)
	})

	t.Run("returns error if Lstat fails", func(t *testing.T) {
		dir := resolvedTempDir(t)
		filePath := filepath.Join(dir, "file.yaml")
		require.NoError(t, os.WriteFile(filePath, []byte("data"), 0644))

		fs := errLstatFs{Fs: afero.NewOsFs()}
		err := RejectSymlinkRecursive(fs, filePath)
		assert.ErrorIs(t, err, anError)
	})
}

// resolvedTempDir returns t.TempDir() with all symlinks resolved.
// Necessary because on macOS t.TempDir() lives under /var which is itself a symlink
// (/var -> /private/var), and RejectSymlinkRecursive walks every parent component.
func resolvedTempDir(t *testing.T) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	return resolved
}

// errLstatFs wraps an afero.Fs whose LstatIfPossible always returns an error.
type errLstatFs struct{ afero.Fs }

func (f errLstatFs) LstatIfPossible(name string) (os.FileInfo, bool, error) {
	return nil, true, anError
}

var anError = errors.New("an error")

func TestRejectSymlinks_EdgeCases(t *testing.T) {

	t.Run("rejects deeply nested symlinked parent (3 levels)", func(t *testing.T) {
		projectDir := resolvedTempDir(t)

		// project/a/b/c -> /nonexistent/outside
		require.NoError(t, os.MkdirAll(filepath.Join(projectDir, "a", "b"), 0755))
		require.NoError(t, os.Symlink("/nonexistent/outside", filepath.Join(projectDir, "a", "b", "c")))

		fs := afero.NewBasePathFs(afero.NewOsFs(), projectDir)
		err := RejectSymlinkRecursive(fs, "a/b/c/secret.txt")
		var symErr *symlinkDetectedError
		require.ErrorAs(t, err, &symErr)
		assert.Equal(t, filepath.FromSlash("a/b/c"), symErr.path)
	})

	t.Run("rejects symlinked intermediate directory (not first parent)", func(t *testing.T) {
		projectDir := resolvedTempDir(t)

		// project/configs/leak -> /nonexistent/outside
		require.NoError(t, os.MkdirAll(filepath.Join(projectDir, "configs"), 0755))
		require.NoError(t, os.Symlink("/nonexistent/outside", filepath.Join(projectDir, "configs", "leak")))

		fs := afero.NewBasePathFs(afero.NewOsFs(), projectDir)
		err := RejectSymlinkRecursive(fs, "configs/leak/sub/x.json")
		var symErr *symlinkDetectedError
		require.ErrorAs(t, err, &symErr)
		assert.Equal(t, filepath.FromSlash("configs/leak"), symErr.path)
	})

	t.Run("rejects path with .. that resolves through symlinked parent", func(t *testing.T) {
		projectDir := resolvedTempDir(t)

		// project/link -> /nonexistent/outside
		require.NoError(t, os.Symlink("/nonexistent/outside", filepath.Join(projectDir, "link")))
		require.NoError(t, os.MkdirAll(filepath.Join(projectDir, "real"), 0755))

		fs := afero.NewBasePathFs(afero.NewOsFs(), projectDir)
		// real/../link/secret.txt cleans to link/secret.txt
		err := RejectSymlinkRecursive(fs, "real/../link/secret.txt")
		require.ErrorAs(t, err, new(*symlinkDetectedError))
	})

	t.Run("handles trailing slash on directory path", func(t *testing.T) {
		projectDir := resolvedTempDir(t)

		// project/link -> /nonexistent/outside
		require.NoError(t, os.Symlink("/nonexistent/outside", filepath.Join(projectDir, "link")))

		fs := afero.NewBasePathFs(afero.NewOsFs(), projectDir)
		err := RejectSymlinkRecursive(fs, "link/")
		assert.ErrorAs(t, err, new(*symlinkDetectedError))
	})

	t.Run("rejects broken symlink (target does not exist)", func(t *testing.T) {
		projectDir := resolvedTempDir(t)

		// project/dangling -> project/missing-target (does not exist)
		require.NoError(t, os.Symlink(filepath.Join(projectDir, "missing-target"), filepath.Join(projectDir, "dangling")))

		fs := afero.NewBasePathFs(afero.NewOsFs(), projectDir)
		err := RejectSymlinkRecursive(fs, "dangling")
		assert.ErrorAs(t, err, new(*symlinkDetectedError))
	})

	t.Run("rejects chained symlinks (symlink -> symlink -> file)", func(t *testing.T) {
		projectDir := resolvedTempDir(t)

		// project/chain -> project/hop -> /nonexistent/secret.txt
		hop := filepath.Join(projectDir, "hop")
		require.NoError(t, os.Symlink("/nonexistent/secret.txt", hop))
		require.NoError(t, os.Symlink(hop, filepath.Join(projectDir, "chain")))

		fs := afero.NewBasePathFs(afero.NewOsFs(), projectDir)
		err := RejectSymlinkRecursive(fs, "chain")
		assert.ErrorAs(t, err, new(*symlinkDetectedError))
	})

	t.Run("rejects symlink with relative target escaping project", func(t *testing.T) {
		baseDir := resolvedTempDir(t)
		projectDir := filepath.Join(baseDir, "project")
		require.NoError(t, os.MkdirAll(projectDir, 0755))

		// project/escape -> ../outside  (relative, resolves to baseDir/outside)
		require.NoError(t, os.Symlink("../outside", filepath.Join(projectDir, "escape")))

		fs := afero.NewBasePathFs(afero.NewOsFs(), projectDir)
		err := RejectSymlinkRecursive(fs, "escape/secret.txt")
		var symErr *symlinkDetectedError
		require.ErrorAs(t, err, &symErr)
		assert.Equal(t, "escape", symErr.path)
	})

	t.Run("rejects symlink that points back into the project", func(t *testing.T) {
		// Even though the target is inside the project, we still reject it because
		// we cannot easily distinguish a benign in-project symlink from an attacker
		// crafting a symlink that resolves to an unrelated path via canonicalization tricks.
		projectDir := resolvedTempDir(t)
		require.NoError(t, os.MkdirAll(filepath.Join(projectDir, "real"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(projectDir, "real", "f.json"), []byte("ok"), 0644))

		// project/alias -> project/real
		require.NoError(t, os.Symlink(filepath.Join(projectDir, "real"), filepath.Join(projectDir, "alias")))

		fs := afero.NewBasePathFs(afero.NewOsFs(), projectDir)
		err := RejectSymlinkRecursive(fs, "alias/f.json")
		assert.ErrorAs(t, err, new(*symlinkDetectedError))
	})

	t.Run("allows path with .. that stays inside project and crosses no symlinks", func(t *testing.T) {
		projectDir := resolvedTempDir(t)
		require.NoError(t, os.MkdirAll(filepath.Join(projectDir, "a", "b"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(projectDir, "a", "x.json"), []byte("ok"), 0644))

		fs := afero.NewBasePathFs(afero.NewOsFs(), projectDir)
		err := RejectSymlinkRecursive(fs, "a/b/../x.json")
		assert.NoError(t, err)
	})

	t.Run("allows path with redundant separators and dot segments", func(t *testing.T) {
		projectDir := resolvedTempDir(t)
		require.NoError(t, os.MkdirAll(filepath.Join(projectDir, "a"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(projectDir, "a", "x.json"), []byte("ok"), 0644))

		fs := afero.NewBasePathFs(afero.NewOsFs(), projectDir)
		err := RejectSymlinkRecursive(fs, "./a//x.json")
		assert.NoError(t, err)
	})

	t.Run("allows nested existing directories with no symlinks", func(t *testing.T) {
		projectDir := resolvedTempDir(t)
		require.NoError(t, os.MkdirAll(filepath.Join(projectDir, "a", "b", "c"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(projectDir, "a", "b", "c", "x.json"), []byte("ok"), 0644))

		fs := afero.NewBasePathFs(afero.NewOsFs(), projectDir)
		err := RejectSymlinkRecursive(fs, "a/b/c/x.json")
		assert.NoError(t, err)
	})

	t.Run("allows existing parents with nonexistent leaf", func(t *testing.T) {
		projectDir := resolvedTempDir(t)
		require.NoError(t, os.MkdirAll(filepath.Join(projectDir, "a", "b"), 0755))

		fs := afero.NewBasePathFs(afero.NewOsFs(), projectDir)
		err := RejectSymlinkRecursive(fs, "a/b/nonexistent.json")
		assert.NoError(t, err)
	})

	t.Run("allows path with entirely nonexistent components", func(t *testing.T) {
		projectDir := resolvedTempDir(t)

		fs := afero.NewBasePathFs(afero.NewOsFs(), projectDir)
		err := RejectSymlinkRecursive(fs, "no/such/dir/file.json")
		assert.NoError(t, err)
	})

	t.Run("allows nonexistent directory path", func(t *testing.T) {
		// A nonexistent directory is equivalent to the nonexistent file case:
		// RejectSymlinkRecursive treats every path component uniformly and only
		// fails on components that actually exist on disk and are symlinks.
		projectDir := resolvedTempDir(t)

		fs := afero.NewBasePathFs(afero.NewOsFs(), projectDir)
		err := RejectSymlinkRecursive(fs, "no/such/directory")
		assert.NoError(t, err)
	})

	t.Run("allows nonexistent directory path with trailing slash", func(t *testing.T) {
		projectDir := resolvedTempDir(t)

		fs := afero.NewBasePathFs(afero.NewOsFs(), projectDir)
		err := RejectSymlinkRecursive(fs, "no/such/directory/")
		assert.NoError(t, err)
	})

	t.Run("rejects symlinked parent even when leaf does not exist", func(t *testing.T) {
		projectDir := resolvedTempDir(t)

		// project/link -> /nonexistent/outside
		require.NoError(t, os.Symlink("/nonexistent/outside", filepath.Join(projectDir, "link")))

		fs := afero.NewBasePathFs(afero.NewOsFs(), projectDir)
		err := RejectSymlinkRecursive(fs, "link/does-not-exist.json")
		assert.ErrorAs(t, err, new(*symlinkDetectedError))
	})

	t.Run("identifies the outermost symlink when multiple parents are symlinks", func(t *testing.T) {
		projectDir := resolvedTempDir(t)
		mid := resolvedTempDir(t)

		// project/outer -> mid, mid/inner -> /nonexistent/outside
		require.NoError(t, os.Symlink("/nonexistent/outside", filepath.Join(mid, "inner")))
		require.NoError(t, os.Symlink(mid, filepath.Join(projectDir, "outer")))

		fs := afero.NewBasePathFs(afero.NewOsFs(), projectDir)
		err := RejectSymlinkRecursive(fs, "outer/inner/s.txt")
		var symErr *symlinkDetectedError
		require.ErrorAs(t, err, &symErr)
		// the outermost symlink "outer" should be reported, not "outer/inner"
		assert.Equal(t, "outer", symErr.path)
	})

	t.Run("rejects symlinked path on raw OsFs with absolute path", func(t *testing.T) {
		projectDir := resolvedTempDir(t)

		// project/link -> /nonexistent/outside
		require.NoError(t, os.Symlink("/nonexistent/outside", filepath.Join(projectDir, "link")))

		fs := afero.NewOsFs()
		err := RejectSymlinkRecursive(fs, filepath.Join(projectDir, "link", "s.txt"))
		assert.ErrorAs(t, err, new(*symlinkDetectedError))
	})
}

func TestParentDirectories(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"/a/b/c", []string{"/a/b/c", "/a/b", "/a", "/"}},
		{"a/b/c", []string{"a/b/c", "a/b", "a"}},
		{"/file.txt", []string{"/file.txt", "/"}},
		{"file.txt", []string{"file.txt"}},
		{"/a/b/../c", []string{"/a/c", "/a", "/"}},
		{"a//b///c", []string{"a/b/c", "a/b", "a"}},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parentDirectories(filepath.FromSlash(tt.input))
			want := make([]string, len(tt.want))
			for i, w := range tt.want {
				want[i] = filepath.FromSlash(w)
			}
			assert.Equal(t, want, got)
		})
	}
}

func TestRejectSymlink(t *testing.T) {
	t.Run("returns nil for nonexistent component", func(t *testing.T) {
		fs := afero.NewOsFs()
		lstater := fs.(afero.Lstater)
		err := rejectSymlink(lstater, filepath.Join(resolvedTempDir(t), "nonexistent"))
		assert.NoError(t, err)
	})

	t.Run("returns nil for regular file", func(t *testing.T) {
		dir := resolvedTempDir(t)
		filePath := filepath.Join(dir, "file.txt")
		require.NoError(t, os.WriteFile(filePath, []byte("data"), 0644))
		fs := afero.NewOsFs()
		lstater := fs.(afero.Lstater)
		err := rejectSymlink(lstater, filePath)
		assert.NoError(t, err)
	})

	t.Run("returns error for symlink", func(t *testing.T) {
		projectDir := resolvedTempDir(t)
		target := filepath.Join(projectDir, "target")
		link := filepath.Join(projectDir, "link")

		// link -> target
		require.NoError(t, os.WriteFile(target, []byte("data"), 0644))
		require.NoError(t, os.Symlink(target, link))

		fs := afero.NewOsFs()
		lstater := fs.(afero.Lstater)
		err := rejectSymlink(lstater, link)
		require.Error(t, err)
		assert.ErrorAs(t, err, new(*symlinkDetectedError))
	})
}
