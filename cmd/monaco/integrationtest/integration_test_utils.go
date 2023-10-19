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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/support"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	manifestloader "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/loader"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
	"github.com/spf13/afero"
	"gotest.tools/assert"
	"path/filepath"
	"testing"
)

// CreateDynatraceClients creates a client set used in e2e tests.
// Note, that the caching mechanism in the client is disabled to eliminate the risk of getting
// wrong information from the cache in cases where we want to get
// resources immediately after they've been created (e.g. to assert that they exist)
func CreateDynatraceClients(t *testing.T, environment manifest.EnvironmentDefinition) *client.ClientSet {
	var clients *client.ClientSet
	var err error
	if environment.Auth.OAuth == nil {
		clients, err = client.CreateClassicClientSet(environment.URL.Value, environment.Auth.Token.Value.Value(), client.ClientOptions{
			SupportArchive:  support.SupportArchive,
			CachingDisabled: true, // disabled to avoid wrong cache reads
		})
	} else {
		clients, err = client.CreatePlatformClientSet(environment.URL.Value, client.PlatformAuth{
			OauthClientID:     environment.Auth.OAuth.ClientID.Value.Value(),
			OauthClientSecret: environment.Auth.OAuth.ClientSecret.Value.Value(),
			Token:             environment.Auth.Token.Value.Value(),
			OauthTokenURL:     environment.Auth.OAuth.GetTokenEndpointValue(),
		}, client.ClientOptions{
			SupportArchive:  support.SupportArchive,
			CachingDisabled: true, // disabled to avoid wrong cache reads
		})
	}
	assert.NilError(t, err, "failed to create test client")
	return clients
}

func LoadManifest(t *testing.T, fs afero.Fs, manifestFile string, specificEnvironment string) manifest.Manifest {
	var specificEnvs []string
	if specificEnvironment != "" {
		specificEnvs = append(specificEnvs, specificEnvironment)
	}

	m, errs := manifestloader.Load(&manifestloader.Context{
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
