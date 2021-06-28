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

package writer

import (
	"fmt"
	"path/filepath"

	config "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/manifest"
	project "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project/v2"
	"github.com/spf13/afero"
)

type WriterContext struct {
	Fs                 afero.Fs
	SourceManifestPath string
	OutputDir          string
	ManifestName       string
	ParametersSerde    map[string]parameter.ParameterSerDe
}

func WriteToDisk(context *WriterContext, manifestToWrite manifest.Manifest, projects []project.Project) []error {
	sanitizedOutputDir := filepath.Clean(context.OutputDir)
	err := context.Fs.MkdirAll(sanitizedOutputDir, 0777)

	if err != nil {
		return []error{err}
	}

	err = manifest.WriteManifest(&manifest.ManifestWriterContext{
		Fs:           context.Fs,
		ManifestPath: filepath.Join(sanitizedOutputDir, context.ManifestName),
	}, manifestToWrite)

	if err != nil {
		return []error{err}
	}

	return writeProjects(context, manifestToWrite.Projects, projects)
}

func writeProjects(context *WriterContext, projectDefinitions map[string]manifest.ProjectDefinition,
	projects []project.Project) []error {
	sanitizedOutputDir := filepath.Clean(context.OutputDir)
	err := context.Fs.MkdirAll(sanitizedOutputDir, 0777)

	if err != nil {
		return []error{err}
	}

	var errors []error

	for _, p := range projects {
		definition, found := projectDefinitions[p.Id]

		if !found {
			fmt.Printf("WARNING: no project definition found for `%s`. skipping....\n", p.Id)
			continue
		}

		configs := collectAllConfigs(p)

		errs := config.WriteConfigs(&config.WriterContext{
			Fs:                             context.Fs,
			OutputFolder:                   context.OutputDir,
			ProjectFolder:                  definition.Path,
			ParametersSerde:                context.ParametersSerde,
			UseShortSyntaxForSpecialParams: true,
		}, configs)

		errors = append(errors, errs...)
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

func collectAllConfigs(p project.Project) (result []config.Config) {
	for _, configsPerApi := range p.Configs {
		for _, configs := range configsPerApi {
			result = append(result, configs...)
		}
	}

	return result
}
