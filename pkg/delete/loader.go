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
	"path/filepath"
	"strings"

	"github.com/mitchellh/mapstructure"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/persistence"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/pointer"

	"github.com/spf13/afero"
	"go.yaml.in/yaml/v2"
)

const deleteDelimiter = "/"

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
		sb.WriteString(fmt.Sprintf("\n\t%s", err))
	}

	return sb.String()
}

func LoadEntriesFromFile(fs afero.Fs, deleteFile string) (DeleteEntries, error) {
	definition, err := readDeleteFile(fs, deleteFile)
	if err != nil {
		return nil, err
	}

	return parseDeleteFileDefinition(definition)
}

func readDeleteFile(fs afero.Fs, deleteFile string) (persistence.FileDefinition, error) {
	targetFile, err := filepath.Abs(deleteFile)
	if err != nil {
		return persistence.FileDefinition{}, fmt.Errorf("could not parse absoulte path to file `%s`: %w", deleteFile, err)
	}

	data, err := afero.ReadFile(fs, targetFile)
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

func parseDeleteFileDefinition(definition persistence.FileDefinition) (DeleteEntries, error) {
	result := DeleteEntries{}
	var errs parseErrors

	for i, e := range definition.DeleteEntries {
		entry, err := convertToDeletePointer(e)
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

func convertToDeletePointer(entry any) (pointer.DeletePointer, error) {
	if str, ok := entry.(string); ok {
		return convertLegacy(str)
	}
	return convert(entry)
}

func convert(entry interface{}) (pointer.DeletePointer, error) {
	var parsed persistence.DeleteEntry
	if err := mapstructure.Decode(entry, &parsed); err != nil {
		return pointer.DeletePointer{}, err
	}

	if parsed.Type == "" {
		return pointer.DeletePointer{}, errors.New("'type' is not supported for this API")
	}
	if a, known := api.NewAPIs()[parsed.Type]; known {
		if err := verifyAPIEntry(parsed, a); err != nil {
			return pointer.DeletePointer{}, fmt.Errorf("failed to parse entry for API '%s': %w", a.ID, err)
		}
	} else {
		if err := verifyCoordinateEntry(parsed); err != nil {
			return pointer.DeletePointer{}, err
		}
	}

	dp := pointer.DeletePointer{
		Project:        parsed.Project,
		Type:           parsed.Type,
		Scope:          parsed.Scope,
		ActionType:     parsed.CustomValues["actionType"],
		Domain:         parsed.CustomValues["domain"],
		OriginObjectId: parsed.ObjectId,
	}
	if _, known := api.NewAPIs()[parsed.Type]; known {
		dp.Identifier = parsed.ConfigName
	} else {
		dp.Identifier = parsed.ConfigId
	}

	return dp, nil
}

func verifyAPIEntry(parsed persistence.DeleteEntry, a api.API) error {
	if parsed.ConfigId != "" {
		return errors.New("'id' is not supported for this API")
	}
	if parsed.Project != "" {
		return errors.New("'project' is not supported for this API")
	}
	if parsed.ConfigName != "" && parsed.ObjectId != "" {
		return errors.New("'name' and 'objectId' can't be used together to define an entry")
	}
	if parsed.ConfigName == "" && parsed.ObjectId == "" {
		return errors.New("a 'name' or a 'objectId' is required, but none was defined")
	}
	// The scope is required for sub-path APIs.
	if a.HasParent() && parsed.Scope == "" {
		return errors.New("API requires a 'scope', but none was defined")
	}
	// Non sub-path APIs must not define the scope
	if !a.HasParent() && parsed.Scope != "" {
		return errors.New("API does not allow 'scope', but it was defined")
	}

	if a.ID == api.KeyUserActionsWeb {
		if v := parsed.CustomValues["actionType"]; v == "" {
			return fmt.Errorf("API of type '%s' requires a '%s', but none was defined", a, "actionType")
		}
		if v := parsed.CustomValues["domain"]; v == "" {
			return fmt.Errorf("API of type '%s' requires a '%s', but none was defined", a, "domain")
		}
	}
	return nil
}

func verifyCoordinateEntry(parsed persistence.DeleteEntry) error {
	if parsed.ConfigName != "" {
		return errors.New("'name' is not supported for this API")
	}
	if parsed.ObjectId == "" && (parsed.ConfigId == "" && parsed.Project == "") {
		return errors.New("either an 'objectId' or a pair 'id'-'project' is required")
	} else if parsed.ObjectId != "" {
		if parsed.ConfigId != "" || parsed.Project != "" {
			return errors.New("the pair 'id'-'project' and 'objectId' can't be used together to define an entry")
		}
	} else {
		if parsed.ConfigId == "" {
			return fmt.Errorf("delete entry requires config 'id' to be defined")
		}
		if parsed.Project == "" {
			return fmt.Errorf("delete entry requires 'project' to be defined")
		}
	}
	return nil
}

func convertLegacy(entry string) (pointer.DeletePointer, error) {
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
