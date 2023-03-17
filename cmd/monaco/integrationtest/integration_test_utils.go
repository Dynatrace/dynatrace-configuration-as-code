//go:build integration || integration_v1 || cleanup || download_restore || nightly

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

package integrationtest

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client"
	config "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/manifest"
	project "github.com/dynatrace/dynatrace-configuration-as-code/pkg/project/v2"
	"github.com/spf13/afero"
	"gotest.tools/assert"
	"path/filepath"
	"testing"
)

func CreateDynatraceClient(t *testing.T, environment manifest.EnvironmentDefinition) client.Client {
	envURL := environment.URL.Value

	c, err := client.NewDynatraceClient(client.NewTokenAuthClient(environment.Auth.Token.Value), envURL, client.WithAutoServerVersion())
	assert.NilError(t, err, "failed to create test client")

	return c
}

func LoadManifest(t *testing.T, fs afero.Fs, manifestFile string, specificEnvironment string) manifest.Manifest {
	var specificEnvs []string
	if specificEnvironment != "" {
		specificEnvs = append(specificEnvs, specificEnvironment)
	}

	m, errs := manifest.LoadManifest(&manifest.ManifestLoaderContext{
		Fs:           fs,
		ManifestPath: manifestFile,
		Environments: specificEnvs,
	})
	testutils.FailTestOnAnyError(t, errs, "failed to load manifest")

	return m
}

func LoadProjects(t *testing.T, fs afero.Fs, manifestPath string, loadedManifest manifest.Manifest) []project.Project {
	cwd, err := filepath.Abs(filepath.Dir(manifestPath))
	assert.NilError(t, err)

	projects, errs := project.LoadProjects(fs, project.ProjectLoaderContext{
		KnownApis:       api.NewAPIs().GetApiNameLookup(),
		WorkingDir:      cwd,
		Manifest:        loadedManifest,
		ParametersSerde: config.DefaultParameterParsers,
	})
	testutils.FailTestOnAnyError(t, errs, "loading of projects failed")
	return projects
}
