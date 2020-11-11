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
	"gopkg.in/src-d/go-billy.v4"
	"gopkg.in/src-d/go-billy.v4/memfs"
	"io/ioutil"
	"os"
	"path/filepath"
)

//go:generate mockgen -source=files.go -destination=files_mock.go -package=util FileReader

// A FileReader is an interface which encapsulates io/ioutil to make the code using it testable.
// Since the reading of the file is now behind an interface, we can easily mock it.
type FileReader interface {
	ReadFile(fileName string) (content []byte, err error)
	ReadDir(fileName string) ([]os.FileInfo, error)
}

// fileReaderImpl implements interface FileReader
type fileReaderImpl struct{}

// NewFileReader creates a new FileReader. A FileReader is an interface which encapsulates io/ioutil to make the code
// using it testable. Since the reading of the file is now behind an interface, we can easily mock it.
func NewFileReader() FileReader {
	return &fileReaderImpl{}
}

// ReadFile is a wrapper around ioutil.ReadFile(fileName)
func (a *fileReaderImpl) ReadFile(fileName string) (content []byte, err error) {
	return ioutil.ReadFile(fileName)
}

// ReadDir is a wrapper around ioutil.ReadDir(fileName)
func (a *fileReaderImpl) ReadDir(fileName string) ([]os.FileInfo, error) {
	return ioutil.ReadDir(fileName)
}

// NewInMemoryFileReader creates a new in-memory file reader which reads
// all the original files once and stores them internally in-memory. This allows
// us to perform any modifications on the files later on
func NewInMemoryFileReader(originalDir string, transformers []func(string) string) (FileReader, error) {

	filesystem, err := loadToMemory(originalDir, transformers)
	if err != nil {
		return nil, err
	}

	return &inMemoryFileReaderImpl{
		filesystem: filesystem,
	}, nil
}

// loadToMemory loads the original files into memory so that subsequent
// file ready are not necessary any more. It takes care of applying a set of
// transformers to each of the lines of the files which get transferred over
// into memory
func loadToMemory(path string, transformers []func(string) string) (billy.Filesystem, error) {

	memory := memfs.New()
	err := loadDirToMemory(memory, path, transformers)

	return memory, err
}

func loadDirToMemory(memory billy.Filesystem, path string, transformers []func(string) string) error {

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}

	for _, file := range files {

		fullPath := filepath.Join(path, file.Name())

		if file.IsDir() {
			err := loadDirToMemory(memory, fullPath, transformers)
			if err != nil {
				return err
			}
			continue
		}

		result := ""
		err := func() error {

			inFile, err := os.Open(fullPath)
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

		dst, err := memory.Create(fullPath)
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

func (a *inMemoryFileReaderImpl) ReadFile(fileName string) (content []byte, err error) {

	file, err := a.filesystem.Open(fileName)

	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(file)
}

func (a *inMemoryFileReaderImpl) ReadDir(fileName string) ([]os.FileInfo, error) {
	return a.filesystem.ReadDir(fileName)
}

// fileReaderImpl implements interface FileReader
type inMemoryFileReaderImpl struct {
	filesystem billy.Filesystem
}
