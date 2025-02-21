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
	"fmt"
	"path/filepath"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/secret"
)

type loaderContext struct {
	fs         afero.Fs
	deleteFile string
}

type DeleteEntryParserError struct {
	Value  string `json:"value"`
	Index  int    `json:"index"`
	Reason string `json:"reason"`
}

func newDeleteEntryParserError(value string, index int, reason string) DeleteEntryParserError {
	return DeleteEntryParserError{
		Value:  value,
		Index:  index,
		Reason: reason,
	}
}

func (e DeleteEntryParserError) Error() string {
	return fmt.Sprintf("invalid delete entry `%s` on index `%d`: %s",
		e.Value, e.Index, e.Reason)
}

func LoadResourcesToDelete(fs afero.Fs, deleteFile string) (Resources, error) {
	context := &loaderContext{
		fs:         fs,
		deleteFile: filepath.Clean(deleteFile),
	}

	definition, err := readDeleteFile(context)

	if err != nil {
		return Resources{}, err
	}

	return parseDeleteFileDefinition(definition)
}

func readDeleteFile(context *loaderContext) (FileDefinition, error) {

	data, err := afero.ReadFile(context.fs, context.deleteFile)

	if err != nil {
		return FileDefinition{}, err
	}

	if len(data) == 0 {
		return FileDefinition{}, fmt.Errorf("file `%s` is empty", context.deleteFile)
	}

	var result FileDefinition

	err = yaml.Unmarshal(data, &result)

	if err != nil {
		return FileDefinition{}, err
	}

	return result, nil
}

func parseDeleteFileDefinition(definition FileDefinition) (Resources, error) {
	var groups []Group
	var users []User
	var serviceUsers []ServiceUser
	var accountPolicies []AccountPolicy
	var environmentPolicies []EnvironmentPolicy

	for i, e := range definition.DeleteEntries {
		var parsed DeleteEntry
		err := mapstructure.Decode(e, &parsed)
		if err != nil {
			return Resources{}, newDeleteEntryParserError(fmt.Sprintf("%v", e), i, err.Error())
		}
		switch parsed.Type {
		case "user":
			var parsed UserDeleteEntry
			err := mapstructure.Decode(e, &parsed)
			if err != nil {
				return Resources{}, newDeleteEntryParserError(fmt.Sprintf("%v", e), i, err.Error())
			}
			users = append(users, User{Email: secret.Email(parsed.Email)})
		case "service-user":
			if !featureflags.ServiceUsers.Enabled() {
				return Resources{}, newDeleteEntryParserError(fmt.Sprintf("%v", e), i, fmt.Sprintf(`unknown type %q - needs to be one of "user", "group" or "policy"`, parsed.Type))
			}
			var parsed ServiceUserDeleteEntry
			err := mapstructure.Decode(e, &parsed)
			if err != nil {
				return Resources{}, newDeleteEntryParserError(fmt.Sprintf("%v", e), i, err.Error())
			}
			serviceUsers = append(serviceUsers, ServiceUser{Name: parsed.Name})
		case "group":
			var parsed GroupDeleteEntry
			err := mapstructure.Decode(e, &parsed)
			if err != nil {
				return Resources{}, newDeleteEntryParserError(fmt.Sprintf("%v", e), i, err.Error())
			}
			groups = append(groups, Group{Name: parsed.Name})
		case "policy":
			var parsed PolicyDeleteEntry
			err := mapstructure.Decode(e, &parsed)
			if err != nil {
				return Resources{}, newDeleteEntryParserError(fmt.Sprintf("%v", e), i, err.Error())
			}
			switch parsed.Level.Type {
			case "account":
				accountPolicies = append(accountPolicies, AccountPolicy{Name: parsed.Name})
			case "environment":
				environmentPolicies = append(environmentPolicies, EnvironmentPolicy{Name: parsed.Name, Environment: parsed.Level.Environment})
			default:
				return Resources{}, newDeleteEntryParserError(fmt.Sprintf("%v", e), i, fmt.Sprintf(`unknown policy level %q - needs to be either "account" or "environment"`, parsed.Level))
			}
		default:
			if featureflags.ServiceUsers.Enabled() {
				return Resources{}, newDeleteEntryParserError(fmt.Sprintf("%v", e), i, fmt.Sprintf(`unknown type %q - needs to be one of "user", "service-user", "group" or "policy"`, parsed.Type))
			}
			return Resources{}, newDeleteEntryParserError(fmt.Sprintf("%v", e), i, fmt.Sprintf(`unknown type %q - needs to be one of "user", "group" or "policy"`, parsed.Type))
		}

	}

	return Resources{
		Users:               users,
		ServiceUsers:        serviceUsers,
		Groups:              groups,
		AccountPolicies:     accountPolicies,
		EnvironmentPolicies: environmentPolicies,
	}, nil
}
