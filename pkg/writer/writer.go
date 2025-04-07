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
	"path/filepath"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	manifestwriter "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/writer"
	configwriter "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/persistence/config/writer"

	"github.com/spf13/afero"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
)

type WriterContext struct {
	Fs                 afero.Fs
	SourceManifestPath string
	OutputDir          string
	ManifestName       string
	ParametersSerde    map[string]parameter.ParameterSerDe
}

func WriteToDisk(context *WriterContext, manifestToWrite manifest.Manifest, environments []project.Environment) []error {
	sanitizedOutputDir := filepath.Clean(context.OutputDir)
	err := context.Fs.MkdirAll(sanitizedOutputDir, 0777)

	if err != nil {
		return []error{err}
	}

	err = manifestwriter.Write(&manifestwriter.Context{
		Fs:           context.Fs,
		ManifestPath: filepath.Join(sanitizedOutputDir, context.ManifestName),
	}, manifestToWrite)

	if err != nil {
		return []error{err}
	}

	return writeProjects(context, manifestToWrite.Projects, environments)
}

func writeProjects(context *WriterContext, projectDefinitions manifest.ProjectDefinitionByProjectID,
	environments []project.Environment) []error {
	sanitizedOutputDir := filepath.Clean(context.OutputDir)
	err := context.Fs.MkdirAll(sanitizedOutputDir, 0777)

	if err != nil {
		return []error{err}
	}

	var errors []error

	for _, p := range environments {
		definition, found := projectDefinitions[p.Name]

		if !found {
			log.Warn("no project definition found for `%s`. skipping....\n", p.Name)
			continue
		}

		configs := p.AllConfigs()

		errs := configwriter.WriteConfigs(&configwriter.WriterContext{
			Fs:              context.Fs,
			OutputFolder:    context.OutputDir,
			ProjectFolder:   definition.Path,
			ParametersSerde: context.ParametersSerde,
		}, configs)

		errors = append(errors, errs...)
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}
