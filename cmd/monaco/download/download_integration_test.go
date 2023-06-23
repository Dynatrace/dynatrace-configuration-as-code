//go:build unit

//@license
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
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/environment"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client/dtclient"
	config "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter/reference"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/download/classic"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/download/settings"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/manifest"
	projectLoader "github.com/dynatrace/dynatrace-configuration-as-code/pkg/project/v2"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/spf13/afero"
	"gotest.tools/assert"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// compareOptions holds all options we require for the tests to not be flaky.
// E.g. slices may be in any order, template may have any implementation.
// We want to be pragmatic in comparing them - so we define these options to make it very simple.
var compareOptions = []cmp.Option{
	cmp.Comparer(func(a, b template.Template) bool {
		return jsonEqual(a.Content(), b.Content())
	}),
	cmpopts.SortSlices(func(a, b config.Config) bool {
		return strings.Compare(a.Coordinate.String(), b.Coordinate.String()) < 0
	}),
	cmpopts.SortSlices(func(a, b coordinate.Coordinate) bool {
		return strings.Compare(a.String(), b.String()) < 0
	}),
}

type contentOnlyTemplate struct {
	content string
}

func (c contentOnlyTemplate) Id() string {
	panic("implement me")
}

func (c contentOnlyTemplate) Name() string {
	panic("implement me")
}

func (c contentOnlyTemplate) Content() string {
	return c.content
}

func (c contentOnlyTemplate) UpdateContent(_ string) {
	panic("implement me")
}

var _ template.Template = (*contentOnlyTemplate)(nil)

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

	dtClient, _ := dtclient.NewDynatraceClientForTesting(server.URL, server.Client())

	downloaders := downloaders{settings.NewDownloader(dtClient), classic.NewDownloader(dtClient, classic.WithAPIs(apiMap))}

	// WHEN we download everything
	err := doDownloadConfigs(fs, downloaders, setupTestingDownloadOptions(t, server, projectName))

	assert.NilError(t, err)

	// THEN we can load the project again and verify its content
	projects, errs := loadDownloadedProjects(fs, apiMap)
	if len(errs) != 0 {
		for _, err := range errs {
			t.Errorf("%v", err)
		}
		return
	}

	assert.Equal(t, len(projects), 1)
	p := projects[0]
	assert.Equal(t, p.Id, projectName)
	assert.Equal(t, len(p.Configs), 1)

	configs, found := p.Configs[projectName]
	assert.Equal(t, found, true)
	assert.Equal(t, len(configs), 1)

	var _ config.Type = config.ClassicApiType{}

	assert.DeepEqual(t, configs, projectLoader.ConfigsPerType{
		fakeApi.ID: []config.Config{
			{
				Coordinate: coordinate.Coordinate{Project: projectName, Type: fakeApi.ID, ConfigId: "id-1"},
				Skip:       false,
				Parameters: map[string]parameter.Parameter{
					"name": &value.ValueParameter{Value: "Test-1"},
				},
				Group:       "default",
				Environment: projectName,
				Template:    contentOnlyTemplate{`{"custom-response": true, "name": "{{.name}}"}`},
				Type:        config.ClassicApiType{Api: fakeApi.ID},
			},
		},
	}, compareOptions...)
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

	dtClient, _ := dtclient.NewDynatraceClientForTesting(server.URL, server.Client())

	downloaders := downloaders{settings.NewDownloader(dtClient), classic.NewDownloader(dtClient, classic.WithAPIs(apiMap))}

	// WHEN we download everything
	err := doDownloadConfigs(fs, downloaders, setupTestingDownloadOptions(t, server, projectName))

	assert.NilError(t, err)

	// THEN we can load the project again and verify its content
	projects, errs := loadDownloadedProjects(fs, apiMap)
	if len(errs) != 0 {
		for _, err := range errs {
			t.Errorf("%v", err)
		}
		return
	}

	assert.Equal(t, len(projects), 1)
	p := projects[0]
	assert.Equal(t, p.Id, projectName)
	assert.Equal(t, len(p.Configs), 1)

	configs, found := p.Configs[projectName]
	assert.Equal(t, found, true)

	assert.DeepEqual(t, configs, projectLoader.ConfigsPerType{
		fakeApi.ID: []config.Config{
			{
				Coordinate: coordinate.Coordinate{Project: projectName, Type: fakeApi.ID, ConfigId: "id-1"},
				Skip:       false,
				Parameters: map[string]parameter.Parameter{
					"name": &value.ValueParameter{Value: "Test-1"},
				},
				Group:       "default",
				Environment: projectName,
				Template:    contentOnlyTemplate{`{"custom-response": true, "name": "{{.name}}"}`},
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
				Template:    contentOnlyTemplate{`{"custom-response": true, "name": "{{.name}}", "reference-to-id1": "{{.fakeid__id1__id}}"}`},
				Type:        config.ClassicApiType{Api: "fake-id"},
			},
		},
	}, compareOptions...)
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

	dtClient, _ := dtclient.NewDynatraceClientForTesting(server.URL, server.Client())

	downloaders := downloaders{settings.NewDownloader(dtClient), classic.NewDownloader(dtClient, classic.WithAPIs(apiMap))}

	// WHEN we download everything
	err := doDownloadConfigs(fs, downloaders, setupTestingDownloadOptions(t, server, projectName))

	assert.NilError(t, err)

	projects, errs := loadDownloadedProjects(fs, apiMap)
	if len(errs) != 0 {
		for _, err := range errs {
			t.Errorf("%v", err)
		}
		return
	}

	assert.Equal(t, len(projects), 1)
	p := projects[0]
	assert.Equal(t, p.Id, projectName)
	assert.Equal(t, len(p.Configs), 1)

	configs, found := p.Configs[projectName]
	assert.Equal(t, found, true)

	assert.DeepEqual(t, configs, projectLoader.ConfigsPerType{
		fakeApi1.ID: []config.Config{
			{
				Coordinate: coordinate.Coordinate{Project: projectName, Type: fakeApi1.ID, ConfigId: "id-1"},
				Skip:       false,
				Parameters: map[string]parameter.Parameter{
					"name": &value.ValueParameter{Value: "Test-1"},
				},
				Group:       "default",
				Environment: projectName,
				Template:    contentOnlyTemplate{`{"custom-response": true, "name": "{{.name}}"}`},
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
				Template:    contentOnlyTemplate{`{"custom-response": false, "name": "{{.name}}", "reference-to-id1": "{{.fakeid1__id1__id}}"}`},
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
				Template:    contentOnlyTemplate{`{"custom-response": "No!", "name": "{{.name}}", "subobject": {"something": "{{.fakeid1__id1__id}}"}}`},
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
				Template:    contentOnlyTemplate{`{"custom-response": true, "name": "{{.name}}", "reference-to-id3": "{{.fakeid2__id3__id}}"}`},
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
				Template:    contentOnlyTemplate{`{"name": "{{.name}}", "custom-response": true, "reference-to-id6-of-another-api": ["{{.fakeid2__id4__id}}" ,{"o":  "{{.fakeid1__id2__id}}"}]}`},
				Type:        config.ClassicApiType{Api: "fake-id-3"},
			},
		},
	}, compareOptions...)
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

	dtClient, _ := dtclient.NewDynatraceClientForTesting(server.URL, server.Client())

	downloaders := downloaders{settings.NewDownloader(dtClient), classic.NewDownloader(dtClient, classic.WithAPIs(apiMap))}

	// WHEN we download everything
	err := doDownloadConfigs(fs, downloaders, setupTestingDownloadOptions(t, server, projectName))

	assert.NilError(t, err)

	// THEN we can load the project again and verify its content
	projects, errs := loadDownloadedProjects(fs, apiMap)
	if len(errs) != 0 {
		for _, err := range errs {
			t.Errorf("%v", err)
		}
		return
	}

	assert.Equal(t, len(projects), 1)
	p := projects[0]
	assert.Equal(t, p.Id, projectName)
	assert.Equal(t, len(p.Configs), 1)

	configs, found := p.Configs[projectName]
	assert.Equal(t, found, true)
	assert.Equal(t, len(configs), 1)

	assert.DeepEqual(t, configs, projectLoader.ConfigsPerType{
		fakeApi.ID: []config.Config{
			{
				Coordinate: coordinate.Coordinate{Project: projectName, Type: fakeApi.ID, ConfigId: "fake-id"},
				Skip:       false,
				Parameters: map[string]parameter.Parameter{
					"name": &value.ValueParameter{Value: "fake-id"},
				},
				Group:       "default",
				Environment: projectName,
				Template:    contentOnlyTemplate{`{"custom-response": true, "name": "{{.name}}"}`},
				Type:        config.ClassicApiType{Api: "fake-id"},
			},
		},
	}, compareOptions...)
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

	dtClient, _ := dtclient.NewDynatraceClientForTesting(server.URL, server.Client())

	downloaders := downloaders{settings.NewDownloader(dtClient), classic.NewDownloader(dtClient, classic.WithAPIs(apiMap))}

	// WHEN we download everything
	err := doDownloadConfigs(fs, downloaders, setupTestingDownloadOptions(t, server, projectName))

	assert.NilError(t, err)

	// THEN we can load the project again and verify its content
	projects, errs := loadDownloadedProjects(fs, apiMap)
	if len(errs) != 0 {
		for _, err := range errs {
			t.Errorf("%v", err)
		}
		return
	}

	assert.Equal(t, len(projects), 1)
	p := projects[0]
	assert.Equal(t, p.Id, projectName)
	assert.Equal(t, len(p.Configs), 1)

	configs, found := p.Configs[projectName]
	assert.Equal(t, found, true)
	assert.Equal(t, len(configs), 1)

	assert.DeepEqual(t, configs, projectLoader.ConfigsPerType{
		syntheticLocationApi.ID: []config.Config{
			{
				Coordinate: coordinate.Coordinate{Project: projectName, Type: syntheticLocationApi.ID, ConfigId: "id-2"},
				Skip:       false,
				Parameters: map[string]parameter.Parameter{
					"name": &value.ValueParameter{Value: "Private location - should be stored"},
				},
				Group:       "default",
				Environment: projectName,
				Template:    contentOnlyTemplate{`{"type": "PRIVATE", "name": "{{.name}}"}`},
				Type:        config.ClassicApiType{Api: "synthetic-location"},
			},
		},
	}, compareOptions...)
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

	dtClient, _ := dtclient.NewDynatraceClientForTesting(server.URL, server.Client())

	downloaders := downloaders{settings.NewDownloader(dtClient), classic.NewDownloader(dtClient, classic.WithAPIs(apiMap))}
	// WHEN we download everything
	err := doDownloadConfigs(fs, downloaders, setupTestingDownloadOptions(t, server, projectName))

	assert.NilError(t, err)

	// THEN we can load the project again and verify its content
	projects, errs := loadDownloadedProjects(fs, apiMap)
	if len(errs) != 0 {
		for _, err := range errs {
			t.Errorf("%v", err)
		}
		return
	}

	assert.Equal(t, len(projects), 1)
	p := projects[0]
	assert.Equal(t, p.Id, projectName)
	assert.Equal(t, len(p.Configs), 1)

	configs, found := p.Configs[projectName]
	assert.Equal(t, found, true)
	assert.Equal(t, len(configs), 1)

	assert.Equal(t, len(configs["dashboard"]), 3)

	assert.DeepEqual(t, configs, projectLoader.ConfigsPerType{
		dashboardApi.ID: []config.Config{
			{
				Coordinate: coordinate.Coordinate{Project: projectName, Type: dashboardApi.ID, ConfigId: "id-1"},
				Skip:       false,
				Parameters: map[string]parameter.Parameter{
					"name": &value.ValueParameter{Value: "Non-unique dashboard-name"},
				},
				Group:       "default",
				Environment: projectName,
				Template:    contentOnlyTemplate{`{"dashboardMetadata": {"name": "{{.name}}", "owner": "Q"}, "tiles": []}`},
				Type:        config.ClassicApiType{Api: "dashboard"},
			},
			{
				Coordinate: coordinate.Coordinate{Project: projectName, Type: dashboardApi.ID, ConfigId: "id-2"},
				Skip:       false,
				Parameters: map[string]parameter.Parameter{
					"name": &value.ValueParameter{Value: "Non-unique dashboard-name"},
				},
				Group:       "default",
				Environment: projectName,
				Template:    contentOnlyTemplate{`{"dashboardMetadata": {"name": "{{.name}}", "owner": "Admiral Jean-Luc Picard"}, "tiles": []}`},
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
				Template:    contentOnlyTemplate{`{"dashboardMetadata": {"name": "{{.name}}","owner": "Not Dynatrace","preset": true},"tiles": []}`},
				Type:        config.ClassicApiType{Api: "dashboard"},
			},
		},
	}, compareOptions...)
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

	dtClient, _ := dtclient.NewDynatraceClientForTesting(server.URL, server.Client())

	downloaders := downloaders{settings.NewDownloader(dtClient), classic.NewDownloader(dtClient, classic.WithAPIs(apiMap))}

	t.Setenv(featureflags.DownloadFilterClassicConfigs().EnvName(), "false")

	// WHEN we download everything
	err := doDownloadConfigs(fs, downloaders, setupTestingDownloadOptions(t, server, projectName))

	assert.NilError(t, err)

	// THEN we can load the project again and verify its content
	projects, errs := loadDownloadedProjects(fs, apiMap)
	if len(errs) != 0 {
		for _, err := range errs {
			t.Errorf("%v", err)
		}
		return
	}

	assert.Equal(t, len(projects), 1)
	p := projects[0]
	assert.Equal(t, p.Id, projectName)
	assert.Equal(t, len(p.Configs), 1)

	configs, found := p.Configs[projectName]
	assert.Equal(t, found, true)
	assert.Equal(t, len(configs), 1)

	assert.Equal(t, len(configs["dashboard"]), 5)

	assert.DeepEqual(t, configs, projectLoader.ConfigsPerType{
		dashboardApi.ID: []config.Config{
			{
				Coordinate: coordinate.Coordinate{Project: projectName, Type: dashboardApi.ID, ConfigId: "id-1"},
				Skip:       false,
				Parameters: map[string]parameter.Parameter{
					"name": &value.ValueParameter{Value: "Non-unique dashboard-name"},
				},
				Group:       "default",
				Environment: projectName,
				Template:    contentOnlyTemplate{`{"dashboardMetadata": {"name": "{{.name}}", "owner": "Q"}, "tiles": []}`},
				Type:        config.ClassicApiType{Api: "dashboard"},
			},
			{
				Coordinate: coordinate.Coordinate{Project: projectName, Type: dashboardApi.ID, ConfigId: "id-2"},
				Skip:       false,
				Parameters: map[string]parameter.Parameter{
					"name": &value.ValueParameter{Value: "Non-unique dashboard-name"},
				},
				Group:       "default",
				Environment: projectName,
				Template:    contentOnlyTemplate{`{"dashboardMetadata": {"name": "{{.name}}", "owner": "Admiral Jean-Luc Picard"}, "tiles": []}`},
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
				Template:    contentOnlyTemplate{`{"dashboardMetadata": {"name": "{{.name}}","owner": "Dynatrace"},"tiles": []}`},
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
				Template:    contentOnlyTemplate{`{"dashboardMetadata": {"name": "{{.name}}","owner": "Not Dynatrace","preset": true},"tiles": []}`},
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
				Template:    contentOnlyTemplate{`{"dashboardMetadata": {"name": "{{.name}}","owner": "Dynatrace","preset": true},"tiles": []}`},
				Type:        config.ClassicApiType{Api: "dashboard"},
			},
		},
	}, compareOptions...)
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

	dtClient, _ := dtclient.NewDynatraceClientForTesting(server.URL, server.Client())

	downloaders := downloaders{settings.NewDownloader(dtClient), classic.NewDownloader(dtClient, classic.WithAPIs(apiMap))}

	// WHEN we download everything
	err := doDownloadConfigs(fs, downloaders, setupTestingDownloadOptions(t, server, projectName))

	assert.NilError(t, err)

	// THEN we can load the project again and verify its content
	projects, errs := loadDownloadedProjects(fs, apiMap)
	if len(errs) != 0 {
		for _, err := range errs {
			t.Errorf("%v", err)
		}
		return
	}

	assert.Equal(t, len(projects), 1)
	p := projects[0]
	assert.Equal(t, p.Id, projectName)
	assert.Equal(t, len(p.Configs), 1)

	configs, found := p.Configs[projectName]
	assert.Equal(t, found, true)
	assert.Equal(t, len(configs), 1)

	assert.DeepEqual(t, configs, projectLoader.ConfigsPerType{
		dashboardApi.ID: []config.Config{
			{
				Coordinate: coordinate.Coordinate{Project: projectName, Type: dashboardApi.ID, ConfigId: "b836ff25-24e3-496d-8dce-d94110815ab5"},
				Skip:       false,
				Parameters: map[string]parameter.Parameter{
					"name": &value.ValueParameter{Value: "Test4"},
				},
				Group:       "default",
				Environment: projectName,
				Template:    contentOnlyTemplate{`{}`},
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
				Template:    contentOnlyTemplate{`{}`},
				Type:        config.ClassicApiType{Api: "anomaly-detection-metrics"},
			},
		},
	}, compareOptions...)
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
					Template:    contentOnlyTemplate{`{"updateWindows":{"windows":[{"id":"3","name":"Daily maintenance window"}]}}`},
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
					Template:    contentOnlyTemplate{`{}`},
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
					Template:    contentOnlyTemplate{`{"updateWindows":{}}`},
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

			dtClient, _ := dtclient.NewDynatraceClientForTesting(server.URL, server.Client())

			downloaders := downloaders{settings.NewDownloader(dtClient), classic.NewDownloader(dtClient, classic.WithAPIs(apiMap))}

			// WHEN we download everything
			err := doDownloadConfigs(fs, downloaders, setupTestingDownloadOptions(t, server, testcase.projectName))

			assert.NilError(t, err)

			// THEN we can load the project again and verify its content
			projects, errs := loadDownloadedProjects(fs, apiMap)

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

			assert.Equal(t, len(projects), 1)
			p := projects[0]
			assert.Equal(t, p.Id, testcase.projectName)
			assert.Equal(t, len(p.Configs), 1)

			configs, found := p.Configs[testcase.projectName]
			assert.Equal(t, found, true)
			assert.Equal(t, len(configs), 1)

			assert.DeepEqual(t, configs, projectLoader.ConfigsPerType{
				hostAutoUpdateApi.ID: testcase.expectedConfigs,
			}, compareOptions...)
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

	dtClient, _ := dtclient.NewDynatraceClientForTesting(server.URL, server.Client())
	downloaders := downloaders{settings.NewDownloader(dtClient), classic.NewDownloader(dtClient, classic.WithAPIs(apis))}

	err := doDownloadConfigs(fs, downloaders, options)

	assert.NilError(t, err)

	// THEN we can load the project again and verify its content
	man, errs := manifest.LoadManifest(&manifest.LoaderContext{
		Fs:           fs,
		ManifestPath: filepath.Join(testBasePath, "manifest.yaml"),
	})
	if len(errs) != 0 {
		for _, err := range errs {
			t.Fatalf("%v", err)
		}
	}

	projects, errs := projectLoader.LoadProjects(fs, projectLoader.ProjectLoaderContext{
		KnownApis:       apis.GetApiNameLookup(),
		WorkingDir:      testBasePath,
		Manifest:        man,
		ParametersSerde: config.DefaultParameterParsers,
	})
	if len(errs) != 0 {
		for _, err := range errs {
			t.Fatalf("%v", err)
		}
	}

	writtenManifest, err := afero.ReadFile(fs, filepath.Join(testBasePath, "manifest.yaml"))
	assert.NilError(t, err)
	assert.Assert(t, string(writtenManifest) != "OVERWRITE ME", "Expected manifest to be overwritten with new data")

	assert.Equal(t, len(projects), 1)
	p := projects[0]
	assert.Equal(t, p.Id, projectName)
	assert.Equal(t, len(p.Configs), 1)

	configs, found := p.Configs[projectName]
	assert.Equal(t, found, true)
	assert.Equal(t, len(configs), 1)

	assert.DeepEqual(t, configs, projectLoader.ConfigsPerType{
		fakeApi.ID: []config.Config{
			{
				Coordinate: coordinate.Coordinate{Project: projectName, Type: fakeApi.ID, ConfigId: "id-1"},
				Skip:       false,
				Parameters: map[string]parameter.Parameter{
					"name": &value.ValueParameter{Value: "Test-1"},
				},
				Group:       "default",
				Environment: projectName,
				Template:    contentOnlyTemplate{`{"custom-response": true, "name": "{{.name}}"}`},
				Type:        config.ClassicApiType{Api: "fake-id"},
			},
		},
	}, compareOptions...)
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
		"/fake-api":                "fake-api/__LIST.json",
		"/fake-api/id-1":           "fake-api/id-1.json",
		"/fake-api/id-2":           "fake-api/id-2.json",
		"/api/v2/settings/schemas": "settings/__SCHEMAS.json",
		"/api/v2/settings/objects": "settings/objects.json",
	}

	// Server
	server := dtclient.NewIntegrationTestServer(t, testBasePath, responses)

	fs := afero.NewMemMapFs()

	opts := setupTestingDownloadOptions(t, server, projectName)
	opts.onlySettings = false
	opts.onlyAPIs = false

	dtClient, _ := dtclient.NewDynatraceClientForTesting(server.URL, server.Client())

	downloaders := downloaders{settings.NewDownloader(dtClient), classic.NewDownloader(dtClient, classic.WithAPIs(apis))}

	err := doDownloadConfigs(fs, downloaders, opts)

	assert.NilError(t, err)

	// THEN we can load the project again and verify its content
	projects, errs := loadDownloadedProjects(fs, apis)
	if len(errs) != 0 {
		for _, err := range errs {
			t.Errorf("%v", err)
		}
		return
	}

	assert.Equal(t, len(projects), 1)
	p := projects[0]
	assert.Equal(t, p.Id, projectName)
	assert.Equal(t, len(p.Configs), 1)

	configs, found := p.Configs[projectName]
	assert.Equal(t, found, true)
	assert.Equal(t, len(configs), 2, "Expected one config API and one Settings schema to be downloaded")

	_, fakeApiDownloaded := configs[fakeApi.ID]
	assert.Assert(t, fakeApiDownloaded)
	assert.Equal(t, len(configs[fakeApi.ID]), 2, "Expected 2 config objects")

	_, settingsDownloaded := configs["settings-schema"]
	assert.Assert(t, settingsDownloaded)
	assert.Equal(t, len(configs["settings-schema"]), 3, "Expected 3 settings objects")
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
	opts.onlySettings = false
	opts.onlyAPIs = true
	dtClient, _ := dtclient.NewDynatraceClientForTesting(server.URL, server.Client())

	downloaders := downloaders{settings.NewDownloader(dtClient), classic.NewDownloader(dtClient, classic.WithAPIs(apis))}

	err := doDownloadConfigs(fs, downloaders, opts)

	assert.NilError(t, err)

	// THEN we can load the project again and verify its content
	projects, errs := loadDownloadedProjects(fs, apis)
	if len(errs) != 0 {
		for _, err := range errs {
			t.Errorf("%v", err)
		}
		return
	}

	assert.Equal(t, len(projects), 1)
	p := projects[0]
	assert.Equal(t, p.Id, projectName)
	assert.Equal(t, len(p.Configs), 1)

	configs, found := p.Configs[projectName]
	assert.Equal(t, found, true)
	assert.Equal(t, len(configs), 1, "Expected one config API to be downloaded")

	_, fakeApiDownloaded := configs[fakeApi.ID]
	assert.Assert(t, fakeApiDownloaded)
	assert.Equal(t, len(configs[fakeApi.ID]), 2, "Expected 2 config objects")

	_, settingsDownloaded := configs["settings-schema"]
	assert.Assert(t, !settingsDownloaded, "Expected no Settings to the downloaded, when onlyAPIs is set")
}

func TestDownloadIntegrationDoesNotDownloadUnmodifiableSettings(t *testing.T) {
	// GIVEN Responses
	const projectName = "integration-test-unmodifiable-settings"
	const testBasePath = "test-resources/" + projectName

	responses := map[string]string{
		"/api/v2/settings/schemas": "settings/__SCHEMAS.json",
		"/api/v2/settings/objects": "settings/objects.json",
	}

	// GIVEN Server
	server := dtclient.NewIntegrationTestServer(t, testBasePath, responses)

	fs := afero.NewMemMapFs()

	opts := setupTestingDownloadOptions(t, server, projectName)
	opts.onlySettings = true
	opts.onlyAPIs = false

	dtClient, _ := dtclient.NewDynatraceClientForTesting(server.URL, server.Client())

	downloaders := downloaders{settings.NewDownloader(dtClient), classic.NewDownloader(dtClient, classic.WithAPIs(nil))}

	err := doDownloadConfigs(fs, downloaders, opts)

	assert.NilError(t, err)

	// THEN we can load the project again and verify its content
	projects, errs := loadDownloadedProjects(fs, api.APIs{})
	if len(errs) != 0 {
		for _, err := range errs {
			t.Errorf("%v", err)
		}
		return
	}

	assert.Equal(t, len(projects), 1)
	p := projects[0]
	assert.Equal(t, p.Id, projectName)
	assert.Equal(t, len(p.Configs), 1)

	configs, found := p.Configs[projectName]
	assert.Equal(t, found, true)
	assert.Equal(t, len(configs), 1, "Expected one Settings schema to be downloaded")

	_, settingsDownloaded := configs["settings-schema"]
	assert.Assert(t, settingsDownloaded)
	assert.Equal(t, len(configs["settings-schema"]), 2, "Expected 2 settings objects")

	expectedConfigs := map[string]struct{}{"so_1": {}, "so_3": {}}
	for _, cfg := range configs["settings-schema"] {
		_, found := expectedConfigs[cfg.OriginObjectId]
		assert.Assert(t, found, "did not expect config %s to be downloaded", cfg.OriginObjectId)
	}
}

func TestDownloadIntegrationDownloadsUnmodifiableSettingsIfFFTurnedOff(t *testing.T) {
	// GIVEN Responses
	const projectName = "integration-test-unmodifiable-settings"
	const testBasePath = "test-resources/" + projectName

	responses := map[string]string{
		"/api/v2/settings/schemas": "settings/__SCHEMAS.json",
		"/api/v2/settings/objects": "settings/objects.json",
	}

	// GIVEN Server
	server := dtclient.NewIntegrationTestServer(t, testBasePath, responses)

	fs := afero.NewMemMapFs()

	opts := setupTestingDownloadOptions(t, server, projectName)
	opts.onlySettings = true
	opts.onlyAPIs = false

	dtClient, _ := dtclient.NewDynatraceClientForTesting(server.URL, server.Client())

	downloaders := downloaders{settings.NewDownloader(dtClient), classic.NewDownloader(dtClient, classic.WithAPIs(nil))}

	// GIVEN filter feature flag is turned OFF
	t.Setenv(featureflags.DownloadFilterSettingsUnmodifiable().EnvName(), "false")

	err := doDownloadConfigs(fs, downloaders, opts)

	assert.NilError(t, err)

	// THEN we can load the project again and verify its content
	projects, errs := loadDownloadedProjects(fs, api.APIs{})
	if len(errs) != 0 {
		for _, err := range errs {
			t.Errorf("%v", err)
		}
		return
	}

	assert.Equal(t, len(projects), 1)
	p := projects[0]
	assert.Equal(t, p.Id, projectName)
	assert.Equal(t, len(p.Configs), 1)

	configs, found := p.Configs[projectName]
	assert.Equal(t, found, true)
	assert.Equal(t, len(configs), 1, "Expected one Settings schema to be downloaded")

	_, settingsDownloaded := configs["settings-schema"]
	assert.Assert(t, settingsDownloaded)
	assert.Equal(t, len(configs["settings-schema"]), 3, "Expected 3 settings objects")

	expectedConfigs := map[string]struct{}{"so_1": {}, "so_2": {}, "so_3": {}}
	for _, cfg := range configs["settings-schema"] {
		_, found := expectedConfigs[cfg.OriginObjectId]
		assert.Assert(t, found, "did not expect config %s to be downloaded", cfg.OriginObjectId)
	}
}

func setupTestingDownloadOptions(t *testing.T, server *httptest.Server, projectName string) downloadConfigsOptions {
	t.Setenv("TOKEN_ENV_VAR", "mock env var")
	t.Setenv(environment.ConcurrentRequestsEnvKey, "50")

	return downloadConfigsOptions{
		downloadOptionsShared: downloadOptionsShared{
			environmentURL: server.URL,
			auth: manifest.Auth{
				Token: manifest.AuthSecret{
					Name:  "TOKEN_ENV_VAR",
					Value: "token",
				},
			},
			outputFolder: "out",
			projectName:  projectName,
		},
		onlyAPIs: true,
	}
}

func loadDownloadedProjects(fs afero.Fs, apis api.APIs) ([]projectLoader.Project, []error) {
	man, errs := manifest.LoadManifest(&manifest.LoaderContext{
		Fs:           fs,
		ManifestPath: "out/manifest.yaml",
	})
	if errs != nil {
		return nil, errs
	}

	return projectLoader.LoadProjects(fs, projectLoader.ProjectLoaderContext{
		KnownApis:       apis.GetApiNameLookup(),
		WorkingDir:      "out",
		Manifest:        man,
		ParametersSerde: config.DefaultParameterParsers,
	})
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
