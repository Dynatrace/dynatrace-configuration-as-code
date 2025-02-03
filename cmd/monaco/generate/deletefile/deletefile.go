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

package deletefile

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/afero"
	"golang.org/x/exp/maps"
	"gopkg.in/yaml.v2"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/timeutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/reference"
	valueParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/persistence"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
)

type createDeleteFileOptions struct {
	environmentNames []string
	fileName         string
	includeTypes     []string
	excludeTypes     []string
	outputFolder     string
}

func createDeleteFile(fs afero.Fs, projects []project.Project, apis api.APIs, options createDeleteFileOptions) error {
	content, err := generateDeleteFileContent(apis, projects, options)
	if err != nil {
		log.WithFields(field.Error(err)).Error("Failed to generate delete file content: %v", err)
		return err
	}

	folderPath, err := filepath.Abs(options.outputFolder)
	if err != nil {
		return fmt.Errorf("failed to access output path: %q: %w", options.outputFolder, err)
	}

	if options.outputFolder != "" {
		if exits, _ := afero.Exists(fs, folderPath); !exits {
			err = fs.Mkdir(folderPath, 0777)
			if err != nil {
				return fmt.Errorf("failed to create output folder: %q", folderPath)
			}
		}
	}

	file := filepath.Join(folderPath, options.fileName)

	exists, err := afero.Exists(fs, file)
	if err != nil {
		return fmt.Errorf("failed to check if file %q exists: %w", options.fileName, err)
	}

	if exists {
		time := timeutils.TimeAnchor().Format("20060102-150405")
		var newFileName string
		if lastDot := strings.LastIndex(options.fileName, "."); lastDot > -1 {
			newFileName = fmt.Sprintf("%s_%s%s", options.fileName[:lastDot], time, options.fileName[lastDot:])
		} else {
			newFileName = fmt.Sprintf("%s_%s", options.fileName, time)
		}

		newFile := filepath.Join(folderPath, newFileName)
		log.WithFields(field.F("file", newFile), field.F("existingFile", options.fileName)).Debug("Output file %q already exists, creating %q instead", options.fileName, newFile)
		file = newFile
	}

	err = afero.WriteFile(fs, file, content, 0666)
	if err != nil {
		return fmt.Errorf("failed to create delete file %q: %w", file, err)
	}
	log.WithFields(field.F("file", file)).Info("Delete file written to %q", file)

	return nil
}

func generateDeleteFileContent(apis api.APIs, projects []project.Project, options createDeleteFileOptions) ([]byte, error) {

	log.Info("Generating delete file...")

	var entries []persistence.DeleteEntry
	if len(options.environmentNames) == 0 {
		entries = generateDeleteEntries(apis, projects, options)
	} else {
		entries = generateDeleteEntriesForEnvironments(apis, projects, options)
	}

	f := persistence.FullFileDefinition{DeleteEntries: entries}
	b, err := yaml.Marshal(&f)
	if err != nil {
		return nil, fmt.Errorf("failed to marshall delete file definition to YAML: %w", err)
	}

	return b, nil
}

func generateDeleteEntries(apis api.APIs, projects []project.Project, options createDeleteFileOptions) []persistence.DeleteEntry {
	entries := make(map[string]persistence.DeleteEntry) // set to ensure cfgs without environment overwrites are only added once

	inclTypesLookup := toStrLookupMap(options.includeTypes)
	exclTypesLookup := toStrLookupMap(options.excludeTypes)

	for _, p := range projects {
		log.Info("Adding delete entries for project %q...", p.Id)
		p.ForEveryConfigDo(func(c config.Config) {
			if skipping(c.Coordinate.Type, inclTypesLookup, exclTypesLookup) {
				if c.Coordinate.Type == string(config.OpenPipelineTypeID) {
					log.Info("Skipped creating delete entry for %q as openpipeline configurations cannot be deleted", c.Coordinate)
				}
				return
			}

			entry, err := createDeleteEntry(c, apis, p)
			if err != nil {
				log.WithFields(field.Error(err)).Warn("Failed to automatically create delete entry for %q: %s", c.Coordinate, err)
				return
			}
			entries[toMapKey(entry)] = entry
		})
	}

	return maps.Values(entries)
}

func generateDeleteEntriesForEnvironments(apis api.APIs, projects []project.Project, options createDeleteFileOptions) []persistence.DeleteEntry {
	entries := make(map[string]persistence.DeleteEntry) // set to ensure cfgs without environment overwrites are only added once

	inclTypesLookup := toStrLookupMap(options.includeTypes)
	exclTypesLookup := toStrLookupMap(options.excludeTypes)

	for _, p := range projects {
		for _, env := range options.environmentNames {
			log.Info("Adding delete entries for project %q and environment %q...", p.Id, env)
			p.ForEveryConfigInEnvironmentDo(env, func(c config.Config) {
				if skipping(c.Coordinate.Type, inclTypesLookup, exclTypesLookup) {
					if c.Coordinate.Type == string(config.OpenPipelineTypeID) {
						log.Info("Skipped creating delete entry for %q as openpipeline configurations cannot be deleted", c.Coordinate)
					}
					return
				}
				entry, err := createDeleteEntry(c, apis, p)
				if err != nil {
					log.WithFields(field.Error(err)).Warn("Failed to automatically create delete entry for %q: %s", c.Coordinate, err)
					return
				}
				entries[toMapKey(entry)] = entry
			})
		}
	}

	return maps.Values(entries)
}

func toStrLookupMap(sl []string) map[string]struct{} {
	res := map[string]struct{}{}
	for _, t := range sl {
		res[t] = struct{}{}
	}
	return res
}

func skipping(ttype string, included, excluded map[string]struct{}) bool {
	if _, ok := excluded[ttype]; ok {
		return true
	}
	if len(included) > 0 {
		if _, ok := included[ttype]; !ok {
			return true
		}
	}
	return false
}

// toMapKey creates a unique key for persistence.DeleteEntries so that they can be managed in maps.
// Note, that persistence.DeleteEntry itself cannot be used as a map key, because it contains a map as field for which
// no == and != comparison is defined.
func toMapKey(d persistence.DeleteEntry) string {
	values := make(map[string]string)
	values["project"] = d.Project
	values["type"] = d.Type
	values["configId"] = d.ConfigId
	values["configName"] = d.ConfigName
	values["scope"] = d.Scope
	maps.Copy(values, d.CustomValues)
	j, _ := json.Marshal(sortedKeyValues(values)) //nolint:errchkjson
	return string(j)
}

func sortedKeyValues(myMap map[string]string) []string {
	keys := maps.Keys(myMap)
	sort.Strings(keys)
	var sortedValues []string
	for _, k := range keys {
		sortedValues = append(sortedValues, k+":"+myMap[k])
	}
	return sortedValues
}

func createDeleteEntry(c config.Config, apis api.APIs, project project.Project) (persistence.DeleteEntry, error) {
	if apis.Contains(c.Coordinate.Type) {
		return createConfigAPIEntry(c, apis, project)
	}

	return persistence.DeleteEntry{
		Project:  c.Coordinate.Project,
		Type:     c.Coordinate.Type,
		ConfigId: c.Coordinate.ConfigId,
	}, nil
}

func createConfigAPIEntry(c config.Config, apis api.APIs, project project.Project) (persistence.DeleteEntry, error) {
	nameParam := c.Parameters[config.NameParameter]

	if nameParam.GetType() == reference.ReferenceParameterType {
		// we don't sort configs or create entities, so references will never find other configs they point to -> user has to write those manually
		return persistence.DeleteEntry{}, fmt.Errorf("unable to resolve reference parameters")
	}

	val, err := nameParam.ResolveValue(parameter.ResolveContext{ParameterName: config.NameParameter})
	if err != nil {
		return persistence.DeleteEntry{}, fmt.Errorf("unable to resolve 'name' parameter: %w", err)
	}

	name, ok := val.(string)
	if !ok {
		return persistence.DeleteEntry{}, fmt.Errorf("value of 'name' parameter '%v' was not a string", val)
	}

	if name == "" {
		return persistence.DeleteEntry{}, fmt.Errorf("'name' parameter was empty")
	}

	var scopeValue string
	var customValues = make(map[string]string)
	if apis[c.Coordinate.Type].HasParent() {
		scopeParam, ok := c.Parameters[config.ScopeParameter]
		if !ok {
			return persistence.DeleteEntry{}, fmt.Errorf("no scope parameter found")
		}

		refs := scopeParam.GetReferences()
		if len(refs) < 1 {
			return persistence.DeleteEntry{}, fmt.Errorf("scope parameter has no references")
		}

		refCfg, ok := project.GetConfigFor(c.Environment, refs[0].Config)
		if !ok {
			return persistence.DeleteEntry{}, fmt.Errorf("no config for referenced scope found")
		}

		refCfgNameParam, ok := refCfg.Parameters[config.NameParameter]
		if !ok {
			return persistence.DeleteEntry{}, fmt.Errorf("no name parameter for reference config found")
		}

		refCfgNamParamVal, ok := refCfgNameParam.(*valueParam.ValueParameter)
		if !ok {
			return persistence.DeleteEntry{}, fmt.Errorf("name parameter of referenced config is no value parameter")
		}

		nameOfRefCfg, err := refCfgNamParamVal.ResolveValue(parameter.ResolveContext{})
		if err != nil {
			log.Warn("Unable to create delete entry for '%s': %v", c.Coordinate, err)
			return persistence.DeleteEntry{}, err
		}

		nameOfRefCfgStr, ok := nameOfRefCfg.(string)
		if !ok {
			return persistence.DeleteEntry{}, fmt.Errorf("resolved name parameter is no string")
		}
		scopeValue = nameOfRefCfgStr

		if apis[c.Coordinate.Type].ID == api.KeyUserActionsWeb {
			// while generating the delete file (which happens offline) we are not able
			// to resolve Reference Parameters
			for pName, p := range c.Parameters {
				if p.GetType() == reference.ReferenceParameterType {
					delete(c.Parameters, pName)
				}
			}
			props, resolvErr := c.ResolveParameterValues(entities.New())
			if resolvErr != nil {
				return persistence.DeleteEntry{}, errors.Join(resolvErr...)
			}
			rendered, err := c.Render(props)
			if err != nil {
				return persistence.DeleteEntry{}, err
			}

			var jsonM map[string]string
			err = json.Unmarshal([]byte(rendered), &jsonM)
			if err != nil {
				return persistence.DeleteEntry{}, err
			}

			customValues["domain"] = jsonM["domain"]
			customValues["actionType"] = jsonM["actionType"]
		}
	}

	return persistence.DeleteEntry{
		Type:         c.Coordinate.Type,
		ConfigName:   name,
		Scope:        scopeValue,
		CustomValues: customValues,
	}, nil
}
