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

package project

import (
	"fmt"
	"github.com/pkg/errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
)

// LoadProjectsToDeploy returns a list of projects for deployment
// if projects specified with -p parameter then it takes only those projects and
// it also resolves all project dependencies
// if no -p parameter specified, then it creates a list of all projects
func LoadProjectsToDeploy(specificProjectToDeploy string, apis map[string]api.Api, path string, fileReader util.FileReader) (projectsToDeploy []Project, err error) {
	projectsFolder := filepath.Join(".", path)
	projectsToDeploy = make([]Project, 0)

	util.Log.Debug("Reading projects...")

	// creates list of all available projects
	availableProjectFolders, err := getAllProjectFoldersRecursively(projectsFolder)
	if err != nil {
		return nil, err
	}
	availableProjects := make([]Project, 0)
	for _, fullQualifiedProjectFolderName := range availableProjectFolders {
		util.Log.Debug("  project - %s", fullQualifiedProjectFolderName)
		projectFolderName := extractFolderNameFromFullPath(fullQualifiedProjectFolderName)
		project, err := NewProject(fullQualifiedProjectFolderName, projectFolderName, apis, path, fileReader)
		if err != nil {
			return nil, err
		}
		availableProjects = append(availableProjects, project)
	}

	// return all projects if no projects specified by -p parameter
	// otherwise only add projects specified by parameter
	if specificProjectToDeploy == "" {
		projectsToDeploy = availableProjects
		return returnSortedProjects(projectsToDeploy)
	}

	projectsToDeploy, err = createProjectsListFromFolderList(path, specificProjectToDeploy, projectsFolder, apis, availableProjectFolders, fileReader)
	if err != nil {
		return nil, err
	}

	// goes through the list of projectToDeploy and searches for dependencies
	// it searches the list recursively as long as dependencies are found
	foundDependency := true
	for foundDependency {
		foundDependency = false
		for _, project := range projectsToDeploy {
			for _, availableProject := range availableProjects {
				if project.HasDependencyOn(availableProject) && !isProjectAlreadyAdded(availableProject, projectsToDeploy) {
					projectsToDeploy = append(projectsToDeploy, availableProject)
					foundDependency = true
				}
			}
		}
	}

	return returnSortedProjects(projectsToDeploy)
}

func returnSortedProjects(projectsToDeploy []Project) ([]Project, error) {
	util.Log.Debug("Sorting projects...")
	projectsToDeploy, err := sortProjects(projectsToDeploy)
	if err != nil {
		return nil, err
	}

	return projectsToDeploy, nil
}

// takes project folder parameter and creates []Project slice
// if project specified contains subprojects, then it adds subprojects instead
func createProjectsListFromFolderList(path, specificProjectToDeploy string, projectsFolder string, apis map[string]api.Api, availableProjectFolders []string, fileReader util.FileReader) ([]Project, error) {
	projectsToDeploy := make([]Project, 0)
	multiProjects := strings.Split(specificProjectToDeploy, ",")
	for _, projectFolderName := range multiProjects {

		projectFolderName = strings.TrimSpace(projectFolderName)
		fullQualifiedProjectFolderName := filepath.Join(projectsFolder, projectFolderName)

		// if specified project has subprojects then add them instead
		if !hasSubprojectFolder(fullQualifiedProjectFolderName, availableProjectFolders) {
			_, err := os.Stat(fullQualifiedProjectFolderName)

			if err != nil {
				return nil, errors.WithMessagef(err, "Project %s does not exist!", specificProjectToDeploy)
			}

			newProject, err := NewProject(fullQualifiedProjectFolderName, projectFolderName, apis, path, fileReader)
			if err != nil {
				return nil, err
			}
			projectsToDeploy = append(projectsToDeploy, newProject)
		} else {
			// get list of folders only for this path
			subProjectFolders, err := getAllProjectFoldersRecursively(fullQualifiedProjectFolderName)
			if err != nil {
				return nil, err
			}
			for _, fullQualifiedSubProjectFolderName := range subProjectFolders {

				subProjectFolderName := extractFolderNameFromFullPath(fullQualifiedSubProjectFolderName)
				newProject, err := NewProject(fullQualifiedSubProjectFolderName, subProjectFolderName, apis, path, fileReader)
				if err != nil {
					return nil, err
				}
				projectsToDeploy = append(projectsToDeploy, newProject)
			}
		}

	}
	return projectsToDeploy, nil
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
	for _, p := range projectFolders {
		if strings.HasPrefix(p, projectFolder+string(filepath.Separator)) && p != projectFolder {
			return true
		}
	}
	return false
}

// walks through a path recursively and searches for all folders
// ignores folders with configurations (containing api configs) and hidden folders
// fails if a folder with both sub projects and api configs are found
func getAllProjectFoldersRecursively(path string) ([]string, error) {
	var allProjectsFolders []string
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() && !strings.HasPrefix(path, ".") && !api.ContainsApiName(path) {
			allProjectsFolders = append(allProjectsFolders, path)
			err := subprojectsMixedWithApi(path)
			return err
		}
		return nil
	})
	if err != nil {
		return allProjectsFolders, err
	}

	return filterProjectsWithSubproject(allProjectsFolders), nil
}

func subprojectsMixedWithApi(path string) error {
	apiFound, subprojectFound := false, false
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	dirs, err := f.Readdir(0)
	if err != nil {
		return err
	}
	for _, d := range dirs {
		if api.IsApi(d.Name()) {
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

// Searches for project in projects array and returns true if
// project found. Projects are compared by IDs
func isProjectAlreadyAdded(findProject Project, projects []Project) bool {
	for _, project := range projects {
		if project.GetId() == findProject.GetId() {
			return true
		}
	}
	return false
}
