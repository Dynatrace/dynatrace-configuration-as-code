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

var standardApiPropertyNameOfGetAllResponse = "values"

type apiInput struct {
	apiPath                      string
	propertyNameOfGetAllResponse string
	isSingleConfigurationApi     bool
	isNonUniqueNameApi           bool
	deprecatedBy                 string
	skipDownload                 bool
}

type Api struct {
	id                           string
	apiPath                      string
	propertyNameOfGetAllResponse string
	isSingleConfigurationApi     bool
	isNonUniqueNameApi           bool
	deprecatedBy                 string
	skipDownload                 bool
}

func NewApis() APIs {
	return getApiMap(configEndpoints)
}

func NewV1Apis() APIs {
	return getApiMap(v1ApiMap)
}

func getApiMap(fromApiInputs map[string]apiInput) APIs {

	apis := make(APIs)

	for id, details := range fromApiInputs {
		apis[id] = newApi(id, details)
	}

	return apis
}

func newApi(id string, input apiInput) *Api {
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
) *Api {
	return NewApi(id, apiPath, standardApiPropertyNameOfGetAllResponse, false, isNonUniqueNameApi, isDeprecatedBy, skipDownload)
}

// NewSingleConfigurationApi creates an API with isSingleConfigurationApi set to true
func NewSingleConfigurationApi(
	id string,
	apiPath string,
	isDeprecatedBy string,
	skipDownload bool,
) *Api {
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
) *Api {

	// TODO log warning if the user tries to create an API with a id not present in map above
	// This means that a user runs monaco with an untested api

	return &Api{
		id:                           id,
		apiPath:                      apiPath,
		propertyNameOfGetAllResponse: propertyNameOfGetAllResponse,
		isSingleConfigurationApi:     isSingleConfigurationApi,
		isNonUniqueNameApi:           isNonUniqueNameApi,
		deprecatedBy:                 isDeprecatedBy,
		skipDownload:                 skipDownload,
	}
}

func (a *Api) GetUrl(environmentUrl string) string {
	return environmentUrl + a.apiPath
}

func (a *Api) GetId() string {
	return a.id
}

func (a *Api) GetPropertyNameOfGetAllResponse() string {
	return a.propertyNameOfGetAllResponse
}

func (a *Api) IsStandardApi() bool {
	return a.propertyNameOfGetAllResponse == standardApiPropertyNameOfGetAllResponse
}

// Single configuration APIs are those APIs that configure an environment global setting.
// Such settings require additional handling and can't be deleted.
func (a *Api) IsSingleConfigurationApi() bool {
	return a.isSingleConfigurationApi
}

// Non unique name APIs are those APIs that don't work with an environment wide unique id.
// For such APIs, the name attribute can't be used as a id (Monaco default behavior), hence
// such APIs require additional handling.
func (a *Api) IsNonUniqueNameApi() bool {
	return a.isNonUniqueNameApi
}

func (a *Api) DeprecatedBy() string {
	return a.deprecatedBy
}

// ShouldSkipDownload indicates whether an API should be downloaded or not.
//
// Some APIs are not re-uploadable by design, either as they require hidden credentials,
// or if they require a special format, e.g. a zip file.
//
// Those configs include all configs handling credentials, as well as the extension-API.
func (a *Api) ShouldSkipDownload() bool {
	return a.skipDownload
}
