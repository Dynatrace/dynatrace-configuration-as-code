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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/persistence"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
	"path/filepath"
)

func createDeleteFile(fs afero.Fs, manifestPath string, projectNames []string, filename, outputFolder string) error {

	m, errs := manifest.LoadManifest(&manifest.LoaderContext{
		Fs:           fs,
		ManifestPath: manifestPath,
		Opts: manifest.LoaderOptions{
			DontResolveEnvVars: true,
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

	env := m.Environments.Names()[0] // take the first environment, as overwrites do not impact the configs that exist (as skipped configs are still loaded)
	content, err := generateDeleteFileContent(env, projects, apis)
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
		newFile := filepath.Join(folderPath, fmt.Sprintf("%s_%s", filename, time))
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

func generateDeleteFileContent(environment string, projects []project.Project, apis api.APIs) ([]byte, error) {

	log.Info("Generating delete file...")

	var entries []persistence.DeleteEntry

	for _, p := range projects {
		log.Info("Adding delete entries for project %q...", p.Id)
		cfgsPerType := p.Configs[environment]
		for _, cfgs := range cfgsPerType {
			for _, c := range cfgs {
				if apis.Contains(c.Coordinate.Type) {
					val, err := c.Parameters[config.NameParameter].ResolveValue(parameter.ResolveContext{ParameterName: config.NameParameter})
					if err != nil {
						log.WithFields(field.Error(err)).Warn("Failed to automatically create delete entry for %q - unable to get name: %v", c.Coordinate, err)
						continue
					}
					name, ok := val.(string)
					if !ok {
						log.Warn("Failed to automatically create delete entry for %q - value of 'name' parameter '%v' was not a string", c.Coordinate, val)
						continue
					}

					if name == "" {
						log.Warn("Failed to automatically create delete entry for %q - 'name' parameter was empty", c.Coordinate)
						continue
					}

					entries = append(entries, persistence.DeleteEntry{
						Type:       c.Coordinate.Type,
						ConfigName: name,
					})
				} else {
					entries = append(entries, persistence.DeleteEntry{
						Project:  c.Coordinate.Project,
						Type:     c.Coordinate.Type,
						ConfigId: c.Coordinate.ConfigId,
					})
				}
			}
		}
	}

	f := persistence.FullFileDefinition{DeleteEntries: entries}
	b, err := yaml.Marshal(&f)
	if err != nil {
		return nil, fmt.Errorf("failed to marshall delete file definition to YAML: %w", err)
	}

	return b, nil
}
