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

package delete

import (
	"errors"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/persistence"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/pointer"
	"github.com/mitchellh/mapstructure"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
)

const deleteDelimiter = "/"

type loaderContext struct {
	fs         afero.Fs
	deleteFile string
	knownApis  api.APIs
}

// entryParserError is an error that occurred while parsing a delete file entry
type entryParserError struct {
	// Value of the DeleteEntry that failed to be parsed
	Value string `json:"value"`
	// Index of the entry that failed to be parsed
	Index int `json:"index"`
	// Reason describing what went wrong
	Reason error `json:"reason"`
}

func (e entryParserError) Error() string {
	return fmt.Sprintf("invalid delete entry `%s` on index `%d`: %s", e.Value, e.Index, e.Reason)
}

// parseErrors is a wrapper for multiple errors. It formats the children in a little bit nicer way.
type parseErrors []error

func (p parseErrors) Error() string {
	sb := strings.Builder{}

	sb.WriteString("failed to parse delete file:")
	for _, err := range p {
		sb.WriteString(fmt.Sprintf("\n\t%s", err.Error()))
	}

	return sb.String()
}

func LoadEntriesToDelete(fs afero.Fs, deleteFile string) (DeleteEntries, error) {
	context := &loaderContext{
		fs:         fs,
		deleteFile: filepath.Clean(deleteFile),
		knownApis:  api.NewAPIs(),
	}

	definition, err := readDeleteFile(context)

	if err != nil {
		return nil, err
	}

	return parseDeleteFileDefinition(context, definition)
}

func readDeleteFile(context *loaderContext) (persistence.FileDefinition, error) {
	targetFile, err := filepath.Abs(context.deleteFile)
	if err != nil {
		return persistence.FileDefinition{}, fmt.Errorf("could not parse absoulte path to file `%s`: %w", context.deleteFile, err)
	}

	data, err := afero.ReadFile(context.fs, targetFile)

	if err != nil {
		return persistence.FileDefinition{}, err
	}

	if len(data) == 0 {
		return persistence.FileDefinition{}, fmt.Errorf("file `%s` is empty", targetFile)
	}

	var result persistence.FileDefinition

	err = yaml.UnmarshalStrict(data, &result)

	if err != nil {
		return persistence.FileDefinition{}, err
	}

	return result, nil
}

func parseDeleteFileDefinition(ctx *loaderContext, definition persistence.FileDefinition) (DeleteEntries, error) {
	result := DeleteEntries{}
	var errs parseErrors

	for i, e := range definition.DeleteEntries {
		entry, err := parseDeleteEntry(ctx, e)

		if err != nil {
			errs = append(errs, entryParserError{
				Value:  fmt.Sprintf("%v", e),
				Index:  i,
				Reason: err,
			})
			continue
		}

		result[entry.Type] = append(result[entry.Type], entry)
	}

	if errs != nil {
		return nil, errs
	}

	return result, nil
}

func parseDeleteEntry(ctx *loaderContext, entry any) (pointer.DeletePointer, error) {

	ptr, err := parseFullEntry(ctx, entry)

	if str, ok := entry.(string); ok && err != nil {
		return parseSimpleEntry(str)
	}

	return ptr, err
}

func parseFullEntry(ctx *loaderContext, entry interface{}) (pointer.DeletePointer, error) {

	var parsed persistence.DeleteEntry
	err := mapstructure.Decode(entry, &parsed)
	if err != nil {
		return pointer.DeletePointer{}, err
	}

	if a, known := ctx.knownApis[parsed.Type]; known {
		p, err := parseAPIEntry(parsed, a)
		if err != nil {
			return pointer.DeletePointer{}, fmt.Errorf("failed to parse entry for API %q: %w", a.ID, err)
		}
		return p, nil
	}

	return parseCoordinateEntry(parsed)
}

func parseAPIEntry(parsed persistence.DeleteEntry, a api.API) (pointer.DeletePointer, error) {
	if parsed.ConfigName == "" {
		return pointer.DeletePointer{}, fmt.Errorf("delete entry of API type requiress config 'name' to be defined")
	}

	if parsed.ConfigId != "" {
		log.Warn("Delete entry %q of API type defines config 'id' - only 'name' will be used.")
	}

	// The scope is required for sub-path APIs.
	if a.SubPathAPI && parsed.Scope == "" {
		return pointer.DeletePointer{}, errors.New("API requires a scope, but non was defined")
	}
	// Non sub-path APIs must not define the scope
	if !a.SubPathAPI && parsed.Scope != "" {
		return pointer.DeletePointer{}, errors.New("API does not allow a scope, but a scope was defined")
	}

	return pointer.DeletePointer{
		Type:       parsed.Type,
		Identifier: parsed.ConfigName,
		Scope:      parsed.Scope,
	}, nil
}

func parseCoordinateEntry(parsed persistence.DeleteEntry) (pointer.DeletePointer, error) {
	if parsed.ConfigId == "" {
		return pointer.DeletePointer{}, fmt.Errorf("delete entry requires config 'id' to be defined")
	}
	if parsed.Project == "" {
		return pointer.DeletePointer{}, fmt.Errorf("delete entry requires 'project' to be defined")
	}
	if parsed.ConfigName != "" {
		log.Warn("Delete entry defines config 'name' - only 'id' will be used.")
	}
	return pointer.DeletePointer{
		Project:    parsed.Project,
		Type:       parsed.Type,
		Identifier: parsed.ConfigId,
	}, nil
}

func parseSimpleEntry(entry string) (pointer.DeletePointer, error) {
	if !strings.Contains(entry, deleteDelimiter) {
		return pointer.DeletePointer{}, fmt.Errorf("invalid format. doesn't contain `%s`", deleteDelimiter)
	}

	parts := strings.SplitN(entry, deleteDelimiter, 2)

	// since the string must contain at least one delimiter and we
	// split the entity by max two, we do not need to test for len of parts
	apiId := parts[0]
	deleteIdentifier := parts[1]

	return pointer.DeletePointer{
		Type:       apiId,
		Identifier: deleteIdentifier,
	}, nil
}
