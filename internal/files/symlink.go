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
	"slices"

	"github.com/spf13/afero"
)

// symlinkDetectedError is returned when a symlink is found at a path component.
type symlinkDetectedError struct {
	path string
}

func (e *symlinkDetectedError) Error() string {
	return fmt.Sprintf("%q is a symbolic link, which is not allowed for security reasons", e.path)
}

// RejectSymlinkRecursive checks whether path, or any of its parent directory components, is a symbolic link.
// It returns an error if so.
// Symlinks are rejected to prevent reading files outside the project boundary, such as credential files.
//
// On filesystems that do not support Lstat (e.g. afero.MemMapFs), this is a no-op.
func RejectSymlinkRecursive(fs afero.Fs, path string) error {
	// if the file system (such as MemMapFs) does not support it, nothing to check
	lstater, ok := fs.(afero.Lstater)
	if !ok {
		return nil
	}

	components := parentDirectories(path)

	// iterate from outermost (root-most) to leaf so we fail on the highest offending segment
	for _, component := range slices.Backward(components) {
		if err := rejectSymlink(lstater, component); err != nil {
			return err
		}
	}

	return nil
}

// parentDirectories returns all parent directories and the path itself, from leaf to root.
// For example, "/a/b/c" returns ["/a/b/c", "/a/b", "/a"].
func parentDirectories(path string) []string {
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

// rejectSymlink returns an error if the path is a symlink, nil otherwise.
func rejectSymlink(lstater afero.Lstater, path string) error {
	lstat, lstatCalled, err := lstater.LstatIfPossible(path)
	if err != nil {
		if os.IsNotExist(err) {
			// component does not exist on disk — nothing to check here
			return nil
		}
		return fmt.Errorf("could not check file %q: %w", path, err)
	}

	if lstatCalled && lstat.Mode()&os.ModeSymlink != 0 {
		return &symlinkDetectedError{path}
	}

	return nil
}
