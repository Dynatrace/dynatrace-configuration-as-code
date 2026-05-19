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

	cleaned := filepath.Clean(path)

	// collect path and all of its parent directories up to (but excluding) the filesystem root
	var components []string
	for current := cleaned; ; {
		components = append(components, current)
		parent := filepath.Dir(current)
		if parent == current || parent == "." {
			break
		}
		current = parent
	}

	// check from outermost component down to the leaf so we fail on the highest offending segment
	for i := len(components) - 1; i >= 0; i-- {
		component := components[i]

		exists, err := afero.Exists(fs, component)
		if err != nil || !exists {
			continue
		}

		fi, lstatCalled, err := lstater.LstatIfPossible(component)
		if err != nil {
			return fmt.Errorf("could not check file %q: %w", component, err)
		}

		if lstatCalled && fi.Mode()&os.ModeSymlink != 0 {
			if component == cleaned {
				return fmt.Errorf("file %q is a symbolic link, which is not allowed for security reasons", path)
			}
			return fmt.Errorf("path %q contains a symbolic link at %q, which is not allowed for security reasons", path, component)
		}
	}

	return nil
}
