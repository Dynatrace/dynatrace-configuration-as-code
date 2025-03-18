//go:build unit

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

package dtclient

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"golang.org/x/exp/maps"

	corerest "github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/slices"
)

var testRetrySettings = RetrySettings{
	Normal: RetrySetting{
		WaitTime:   0,
		MaxRetries: DefaultRetrySettings.Normal.MaxRetries,
	},
	Long: RetrySetting{
		WaitTime:   0,
		MaxRetries: DefaultRetrySettings.Long.MaxRetries,
	},
	VeryLong: RetrySetting{
		WaitTime:   0,
		MaxRetries: DefaultRetrySettings.VeryLong.MaxRetries,
	},
}

type integrationTestResources struct {
	basePath   string
	urlMapping map[string]string
	t          *testing.T
	calls      map[string]int
	callsMutex *sync.Mutex
}

func (i integrationTestResources) Read(urlPath string) ([]byte, bool) {
	path, found := i.urlMapping[urlPath]

	if !found {
		i.t.Errorf("URL path '%s' not mapped", urlPath)
		return nil, false
	}

	return readFileOrPanic(filepath.Join(i.basePath, path)), true
}

func (i integrationTestResources) handler() func(res http.ResponseWriter, req *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			http.Error(res, "Unsupported", http.StatusMethodNotAllowed)
			return
		}

		path := req.URL.Path

		i.callsMutex.Lock()
		i.calls[path]++
		i.callsMutex.Unlock()

		if content, found := i.Read(path); !found {
			log.Error("Failed to find resource '%s'", path)
			http.Error(res, "Not found", http.StatusNotFound)
			return
		} else {
			_, err := res.Write(content) // nosemgrep: go.lang.security.audit.xss.no-direct-write-to-responsewriter.no-direct-write-to-responsewriter
			if err != nil {
				http.Error(res, err.Error(), http.StatusInternalServerError)
			}
		}
	}
}

func (i integrationTestResources) verify() {
	mapped := maps.Keys(i.urlMapping)
	accessed := maps.Keys(i.calls)

	accessedNotMapped := slices.Difference(accessed, mapped)
	mappedNotAccessed := slices.Difference(mapped, accessed)

	for _, v := range accessedNotMapped {
		i.t.Errorf("Path accessed but not mapped: %v", v)
	}

	for _, v := range mappedNotAccessed {
		i.t.Errorf("Path mapped but never accessed: %v", v)
	}
}

func readFileOrPanic(path ...string) []byte {
	content, err := os.ReadFile(filepath.Join(path...))
	if err != nil {
		panic("Failed to read file during test setup: " + err.Error())
	}

	return content
}

// NewIntegrationTestServer creates a new test server and returns it.
// The server is closed automatically upon exiting the testing environment, as well as all defined mappings are checked.
//
// The mapping works as followings: The keys of the map are the URIs the client accesses, while the keys are the path to the
// files *without* the basePath. What file names are used is not important, though a convention is to use __LIST.json for the
// list of all resources for a given API.
func NewIntegrationTestServer(t *testing.T, basePath string, mappings map[string]string) *httptest.Server {
	serverResources := integrationTestResources{
		t:          t,
		basePath:   basePath,
		urlMapping: mappings,
		calls:      map[string]int{},
		callsMutex: &sync.Mutex{},
	}

	testServer := httptest.NewTLSServer(http.HandlerFunc(serverResources.handler()))

	t.Cleanup(serverResources.verify)
	t.Cleanup(testServer.Close)

	return testServer
}

func NewPlatformSettingsClientForTesting(environmentUrl string, client *http.Client, opts ...func(d *SettingsClient)) (*SettingsClient, error) {
	u, err := url.Parse(environmentUrl)
	if err != nil {
		return nil, err
	}

	restClient := corerest.NewClient(u, client, corerest.WithRateLimiter())
	return NewPlatformSettingsClient(
		restClient,
		opts...)
}

func NewClassicConfigClientForTesting(environmentUrl string, client *http.Client, opts ...func(d *ConfigClient)) (*ConfigClient, error) {
	u, err := url.Parse(environmentUrl)
	if err != nil {
		return nil, err
	}

	restClient := corerest.NewClient(u, client, corerest.WithRateLimiter())
	return NewClassicConfigClient(
		restClient,
		opts...)
}

type MuxRouteOptions struct {
	Response       any  // default empty object
	ResponseStatus int  // default 200
	FailOnCall     bool // default false
}
