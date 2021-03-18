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
	"log"
	"path/filepath"
	"regexp"

	"github.com/spf13/afero"
)

//go:generate mockgen -source=creator.go -destination=creator_mock.go -package=files FileCreator

//FileCreator is an interface to encapsulate the file creation process. Is has 2 implementations
//in memory and one that uses files on disk.
type FileCreator interface {
	CreateFolder(path string) (fullpath string, err error)
	CreateFile(byteArray []byte, path string, name string, fileType string) (cleanName string, err error)
}

// fileCreatorImpl implements interface FileCreator using disk as storage
type fileCreator struct {
	fileManager afero.Fs
}

//NewDiskFileCreator creates a new FileCreator instance
func NewDiskFileCreator() FileCreator {
	el := &fileCreator{}
	el.fileManager = afero.NewOsFs()
	return el
}

//CreateFolder creates a folder in the specified path
func (a *fileCreator) CreateFolder(path string) (fullpath string, err error) {
	//path should be sanitized
	exist, err := afero.Exists(a.fileManager, path)
	if !exist && err == nil {
		err = a.fileManager.Mkdir(path, 0777)
	}
	if err != nil {
		return "", err
	}
	return path, nil
}

//CreateFile allows to write a file on disk using the specified path
func (a *fileCreator) CreateFile(byteArray []byte, path string, name string, fileType string) (cleanName string, err error) {
	cleanName = sanitizeName(name)
	fullPath := filepath.Join(path, cleanName+fileType)
	err = afero.WriteFile(a.fileManager, fullPath, byteArray, 0664)
	if err != nil {
		return "", err
	}

	return cleanName, nil
}

//NewInMemoryFileCreator creates  a new instance of FileCreator
func NewInMemoryFileCreator() FileCreator {
	el := &fileCreator{}
	el.fileManager = afero.NewMemMapFs()
	return el
}

//SanitizeName removes special characters and max 50 characters in name, no special characters
func sanitizeName(name string) string {
	reg, err := regexp.Compile("[^a-zA-Z0-9-]+")
	if err != nil {
		log.Fatal(err)
	}
	processedString := reg.ReplaceAllString(name, "")
	runes := []rune(processedString)
	if len(runes) > 50 {
		processedString = string(runes[:50])
	}
	return processedString

}
