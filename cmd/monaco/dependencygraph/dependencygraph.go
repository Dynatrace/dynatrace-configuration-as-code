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

package dependencygraph

import (
	"errors"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/errutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/timeutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/graph"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
	"github.com/spf13/afero"
	"path/filepath"
)

func writeGraphFiles(fs afero.Fs, manifestPath string, environmentNames []string, environmentGroups []string, outputFolder string) error {

	m, errs := manifest.LoadManifest(&manifest.LoaderContext{
		Fs:           fs,
		ManifestPath: manifestPath,
		Environments: environmentNames,
		Groups:       environmentGroups,
	})
	if len(errs) > 0 {
		errutils.PrintErrors(errs)
		return fmt.Errorf("failed to load manifest %q: %w", manifestPath, errors.Join(errs...))
	}

	projects, errs := project.LoadProjects(fs, project.ProjectLoaderContext{
		KnownApis:       api.NewAPIs().GetApiNameLookup(),
		WorkingDir:      filepath.Dir(manifestPath),
		Manifest:        m,
		ParametersSerde: config.DefaultParameterParsers,
	})

	if len(errs) > 0 {
		errutils.PrintErrors(errs)
		return fmt.Errorf("failed to load projects")
	}

	graphs := graph.New(projects, m.Environments.Names())

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

	for _, e := range m.Environments.Names() {
		b, err := graphs.EncodeToDOT(e)
		if err != nil {
			return fmt.Errorf("failed to encode dependency graph to DOT for environment %q: %w", e, err)
		}
		file := filepath.Join(folderPath, fmt.Sprintf("dependency_graph_%s.dot", e))

		exists, err := afero.Exists(fs, file)
		if err != nil {
			return fmt.Errorf("failed to validate if output file %q already exists: %w", file, err)
		}
		if exists {
			time := timeutils.TimeAnchor().Format("20060102-150405")
			newFile := filepath.Join(folderPath, fmt.Sprintf("dependency_graph_%s_%s.dot", e, time))
			log.Debug("Output file %q already exists, creating %q instead", file, newFile)
			file = newFile
		}

		err = afero.WriteFile(fs, file, b, 0666)
		if err != nil {
			return fmt.Errorf("failed to create dependency graph file %q: %w", file, err)
		}
		log.Info("Dependency graph for environment %q written to %q", e, file)
	}

	return nil
}
