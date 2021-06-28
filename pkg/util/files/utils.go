// @license
// Copyright 2021 Dynatrace LLC
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package files

import (
	"strings"

	"github.com/spf13/afero"
)

func DoesFileExist(fs afero.Fs, path string) (bool, error) {
	exists, err := afero.Exists(fs, path)

	if err != nil {
		return false, err
	}

	if !exists {
		return false, nil
	}

	isDir, err := afero.IsDir(fs, path)

	if err != nil {
		return false, err
	}

	return !isDir, nil
}

func IsYaml(file string) bool {
	return strings.HasSuffix(file, ".yaml")
}
