//go:build unit || integration || integration_v1 || download_restore || cleanup || nightly

/*
 * @license
 * Copyright 2023 Dynatrace LLC
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

package testutils

import (
	"testing"

	"github.com/spf13/afero"
)

// CreateTestFileSystem creates a virtual filesystem with 2 layers.
// The first layer allows to read file from the disk
// the second layer allows to modify files on a virtual filesystem
func CreateTestFileSystem() afero.Fs {
	base := afero.NewOsFs()
	baseLayer := afero.NewReadOnlyFs(base)
	return afero.NewCopyOnWriteFs(baseLayer, afero.NewMemMapFs())
}

// TempFs creates a new [afero.Fs] file system within a temporary directory.
// The temp directory will be cleaned up automatically.
// Use this to create a file system for testing rather than afero.MemMapFs as it catches more bugs as it works with actual files.
func TempFs(t *testing.T) afero.Fs {
	return afero.NewBasePathFs(afero.NewOsFs(), t.TempDir())
}
