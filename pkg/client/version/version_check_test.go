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

package version

import (
	"context"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	corerest "github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/version"
)

func TestGetDynatraceVersion(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse string
		want           version.Version
		wantErr        bool
	}{
		{
			"GetDynatraceVersion_AsExpected_1",
			`{ "version": "1.236.0.20220203-192004" }`,
			version.Version{Major: 1, Minor: 236, Patch: 0},
			false,
		},
		{
			"GetDynatraceVersion_AsExpected_2",
			`{ "version": "1.236.5.20220203-192004" }`,
			version.Version{Major: 1, Minor: 236, Patch: 5},
			false,
		},
		{
			"GetDynatraceVersion_AsExpected_3",
			`{ "version": "2.234.0.20220203-192004" }`,
			version.Version{Major: 2, Minor: 234, Patch: 0},
			false,
		},
		{
			"GetDynatraceVersion_FailOnIncompleteVersionString",
			`{ "version": "236.0.20220203-192004" }`,
			version.Version{},
			true,
		},
		{
			"GetDynatraceVersion_FailOnInvalidVersionString",
			`{ "version": "hello.236.0.20220203-192004 }"`,
			version.Version{},
			true,
		},
		{
			"GetDynatraceVersion_IgnoreUnknownJsonProperties",
			`{ "version": "1.236.0.20220203-192004", "thing": "some" }`,
			version.Version{Major: 1, Minor: 236, Patch: 0},
			false,
		},
		{
			"GetDynatraceVersion_FailOnIncompleteJsonResponse",
			`{ "version": "1.236.0.20220203-192004" `,
			version.Version{},
			true,
		},
		{
			"GetDynatraceVersion_FailOnUnexpectedJsonResponse_1",
			`{ "1.236.0.20220203-192004" }"`,
			version.Version{},
			true,
		},
		{
			"GetDynatraceVersion_FailOnUnexpectedJsonResponse_2",
			`{ "version": { "major": 1, "minor": 236, "patch": 0 } }`,
			version.Version{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			server := testutils.NewHTTPTestServer(t, []testutils.ResponseDef{
				{
					GET: func(t *testing.T, request *http.Request) testutils.Response {
						if request.URL.Path == versionPathClassic {
							return testutils.Response{
								ResponseCode: http.StatusOK,
								ResponseBody: tt.serverResponse,
							}
						}

						return testutils.Response{
							ResponseCode: http.StatusNotFound,
						}
					},
				},
			})
			defer server.Close()

			got, err := GetDynatraceVersion(context.TODO(), corerest.NewClient(server.URL(), server.Client()))
			if (err != nil) != tt.wantErr {
				t.Errorf("GetDynatraceVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetDynatraceVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetDynatraceVersionWorksWithTrailingSlash(t *testing.T) {
	server := testutils.NewHTTPTestServer(t, []testutils.ResponseDef{
		{
			GET: func(t *testing.T, request *http.Request) testutils.Response {
				if request.URL.Path == versionPathClassic {
					return testutils.Response{
						ResponseCode: http.StatusOK,
						ResponseBody: `{ "version": "1.236.5.20220203-192004" }`,
					}
				}

				return testutils.Response{
					ResponseCode: http.StatusNotFound,
				}
			},
		},
	})
	defer server.Close()

	urlWithSlash, err := url.Parse(server.URL().String() + "/")
	require.NoError(t, err)

	got, err := GetDynatraceVersion(context.TODO(), corerest.NewClient(urlWithSlash, server.Client()))
	assert.Equal(t, version.Version{Major: 1, Minor: 236, Patch: 5}, got)
	assert.NoError(t, err)
}

func Test_parseDynatraceVersion(t *testing.T) {
	tests := []struct {
		versionString string
		wantVersion   version.Version
		wantErr       bool
	}{
		{
			"1.236.0.20220203-192004",
			version.Version{Major: 1, Minor: 236, Patch: 0},
			false,
		},
		{
			"1.236.5.20220203-192004",
			version.Version{Major: 1, Minor: 236, Patch: 5},
			false,
		},
		{
			"2.234.0.20220203-192004",
			version.Version{Major: 2, Minor: 234, Patch: 0},
			false,
		},
		{
			"1.234.0.20220203-192004",
			version.Version{Major: 1, Minor: 234, Patch: 0},
			false,
		},
		{
			"2.241345.353.20220203-192004",
			version.Version{Major: 2, Minor: 241345, Patch: 353},
			false,
		},
		{
			"236.0.20220203-192004",
			version.Version{},
			true,
		},
		{
			"1.2.236.0.20220203-192004",
			version.Version{},
			true,
		},
		{
			"hello.236.0.20220203-192004",
			version.Version{},
			true,
		},
		{
			"version 42",
			version.Version{},
			true,
		},
		{
			"1.236.0",
			version.Version{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run("parseVersion("+tt.versionString+")", func(t *testing.T) {
			gotVersion, err := parseDynatraceClassicVersion(tt.versionString)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotVersion, tt.wantVersion) {
				t.Errorf("parseVersion() gotVersion = %v, want %v", gotVersion, tt.wantVersion)
			}
		})
	}
}
