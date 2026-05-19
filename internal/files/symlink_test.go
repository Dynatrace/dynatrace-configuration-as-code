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

func TestRejectSymlinks_EdgeCases(t *testing.T) {

	t.Run("rejects deeply nested symlinked parent (3 levels)", func(t *testing.T) {
		projectDir := resolvedTempDir(t)
		outsideDir := resolvedTempDir(t)

		require.NoError(t, os.MkdirAll(filepath.Join(projectDir, "a", "b"), 0755))
		// project/a/b/c -> outsideDir
		require.NoError(t, os.Symlink(outsideDir, filepath.Join(projectDir, "a", "b", "c")))
		require.NoError(t, os.WriteFile(filepath.Join(outsideDir, "secret.txt"), []byte("S"), 0644))

		fs := afero.NewBasePathFs(afero.NewOsFs(), projectDir)
		err := RejectSymlink(fs, "a/b/c/secret.txt")
		require.ErrorContains(t, err, "symbolic link")
		assert.Contains(t, err.Error(), "a/b/c")
	})

	t.Run("rejects symlinked intermediate directory (not first parent)", func(t *testing.T) {
		projectDir := resolvedTempDir(t)
		outsideDir := resolvedTempDir(t)

		require.NoError(t, os.MkdirAll(filepath.Join(projectDir, "configs"), 0755))
		// project/configs/leak -> outsideDir
		require.NoError(t, os.Symlink(outsideDir, filepath.Join(projectDir, "configs", "leak")))
		require.NoError(t, os.MkdirAll(filepath.Join(outsideDir, "sub"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(outsideDir, "sub", "x.json"), []byte("X"), 0644))

		fs := afero.NewBasePathFs(afero.NewOsFs(), projectDir)
		err := RejectSymlink(fs, "configs/leak/sub/x.json")
		require.ErrorContains(t, err, "symbolic link")
		assert.Contains(t, err.Error(), "configs/leak")
	})

	t.Run("rejects path with .. that resolves through symlinked parent", func(t *testing.T) {
		projectDir := resolvedTempDir(t)
		outsideDir := resolvedTempDir(t)

		require.NoError(t, os.Symlink(outsideDir, filepath.Join(projectDir, "link")))
		require.NoError(t, os.MkdirAll(filepath.Join(projectDir, "real"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(outsideDir, "secret.txt"), []byte("S"), 0644))

		fs := afero.NewBasePathFs(afero.NewOsFs(), projectDir)
		// real/../link/secret.txt cleans to link/secret.txt
		err := RejectSymlink(fs, "real/../link/secret.txt")
		require.ErrorContains(t, err, "symbolic link")
	})

	t.Run("handles trailing slash on directory path", func(t *testing.T) {
		projectDir := resolvedTempDir(t)
		outsideDir := resolvedTempDir(t)

		require.NoError(t, os.Symlink(outsideDir, filepath.Join(projectDir, "link")))

		fs := afero.NewBasePathFs(afero.NewOsFs(), projectDir)
		err := RejectSymlink(fs, "link/")
		assert.ErrorContains(t, err, "symbolic link")
	})

	t.Run("rejects broken symlink (target does not exist)", func(t *testing.T) {
		projectDir := resolvedTempDir(t)

		// link points to a path that does not exist
		require.NoError(t, os.Symlink(filepath.Join(projectDir, "missing-target"), filepath.Join(projectDir, "dangling")))

		fs := afero.NewBasePathFs(afero.NewOsFs(), projectDir)
		err := RejectSymlink(fs, "dangling")
		assert.ErrorContains(t, err, "symbolic link")
	})

	t.Run("rejects chained symlinks (symlink -> symlink -> file)", func(t *testing.T) {
		projectDir := resolvedTempDir(t)
		outsideDir := resolvedTempDir(t)

		secret := filepath.Join(outsideDir, "secret.txt")
		require.NoError(t, os.WriteFile(secret, []byte("S"), 0644))

		hop := filepath.Join(outsideDir, "hop")
		require.NoError(t, os.Symlink(secret, hop))
		require.NoError(t, os.Symlink(hop, filepath.Join(projectDir, "chain")))

		fs := afero.NewBasePathFs(afero.NewOsFs(), projectDir)
		err := RejectSymlink(fs, "chain")
		assert.ErrorContains(t, err, "symbolic link")
	})

	t.Run("rejects symlink with relative target escaping project", func(t *testing.T) {
		baseDir := resolvedTempDir(t)
		projectDir := filepath.Join(baseDir, "project")
		outsideDir := filepath.Join(baseDir, "outside")
		require.NoError(t, os.MkdirAll(projectDir, 0755))
		require.NoError(t, os.MkdirAll(outsideDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(outsideDir, "secret.txt"), []byte("S"), 0644))

		// project/escape -> ../outside  (relative target)
		require.NoError(t, os.Symlink("../outside", filepath.Join(projectDir, "escape")))

		fs := afero.NewBasePathFs(afero.NewOsFs(), projectDir)
		err := RejectSymlink(fs, "escape/secret.txt")
		require.ErrorContains(t, err, "symbolic link")
		assert.Contains(t, err.Error(), "escape")
	})

	t.Run("rejects symlink that points back into the project", func(t *testing.T) {
		// Even though the target is inside the project, we still reject it because
		// we cannot easily distinguish a benign in-project symlink from an attacker
		// crafting a symlink that resolves to an unrelated path via canonicalization tricks.
		projectDir := resolvedTempDir(t)
		require.NoError(t, os.MkdirAll(filepath.Join(projectDir, "real"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(projectDir, "real", "f.json"), []byte("ok"), 0644))
		require.NoError(t, os.Symlink(filepath.Join(projectDir, "real"), filepath.Join(projectDir, "alias")))

		fs := afero.NewBasePathFs(afero.NewOsFs(), projectDir)
		err := RejectSymlink(fs, "alias/f.json")
		assert.ErrorContains(t, err, "symbolic link")
	})

	t.Run("allows path with .. that stays inside project and crosses no symlinks", func(t *testing.T) {
		projectDir := resolvedTempDir(t)
		require.NoError(t, os.MkdirAll(filepath.Join(projectDir, "a", "b"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(projectDir, "a", "x.json"), []byte("ok"), 0644))

		fs := afero.NewBasePathFs(afero.NewOsFs(), projectDir)
		err := RejectSymlink(fs, "a/b/../x.json")
		assert.NoError(t, err)
	})

	t.Run("allows path with redundant separators and dot segments", func(t *testing.T) {
		projectDir := resolvedTempDir(t)
		require.NoError(t, os.MkdirAll(filepath.Join(projectDir, "a"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(projectDir, "a", "x.json"), []byte("ok"), 0644))

		fs := afero.NewBasePathFs(afero.NewOsFs(), projectDir)
		err := RejectSymlink(fs, "./a//x.json")
		assert.NoError(t, err)
	})

	t.Run("allows nested existing directories with no symlinks", func(t *testing.T) {
		projectDir := resolvedTempDir(t)
		require.NoError(t, os.MkdirAll(filepath.Join(projectDir, "a", "b", "c"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(projectDir, "a", "b", "c", "x.json"), []byte("ok"), 0644))

		fs := afero.NewBasePathFs(afero.NewOsFs(), projectDir)
		err := RejectSymlink(fs, "a/b/c/x.json")
		assert.NoError(t, err)
	})

	t.Run("allows existing parents with nonexistent leaf", func(t *testing.T) {
		projectDir := resolvedTempDir(t)
		require.NoError(t, os.MkdirAll(filepath.Join(projectDir, "a", "b"), 0755))

		fs := afero.NewBasePathFs(afero.NewOsFs(), projectDir)
		err := RejectSymlink(fs, "a/b/nonexistent.json")
		assert.NoError(t, err)
	})

	t.Run("allows path with entirely nonexistent components", func(t *testing.T) {
		projectDir := resolvedTempDir(t)

		fs := afero.NewBasePathFs(afero.NewOsFs(), projectDir)
		err := RejectSymlink(fs, "no/such/dir/file.json")
		assert.NoError(t, err)
	})

	t.Run("rejects symlinked parent even when leaf does not exist", func(t *testing.T) {
		projectDir := resolvedTempDir(t)
		outsideDir := resolvedTempDir(t)

		require.NoError(t, os.Symlink(outsideDir, filepath.Join(projectDir, "link")))

		fs := afero.NewBasePathFs(afero.NewOsFs(), projectDir)
		err := RejectSymlink(fs, "link/does-not-exist.json")
		assert.ErrorContains(t, err, "symbolic link")
	})

	t.Run("identifies the outermost symlink when multiple parents are symlinks", func(t *testing.T) {
		projectDir := resolvedTempDir(t)
		mid := resolvedTempDir(t)
		outsideDir := resolvedTempDir(t)

		// outside/inner -> outsideDir, project/outer -> mid
		require.NoError(t, os.Symlink(outsideDir, filepath.Join(mid, "inner")))
		require.NoError(t, os.Symlink(mid, filepath.Join(projectDir, "outer")))
		require.NoError(t, os.WriteFile(filepath.Join(outsideDir, "s.txt"), []byte("S"), 0644))

		fs := afero.NewBasePathFs(afero.NewOsFs(), projectDir)
		err := RejectSymlink(fs, "outer/inner/s.txt")
		require.ErrorContains(t, err, "symbolic link")
		// the outermost symlink "outer" should be reported, not "outer/inner"
		assert.Contains(t, err.Error(), `"outer"`)
	})

	t.Run("rejects symlinked path on raw OsFs with absolute path", func(t *testing.T) {
		projectDir := resolvedTempDir(t)
		outsideDir := resolvedTempDir(t)
		require.NoError(t, os.Symlink(outsideDir, filepath.Join(projectDir, "link")))
		require.NoError(t, os.WriteFile(filepath.Join(outsideDir, "s.txt"), []byte("S"), 0644))

		fs := afero.NewOsFs()
		err := RejectSymlink(fs, filepath.Join(projectDir, "link", "s.txt"))
		assert.ErrorContains(t, err, "symbolic link")
	})

	t.Run("rejects symlinked parent on MemMapFs is a no-op (MemMapFs has no Lstater symlink semantics)", func(t *testing.T) {
		// MemMapFs does not model symlinks, so this is documenting that RejectSymlink
		// is effectively pass-through on it. Production code uses OsFs/BasePathFs.
		fs := afero.NewMemMapFs()
		require.NoError(t, afero.WriteFile(fs, "a/b/c/file.yaml", []byte("ok"), 0644))
		err := RejectSymlink(fs, "a/b/c/file.yaml")
		assert.NoError(t, err)
	})
}
