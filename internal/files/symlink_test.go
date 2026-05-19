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
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRejectSymlinks(t *testing.T) {

	t.Run("returns no error for nonexistent file on MemMapFs", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		err := RejectSymlink(fs, "/nonexistent/file.yaml")
		assert.NoError(t, err)
	})

	t.Run("returns no error for regular file on MemMapFs", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		require.NoError(t, afero.WriteFile(fs, "/file.yaml", []byte("data"), 0644))

		err := RejectSymlink(fs, "/file.yaml")
		assert.NoError(t, err)
	})

	t.Run("returns no error for nonexistent file on OsFs", func(t *testing.T) {
		fs := afero.NewOsFs()
		err := RejectSymlink(fs, filepath.Join(resolvedTempDir(t), "nonexistent.yaml"))
		assert.NoError(t, err)
	})

	t.Run("returns no error for regular file on OsFs", func(t *testing.T) {
		dir := resolvedTempDir(t)
		filePath := filepath.Join(dir, "regular.yaml")
		require.NoError(t, os.WriteFile(filePath, []byte("content"), 0644))

		fs := afero.NewOsFs()
		err := RejectSymlink(fs, filePath)
		assert.NoError(t, err)
	})

	t.Run("returns no error for directory on OsFs", func(t *testing.T) {
		fs := afero.NewOsFs()
		err := RejectSymlink(fs, resolvedTempDir(t))
		assert.NoError(t, err)
	})

	t.Run("rejects symlink on OsFs", func(t *testing.T) {
		dir := resolvedTempDir(t)
		target := filepath.Join(dir, "target.yaml")
		link := filepath.Join(dir, "link.yaml")

		require.NoError(t, os.WriteFile(target, []byte("secret"), 0644))
		require.NoError(t, os.Symlink(target, link))

		fs := afero.NewOsFs()
		err := RejectSymlink(fs, link)
		assert.ErrorContains(t, err, "symbolic link")
	})

	t.Run("rejects symlink pointing outside project on BasePathFs", func(t *testing.T) {
		projectDir := resolvedTempDir(t)
		outsideDir := resolvedTempDir(t)

		secretPath := filepath.Join(outsideDir, "host-secret.txt")
		require.NoError(t, os.WriteFile(secretPath, []byte("TOP-SECRET"), 0644))

		link := filepath.Join(projectDir, "loot")
		require.NoError(t, os.Symlink(secretPath, link))

		fs := afero.NewBasePathFs(afero.NewOsFs(), projectDir)
		err := RejectSymlink(fs, "loot")
		assert.ErrorContains(t, err, "symbolic link")
	})

	t.Run("rejects symlinked parent directory on BasePathFs", func(t *testing.T) {
		projectDir := resolvedTempDir(t)
		outsideDir := resolvedTempDir(t)

		secretPath := filepath.Join(outsideDir, "secret.txt")
		require.NoError(t, os.WriteFile(secretPath, []byte("TOP-SECRET"), 0644))

		// project/secrets -> outsideDir
		require.NoError(t, os.Symlink(outsideDir, filepath.Join(projectDir, "secrets")))

		fs := afero.NewBasePathFs(afero.NewOsFs(), projectDir)
		err := RejectSymlink(fs, "secrets/secret.txt")
		assert.ErrorContains(t, err, "symbolic link")
	})

	t.Run("rejects nested symlinked parent directory on BasePathFs", func(t *testing.T) {
		projectDir := resolvedTempDir(t)
		outsideDir := resolvedTempDir(t)

		nested := filepath.Join(outsideDir, "inner")
		require.NoError(t, os.MkdirAll(nested, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(nested, "secret.txt"), []byte("X"), 0644))

		require.NoError(t, os.MkdirAll(filepath.Join(projectDir, "a"), 0755))
		// project/a/b -> outsideDir
		require.NoError(t, os.Symlink(outsideDir, filepath.Join(projectDir, "a", "b")))

		fs := afero.NewBasePathFs(afero.NewOsFs(), projectDir)
		err := RejectSymlink(fs, "a/b/inner/secret.txt")
		assert.ErrorContains(t, err, "symbolic link")
	})

	t.Run("returns no error for regular file on BasePathFs", func(t *testing.T) {
		dir := resolvedTempDir(t)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "file.yaml"), []byte("ok"), 0644))

		fs := afero.NewBasePathFs(afero.NewOsFs(), dir)
		err := RejectSymlink(fs, "file.yaml")
		assert.NoError(t, err)
	})

	t.Run("rejects symlink through ReadOnlyFs wrapper", func(t *testing.T) {
		dir := resolvedTempDir(t)
		target := filepath.Join(dir, "target.yaml")
		link := filepath.Join(dir, "link.yaml")

		require.NoError(t, os.WriteFile(target, []byte("data"), 0644))
		require.NoError(t, os.Symlink(target, link))

		fs := afero.NewReadOnlyFs(afero.NewOsFs())
		err := RejectSymlink(fs, link)
		assert.ErrorContains(t, err, "symbolic link")
	})

	t.Run("returns no error for regular file through ReadOnlyFs wrapper", func(t *testing.T) {
		dir := resolvedTempDir(t)
		filePath := filepath.Join(dir, "file.yaml")
		require.NoError(t, os.WriteFile(filePath, []byte("ok"), 0644))

		fs := afero.NewReadOnlyFs(afero.NewOsFs())
		err := RejectSymlink(fs, filePath)
		assert.NoError(t, err)
	})

	t.Run("returns no error if fs does not support Lstater", func(t *testing.T) {
		inner := afero.NewMemMapFs()
		require.NoError(t, afero.WriteFile(inner, "file.yaml", []byte("data"), 0644))

		// afero.RegexpFs does not implement afero.Lstater
		fs := afero.NewRegexpFs(inner, nil)
		err := RejectSymlink(fs, "file.yaml")
		assert.NoError(t, err)
	})

	t.Run("returns error if Lstat fails", func(t *testing.T) {
		inner := afero.NewMemMapFs()
		require.NoError(t, afero.WriteFile(inner, "file.yaml", []byte("data"), 0644))

		fs := errLstatFs{Fs: inner}
		err := RejectSymlink(fs, "file.yaml")
		assert.ErrorContains(t, err, "could not check file")
	})
}

// resolvedTempDir returns t.TempDir() with all symlinks resolved.
// Necessary because on macOS t.TempDir() lives under /var which is itself a symlink
// (/var -> /private/var), and RejectSymlink walks every parent component.
func resolvedTempDir(t *testing.T) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	return resolved
}

// errLstatFs wraps an afero.Fs whose LstatIfPossible always returns an error.
type errLstatFs struct{ afero.Fs }

func (f errLstatFs) LstatIfPossible(name string) (os.FileInfo, bool, error) {
	return nil, true, fmt.Errorf("simulated lstat error")
}
