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

package v1

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"

	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/api"
)

// LoadProjectsToConvert returns a list of projects to be converted to v2
func LoadProjectsToConvert(fs afero.Fs, apis api.APIs, path string) ([]Project, error) {
	_, projects, err := loadAllProjects(fs, apis, path, template.UnmarshalYamlWithoutTemplating)
	return projects, err
}

func loadAllProjects(fs afero.Fs, apis api.APIs, projectsFolder string, unmarshalYaml template.UnmarshalYamlFunc) (projectFolders []string, projects []Project, err error) {
	projectsFolder = filepath.Clean(projectsFolder)

	log.Debug("Reading projects...")

	// creates list of all available projects
	availableProjectFolders, err := getAllProjectFoldersRecursively(fs, apis, projectsFolder)
	if err != nil {
		return nil, nil, err
	}

	availableProjects := make([]Project, 0)
	for _, fullQualifiedProjectFolderName := range availableProjectFolders {
		log.Debug("  project - %s", fullQualifiedProjectFolderName)
		projectFolderName := extractFolderNameFromFullPath(fullQualifiedProjectFolderName)
		project, err := newProject(fs, fullQualifiedProjectFolderName, projectFolderName, apis, projectsFolder, unmarshalYaml)
		if err != nil {
			return nil, nil, err
		}
		availableProjects = append(availableProjects, project)
	}

	return availableProjectFolders, availableProjects, nil
}

func extractFolderNameFromFullPath(fullQualifiedProjectFolderName string) string {

	// split the full qualified sub project folder name into the individual folders:
	folders := strings.Split(fullQualifiedProjectFolderName, string(os.PathSeparator))

	// The last element is the name of the sub folder:
	folderName := folders[len(folders)-1]

	return folderName
}

// removes projects containing subprojects
// needed to prevent duplication of configurations
// e.g. if project x has projects y and z as subprojects, then
// add only projects y and z
func filterProjectsWithSubproject(allProjectFolders []string) []string {
	var list []string
	for _, projectFolder := range allProjectFolders {
		if !hasSubprojectFolder(projectFolder, allProjectFolders) {
			list = append(list, projectFolder)
		}

	}
	return list
}

// checks if project folder contains subproject(s)
func hasSubprojectFolder(projectFolder string, projectFolders []string) bool {
	cleanedProjectFolder := filepath.Clean(projectFolder)

	for _, p := range projectFolders {
		cleanedFolder := filepath.Clean(p)
		if filepath.Dir(cleanedFolder) == cleanedProjectFolder && cleanedFolder != cleanedProjectFolder {
			return true
		}
	}
	return false
}

// walks through a path recursively and searches for all folders
// ignores folders with configurations (containing api configs) and hidden folders
// fails if a folder with both sub projects and api configs are found
func getAllProjectFoldersRecursively(fs afero.Fs, availableApis api.APIs, path string) ([]string, error) {
	var allProjectsFolders []string
	err := afero.Walk(fs, path, func(path string, info os.FileInfo, err error) error {
		if info == nil {
			return fmt.Errorf("Project path does not exist: %s. (This needs to be a relative path from the current directory)", path)
		}
		if info.IsDir() && !isIgnoredPath(path) && !containsApiName(availableApis, path) {
			allProjectsFolders = append(allProjectsFolders, path)
			err := subprojectsMixedWithApi(fs, availableApis, path)
			return err
		}
		return nil
	})
	if err != nil {
		return allProjectsFolders, err
	}

	return filterProjectsWithSubproject(allProjectsFolders), nil
}

// containsApiName tests if part of project folder path contains an API
// folders with API in path are not valid projects
func containsApiName(apis api.APIs, path string) bool {
	for a := range apis {
		if strings.Contains(path, a) {
			return true
		}
	}

	return false
}

func subprojectsMixedWithApi(fs afero.Fs, availableApis api.APIs, path string) error {
	apiFound, subprojectFound := false, false
	if _, err := fs.Open(path); err != nil {
		return err
	}
	dirs, err := afero.ReadDir(fs, path)
	if err != nil {
		return err
	}
	for _, d := range dirs {
		if isIgnoredPath(d.Name()) {
			continue
		}

		if availableApis.Contains(d.Name()) {
			apiFound = true
		} else if d.IsDir() {
			subprojectFound = true
		}

		if apiFound && subprojectFound {
			return fmt.Errorf("found folder with projects and configurations in %s", path)
		}
	}
	return nil
}

// isIgnoredPath checks if the path starts with a dot, or if the current evaluated element starts with a dot
func isIgnoredPath(path string) bool {
	baseName := filepath.Base(path)

	return strings.HasPrefix(path, ".") || strings.HasPrefix(baseName, ".")
}
