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

	"github.com/spf13/afero"
)

// RejectSymlink checks whether path or any of its parent directory components is a symbolic link
// and returns an error if so. Symlinks are rejected to prevent reading files outside the project
// boundary such as credential files.
//
// Each component of the path is checked independently because lstat on the final component
// transparently resolves intermediate symlinked directories, which would otherwise allow a
// symlinked parent directory to bypass the check (see CWE-59).
//
// On filesystems that do not support Lstat (e.g. afero.MemMapFs), this is a no-op.
func RejectSymlink(fs afero.Fs, path string) error {
	// if the file system (such as MemMapFs) does not support it, nothing to check
	lstater, ok := fs.(afero.Lstater)
	if !ok {
		return nil
	}

	components := pathComponents(path)
	return checkComponentsForSymlinks(lstater, path, components)
}

// pathComponents returns all parent directories and the path itself, from leaf to root.
// For example, "/a/b/c" returns ["/a/b/c", "/a/b", "/a"].
func pathComponents(path string) []string {
	cleaned := filepath.Clean(path)

	var components []string
	for current := cleaned; ; {
		components = append(components, current)
		parent := filepath.Dir(current)
		if parent == current || parent == "." {
			break
		}
		current = parent
	}
	return components
}

// checkComponentsForSymlinks verifies that no path component is a symlink.
// Checks from outermost component down to the leaf so errors report the highest offending segment.
func checkComponentsForSymlinks(lstater afero.Lstater, originalPath string, components []string) error {
	// iterate from outermost (root-most) to leaf
	for i := len(components) - 1; i >= 0; i-- {
		component := components[i]

		fi, lstatCalled, err := lstater.LstatIfPossible(component)
		if err != nil {
			if os.IsNotExist(err) {
				// component does not exist on disk — nothing to check here
				continue
			}
			return fmt.Errorf("could not check file %q: %w", component, err)
		}

		if lstatCalled && fi.Mode()&os.ModeSymlink != 0 {
			return symlinksError(originalPath, component)
		}
	}

	return nil
}

// symlinksError formats an error message for a symlink found at a path component.
func symlinksError(fullPath, symlinkComponent string) error {
	if symlinkComponent == filepath.Clean(fullPath) {
		return fmt.Errorf("file %q is a symbolic link, which is not allowed for security reasons", fullPath)
	}
	return fmt.Errorf("path %q contains a symbolic link at %q, which is not allowed for security reasons", fullPath, symlinkComponent)
}
