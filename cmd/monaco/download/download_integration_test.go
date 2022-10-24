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
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter/value"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/template"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/manifest"
	projectLoader "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/rest"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/log"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/maps"
	"github.com/google/go-cmp/cmp"
	"github.com/spf13/afero"
	"gotest.tools/assert"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
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
	responses := map[string][]byte{
		"/fake-id":      readFileOrPanic(testBasePath, "fake-api/__LIST.json"),
		"/fake-id/id-1": readFileOrPanic(testBasePath, "fake-api/id-1.json"),
	}

	// Server
	server := rest.NewDynatraceTLSServerForTesting(t, func(res http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			http.Error(res, "Unsupported", http.StatusMethodNotAllowed)
			return
		}

		if content, found := responses[req.RequestURI]; !found {
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
