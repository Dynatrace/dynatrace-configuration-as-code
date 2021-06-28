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

package v2

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
)

const deleteDelimiter = "/"

type loaderContext struct {
	fs         afero.Fs
	workingDir string
	deleteFile string
	knownApis  map[string]struct{}
}

type deleteFileDefinition struct {
	DeleteEntries []string `yaml:"delete"`
}

type DeleteEntryParserError struct {
	Value  string
	Index  int
	Reason string
}

func (e *DeleteEntryParserError) Error() string {
	return fmt.Sprintf("invalid delete entry `%s` on index `%d`: %s",
		e.Value, e.Index, e.Reason)
}

func LoadEntriesToDelete(fs afero.Fs, knownApis []string, workingDir string, deleteFile string) (map[string][]DeletePointer, []error) {
	context := &loaderContext{
		fs:         fs,
		workingDir: filepath.Clean(workingDir),
		deleteFile: filepath.Clean(deleteFile),
		knownApis:  toSetMap(knownApis),
	}

	definition, err := parseDeleteFile(context)

	if err != nil {
		return nil, []error{err}
	}

	return parseDeleteFileDefinition(context, definition)
}

func toSetMap(strs []string) map[string]struct{} {
	result := make(map[string]struct{})

	for _, s := range strs {
		result[s] = struct{}{}
	}

	return result
}

func parseDeleteFile(context *loaderContext) (deleteFileDefinition, error) {
	targetFile := context.deleteFile

	if !filepath.IsAbs(targetFile) {
		targetFile = filepath.Join(context.workingDir, targetFile)
	}

	data, err := afero.ReadFile(context.fs, targetFile)

	if err != nil {
		return deleteFileDefinition{}, err
	}

	if len(data) == 0 {
		return deleteFileDefinition{}, fmt.Errorf("file `%s` is empty", targetFile)
	}

	var result deleteFileDefinition

	err = yaml.UnmarshalStrict(data, &result)

	if err != nil {
		return deleteFileDefinition{}, err
	}

	return result, nil
}

func parseDeleteFileDefinition(context *loaderContext, definition deleteFileDefinition) (map[string][]DeletePointer, []error) {
	var result = make(map[string][]DeletePointer)
	var errors []error

	for i, e := range definition.DeleteEntries {
		entry, err := parseDeleteEntry(context, i, e)

		if err != nil {
			errors = append(errors, err)
			continue
		}

		result[entry.ApiId] = append(result[entry.ApiId], entry)
	}

	if errors != nil {
		return nil, errors
	}

	return result, nil
}

func parseDeleteEntry(context *loaderContext, index int, entry string) (DeletePointer, error) {
	if !strings.Contains(entry, deleteDelimiter) {
		return DeletePointer{}, &DeleteEntryParserError{
			Value:  entry,
			Index:  index,
			Reason: fmt.Sprintf("invalid format. doesn't contain `%s`", deleteDelimiter),
		}
	}

	parts := strings.SplitN(entry, deleteDelimiter, 2)

	// since the string must contain at least one delimiter and we
	// split the entity by max two, we do not need to test for len of parts
	apiId := parts[0]
	entityName := parts[1]

	if _, found := context.knownApis[apiId]; !found {
		return DeletePointer{}, &DeleteEntryParserError{
			Value:  entry,
			Index:  index,
			Reason: fmt.Sprintf("unknown api `%s`", apiId),
		}
	}

	return DeletePointer{
		ApiId: apiId,
		Name:  entityName,
	}, nil
}
