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
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	config "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/coordinate"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter/reference"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter/value"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/template"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/manifest"
	projectLoader "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/rest"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/log"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/maps"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/spf13/afero"
	"gotest.tools/assert"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

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

var templateContentComparator = cmp.Comparer(func(a, b template.Template) bool {
	return jsonEqual(a.Content(), b.Content())
})

type integrationTestServer struct {
	basePath   string
	urlMapping map[string]string
	t          *testing.T
}

func (i integrationTestServer) Read(uri string) ([]byte, bool) {
	path, found := i.urlMapping[uri]

	if !found {
		i.t.Errorf("Uri '%s' not mapped", uri)
		return nil, false
	}

	return readFileOrPanic(filepath.Join(i.basePath, path)), true
}

func newTestServer(t *testing.T, basePath string, urlMapping map[string]string) integrationTestServer {
	return integrationTestServer{
		t:          t,
		basePath:   basePath,
		urlMapping: urlMapping,
	}
}

func TestDownloadIntegrationSimple(t *testing.T) {
	// GIVEN apis, server responses, file system
	const projectName = "integration-test-1"
	const testBasePath = "test-resources/" + projectName

	// APIs
	fakeApi := api.NewStandardApi("fake-id", "/fake-id", false, "", false)
	apiMap := api.ApiMap{
		fakeApi.GetId(): fakeApi,
	}

	// Responses
	responses := map[string]string{
		"/fake-id":      "fake-api/__LIST.json",
		"/fake-id/id-1": "fake-api/id-1.json",
	}

	testServer := newTestServer(t, testBasePath, responses)

	// Server
	server := rest.NewDynatraceTLSServerForTesting(t, func(res http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			http.Error(res, "Unsupported", http.StatusMethodNotAllowed)
			return
		}

		if content, found := testServer.Read(req.RequestURI); !found {
			log.Error("Failed to find resource '%s'", req.RequestURI)
			http.Error(res, "Not found", http.StatusNotFound)
			return
		} else {
			_, err := res.Write(content)
			if err != nil {
				http.Error(res, err.Error(), http.StatusInternalServerError)
			}
		}
	})
	fs := afero.NewMemMapFs()

	// WHEN we download everything
	err := doDownload(fs, server.URL, projectName, "token", "TOKEN_ENV_VAR", "out", apiMap, func(environmentUrl, token string) (rest.DynatraceClient, error) {
		return rest.NewDynatraceClientForTesting(environmentUrl, token, server.Client())
	})

	assert.NilError(t, err)

	// THEN we can load the project again and verify its content
	man, errs := manifest.LoadManifest(&manifest.ManifestLoaderContext{
		Fs:           fs,
		ManifestPath: "out/manifest.yaml",
	})
	if errs != nil {
		t.Errorf("Errors occured: %v", errs)
		return
	}

	projects, errs := projectLoader.LoadProjects(fs, projectLoader.ProjectLoaderContext{
		Apis:            maps.Keys(apiMap),
		WorkingDir:      "out",
		Manifest:        man,
		ParametersSerde: config.DefaultParameterParsers,
	})
	if errs != nil {
		t.Errorf("Errors occured: %v", errs)
		return
	}

	assert.Equal(t, len(projects), 1)
	p := projects[0]
	assert.Equal(t, p.Id, projectName)
	assert.Equal(t, len(p.Configs), 1)

	configs, found := p.Configs[projectName]
	assert.Equal(t, found, true)
	assert.Equal(t, len(configs), 1)

	assert.DeepEqual(t, configs, projectLoader.ConfigsPerApis{
		fakeApi.GetId(): []config.Config{
			{
				Coordinate: coordinate.Coordinate{Project: projectName, Api: fakeApi.GetId(), Config: "id-1"},
				Skip:       false,
				Parameters: map[string]parameter.Parameter{
					"name": &value.ValueParameter{Value: "Test-1"},
				},
				Group:       "default",
				Environment: projectName,
				References:  []coordinate.Coordinate{},
				Template:    contentOnlyTemplate{`{"custom-response": true, "name": "{{.name}}"}`},
			},
		},
	}, templateContentComparator)
}

func TestDownloadIntegrationWithReference(t *testing.T) {
	// GIVEN apis, server responses, file system
	const projectName = "integration-test-2"
	const testBasePath = "test-resources/" + projectName

	// APIs
	fakeApi := api.NewStandardApi("fake-id", "/fake-id", false, "", false)
	apiMap := api.ApiMap{
		fakeApi.GetId(): fakeApi,
	}

	// Responses
	responses := map[string]string{
		"/fake-id":      "fake-api/__LIST.json",
		"/fake-id/id-1": "fake-api/id-1.json",
		"/fake-id/id-2": "fake-api/id-2.json",
	}

	testServer := newTestServer(t, testBasePath, responses)

	// Server
	server := rest.NewDynatraceTLSServerForTesting(t, func(res http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			http.Error(res, "Unsupported", http.StatusMethodNotAllowed)
			return
		}

		if content, found := testServer.Read(req.RequestURI); !found {
			http.Error(res, "Not found", http.StatusNotFound)
			return
		} else {
			_, err := res.Write(content)
			if err != nil {
				http.Error(res, err.Error(), http.StatusInternalServerError)
			}
		}
	})
	fs := afero.NewMemMapFs()

	// WHEN we download everything
	err := doDownload(fs, server.URL, projectName, "token", "TOKEN_ENV_VAR", "out", apiMap, func(environmentUrl, token string) (rest.DynatraceClient, error) {
		return rest.NewDynatraceClientForTesting(environmentUrl, token, server.Client())
	})

	assert.NilError(t, err)

	// THEN we can load the project again and verify its content
	man, errs := manifest.LoadManifest(&manifest.ManifestLoaderContext{
		Fs:           fs,
		ManifestPath: "out/manifest.yaml",
	})
	if errs != nil {
		t.Errorf("Errors occured: %v", errs)
		return
	}

	projects, errs := projectLoader.LoadProjects(fs, projectLoader.ProjectLoaderContext{
		Apis:            maps.Keys(apiMap),
		WorkingDir:      "out",
		Manifest:        man,
		ParametersSerde: config.DefaultParameterParsers,
	})
	if errs != nil {
		t.Errorf("Errors occured: %v", errs)
		return
	}

	assert.Equal(t, len(projects), 1)
	p := projects[0]
	assert.Equal(t, p.Id, projectName)
	assert.Equal(t, len(p.Configs), 1)

	configs, found := p.Configs[projectName]
	assert.Equal(t, found, true)

	assert.DeepEqual(t, configs, projectLoader.ConfigsPerApis{
		fakeApi.GetId(): []config.Config{
			{
				Coordinate: coordinate.Coordinate{Project: projectName, Api: fakeApi.GetId(), Config: "id-1"},
				Skip:       false,
				Parameters: map[string]parameter.Parameter{
					"name": &value.ValueParameter{Value: "Test-1"},
				},
				Group:       "default",
				Environment: projectName,
				References:  []coordinate.Coordinate{},
				Template:    contentOnlyTemplate{`{"custom-response": true, "name": "{{.name}}"}`},
			},
			{
				Coordinate: coordinate.Coordinate{Project: projectName, Api: fakeApi.GetId(), Config: "id-2"},
				Skip:       false,
				Parameters: map[string]parameter.Parameter{
					"name":            &value.ValueParameter{Value: "Test-2"},
					"fakeid__id1__id": reference.New(projectName, fakeApi.GetId(), "id-1", "id"),
				},
				Group:       "default",
				Environment: projectName,
				References: []coordinate.Coordinate{
					{Project: projectName, Api: fakeApi.GetId(), Config: "id-1"},
				},
				Template: contentOnlyTemplate{`{"custom-response": true, "name": "{{.name}}", "reference-to-id1": "{{.fakeid__id1__id}}"}`},
			},
		},
	}, templateContentComparator, cmpopts.SortSlices(func(a, b config.Config) bool {
		return strings.Compare(a.Coordinate.String(), b.Coordinate.String()) < 0
	}))
}

func TestDownloadIntegrationWithMultipleApisAndReferences(t *testing.T) {
	// GIVEN apis, server responses, file system
	const projectName = "integration-test-3"
	const testBasePath = "test-resources/" + projectName

	// APIs
	fakeApi1 := api.NewStandardApi("fake-id-1", "/fake-api-1", false, "", false)
	fakeApi2 := api.NewStandardApi("fake-id-2", "/fake-api-2", false, "", false)
	fakeApi3 := api.NewStandardApi("fake-id-3", "/fake-api-3", false, "", false)
	apiMap := api.ApiMap{
		fakeApi1.GetId(): fakeApi1,
		fakeApi2.GetId(): fakeApi2,
		fakeApi3.GetId(): fakeApi3,
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

	testServer := newTestServer(t, testBasePath, responses)

	// Server
	server := rest.NewDynatraceTLSServerForTesting(t, func(res http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			http.Error(res, "Unsupported", http.StatusMethodNotAllowed)
			return
		}

		if content, found := testServer.Read(req.RequestURI); !found {
			log.Error("Failed to find resource '%s'", req.RequestURI)
			http.Error(res, "Not found", http.StatusNotFound)
			return
		} else {
			_, err := res.Write(content)
			if err != nil {
				http.Error(res, err.Error(), http.StatusInternalServerError)
			}
		}
	})
	fs := afero.NewMemMapFs()

	// WHEN we download everything
	err := doDownload(fs, server.URL, projectName, "token", "TOKEN_ENV_VAR", "out", apiMap, func(environmentUrl, token string) (rest.DynatraceClient, error) {
		return rest.NewDynatraceClientForTesting(environmentUrl, token, server.Client())
	})

	assert.NilError(t, err)

	// THEN we can load the project again and verify its content
	man, errs := manifest.LoadManifest(&manifest.ManifestLoaderContext{
		Fs:           fs,
		ManifestPath: "out/manifest.yaml",
	})
	if errs != nil {
		t.Errorf("Errors occured: %v", errs)
		return
	}

	projects, errs := projectLoader.LoadProjects(fs, projectLoader.ProjectLoaderContext{
		Apis:            maps.Keys(apiMap),
		WorkingDir:      "out",
		Manifest:        man,
		ParametersSerde: config.DefaultParameterParsers,
	})
	if errs != nil {
		t.Errorf("Errors occured: %v", errs)
		return
	}

	assert.Equal(t, len(projects), 1)
	p := projects[0]
	assert.Equal(t, p.Id, projectName)
	assert.Equal(t, len(p.Configs), 1)

	configs, found := p.Configs[projectName]
	assert.Equal(t, found, true)

	assert.DeepEqual(t, configs, projectLoader.ConfigsPerApis{
		fakeApi1.GetId(): []config.Config{
			{
				Coordinate: coordinate.Coordinate{Project: projectName, Api: fakeApi1.GetId(), Config: "id-1"},
				Skip:       false,
				Parameters: map[string]parameter.Parameter{
					"name": &value.ValueParameter{Value: "Test-1"},
				},
				Group:       "default",
				Environment: projectName,
				References:  []coordinate.Coordinate{},
				Template:    contentOnlyTemplate{`{"custom-response": true, "name": "{{.name}}"}`},
			},
			{
				Coordinate: coordinate.Coordinate{Project: projectName, Api: fakeApi1.GetId(), Config: "id-2"},
				Skip:       false,
				Parameters: map[string]parameter.Parameter{
					"name":             &value.ValueParameter{Value: "Test-2"},
					"fakeid1__id1__id": reference.New(projectName, fakeApi1.GetId(), "id-1", "id"),
				},
				Group:       "default",
				Environment: projectName,
				References: []coordinate.Coordinate{
					{Project: projectName, Api: fakeApi1.GetId(), Config: "id-1"},
				},
				Template: contentOnlyTemplate{`{"custom-response": false, "name": "{{.name}}", "reference-to-id1": "{{.fakeid1__id1__id}}"}`},
			},
		},
		fakeApi2.GetId(): []config.Config{
			{
				Coordinate: coordinate.Coordinate{Project: projectName, Api: fakeApi2.GetId(), Config: "id-3"},
				Skip:       false,
				Parameters: map[string]parameter.Parameter{
					"name":             &value.ValueParameter{Value: "Test-3"},
					"fakeid1__id1__id": reference.New(projectName, fakeApi1.GetId(), "id-1", "id"),
				},
				Group:       "default",
				Environment: projectName,
				References: []coordinate.Coordinate{
					{Project: projectName, Api: fakeApi1.GetId(), Config: "id-1"},
				},
				Template: contentOnlyTemplate{`{"custom-response": "No!", "name": "{{.name}}", "subobject": {"something": "{{.fakeid1__id1__id}}"}}`},
			},
			{
				Coordinate: coordinate.Coordinate{Project: projectName, Api: fakeApi2.GetId(), Config: "id-4"},
				Skip:       false,
				Parameters: map[string]parameter.Parameter{
					"name":             &value.ValueParameter{Value: "Test-4"},
					"fakeid2__id3__id": reference.New(projectName, fakeApi2.GetId(), "id-3", "id"),
				},
				Group:       "default",
				Environment: projectName,
				References: []coordinate.Coordinate{
					{Project: projectName, Api: fakeApi2.GetId(), Config: "id-3"},
				},
				Template: contentOnlyTemplate{`{"custom-response": true, "name": "{{.name}}", "reference-to-id3": "{{.fakeid2__id3__id}}"}`},
			},
		},
		fakeApi3.GetId(): []config.Config{
			{
				Coordinate: coordinate.Coordinate{Project: projectName, Api: fakeApi3.GetId(), Config: "id-5"},
				Skip:       false,
				Parameters: map[string]parameter.Parameter{
					"name":             &value.ValueParameter{Value: "Test-5"},
					"fakeid1__id2__id": reference.New(projectName, fakeApi1.GetId(), "id-2", "id"),
					"fakeid2__id4__id": reference.New(projectName, fakeApi2.GetId(), "id-4", "id"),
				},
				Group:       "default",
				Environment: projectName,
				References: []coordinate.Coordinate{
					{Project: projectName, Api: fakeApi1.GetId(), Config: "id-2"},
					{Project: projectName, Api: fakeApi2.GetId(), Config: "id-4"},
				},
				Template: contentOnlyTemplate{`{"name": "{{.name}}", "custom-response": true, "reference-to-id6-of-another-api": ["{{.fakeid2__id4__id}}" ,{"o":  "{{.fakeid1__id2__id}}"}]}
`},
			},
		},
	}, templateContentComparator, cmpopts.SortSlices(func(a, b config.Config) bool {
		return strings.Compare(a.Coordinate.String(), b.Coordinate.String()) < 0
	}))
}

func readFileOrPanic(path ...string) []byte {
	content, err := os.ReadFile(filepath.Join(path...))
	if err != nil {
		panic("Failed to read file during test setup: " + err.Error())
	}

	return content
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
