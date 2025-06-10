//go:build unit

// @license
// Copyright 2022 Dynatrace LLC
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

package download

import (
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/environment"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/reference"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	manifestloader "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/loader"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project"
)

// compareOptions holds all options we require for the tests to not be flaky.
// E.g. slices may be in any order, template may have any implementation.
// We want to be pragmatic in comparing them - so we define these options to make it very simple.
var compareOptions = []cmp.Option{
	cmp.Comparer(func(a, b template.Template) bool {
		cA, _ := a.Content()
		cB, _ := b.Content()
		return jsonEqual(cA, cB)
	}),
	cmpopts.SortSlices(func(a, b config.Config) bool {
		return strings.Compare(a.Coordinate.String(), b.Coordinate.String()) < 0
	}),
	cmpopts.SortSlices(func(a, b coordinate.Coordinate) bool {
		return strings.Compare(a.String(), b.String()) < 0
	}),
}

func TestDownloadIntegrationSimple(t *testing.T) {
	// GIVEN apis, server responses, file system
	const projectName = "integration-test-1"
	const testBasePath = "test-resources/" + projectName

	// APIs
	fakeApi := api.API{ID: "fake-id", URLPath: "/fake-id", PropertyNameOfGetAllResponse: api.StandardApiPropertyNameOfGetAllResponse}
	apiMap := api.APIs{
		fakeApi.ID: fakeApi,
	}

	// Responses
	responses := map[string]string{
		"/fake-id":      "fake-api/__LIST.json",
		"/fake-id/id-1": "fake-api/id-1.json",
	}

	// Server
	server := dtclient.NewIntegrationTestServer(t, testBasePath, responses)

	fs := afero.NewMemMapFs()

	configClient, err := dtclient.NewClassicConfigClientForTesting(server.URL, server.Client())
	require.NoError(t, err)

	settingsClient, err := dtclient.NewPlatformSettingsClientForTesting(server.URL, server.Client())
	require.NoError(t, err)

	// WHEN we download everything
	err = doDownloadConfigs(t.Context(), fs, &client.ClientSet{ConfigClient: configClient, SettingsClient: settingsClient}, apiMap, setupTestingDownloadOptions(t, server, projectName))
	assert.NoError(t, err)

	// THEN we can load the project again and verify its content
	projects, errs := loadDownloadedProjects(t, fs, apiMap)
	if len(errs) != 0 {
		for _, err := range errs {
			t.Errorf("%v", err)
		}
		return
	}

	assert.Len(t, projects, 1)
	p := projects[0]
	assert.Equal(t, p.Id, projectName)
	assert.Len(t, p.Configs, 1)

	configs, found := p.Configs[projectName]
	assert.True(t, found)
	assert.Len(t, configs, 1)

	var _ config.Type = config.ClassicApiType{}

	diff := cmp.Diff(configs, project.ConfigsPerType{
		fakeApi.ID: []config.Config{
			{
				Coordinate: coordinate.Coordinate{Project: projectName, Type: fakeApi.ID, ConfigId: "id-1"},
				Skip:       false,
				Parameters: map[string]parameter.Parameter{
					"name": &value.ValueParameter{Value: "Test-1"},
				},
				Group:       "default",
				Environment: projectName,
				Template:    template.NewInMemoryTemplate("template-1", `{"custom-response": true, "name": "{{.name}}"}`),
				Type:        config.ClassicApiType{Api: fakeApi.ID},
			},
		},
	}, compareOptions...)

	if diff != "" {
		assert.Fail(t, "Objects do not match match: %s", diff)
	}
}

func TestDownloadIntegrationWithReference(t *testing.T) {
	// GIVEN apis, server responses, file system
	const projectName = "integration-test-2"
	const testBasePath = "test-resources/" + projectName

	// APIs
	fakeApi := api.API{ID: "fake-id", URLPath: "/fake-id", PropertyNameOfGetAllResponse: api.StandardApiPropertyNameOfGetAllResponse}
	apiMap := api.APIs{
		fakeApi.ID: fakeApi,
	}

	// Responses
	responses := map[string]string{
		"/fake-id":      "fake-api/__LIST.json",
		"/fake-id/id-1": "fake-api/id-1.json",
		"/fake-id/id-2": "fake-api/id-2.json",
	}

	// Server
	server := dtclient.NewIntegrationTestServer(t, testBasePath, responses)

	fs := afero.NewMemMapFs()

	configClient, err := dtclient.NewClassicConfigClientForTesting(server.URL, server.Client())
	require.NoError(t, err)

	settingsClient, err := dtclient.NewPlatformSettingsClientForTesting(server.URL, server.Client())
	require.NoError(t, err)

	// WHEN we download everything
	err = doDownloadConfigs(t.Context(), fs, &client.ClientSet{ConfigClient: configClient, SettingsClient: settingsClient}, apiMap, setupTestingDownloadOptions(t, server, projectName))
	assert.NoError(t, err)

	// THEN we can load the project again and verify its content
	projects, errs := loadDownloadedProjects(t, fs, apiMap)
	if len(errs) != 0 {
		for _, err := range errs {
			t.Errorf("%v", err)
		}
		return
	}

	assert.Len(t, projects, 1)
	p := projects[0]
	assert.Equal(t, p.Id, projectName)
	assert.Len(t, p.Configs, 1)

	configs, found := p.Configs[projectName]
	assert.True(t, found)

	diff := cmp.Diff(configs, project.ConfigsPerType{
		fakeApi.ID: []config.Config{
			{
				Coordinate: coordinate.Coordinate{Project: projectName, Type: fakeApi.ID, ConfigId: "id-1"},
				Skip:       false,
				Parameters: map[string]parameter.Parameter{
					"name": &value.ValueParameter{Value: "Test-1"},
				},
				Group:       "default",
				Environment: projectName,
				Template:    template.NewInMemoryTemplate("template-1", `{"custom-response": true, "name": "{{.name}}"}`),
				Type:        config.ClassicApiType{Api: "fake-id"},
			},
			{
				Coordinate: coordinate.Coordinate{Project: projectName, Type: fakeApi.ID, ConfigId: "id-2"},
				Skip:       false,
				Parameters: map[string]parameter.Parameter{
					"name":            &value.ValueParameter{Value: "Test-2"},
					"fakeid__id1__id": reference.New(projectName, fakeApi.ID, "id-1", "id"),
				},
				Group:       "default",
				Environment: projectName,
				Template:    template.NewInMemoryTemplate("template-2", `{"custom-response": true, "name": "{{.name}}", "reference-to-id1": "{{.fakeid__id1__id}}"}`),
				Type:        config.ClassicApiType{Api: "fake-id"},
			},
		},
	}, compareOptions...)

	if diff != "" {
		assert.Fail(t, "Objects do not match: %s", diff)
	}

}

func TestDownloadIntegrationWithMultipleApisAndReferences(t *testing.T) {
	// GIVEN apis, server responses, file system
	const projectName = "integration-test-3"
	const testBasePath = "test-resources/" + projectName

	// APIs
	fakeApi1 := api.API{ID: "fake-id-1", URLPath: "/fake-api-1", PropertyNameOfGetAllResponse: api.StandardApiPropertyNameOfGetAllResponse}
	fakeApi2 := api.API{ID: "fake-id-2", URLPath: "/fake-api-2", PropertyNameOfGetAllResponse: api.StandardApiPropertyNameOfGetAllResponse}
	fakeApi3 := api.API{ID: "fake-id-3", URLPath: "/fake-api-3", PropertyNameOfGetAllResponse: api.StandardApiPropertyNameOfGetAllResponse}
	apiMap := api.APIs{
		fakeApi1.ID: fakeApi1,
		fakeApi2.ID: fakeApi2,
		fakeApi3.ID: fakeApi3,
	}

	// Responses
	responses := map[string]string{
		"/fake-api-1":      "fake-api-1/__LIST.json",
		"/fake-api-1/id-1": "fake-api-1/id-1.json",
		"/fake-api-1/id-2": "fake-api-1/id-2.json",

		"/fake-api-2":      "fake-api-2/__LIST.json",
		"/fake-api-2/id-3": "fake-api-2/id-3.json",
		"/fake-api-2/id-4": "fake-api-2/id-4.json",

		"/fake-api-3":      "fake-api-3/__LIST.json",
		"/fake-api-3/id-5": "fake-api-3/id-5.json",
	}

	// Server
	server := dtclient.NewIntegrationTestServer(t, testBasePath, responses)
	fs := afero.NewMemMapFs()

	configClient, err := dtclient.NewClassicConfigClientForTesting(server.URL, server.Client())
	require.NoError(t, err)

	settingsClient, err := dtclient.NewPlatformSettingsClientForTesting(server.URL, server.Client())
	require.NoError(t, err)

	// WHEN we download everything
	err = doDownloadConfigs(t.Context(), fs, &client.ClientSet{ConfigClient: configClient, SettingsClient: settingsClient}, apiMap, setupTestingDownloadOptions(t, server, projectName))

	assert.NoError(t, err)

	projects, errs := loadDownloadedProjects(t, fs, apiMap)
	if len(errs) != 0 {
		for _, err := range errs {
			t.Errorf("%v", err)
		}
		return
	}

	assert.Len(t, projects, 1)
	p := projects[0]
	assert.Equal(t, p.Id, projectName)
	assert.Len(t, p.Configs, 1)

	configs, found := p.Configs[projectName]
	assert.True(t, found)

	diff := cmp.Diff(configs, project.ConfigsPerType{
		fakeApi1.ID: []config.Config{
			{
				Coordinate: coordinate.Coordinate{Project: projectName, Type: fakeApi1.ID, ConfigId: "id-1"},
				Skip:       false,
				Parameters: map[string]parameter.Parameter{
					"name": &value.ValueParameter{Value: "Test-1"},
				},
				Group:       "default",
				Environment: projectName,
				Template:    template.NewInMemoryTemplate("id", `{"custom-response": true, "name": "{{.name}}"}`),
				Type:        config.ClassicApiType{Api: "fake-id-1"},
			},
			{
				Coordinate: coordinate.Coordinate{Project: projectName, Type: fakeApi1.ID, ConfigId: "id-2"},
				Skip:       false,
				Parameters: map[string]parameter.Parameter{
					"name":             &value.ValueParameter{Value: "Test-2"},
					"fakeid1__id1__id": reference.New(projectName, fakeApi1.ID, "id-1", "id"),
				},
				Group:       "default",
				Environment: projectName,
				Template:    template.NewInMemoryTemplate("id", `{"custom-response": false, "name": "{{.name}}", "reference-to-id1": "{{.fakeid1__id1__id}}"}`),
				Type:        config.ClassicApiType{Api: "fake-id-1"},
			},
		},
		fakeApi2.ID: []config.Config{
			{
				Coordinate: coordinate.Coordinate{Project: projectName, Type: fakeApi2.ID, ConfigId: "id-3"},
				Skip:       false,
				Parameters: map[string]parameter.Parameter{
					"name":             &value.ValueParameter{Value: "Test-3"},
					"fakeid1__id1__id": reference.New(projectName, fakeApi1.ID, "id-1", "id"),
				},
				Group:       "default",
				Environment: projectName,
				Template:    template.NewInMemoryTemplate("id", `{"custom-response": "No!", "name": "{{.name}}", "subobject": {"something": "{{.fakeid1__id1__id}}"}}`),
				Type:        config.ClassicApiType{Api: "fake-id-2"},
			},
			{
				Coordinate: coordinate.Coordinate{Project: projectName, Type: fakeApi2.ID, ConfigId: "id-4"},
				Skip:       false,
				Parameters: map[string]parameter.Parameter{
					"name":             &value.ValueParameter{Value: "Test-4"},
					"fakeid2__id3__id": reference.New(projectName, fakeApi2.ID, "id-3", "id"),
				},
				Group:       "default",
				Environment: projectName,
				Template:    template.NewInMemoryTemplate("id", `{"custom-response": true, "name": "{{.name}}", "reference-to-id3": "{{.fakeid2__id3__id}}"}`),
				Type:        config.ClassicApiType{Api: "fake-id-2"},
			},
		},
		fakeApi3.ID: []config.Config{
			{
				Coordinate: coordinate.Coordinate{Project: projectName, Type: fakeApi3.ID, ConfigId: "id-5"},
				Skip:       false,
				Parameters: map[string]parameter.Parameter{
					"name":             &value.ValueParameter{Value: "Test-5"},
					"fakeid1__id2__id": reference.New(projectName, fakeApi1.ID, "id-2", "id"),
					"fakeid2__id4__id": reference.New(projectName, fakeApi2.ID, "id-4", "id"),
				},
				Group:       "default",
				Environment: projectName,
				Template:    template.NewInMemoryTemplate("id", `{"name": "{{.name}}", "custom-response": true, "reference-to-id6-of-another-api": ["{{.fakeid2__id4__id}}" ,{"o":  "{{.fakeid1__id2__id}}"}]}`),
				Type:        config.ClassicApiType{Api: "fake-id-3"},
			},
		},
	}, compareOptions...)

	if diff != "" {
		assert.Fail(t, "Objects do not match: %s", diff)
	}
}

func TestDownloadIntegrationSingletonConfig(t *testing.T) {
	// GIVEN apis, server responses, file system
	const projectName = "integration-test-singleton"
	const testBasePath = "test-resources/" + projectName

	// APIs
	fakeApi := api.API{ID: "fake-id", URLPath: "/fake-id", SingleConfiguration: true}
	apiMap := api.APIs{
		fakeApi.ID: fakeApi,
	}

	// Responses
	responses := map[string]string{
		"/fake-id": "fake-api/singleton.json",
	}

	// Server
	server := dtclient.NewIntegrationTestServer(t, testBasePath, responses)

	fs := afero.NewMemMapFs()

	configClient, err := dtclient.NewClassicConfigClientForTesting(server.URL, server.Client())
	require.NoError(t, err)

	settingsClient, err := dtclient.NewPlatformSettingsClientForTesting(server.URL, server.Client())
	require.NoError(t, err)

	// WHEN we download everything
	err = doDownloadConfigs(t.Context(), fs, &client.ClientSet{ConfigClient: configClient, SettingsClient: settingsClient}, apiMap, setupTestingDownloadOptions(t, server, projectName))

	assert.NoError(t, err)

	// THEN we can load the project again and verify its content
	projects, errs := loadDownloadedProjects(t, fs, apiMap)
	if len(errs) != 0 {
		for _, err := range errs {
			t.Errorf("%v", err)
		}
		return
	}

	assert.Len(t, projects, 1)
	p := projects[0]
	assert.Equal(t, p.Id, projectName)
	assert.Len(t, p.Configs, 1)

	configs, found := p.Configs[projectName]
	assert.True(t, found)
	assert.Len(t, configs, 1)

	diff := cmp.Diff(configs, project.ConfigsPerType{
		fakeApi.ID: []config.Config{
			{
				Coordinate: coordinate.Coordinate{Project: projectName, Type: fakeApi.ID, ConfigId: "fake-id"},
				Skip:       false,
				Parameters: map[string]parameter.Parameter{
					"name": &value.ValueParameter{Value: "fake-id"},
				},
				Group:       "default",
				Environment: projectName,
				Template:    template.NewInMemoryTemplate("id", `{"custom-response": true, "name": "{{.name}}"}`),
				Type:        config.ClassicApiType{Api: "fake-id"},
			},
		},
	}, compareOptions...)

	if diff != "" {
		assert.Fail(t, "Objects do not match: %s", diff)
	}
}

func TestDownloadIntegrationSyntheticLocations(t *testing.T) {
	// GIVEN apis, server responses, file system
	const projectName = "integration-test-synthetic-locations"
	const testBasePath = "test-resources/" + projectName

	// APIs
	syntheticLocationApi := api.API{ID: "synthetic-location", URLPath: "/synthetic-location", PropertyNameOfGetAllResponse: api.StandardApiPropertyNameOfGetAllResponse}
	apiMap := api.APIs{syntheticLocationApi.ID: syntheticLocationApi}

	// Responses
	responses := map[string]string{
		"/synthetic-location":      "synthetic-locations/__LIST.json",
		"/synthetic-location/id-1": "synthetic-locations/id-1.json",
		"/synthetic-location/id-2": "synthetic-locations/id-2.json",
		"/synthetic-location/id-3": "synthetic-locations/id-3.json",
	}

	// Server
	server := dtclient.NewIntegrationTestServer(t, testBasePath, responses)

	fs := afero.NewMemMapFs()

	configClient, err := dtclient.NewClassicConfigClientForTesting(server.URL, server.Client())
	require.NoError(t, err)

	settingsClient, err := dtclient.NewPlatformSettingsClientForTesting(server.URL, server.Client())
	require.NoError(t, err)

	// WHEN we download everything
	err = doDownloadConfigs(t.Context(), fs, &client.ClientSet{ConfigClient: configClient, SettingsClient: settingsClient}, apiMap, setupTestingDownloadOptions(t, server, projectName))

	assert.NoError(t, err)

	// THEN we can load the project again and verify its content
	projects, errs := loadDownloadedProjects(t, fs, apiMap)
	if len(errs) != 0 {
		for _, err := range errs {
			t.Errorf("%v", err)
		}
		return
	}

	assert.Len(t, projects, 1)
	p := projects[0]
	assert.Equal(t, p.Id, projectName)
	assert.Len(t, p.Configs, 1)

	configs, found := p.Configs[projectName]
	assert.True(t, found)
	assert.Len(t, configs, 1)

	diff := cmp.Diff(configs, project.ConfigsPerType{
		syntheticLocationApi.ID: []config.Config{
			{
				Coordinate: coordinate.Coordinate{Project: projectName, Type: syntheticLocationApi.ID, ConfigId: "id-2"},
				Skip:       false,
				Parameters: map[string]parameter.Parameter{
					"name": &value.ValueParameter{Value: "Private location - should be stored"},
				},
				Group:       "default",
				Environment: projectName,
				Template:    template.NewInMemoryTemplate("id", `{"type": "PRIVATE", "name": "{{.name}}"}`),
				Type:        config.ClassicApiType{Api: "synthetic-location"},
			},
		},
	}, compareOptions...)

	if diff != "" {
		assert.Fail(t, "Objects do not match: %s", diff)
	}
}

func TestDownloadIntegrationDashboards(t *testing.T) {
	// GIVEN apis, server responses, file system
	const projectName = "integration-test-dashboard"
	const testBasePath = "test-resources/" + projectName

	// APIs
	dashboardApi := api.API{ID: "dashboard", URLPath: "/dashboard", PropertyNameOfGetAllResponse: "dashboards", SingleConfiguration: false, NonUniqueName: false, DeprecatedBy: "", SkipDownload: false}
	apiMap := api.APIs{
		dashboardApi.ID: dashboardApi,
	}

	// Responses
	responses := map[string]string{
		"/dashboard":      "dashboard/__LIST.json",
		"/dashboard/id-1": "dashboard/id-1.json",
		"/dashboard/id-2": "dashboard/id-2.json",
		"/dashboard/id-4": "dashboard/id-4.json",
		// dashbards 3 & 5 MUST NOT BE ACCESSED - filtered out due to being owned by Dynatrace
	}

	// Server
	server := dtclient.NewIntegrationTestServer(t, testBasePath, responses)

	fs := afero.NewMemMapFs()

	configClient, err := dtclient.NewClassicConfigClientForTesting(server.URL, server.Client())
	require.NoError(t, err)

	settingsClient, err := dtclient.NewPlatformSettingsClientForTesting(server.URL, server.Client())
	require.NoError(t, err)

	// WHEN we download everything
	err = doDownloadConfigs(t.Context(), fs, &client.ClientSet{ConfigClient: configClient, SettingsClient: settingsClient}, apiMap, setupTestingDownloadOptions(t, server, projectName))
	assert.NoError(t, err)

	// THEN we can load the project again and verify its content
	projects, errs := loadDownloadedProjects(t, fs, apiMap)
	if len(errs) != 0 {
		for _, err := range errs {
			t.Errorf("%v", err)
		}
		return
	}

	assert.Len(t, projects, 1)
	p := projects[0]
	assert.Equal(t, p.Id, projectName)
	assert.Len(t, p.Configs, 1)

	configs, found := p.Configs[projectName]
	assert.True(t, found)
	assert.Len(t, configs, 1)

	assert.Len(t, configs["dashboard"], 3)

	diff := cmp.Diff(configs, project.ConfigsPerType{
		dashboardApi.ID: []config.Config{
			{
				Coordinate: coordinate.Coordinate{Project: projectName, Type: dashboardApi.ID, ConfigId: "id-1"},
				Skip:       false,
				Parameters: map[string]parameter.Parameter{
					"name": &value.ValueParameter{Value: "Non-unique dashboard-name"},
					config.NonUniqueNameConfigDuplicationParameter: value.New(true),
				},
				Group:       "default",
				Environment: projectName,
				Template:    template.NewInMemoryTemplate("id", `{"dashboardMetadata": {"name": "{{.name}}", "owner": "Q"}, "tiles": []}`),
				Type:        config.ClassicApiType{Api: "dashboard"},
			},
			{
				Coordinate: coordinate.Coordinate{Project: projectName, Type: dashboardApi.ID, ConfigId: "id-2"},
				Skip:       false,
				Parameters: map[string]parameter.Parameter{
					"name": &value.ValueParameter{Value: "Non-unique dashboard-name"},
					config.NonUniqueNameConfigDuplicationParameter: value.New(true),
				},
				Group:       "default",
				Environment: projectName,
				Template:    template.NewInMemoryTemplate("id", `{"dashboardMetadata": {"name": "{{.name}}", "owner": "Admiral Jean-Luc Picard"}, "tiles": []}`),
				Type:        config.ClassicApiType{Api: "dashboard"},
			},
			{
				Coordinate: coordinate.Coordinate{Project: projectName, Type: dashboardApi.ID, ConfigId: "id-4"},
				Skip:       false,
				Parameters: map[string]parameter.Parameter{
					"name": &value.ValueParameter{Value: "Dashboard which is a preset"},
				},
				Group:       "default",
				Environment: projectName,
				Template:    template.NewInMemoryTemplate("id", `{"dashboardMetadata": {"name": "{{.name}}","owner": "Not Dynatrace","preset": true},"tiles": []}`),
				Type:        config.ClassicApiType{Api: "dashboard"},
			},
		},
	}, compareOptions...)

	if diff != "" {
		assert.Fail(t, "Objects do not match: %s", diff)
	}
}

func TestDownloadIntegrationAllDashboardsAreDownloadedIfFilterFFTurnedOff(t *testing.T) {
	// GIVEN apis, server responses, file system
	const projectName = "integration-test-dashboard"
	const testBasePath = "test-resources/" + projectName

	// APIs
	dashboardApi := api.API{ID: "dashboard", URLPath: "/dashboard", PropertyNameOfGetAllResponse: "dashboards", SingleConfiguration: false, NonUniqueName: false, DeprecatedBy: "", SkipDownload: false}
	apiMap := api.APIs{
		dashboardApi.ID: dashboardApi,
	}

	// Responses
	responses := map[string]string{
		"/dashboard":      "dashboard/__LIST.json",
		"/dashboard/id-1": "dashboard/id-1.json",
		"/dashboard/id-2": "dashboard/id-2.json",
		"/dashboard/id-3": "dashboard/id-3.json",
		"/dashboard/id-4": "dashboard/id-4.json",
		"/dashboard/id-5": "dashboard/id-5.json",
	}

	// Server
	server := dtclient.NewIntegrationTestServer(t, testBasePath, responses)

	fs := afero.NewMemMapFs()

	configClient, err := dtclient.NewClassicConfigClientForTesting(server.URL, server.Client())
	require.NoError(t, err)

	settingsClient, err := dtclient.NewPlatformSettingsClientForTesting(server.URL, server.Client())
	require.NoError(t, err)

	t.Setenv(featureflags.DownloadFilterClassicConfigs.EnvName(), "false")

	// WHEN we download everything
	err = doDownloadConfigs(t.Context(), fs, &client.ClientSet{ConfigClient: configClient, SettingsClient: settingsClient}, apiMap, setupTestingDownloadOptions(t, server, projectName))
	assert.NoError(t, err)

	// THEN we can load the project again and verify its content
	projects, errs := loadDownloadedProjects(t, fs, apiMap)
	if len(errs) != 0 {
		for _, err := range errs {
			t.Errorf("%v", err)
		}
		return
	}

	assert.Len(t, projects, 1)
	p := projects[0]
	assert.Equal(t, p.Id, projectName)
	assert.Len(t, p.Configs, 1)

	configs, found := p.Configs[projectName]
	assert.True(t, found)
	assert.Len(t, configs, 1)

	assert.Len(t, configs["dashboard"], 5)

	diff := cmp.Diff(configs, project.ConfigsPerType{
		dashboardApi.ID: []config.Config{
			{
				Coordinate: coordinate.Coordinate{Project: projectName, Type: dashboardApi.ID, ConfigId: "id-1"},
				Skip:       false,
				Parameters: map[string]parameter.Parameter{
					config.NonUniqueNameConfigDuplicationParameter: value.New(true),
					"name": &value.ValueParameter{Value: "Non-unique dashboard-name"},
				},
				Group:       "default",
				Environment: projectName,
				Template:    template.NewInMemoryTemplate("id", `{"dashboardMetadata": {"name": "{{.name}}", "owner": "Q"}, "tiles": []}`),
				Type:        config.ClassicApiType{Api: "dashboard"},
			},
			{
				Coordinate: coordinate.Coordinate{Project: projectName, Type: dashboardApi.ID, ConfigId: "id-2"},
				Skip:       false,
				Parameters: map[string]parameter.Parameter{
					config.NonUniqueNameConfigDuplicationParameter: value.New(true),
					"name": &value.ValueParameter{Value: "Non-unique dashboard-name"},
				},
				Group:       "default",
				Environment: projectName,
				Template:    template.NewInMemoryTemplate("id", `{"dashboardMetadata": {"name": "{{.name}}", "owner": "Admiral Jean-Luc Picard"}, "tiles": []}`),
				Type:        config.ClassicApiType{Api: "dashboard"},
			},
			{
				Coordinate: coordinate.Coordinate{Project: projectName, Type: dashboardApi.ID, ConfigId: "id-3"},
				Skip:       false,
				Parameters: map[string]parameter.Parameter{
					"name": &value.ValueParameter{Value: "Dashboard owned by Dynatrace"},
				},
				Group:       "default",
				Environment: projectName,
				Template:    template.NewInMemoryTemplate("id", `{"dashboardMetadata": {"name": "{{.name}}","owner": "Dynatrace"},"tiles": []}`),
				Type:        config.ClassicApiType{Api: "dashboard"},
			},
			{
				Coordinate: coordinate.Coordinate{Project: projectName, Type: dashboardApi.ID, ConfigId: "id-4"},
				Skip:       false,
				Parameters: map[string]parameter.Parameter{
					"name": &value.ValueParameter{Value: "Dashboard which is a preset"},
				},
				Group:       "default",
				Environment: projectName,
				Template:    template.NewInMemoryTemplate("id", `{"dashboardMetadata": {"name": "{{.name}}","owner": "Not Dynatrace","preset": true},"tiles": []}`),
				Type:        config.ClassicApiType{Api: "dashboard"},
			},
			{
				Coordinate: coordinate.Coordinate{Project: projectName, Type: dashboardApi.ID, ConfigId: "id-5"},
				Skip:       false,
				Parameters: map[string]parameter.Parameter{
					"name": &value.ValueParameter{Value: "Dashboard which is a preset by Dynatrace"},
				},
				Group:       "default",
				Environment: projectName,
				Template:    template.NewInMemoryTemplate("id", `{"dashboardMetadata": {"name": "{{.name}}","owner": "Dynatrace","preset": true},"tiles": []}`),
				Type:        config.ClassicApiType{Api: "dashboard"},
			},
		},
	}, compareOptions...)

	if diff != "" {
		assert.Fail(t, "Objects do not match: %s", diff)
	}
}

func TestDownloadIntegrationAnomalyDetectionMetrics(t *testing.T) {
	// GIVEN apis, server responses, file system
	const projectName = "integration-test-ad-metrics"
	const testBasePath = "test-resources/" + projectName

	// APIs
	dashboardApi := api.API{ID: "anomaly-detection-metrics", URLPath: "/ad-metrics", PropertyNameOfGetAllResponse: api.StandardApiPropertyNameOfGetAllResponse}
	apiMap := api.APIs{
		dashboardApi.ID: dashboardApi,
	}

	// Responses
	responses := map[string]string{
		"/ad-metrics":         "ad-metrics/__LIST.json",
		"/ad-metrics/my.name": "ad-metrics/my.name.json",
		"/ad-metrics/b836ff25-24e3-496d-8dce-d94110815ab5": "ad-metrics/b836ff25-24e3-496d-8dce-d94110815ab5.json",
	}

	// Server
	server := dtclient.NewIntegrationTestServer(t, testBasePath, responses)

	fs := afero.NewMemMapFs()

	configClient, err := dtclient.NewClassicConfigClientForTesting(server.URL, server.Client())
	require.NoError(t, err)

	settingsClient, err := dtclient.NewPlatformSettingsClientForTesting(server.URL, server.Client())
	require.NoError(t, err)

	// WHEN we download everything
	err = doDownloadConfigs(t.Context(), fs, &client.ClientSet{ConfigClient: configClient, SettingsClient: settingsClient}, apiMap, setupTestingDownloadOptions(t, server, projectName))
	assert.NoError(t, err)

	// THEN we can load the project again and verify its content
	projects, errs := loadDownloadedProjects(t, fs, apiMap)
	if len(errs) != 0 {
		for _, err := range errs {
			t.Errorf("%v", err)
		}
		return
	}

	assert.Len(t, projects, 1)
	p := projects[0]
	assert.Equal(t, p.Id, projectName)
	assert.Len(t, p.Configs, 1)

	configs, found := p.Configs[projectName]
	assert.True(t, found)
	assert.Len(t, configs, 1)

	diff := cmp.Diff(configs, project.ConfigsPerType{
		dashboardApi.ID: []config.Config{
			{
				Coordinate: coordinate.Coordinate{Project: projectName, Type: dashboardApi.ID, ConfigId: "b836ff25-24e3-496d-8dce-d94110815ab5"},
				Skip:       false,
				Parameters: map[string]parameter.Parameter{
					"name": &value.ValueParameter{Value: "Test4"},
				},
				Group:       "default",
				Environment: projectName,
				Template:    template.NewInMemoryTemplate("id", `{}`),
				Type:        config.ClassicApiType{Api: "anomaly-detection-metrics"},
			},
			{
				Coordinate: coordinate.Coordinate{Project: projectName, Type: dashboardApi.ID, ConfigId: "my.name"},
				Skip:       false,
				Parameters: map[string]parameter.Parameter{
					"name": &value.ValueParameter{Value: "Test1"},
				},
				Group:       "default",
				Environment: projectName,
				Template:    template.NewInMemoryTemplate("id", `{}`),
				Type:        config.ClassicApiType{Api: "anomaly-detection-metrics"},
			},
		},
	}, compareOptions...)

	if diff != "" {
		assert.Fail(t, "Objects do not match: %s", diff)
	}
}

func TestDownloadIntegrationHostAutoUpdate(t *testing.T) {
	testcases := []struct {
		projectName        string
		shouldProjectExist bool
		expectedConfigs    []config.Config
	}{
		{
			projectName:        "valid",
			shouldProjectExist: true,
			expectedConfigs: []config.Config{
				{
					Coordinate: coordinate.Coordinate{Project: "valid", Type: "hosts-auto-update", ConfigId: "hosts-auto-update"},
					Skip:       false,
					Parameters: map[string]parameter.Parameter{
						"name": &value.ValueParameter{Value: "hosts-auto-update"},
					},
					Group:       "default",
					Environment: "valid",
					Template:    template.NewInMemoryTemplate("id", `{"updateWindows":{"windows":[{"id":"3","name":"Daily maintenance window"}]}}`),
					Type:        config.ClassicApiType{Api: "hosts-auto-update"},
				},
			},
		},
		{
			projectName:        "updateWindows-empty",
			shouldProjectExist: true,
			expectedConfigs: []config.Config{
				{
					Coordinate: coordinate.Coordinate{Project: "updateWindows-empty", Type: "hosts-auto-update", ConfigId: "hosts-auto-update"},
					Skip:       false,
					Parameters: map[string]parameter.Parameter{
						"name": &value.ValueParameter{Value: "hosts-auto-update"},
					},
					Group:       "default",
					Environment: "updateWindows-empty",
					Template:    template.NewInMemoryTemplate("id", `{}`),
					Type:        config.ClassicApiType{Api: "hosts-auto-update"},
				},
			},
		},
		{
			projectName:        "windows-empty",
			shouldProjectExist: false,
			expectedConfigs:    []config.Config{},
		},
		{
			projectName:        "windows-missing",
			shouldProjectExist: true,
			expectedConfigs: []config.Config{
				{
					Coordinate: coordinate.Coordinate{Project: "windows-missing", Type: "hosts-auto-update", ConfigId: "hosts-auto-update"},
					Skip:       false,
					Parameters: map[string]parameter.Parameter{
						"name": &value.ValueParameter{Value: "hosts-auto-update"},
					},
					Group:       "default",
					Environment: "windows-missing",
					Template:    template.NewInMemoryTemplate("id", `{"updateWindows":{}}`),
					Type:        config.ClassicApiType{Api: "hosts-auto-update"},
				},
			},
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.projectName, func(t *testing.T) {
			testBasePath := "test-resources/integration-test-auto-update/" + testcase.projectName

			// APIs
			hostAutoUpdateApi := api.API{ID: "hosts-auto-update", URLPath: "/hosts-auto-update", SingleConfiguration: true}
			apiMap := api.APIs{
				hostAutoUpdateApi.ID: hostAutoUpdateApi,
			}

			// Responses
			responses := map[string]string{
				"/hosts-auto-update": "host-auto-update/singleton.json",
			}

			// Server
			server := dtclient.NewIntegrationTestServer(t, testBasePath, responses)

			fs := afero.NewMemMapFs()

			configClient, err := dtclient.NewClassicConfigClientForTesting(server.URL, server.Client())
			require.NoError(t, err)

			settingsClient, err := dtclient.NewPlatformSettingsClientForTesting(server.URL, server.Client())
			require.NoError(t, err)

			// WHEN we download everything
			err = doDownloadConfigs(t.Context(), fs, &client.ClientSet{ConfigClient: configClient, SettingsClient: settingsClient}, apiMap, setupTestingDownloadOptions(t, server, testcase.projectName))
			assert.NoError(t, err)

			// THEN we can load the project again and verify its content
			projects, errs := loadDownloadedProjects(t, fs, apiMap)

			if !testcase.shouldProjectExist {
				assert.Equal(t, len(errs) > 0, true, "Project loading should have failed")
				return
			}

			if len(errs) != 0 {
				for _, err := range errs {
					t.Errorf("%v", err)
				}
				return
			}

			assert.Len(t, projects, 1)
			p := projects[0]
			assert.Equal(t, p.Id, testcase.projectName)
			assert.Len(t, p.Configs, 1)

			configs, found := p.Configs[testcase.projectName]
			assert.True(t, found)
			assert.Len(t, configs, 1)

			diff := cmp.Diff(configs, project.ConfigsPerType{
				hostAutoUpdateApi.ID: testcase.expectedConfigs,
			}, compareOptions...)

			if diff != "" {
				assert.Fail(t, "Objects do not match: %s", diff)
			}
		})
	}
}

func TestDownloadIntegrationOverwritesFolderAndManifestIfForced(t *testing.T) {
	// GIVEN apis, server responses, file system
	const projectName = "integration-test-1"
	const testBasePath = "test-resources/" + projectName

	// APIs
	fakeApi := api.API{ID: "fake-id", URLPath: "/fake-id", PropertyNameOfGetAllResponse: api.StandardApiPropertyNameOfGetAllResponse}
	apis := api.APIs{
		fakeApi.ID: fakeApi,
	}

	// Responses
	responses := map[string]string{
		"/fake-id":      "fake-api/__LIST.json",
		"/fake-id/id-1": "fake-api/id-1.json",
	}

	// Server
	server := dtclient.NewIntegrationTestServer(t, testBasePath, responses)

	// GIVEN existing files
	fs := afero.NewMemMapFs()
	_ = fs.MkdirAll(testBasePath, 0777)
	_ = fs.MkdirAll(filepath.Join(testBasePath, "fake-id"), 0777)
	_ = afero.WriteFile(fs, filepath.Join(testBasePath, "manifest.yaml"), []byte("OVERWRITE ME"), 0777)
	_ = afero.WriteFile(fs, filepath.Join(testBasePath, "fake-id", "id-1.json"), []byte{}, 0777)

	// WHEN we set the input folder as output and force manifest overwrite on download
	options := setupTestingDownloadOptions(t, server, projectName)
	options.forceOverwriteManifest = true
	options.outputFolder = testBasePath

	configClient, err := dtclient.NewClassicConfigClientForTesting(server.URL, server.Client())
	require.NoError(t, err)

	settingsClient, err := dtclient.NewPlatformSettingsClientForTesting(server.URL, server.Client())
	require.NoError(t, err)

	err = doDownloadConfigs(t.Context(), fs, &client.ClientSet{ConfigClient: configClient, SettingsClient: settingsClient}, apis, options)
	assert.NoError(t, err)

	// THEN we can load the project again and verify its content
	man, errs := manifestloader.Load(&manifestloader.Context{
		Fs:           fs,
		ManifestPath: filepath.Join(testBasePath, "manifest.yaml"),
		Opts:         manifestloader.Options{RequireEnvironmentGroups: true},
	})
	if len(errs) != 0 {
		for _, err := range errs {
			t.Fatalf("%v", err)
		}
	}

	projects, errs := project.LoadProjects(t.Context(), fs, project.ProjectLoaderContext{
		KnownApis:       apis.GetApiNameLookup(),
		WorkingDir:      testBasePath,
		Manifest:        man,
		ParametersSerde: config.DefaultParameterParsers,
	}, nil)
	if len(errs) != 0 {
		for _, err := range errs {
			t.Fatalf("%v", err)
		}
	}

	writtenManifest, err := afero.ReadFile(fs, filepath.Join(testBasePath, "manifest.yaml"))
	assert.NoError(t, err)
	assert.NotEqualf(t, string(writtenManifest), "OVERWRITE ME", "Expected manifest to be overwritten with new data")

	assert.Len(t, projects, 1)
	p := projects[0]
	assert.Equal(t, p.Id, projectName)
	assert.Len(t, p.Configs, 1)

	configs, found := p.Configs[projectName]
	assert.True(t, found)
	assert.Len(t, configs, 1)

	diff := cmp.Diff(configs, project.ConfigsPerType{
		fakeApi.ID: []config.Config{
			{
				Coordinate: coordinate.Coordinate{Project: projectName, Type: fakeApi.ID, ConfigId: "id-1"},
				Skip:       false,
				Parameters: map[string]parameter.Parameter{
					"name": &value.ValueParameter{Value: "Test-1"},
				},
				Group:       "default",
				Environment: projectName,
				Template:    template.NewInMemoryTemplate("id", `{"custom-response": true, "name": "{{.name}}"}`),
				Type:        config.ClassicApiType{Api: "fake-id"},
			},
		},
	}, compareOptions...)

	if diff != "" {
		assert.Fail(t, "Objects do not match: %s", diff)
	}
}

func TestDownloadIntegrationDownloadsAPIsAndSettings(t *testing.T) {
	// GIVEN apis, server responses, file system
	const projectName = "integration-test-full"
	const testBasePath = "test-resources/" + projectName

	// APIs
	fakeApi := api.API{ID: "fake-api", URLPath: "/fake-api", PropertyNameOfGetAllResponse: api.StandardApiPropertyNameOfGetAllResponse}
	apis := api.APIs{
		fakeApi.ID: fakeApi,
	}

	// Responses
	responses := map[string]string{
		"/fake-api":      "fake-api/__LIST.json",
		"/fake-api/id-1": "fake-api/id-1.json",
		"/fake-api/id-2": "fake-api/id-2.json",
		"/platform/classic/environment-api/v2/settings/schemas": "settings/__SCHEMAS.json",
		"/platform/classic/environment-api/v2/settings/objects": "settings/objects.json",
	}

	// Server
	server := dtclient.NewIntegrationTestServer(t, testBasePath, responses)

	fs := afero.NewMemMapFs()

	opts := setupTestingDownloadOptions(t, server, projectName)
	opts.onlyOptions[OnlySettingsFlag] = false
	opts.onlyOptions[OnlyApisFlag] = false

	configClient, err := dtclient.NewClassicConfigClientForTesting(server.URL, server.Client())
	require.NoError(t, err)

	settingsClient, err := dtclient.NewPlatformSettingsClientForTesting(server.URL, server.Client())
	require.NoError(t, err)

	err = doDownloadConfigs(t.Context(), fs, &client.ClientSet{ConfigClient: configClient, SettingsClient: settingsClient}, apis, opts)
	assert.NoError(t, err)

	// THEN we can load the project again and verify its content
	projects, errs := loadDownloadedProjects(t, fs, apis)
	if len(errs) != 0 {
		for _, err := range errs {
			t.Errorf("%v", err)
		}
		return
	}

	assert.Len(t, projects, 1)
	p := projects[0]
	assert.Equal(t, p.Id, projectName)
	assert.Len(t, p.Configs, 1)

	configs, found := p.Configs[projectName]
	assert.True(t, found)
	assert.Equal(t, len(configs), 2, "Expected one config API and one Settings schema to be downloaded")

	_, fakeApiDownloaded := configs[fakeApi.ID]
	assert.True(t, fakeApiDownloaded)
	assert.Equal(t, len(configs[fakeApi.ID]), 2, "Expected 2 config objects")

	_, settingsDownloaded := configs["settings-schema"]
	assert.True(t, settingsDownloaded)
	assert.Equal(t, len(configs["settings-schema"]), 3, "Expected 3 settings objects")
}

func TestDownloadGoTemplateExpressionsAreEscaped(t *testing.T) {
	// GIVEN apis, server responses, file system
	const projectName = "integration-test-go-templating-expressions-are-escaped"
	const testBasePath = "test-resources/" + projectName

	// Responses
	responses := map[string]string{
		"/platform/classic/environment-api/v2/settings/schemas": "settings/__SCHEMAS.json",
		"/platform/classic/environment-api/v2/settings/objects": "settings/objects.json",
	}

	// Server
	server := dtclient.NewIntegrationTestServer(t, testBasePath, responses)

	fs := afero.NewMemMapFs()

	opts := setupTestingDownloadOptions(t, server, projectName)
	opts.onlyOptions[OnlySettingsFlag] = false
	opts.onlyOptions[OnlyApisFlag] = false

	configClient, err := dtclient.NewClassicConfigClientForTesting(server.URL, server.Client())
	require.NoError(t, err)

	settingsClient, err := dtclient.NewPlatformSettingsClientForTesting(server.URL, server.Client())
	require.NoError(t, err)

	err = doDownloadConfigs(t.Context(), fs, &client.ClientSet{ConfigClient: configClient, SettingsClient: settingsClient}, api.APIs{}, opts)
	assert.NoError(t, err)

	// THEN we can load the project again and verify its content
	projects, errs := loadDownloadedProjects(t, fs, api.APIs{})
	if len(errs) != 0 {
		for _, err := range errs {
			t.Errorf("%v", err)
		}
		return
	}

	assert.Len(t, projects, 1)
	p := projects[0]
	assert.Equal(t, p.Id, projectName)
	assert.Len(t, p.Configs, 1)

	configsPerType, found := p.Configs[projectName]
	assert.True(t, found)
	assert.Equal(t, len(configsPerType), 1, "Expected one Settings schema to be downloaded")

	settingsDownloaded, f := configsPerType["settings-schema"]
	assert.True(t, f)
	assert.Len(t, settingsDownloaded, 1, "Expected 1 settings object")

	obj := settingsDownloaded[0]
	content, err := obj.Template.Content()

	assert.JSONEq(t, "{"+
		"\"name\": \"SettingsTest-1\","+
		"\"DQL\": \"fetch bizevents | FILTER like(event.type,\\\"platform.LoginEvent%\\\") | FIELDS CountryIso, Country | SUMMARIZE quantity = toDouble(count()), by:{{`{{`}}CountryIso, alias:countryIso}, {Country, alias:country{{`}}`}} | sort quantity desc\""+
		"}", content)
}

func TestDownloadIntegrationDownloadsOnlyAPIsIfConfigured(t *testing.T) {

	// GIVEN apis, server responses, file system
	const projectName = "integration-test-full"
	const testBasePath = "test-resources/" + projectName

	// APIs
	fakeApi := api.API{ID: "fake-api", URLPath: "/fake-api", PropertyNameOfGetAllResponse: api.StandardApiPropertyNameOfGetAllResponse}
	apis := api.APIs{
		fakeApi.ID: fakeApi,
	}

	// Responses
	responses := map[string]string{
		"/fake-api":      "fake-api/__LIST.json",
		"/fake-api/id-1": "fake-api/id-1.json",
		"/fake-api/id-2": "fake-api/id-2.json",
	}

	// Server
	server := dtclient.NewIntegrationTestServer(t, testBasePath, responses)

	fs := afero.NewMemMapFs()

	opts := setupTestingDownloadOptions(t, server, projectName)
	opts.onlyOptions[OnlySettingsFlag] = false
	opts.onlyOptions[OnlyApisFlag] = true

	configClient, err := dtclient.NewClassicConfigClientForTesting(server.URL, server.Client())
	require.NoError(t, err)

	settingsClient, err := dtclient.NewPlatformSettingsClientForTesting(server.URL, server.Client())
	require.NoError(t, err)

	err = doDownloadConfigs(t.Context(), fs, &client.ClientSet{ConfigClient: configClient, SettingsClient: settingsClient}, apis, opts)
	assert.NoError(t, err)

	// THEN we can load the project again and verify its content
	projects, errs := loadDownloadedProjects(t, fs, apis)
	if len(errs) != 0 {
		for _, err := range errs {
			t.Errorf("%v", err)
		}
		return
	}

	assert.Len(t, projects, 1)
	p := projects[0]
	assert.Equal(t, p.Id, projectName)
	assert.Len(t, p.Configs, 1)

	configs, found := p.Configs[projectName]
	assert.True(t, found)
	assert.Equal(t, len(configs), 1, "Expected one config API to be downloaded")

	_, fakeApiDownloaded := configs[fakeApi.ID]
	assert.True(t, fakeApiDownloaded)
	assert.Equal(t, len(configs[fakeApi.ID]), 2, "Expected 2 config objects")

	_, settingsDownloaded := configs["settings-schema"]
	assert.False(t, settingsDownloaded, "Expected no Settings to the downloaded, when onlyAPIs is set")
}

func TestDownloadIntegrationDoesNotDownloadUnmodifiableSettings(t *testing.T) {
	// GIVEN Responses
	const projectName = "integration-test-unmodifiable-settings"
	const testBasePath = "test-resources/" + projectName

	responses := map[string]string{
		"/platform/classic/environment-api/v2/settings/schemas": "settings/__SCHEMAS.json",
		"/platform/classic/environment-api/v2/settings/objects": "settings/objects.json",
	}

	// GIVEN Server
	server := dtclient.NewIntegrationTestServer(t, testBasePath, responses)

	fs := afero.NewMemMapFs()

	opts := setupTestingDownloadOptions(t, server, projectName)
	opts.onlyOptions[OnlySettingsFlag] = true
	opts.onlyOptions[OnlyApisFlag] = false

	configClient, err := dtclient.NewClassicConfigClientForTesting(server.URL, server.Client())
	require.NoError(t, err)

	settingsClient, err := dtclient.NewPlatformSettingsClientForTesting(server.URL, server.Client())
	require.NoError(t, err)

	err = doDownloadConfigs(t.Context(), fs, &client.ClientSet{ConfigClient: configClient, SettingsClient: settingsClient}, nil, opts)
	assert.NoError(t, err)

	// THEN we can load the project again and verify its content
	projects, errs := loadDownloadedProjects(t, fs, api.APIs{})
	if len(errs) != 0 {
		for _, err := range errs {
			t.Errorf("%v", err)
		}
		return
	}

	assert.Len(t, projects, 1)
	p := projects[0]
	assert.Equal(t, p.Id, projectName)
	assert.Len(t, p.Configs, 1)

	configs, found := p.Configs[projectName]
	assert.True(t, found)
	assert.Equal(t, len(configs), 1, "Expected one Settings schema to be downloaded")

	_, settingsDownloaded := configs["settings-schema"]
	assert.True(t, settingsDownloaded)
	assert.Equal(t, len(configs["settings-schema"]), 2, "Expected 2 settings objects")

	expectedConfigs := map[string]struct{}{"so_1": {}, "so_3": {}}
	for _, cfg := range configs["settings-schema"] {
		_, found := expectedConfigs[cfg.OriginObjectId]
		assert.True(t, found, "did not expect config %s to be downloaded", cfg.OriginObjectId)
	}
}

func TestDownloadIntegrationDownloadsUnmodifiableSettingsIfFFTurnedOff(t *testing.T) {
	// GIVEN Responses
	const projectName = "integration-test-unmodifiable-settings"
	const testBasePath = "test-resources/" + projectName

	responses := map[string]string{
		"/platform/classic/environment-api/v2/settings/schemas": "settings/__SCHEMAS.json",
		"/platform/classic/environment-api/v2/settings/objects": "settings/objects.json",
	}

	// GIVEN Server
	server := dtclient.NewIntegrationTestServer(t, testBasePath, responses)

	fs := afero.NewMemMapFs()

	opts := setupTestingDownloadOptions(t, server, projectName)
	opts.onlyOptions[OnlySettingsFlag] = true
	opts.onlyOptions[OnlyApisFlag] = false

	configClient, err := dtclient.NewClassicConfigClientForTesting(server.URL, server.Client())
	require.NoError(t, err)

	settingsClient, err := dtclient.NewPlatformSettingsClientForTesting(server.URL, server.Client())
	require.NoError(t, err)

	// GIVEN filter feature flag is turned OFF
	t.Setenv(featureflags.DownloadFilterSettingsUnmodifiable.EnvName(), "false")

	err = doDownloadConfigs(t.Context(), fs, &client.ClientSet{ConfigClient: configClient, SettingsClient: settingsClient}, nil, opts)
	assert.NoError(t, err)

	// THEN we can load the project again and verify its content
	projects, errs := loadDownloadedProjects(t, fs, api.APIs{})
	if len(errs) != 0 {
		for _, err := range errs {
			t.Errorf("%v", err)
		}
		return
	}

	assert.Len(t, projects, 1)
	p := projects[0]
	assert.Equal(t, p.Id, projectName)
	assert.Len(t, p.Configs, 1)

	configs, found := p.Configs[projectName]
	assert.True(t, found)
	assert.Equal(t, len(configs), 1, "Expected one Settings schema to be downloaded")

	_, settingsDownloaded := configs["settings-schema"]
	assert.True(t, settingsDownloaded)
	assert.Equal(t, len(configs["settings-schema"]), 3, "Expected 3 settings objects")

	expectedConfigs := map[string]struct{}{"so_1": {}, "so_2": {}, "so_3": {}}
	for _, cfg := range configs["settings-schema"] {
		_, found := expectedConfigs[cfg.OriginObjectId]
		assert.True(t, found, "did not expect config %s to be downloaded", cfg.OriginObjectId)
	}
}

func setupTestingDownloadOptions(t *testing.T, server *httptest.Server, projectName string) downloadConfigsOptions {
	t.Setenv("TOKEN_ENV_VAR", "mock env var")
	t.Setenv(environment.ConcurrentRequestsEnvKey, "50")

	return downloadConfigsOptions{
		downloadOptionsShared: downloadOptionsShared{
			environmentURL: manifest.URLDefinition{
				Type:  manifest.ValueURLType,
				Value: server.URL,
			},
			auth: manifest.Auth{
				ApiToken: &manifest.AuthSecret{
					Name:  "TOKEN_ENV_VAR",
					Value: "token",
				},
			},
			outputFolder: "out",
			projectName:  projectName,
		},
		onlyOptions: OnlyOptions{
			OnlyApisFlag: true,
		},
	}
}

func loadDownloadedProjects(t *testing.T, fs afero.Fs, apis api.APIs) ([]project.Project, []error) {
	man, errs := manifestloader.Load(&manifestloader.Context{
		Fs:           fs,
		ManifestPath: "out/manifest.yaml",
		Opts:         manifestloader.Options{RequireEnvironmentGroups: true},
	})
	if errs != nil {
		return nil, errs
	}

	return project.LoadProjects(t.Context(), fs, project.ProjectLoaderContext{
		KnownApis:       apis.GetApiNameLookup(),
		WorkingDir:      "out",
		Manifest:        man,
		ParametersSerde: config.DefaultParameterParsers,
	}, nil)
}

func jsonEqual(jsonA, jsonB string) bool {
	var a, b map[string]interface{}

	err := json.Unmarshal([]byte(jsonA), &a)
	if err != nil {
		log.Fatal("Failed to unmarshal jsonA: %v", jsonA)
		return false
	}

	err = json.Unmarshal([]byte(jsonB), &b)
	if err != nil {
		log.Fatal("Failed to unmarshal jsonB: %v", jsonB)
		return false
	}

	return reflect.DeepEqual(a, b)
}
