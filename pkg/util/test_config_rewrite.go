//go:build integration
// +build integration

/**
 * @license
 * Copyright 2020 Dynatrace LLC
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
package util

import (
	"bufio"
	"path/filepath"

	"github.com/spf13/afero"
)

//rewriteConfigNames reads the config from the config folder and rewrites the files on a inmemory filesystem.
func RewriteConfigNames(path string, fs afero.Fs, transformers []func(string) string) error {
	files, err := afero.ReadDir(fs, path)
	if err != nil {
		return err
	}

	for _, file := range files {

		fullPath := filepath.Join(path, file.Name())

		if file.IsDir() {
			err := RewriteConfigNames(fullPath, fs, transformers)
			if err != nil {
				return err
			}
			continue
		}

		result := ""
		err := func() error {

			inFile, err := fs.Open(fullPath)
			if err != nil {
				return err
			}
			defer func() {
				err = inFile.Close()
			}()

			scanner := bufio.NewScanner(inFile)
			for scanner.Scan() {

				lineWithReplacedName := applyLineTransformers(scanner.Text(), transformers)
				result += lineWithReplacedName + "\n"
			}
			return nil
		}()
		if err != nil {
			return err
		}

		dst, err := fs.Create(fullPath)
		if err != nil {
			return err
		}

		if _, err := dst.Write([]byte(result)); err != nil {
			return err
		}

		if err := dst.Close(); err != nil {
			return err
		}
	}
	return nil
}
func applyLineTransformers(line string, transformers []func(string) string) string {

	for _, transformer := range transformers {
		line = transformer(line)
	}
	return line
}
