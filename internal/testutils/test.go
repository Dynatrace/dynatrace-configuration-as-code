//go:build integration || integration_v1 || download_restore || unit || nightly

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
	"bufio"
	"github.com/spf13/afero"
	"path/filepath"
	"strings"
	"testing"
)

func FailTestOnAnyError(t *testing.T, errors []error, errorMessage string) {
	if len(errors) == 0 {
		return
	}

	for _, err := range errors {
		t.Logf("%s: %v", errorMessage, err)
	}
	t.FailNow()
}

func ReplaceName(line string, idChange func(string) string) string {

	if strings.Contains(line, "env-token-name:") {
		return line
	}

	if strings.Contains(line, "name:") {

		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "-") {
			trimmed = trimmed[1:]
			trimmed = strings.TrimSpace(trimmed)
		}

		withoutPrefix := strings.TrimLeft(trimmed, "name:")
		name := strings.TrimSpace(withoutPrefix)

		if name == "" { //line only contained the name, can't do anything here and probably a non-shorthand v2 reference
			return line
		}

		if strings.HasPrefix(name, "\"") || strings.HasPrefix(name, "'") {
			name = name[1 : len(name)-1]
		}

		// Dependencies are not substituted
		if isV1Dependency(name) || isV2Dependency(name) {
			return line
		}

		replaced := strings.ReplaceAll(line, name, idChange(name))
		return replaced
	}
	return line
}

// rewriteConfigNames reads the config from the config folder and rewrites the files on a inmemory filesystem.
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

		if strings.Contains(fullPath, "manifest") {
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

func isV1Dependency(name string) bool {
	return strings.HasSuffix(name, ".id") || strings.HasSuffix(name, ".name")
}

func isV2Dependency(name string) bool {
	if !(strings.HasPrefix(name, "[") && strings.HasSuffix(name, "]")) {
		return false
	}
	s := strings.TrimSuffix(name, "]")
	s = strings.TrimSpace(s)
	return strings.HasSuffix(s, `"id"`) || strings.HasSuffix(s, `"name"`)
}
