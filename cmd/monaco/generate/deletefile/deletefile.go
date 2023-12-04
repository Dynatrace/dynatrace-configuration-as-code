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
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/errutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/timeutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/reference"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/persistence"
	manifestloader "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/loader"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
	"github.com/spf13/afero"
	"golang.org/x/exp/maps"
	"gopkg.in/yaml.v2"
	"path/filepath"
	"strings"
)

func createDeleteFile(fs afero.Fs, manifestPath string, projectNames, specificEnvironments []string, filename, outputFolder string) error {

	m, errs := manifestloader.Load(&manifestloader.Context{
		Fs:           fs,
		ManifestPath: manifestPath,
		Opts: manifestloader.Options{
			DoNotResolveEnvVars:      true,
			RequireEnvironmentGroups: true,
		},
	})
	if len(errs) > 0 {
		errutils.PrintErrors(errs)
		return fmt.Errorf("failed to load manifest %q", manifestPath)
	}

	apis := api.NewAPIs()
	projects, errs := project.LoadProjects(fs, project.ProjectLoaderContext{
		KnownApis:       apis.GetApiNameLookup(),
		WorkingDir:      filepath.Dir(manifestPath),
		Manifest:        m,
		ParametersSerde: config.DefaultParameterParsers,
	})

	if len(errs) > 0 {
		errutils.PrintErrors(errs)
		return fmt.Errorf("failed to load projects")
	}

	projects, err := filterProjects(projects, projectNames)

	if err != nil {
		log.WithFields(field.Error(err)).Error("Failed to filter requested projects: %v", err)
		return err
	}

	content, err := generateDeleteFileContent(projects, specificEnvironments, apis)
	if err != nil {
		log.WithFields(field.Error(err)).Error("Failed to generate delete file content: %v", err)
		return err
	}

	folderPath, err := filepath.Abs(outputFolder)
	if err != nil {
		return fmt.Errorf("failed to access output path: %q: %w", outputFolder, err)
	}

	if outputFolder != "" {
		if exits, _ := afero.Exists(fs, folderPath); !exits {
			err = fs.Mkdir(folderPath, 0777)
			if err != nil {
				return fmt.Errorf("failed to create output folder: %q", folderPath)
			}
		}
	}

	file := filepath.Join(folderPath, filename)

	exists, err := afero.Exists(fs, file)
	if err != nil {
		return fmt.Errorf("failed to check if file %q exists: %w", filename, err)
	}

	if exists {
		time := timeutils.TimeAnchor().Format("20060102-150405")
		var newFileName string
		if lastDot := strings.LastIndex(filename, "."); lastDot > -1 {
			newFileName = fmt.Sprintf("%s_%s%s", filename[:lastDot], time, filename[lastDot:])
		} else {
			newFileName = fmt.Sprintf("%s_%s", filename, time)
		}

		newFile := filepath.Join(folderPath, newFileName)
		log.WithFields(field.F("file", newFile), field.F("existingFile", filename)).Debug("Output file %q already exists, creating %q instead", filename, newFile)
		file = newFile
	}

	err = afero.WriteFile(fs, file, content, 0666)
	if err != nil {
		return fmt.Errorf("failed to create delete file %q: %w", file, err)
	}
	log.WithFields(field.F("file", file)).Info("Delete file written to %q", file)

	return nil
}

func filterProjects(projects []project.Project, projectsToUse []string) ([]project.Project, error) {
	if len(projectsToUse) == 0 {
		return projects, nil
	}
	var filteredProjects []project.Project
	for _, id := range projectsToUse {
		for _, p := range projects {
			if p.Id == id {
				filteredProjects = append(filteredProjects, p)
				break
			}
		}
	}

	if len(filteredProjects) == 0 {
		return nil, fmt.Errorf("requested projects %v not found in manifest", projectsToUse)
	}

	return filteredProjects, nil
}

func generateDeleteFileContent(projects []project.Project, specificEnvironments []string, apis api.APIs) ([]byte, error) {

	log.Info("Generating delete file...")

	var entries []persistence.DeleteEntry
	if len(specificEnvironments) == 0 {
		entries = generateDeleteEntries(projects, apis)
	} else {
		entries = generateDeleteEntriesForEnvironments(projects, specificEnvironments, apis)
	}

	f := persistence.FullFileDefinition{DeleteEntries: entries}
	b, err := yaml.Marshal(&f)
	if err != nil {
		return nil, fmt.Errorf("failed to marshall delete file definition to YAML: %w", err)
	}

	return b, nil
}

func generateDeleteEntries(projects []project.Project, apis api.APIs) []persistence.DeleteEntry {
	entries := make(map[persistence.DeleteEntry]struct{}) // set to ensure cfgs without environment overwrites are only added once

	for _, p := range projects {
		log.Info("Adding delete entries for project %q...", p.Id)

		p.ForEveryConfigDo(func(c config.Config) {
			entry, err := createDeleteEntry(c, apis)
			if err != nil {
				log.WithFields(field.Error(err)).Warn("Failed to automatically create delete entry for %q: %s", c.Coordinate, err)
				return
			}
			entries[entry] = struct{}{}
		})
	}

	return maps.Keys(entries)
}

func generateDeleteEntriesForEnvironments(projects []project.Project, specificEnvironments []string, apis api.APIs) []persistence.DeleteEntry {
	entries := make(map[persistence.DeleteEntry]struct{}) // set to ensure cfgs without environment overwrites are only added once

	for _, p := range projects {
		for _, env := range specificEnvironments {
			log.Info("Adding delete entries for project %q and environment %q...", p.Id, env)
			p.ForEveryConfigInEnvironmentDo(env, func(c config.Config) {
				entry, err := createDeleteEntry(c, apis)
				if err != nil {
					log.WithFields(field.Error(err)).Warn("Failed to automatically create delete entry for %q: %s", c.Coordinate, err)
					return
				}
				entries[entry] = struct{}{}
			})
		}
	}

	return maps.Keys(entries)
}

func createDeleteEntry(c config.Config, apis api.APIs) (persistence.DeleteEntry, error) {
	if apis.Contains(c.Coordinate.Type) {
		return createConfigAPIEntry(c)
	}

	return persistence.DeleteEntry{
		Project:  c.Coordinate.Project,
		Type:     c.Coordinate.Type,
		ConfigId: c.Coordinate.ConfigId,
	}, nil
}

func createConfigAPIEntry(c config.Config) (persistence.DeleteEntry, error) {
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

	return persistence.DeleteEntry{
		Type:       c.Coordinate.Type,
		ConfigName: name,
	}, nil
}
