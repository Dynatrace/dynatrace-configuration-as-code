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
	"os"
	"path/filepath"
	"strings"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/files"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/log"
	"github.com/spf13/afero"
)

type Project interface {
	HasDependencyOn(project Project) bool
	GetConfigs() []config.Config
	GetConfig(id string) (config.Config, error)
	GetId() string
	GetCleanId() (string, error)
}

type ProjectImpl struct {
	Id                string
	Configs           []config.Config
	projectRootFolder string
	getRelFilepath    func(basePath string, targetPath string) (string, error)
}

type projectBuilder struct {
	projectRootFolder string
	projectId         string
	configs           []config.Config
	apis              map[string]api.Api
	configFactory     config.ConfigFactory
	fs                afero.Fs
}

// NewProject loads a new project from folder. Returns either project or a reading/sorting error respectively.
func NewProject(fs afero.Fs, fullQualifiedProjectFolderName string, projectFolderName string, apis map[string]api.Api, projectRootFolder string) (Project, error) {

	var configs = make([]config.Config, 0)

	// standardize projectRootFolder
	// trim path separator from projectRoot
	sanitizedProjectRootFolder := strings.TrimRight(projectRootFolder, string(os.PathSeparator))

	builder := projectBuilder{
		projectRootFolder: sanitizedProjectRootFolder,
		projectId:         fullQualifiedProjectFolderName,
		configs:           configs,
		apis:              apis,
		configFactory:     config.NewConfigFactory(),
		fs:                fs,
	}
	err := builder.readFolder(fullQualifiedProjectFolderName, true)
	if err != nil {
		//debug log here?
		return nil, err
	}

	err = builder.sortConfigsAccordingToDependencies()
	if err != nil {
		//debug log here?
		return nil, err
	}

	warnIfProjectNameClashesWithApiName(projectFolderName, apis, sanitizedProjectRootFolder)

	return &ProjectImpl{
		Id:                fullQualifiedProjectFolderName,
		Configs:           builder.configs,
		projectRootFolder: projectRootFolder,
		getRelFilepath:    filepath.Rel,
	}, nil
}

func warnIfProjectNameClashesWithApiName(projectFolderName string, apis map[string]api.Api, projectRootFolder string) {

	lowerCaseProjectFolderName := strings.ToLower(projectFolderName)
	_, ok := apis[lowerCaseProjectFolderName]
	if ok {
		log.Warn("Project %s in folder %s clashes with API name %s. Consider using a different name for your project.", projectFolderName, projectRootFolder, lowerCaseProjectFolderName)
	}
}

func (p *projectBuilder) readFolder(folder string, isProjectRoot bool) error {
	filesInFolder, err := afero.ReadDir(p.fs, folder)

	if util.CheckError(err, "Folder "+folder+" could not be read") {
		return err
	}

	for _, file := range filesInFolder {

		fullFileName := filepath.Join(folder, file.Name())

		if file.IsDir() {
			err = p.readFolder(fullFileName, false)
			if err != nil {
				return err
			}
		} else if !isProjectRoot && files.IsYamlFileExtension(file.Name()) {
			err = p.processYaml(fullFileName)
		}
	}
	return err
}

func (p *projectBuilder) processYaml(filename string) error {

	log.Debug("Processing file: " + filename)

	bytes, err := afero.ReadFile(p.fs, filename)

	if util.CheckError(err, "Error while reading file "+filename) {
		return err
	}

	properties, err := util.UnmarshalYaml(string(bytes), filename)
	if util.CheckError(err, "Error while converting file "+filename) {
		return err
	}

	err, folderPath := p.removeYamlFileFromPath(filename)
	if util.CheckError(err, "Error while stripping yaml from file path "+filename) {
		return err
	}

	err = p.processConfigSection(properties, folderPath)

	return err
}

func (p *projectBuilder) processConfigSection(properties map[string]map[string]string, folderPath string) error {

	templates, ok := properties["config"]
	if !ok {
		log.Error("Property 'config' was not available")
		return fmt.Errorf("property 'config' was not available")
	}

	for configName, location := range templates {

		location = p.standardizeLocation(location, folderPath)

		err, a := p.getExtendedInformationFromLocation(location)
		if util.CheckError(err, "Could not find API fom location") {
			return err
		}

		c, err := p.configFactory.NewConfig(p.fs, configName, p.projectId, location, properties, a)
		if util.CheckError(err, "Could not create config"+configName) {
			return err
		}

		if err != nil {
			return err
		}

		p.configs = append(p.configs, c)
	}
	return nil
}

// standardizeLocation aims to standardize the location of the passed json file
// When it is called with an absolute path (starting with /), we simply strip the "/" away
// Otherwise we assume that the location is relative to the given yaml - so it needs to pe prepended with the folder
// the yaml file is located in
func (p *projectBuilder) standardizeLocation(location string, folderPath string) string {

	if strings.HasPrefix(location, string(os.PathSeparator)) {
		// add project root to location
		location = filepath.Join(p.projectRootFolder, location)
	} else {
		// add folder + location
		location = filepath.Join(folderPath, location)
	}
	return location
}

func (p *projectBuilder) getExtendedInformationFromLocation(location string) (err error, api api.Api) {

	return p.getConfigTypeFromLocation(location)
}

// Strips the "XXX.yaml" from the path
// example: input is "project/dashboards/config.yaml"
//
//	output should be "project/dashboards"
func (p *projectBuilder) removeYamlFileFromPath(location string) (error, string) {

	split := strings.Split(location, string(os.PathSeparator))
	if len(split) <= 1 {
		return fmt.Errorf("path %s too short", location), ""
	}

	return nil, strings.Join(split[:len(split)-1], string(os.PathSeparator))
}

func (p *projectBuilder) getConfigTypeFromLocation(location string) (error, api.Api) {

	split := strings.Split(location, string(os.PathSeparator))
	if len(split) <= 1 {
		return fmt.Errorf("path %s too short", location), nil
	}

	// iterate from end of path:
	for i := len(split) - 2; i >= 0; i-- {

		potentialApi := split[i]
		a, ok := p.apis[potentialApi]
		if ok {
			return nil, a
		}
	}

	return fmt.Errorf("API was unknown. Not found in %s", location), nil
}

func (p *projectBuilder) sortConfigsAccordingToDependencies() error {

	configs, err := sortConfigurations(p.configs)
	if err == nil {
		p.configs = configs
	}
	return err
}

// GetConfigs returns the configs for this project
func (p *ProjectImpl) GetConfigs() []config.Config {
	return p.Configs
}

// GetConfig searches for a config with the given id in the current project
// If no such config is found, an error is returned
func (p *ProjectImpl) GetConfig(id string) (config config.Config, err error) {
	for _, conf := range p.GetConfigs() {
		if id == conf.GetId() {
			return conf, err
		}
	}

	return config, fmt.Errorf("config with id %s not found", id)
}

// GetId returns the id for this project
func (p *ProjectImpl) GetId() string {
	return p.Id
}

// GetCleanId returns a sanitized project id, cleaned from all path attributes
func (p *ProjectImpl) GetCleanId() (string, error) {
	return p.getRelFilepath(p.projectRootFolder, p.Id)
}

// HasDependencyOn checks if one project depends on the given parameter config
// Having a dependency means, that the project having the dependency needs to be applied AFTER the project it depends on
func (p *ProjectImpl) HasDependencyOn(project Project) bool {

	for _, myConfig := range p.Configs {
		for _, otherConfig := range project.GetConfigs() {
			if myConfig.HasDependencyOn(otherConfig) {
				return true
			}
		}
	}
	return false
}
