/**
 * @license
 * Copyright 2020 Dynatrace LLC
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

package api

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/util/maps"
	"strings"
)

//go:generate mockgen -source=api.go -destination=api_mock.go -package=api Api

var standardApiPropertyNameOfGetAllResponse = "values"

type Api interface {
	GetUrl(environmentUrl string) string
	GetId() string
	GetPropertyNameOfGetAllResponse() string
	IsStandardApi() bool
	IsSingleConfigurationApi() bool
	IsNonUniqueNameApi() bool
	DeprecatedBy() string

	// ShouldSkipDownload indicates whether an API should be downloaded or not.
	//
	// Some APIs are not re-uploadable by design, either as they require hidden credentials,
	// or if they require a special format, e.g. a zip file.
	//
	// Those configs include all configs handling credentials, as well as the extension-API.
	ShouldSkipDownload() bool
}

type apiInput struct {
	apiPath                      string
	propertyNameOfGetAllResponse string
	isSingleConfigurationApi     bool
	isNonUniqueNameApi           bool
	deprecatedBy                 string
	skipDownload                 bool
}

type apiImpl struct {
	id                           string
	apiPath                      string
	propertyNameOfGetAllResponse string
	isSingleConfigurationApi     bool
	isNonUniqueNameApi           bool
	deprecatedBy                 string
	skipDownload                 bool
}

var (
	// apiImpl needs to implement the API interface
	_ Api = (*apiImpl)(nil)
)

type ApiMap map[string]Api

func NewApis() ApiMap {
	return getApiMap(configEndpoints)
}

func NewV1Apis() ApiMap {
	return getApiMap(v1ApiMap)
}

func getApiMap(fromApiInputs map[string]apiInput) ApiMap {

	apis := make(map[string]Api)

	for id, details := range fromApiInputs {
		apis[id] = newApi(id, details)
	}

	return apis
}

func GetApiNames(apis map[string]Api) []string {
	return maps.Keys(apis)
}

func GetApiNameLookup(apis map[string]Api) map[string]struct{} {
	lookup := make(map[string]struct{}, len(apis))

	for k := range apis {
		lookup[k] = struct{}{}
	}

	return lookup
}

func newApi(id string, input apiInput) Api {
	if input.isSingleConfigurationApi {
		return NewSingleConfigurationApi(id, input.apiPath, input.deprecatedBy, input.skipDownload)
	}

	if input.propertyNameOfGetAllResponse == "" {
		return NewStandardApi(id, input.apiPath, input.isNonUniqueNameApi, input.deprecatedBy, input.skipDownload)
	}

	return NewApi(id, input.apiPath, input.propertyNameOfGetAllResponse, false, input.isNonUniqueNameApi, input.deprecatedBy, input.skipDownload)
}

// NewStandardApi creates an API with propertyNameOfGetAllResponse set to "values"
func NewStandardApi(
	id string,
	apiPath string,
	isNonUniqueNameApi bool,
	isDeprecatedBy string,
	skipDownload bool,
) Api {
	return NewApi(id, apiPath, standardApiPropertyNameOfGetAllResponse, false, isNonUniqueNameApi, isDeprecatedBy, skipDownload)
}

// NewSingleConfigurationApi creates an API with isSingleConfigurationApi set to true
func NewSingleConfigurationApi(
	id string,
	apiPath string,
	isDeprecatedBy string,
	skipDownload bool,
) Api {
	return NewApi(id, apiPath, "", true, false, isDeprecatedBy, skipDownload)
}

func NewApi(
	id string,
	apiPath string,
	propertyNameOfGetAllResponse string,
	isSingleConfigurationApi bool,
	isNonUniqueNameApi bool,
	isDeprecatedBy string,
	skipDownload bool,
) Api {

	// TODO log warning if the user tries to create an API with a id not present in map above
	// This means that a user runs monaco with an untested api

	return &apiImpl{
		id:                           id,
		apiPath:                      apiPath,
		propertyNameOfGetAllResponse: propertyNameOfGetAllResponse,
		isSingleConfigurationApi:     isSingleConfigurationApi,
		isNonUniqueNameApi:           isNonUniqueNameApi,
		deprecatedBy:                 isDeprecatedBy,
		skipDownload:                 skipDownload,
	}
}

func (a *apiImpl) GetUrl(environmentUrl string) string {
	return environmentUrl + a.apiPath
}

func (a *apiImpl) GetId() string {
	return a.id
}

func (a *apiImpl) GetPropertyNameOfGetAllResponse() string {
	return a.propertyNameOfGetAllResponse
}

func (a *apiImpl) IsStandardApi() bool {
	return a.propertyNameOfGetAllResponse == standardApiPropertyNameOfGetAllResponse
}

// Single configuration APIs are those APIs that configure an environment global setting.
// Such settings require additional handling and can't be deleted.
func (a *apiImpl) IsSingleConfigurationApi() bool {
	return a.isSingleConfigurationApi
}

// Non unique name APIs are those APIs that don't work with an environment wide unique id.
// For such APIs, the name attribute can't be used as a id (Monaco default behavior), hence
// such APIs require additional handling.
func (a *apiImpl) IsNonUniqueNameApi() bool {
	return a.isNonUniqueNameApi
}

func (a *apiImpl) DeprecatedBy() string {
	return a.deprecatedBy
}

func (a *apiImpl) ShouldSkipDownload() bool {
	return a.skipDownload
}

func (m ApiMap) IsApi(dir string) bool {
	_, ok := m[dir]
	return ok
}

// ContainsApiName tests if part of project folder path contains an API
// folders with API in path are not valid projects
func (m ApiMap) ContainsApiName(path string) bool {
	for api := range m {
		if strings.Contains(path, api) {
			return true
		}
	}

	return false
}

// Filter filters the APIs into two maps based on the provided callback.
// If the value is true (the value is filtered), the value is put into the second return value, otherwise the first.
func (m ApiMap) Filter(filter func(api Api) bool) (ApiMap, ApiMap) {
	apis := make(ApiMap, len(m))
	filteredApis := ApiMap{}

	for key, value := range m {
		if filter(value) {
			filteredApis[key] = value
		} else {
			apis[key] = value
		}
	}

	return apis, filteredApis
}

// FilterApisByName filters the object for the api names passed.
// Given an emtpy slice, the object is unchanged.
// The second return value contains all apiNames which were not found in the original map, otherwise an empty slice.
func (m ApiMap) FilterApisByName(apiNames []string) (apis ApiMap, unknownApis []string) {
	unknownApis = make([]string, 0)

	if len(apiNames) == 0 {
		return m, unknownApis
	}

	apis = make(ApiMap, len(m))
	for _, name := range apiNames {
		if api, found := m[name]; found {
			apis[name] = api
		} else {
			unknownApis = append(unknownApis, name)
		}
	}

	return apis, unknownApis
}
