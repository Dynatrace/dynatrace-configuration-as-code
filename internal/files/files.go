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

package files

import (
	"os"
	"strings"

	"github.com/spf13/afero"
)

// YamlExtensions contains all yaml-file extensions without leading dot that we allow.
var YamlExtensions = []string{"yaml", "yml"}

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

// IsYamlFileExtension checks whether a file has a yaml extension specified in YamlExtensions, with leading dot.
func IsYamlFileExtension(file string) bool {
	for _, extension := range YamlExtensions {
		if strings.HasSuffix(file, "."+extension) {
			return true
		}
	}

	return false
}

func ReplacePathSeparators(path string) (newPath string) {
	newPath = strings.ReplaceAll(path, "\\", string(os.PathSeparator))
	newPath = strings.ReplaceAll(newPath, "/", string(os.PathSeparator))
	return newPath
}
