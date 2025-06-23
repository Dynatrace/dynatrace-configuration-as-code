//go:build integration || cleanup || download_restore || unit || nightly || account_integration

/*
 * @license
 * Copyright 2025 Dynatrace LLC
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

package runner

import (
	"bufio"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/rand"
)

func GenerateTestSuffix(t *testing.T, generalSuffix string) string {
	randomNumber, err := rand.Int(int64(10000))
	if err != nil {
		t.Fatalf("Failed to generate random number for the test suffix: %s", err)
	}

	timestamp := time.Now().Format("20060102150405")
	suffix := fmt.Sprintf("_%s_%d", timestamp, randomNumber)
	if generalSuffix != "" {
		suffix = fmt.Sprintf("%s_%s", suffix, generalSuffix)
	}
	return strings.ToLower(suffix)
}

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
