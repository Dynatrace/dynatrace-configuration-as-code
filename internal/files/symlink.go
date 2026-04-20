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

	"github.com/spf13/afero"
)

// RejectSymlink checks whether path is a symbolic link and returns an error if so.
// Symlinks are rejected to prevent reading files outside the project boundary such as credential files.
// On filesystems that do not support Lstat (e.g. afero.MemMapFs), this is a no-op.
func RejectSymlink(fs afero.Fs, path string) error {
	// if file does not exist, nothing to check
	exists, err := afero.Exists(fs, path)
	if err != nil || !exists {
		return nil
	}

	// if the file system (such as MemMapFs) does not support it, nothing to check
	lstater, ok := fs.(afero.Lstater)
	if !ok {
		return nil
	}

	// check file
	fi, lstatCalled, err := lstater.LstatIfPossible(path)
	if err != nil {
		return fmt.Errorf("could not check file %q: %w", path, err)
	}

	if lstatCalled && fi.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("file %q is a symbolic link, which is not allowed for security reasons", path)
	}

	return nil
}
