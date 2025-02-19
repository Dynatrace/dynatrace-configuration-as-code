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

package dependencygraph

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/errutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/multierror"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/timeutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/graph"
	manifestloader "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/loader"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
	"github.com/spf13/afero"
	"path/filepath"
)

// ExportError is returned in case any error occurs while creating a dependency graph file
type ExportError struct {
	message string
	// Reason is the underlying error that occurred
	Reason error `json:"reason"`
	// ManifestFile the export failed for
	ManifestFile string `json:"manifestFile"`
	// Environment the dependency graph failed to be exported for - omitted if the error is not specific to an environment
	Environment string `json:"environment,omitempty"`
	// Filepath of the file that failed to be created - omitted if the error is not related to a file
	Filepath string `json:"filepath,omitempty"`
}

func (e ExportError) Error() string {
	return fmt.Sprintf("%s: %v", e.message, e.Reason)
}

func writeGraphFiles(ctx context.Context, fs afero.Fs, manifestPath string, environmentNames []string, environmentGroups []string, outputFolder string, writeJSONIDs bool) error {

	m, errs := manifestloader.Load(&manifestloader.Context{
		Fs:           fs,
		ManifestPath: manifestPath,
		Environments: environmentNames,
		Groups:       environmentGroups,
		Opts: manifestloader.Options{
			DoNotResolveEnvVars:      true,
			RequireEnvironmentGroups: true,
		},
	})
	if len(errs) > 0 {
		errutils.PrintErrors(errs)
		return ExportError{
			ManifestFile: manifestPath,
			message:      fmt.Sprintf("failed to load manifest %q", manifestPath),
			Reason:       multierror.New(errs...),
		}
	}

	projects, errs := project.LoadProjects(ctx, fs, project.ProjectLoaderContext{
		KnownApis:       api.NewAPIs().GetApiNameLookup(),
		WorkingDir:      filepath.Dir(manifestPath),
		Manifest:        m,
		ParametersSerde: config.DefaultParameterParsers,
	}, nil)

	if len(errs) > 0 {
		errutils.PrintErrors(errs)
		return ExportError{
			ManifestFile: manifestPath,
			message:      "failed to load projects",
			Reason:       multierror.New(errs...),
		}
	}

	var opts []graph.NodeOption
	if writeJSONIDs {
		log.Debug("Encoding DOT Node IDs as JSON")
		opts = append(opts, func(n *graph.ConfigNode) {
			s, err := json.Marshal(n.Config.Coordinate)
			if err == nil {
				n.DOTEncoding = string(s)
			} else {
				log.WithFields(field.Coordinate(n.Config.Coordinate)).Error("Failed to encode Node ID as JSON: %v", err)
				n.DOTEncoding = "{}"
			}
		})
	}

	graphs := graph.New(projects, m.Environments.Names(), opts...)

	folderPath, err := filepath.Abs(outputFolder)
	if err != nil {
		return ExportError{
			ManifestFile: manifestPath,
			message:      fmt.Sprintf("failed to access output path %q", outputFolder),
			Reason:       multierror.New(errs...),
		}
	}

	if outputFolder != "" {
		if exits, _ := afero.Exists(fs, folderPath); !exits {
			err = fs.Mkdir(folderPath, 0777)
			if err != nil {
				return ExportError{
					ManifestFile: manifestPath,
					message:      fmt.Sprintf("failed to create output folder: %q", folderPath),
					Reason:       err,
				}
			}
		}
	}

	for _, e := range m.Environments.Names() {
		b, err := graphs.EncodeToDOT(e)
		if err != nil {
			return ExportError{
				ManifestFile: manifestPath,
				Environment:  e,
				message:      fmt.Sprintf("failed to encode dependency graph to DOT for environment %q", e),
				Reason:       err,
			}
		}
		file := filepath.Join(folderPath, fmt.Sprintf("dependency_graph_%s.dot", e))

		exists, err := afero.Exists(fs, file)
		if err != nil {
			return ExportError{
				ManifestFile: manifestPath,
				Environment:  e,
				Filepath:     file,
				message:      fmt.Sprintf("\"failed to validate if output file %q already exists", file),
				Reason:       err,
			}
		}
		if exists {
			time := timeutils.TimeAnchor().Format("20060102-150405")
			newFile := filepath.Join(folderPath, fmt.Sprintf("dependency_graph_%s_%s.dot", e, time))
			log.WithFields(field.F("file", newFile), field.F("existingFile", file)).Debug("Output file %q already exists, creating %q instead", file, newFile)
			file = newFile
		}

		err = afero.WriteFile(fs, file, b, 0666)
		if err != nil {
			return ExportError{
				ManifestFile: manifestPath,
				Environment:  e,
				Filepath:     file,
				message:      fmt.Sprintf("failed to create dependency graph file %q", file),
				Reason:       err,
			}
		}
		log.WithFields(field.F("file", file)).Info("Dependency graph for environment %q written to %q", e, file)
	}

	return nil
}
