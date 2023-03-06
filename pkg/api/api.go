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

type API struct {
	id                           string
	apiPath                      string
	propertyNameOfGetAllResponse string
	isSingleConfigurationApi     bool
	isNonUniqueNameApi           bool
	deprecatedBy                 string
	skipDownload                 bool
}

func (a *API) GetUrl(environmentUrl string) string {
	return environmentUrl + a.apiPath
}

func (a *API) GetId() string {
	return a.id
}

func (a *API) GetPropertyNameOfGetAllResponse() string {
	return a.propertyNameOfGetAllResponse
}

func (a *API) IsStandardApi() bool {
	return a.propertyNameOfGetAllResponse == standardApiPropertyNameOfGetAllResponse
}

// Single configuration APIs are those APIs that configure an environment global setting.
// Such settings require additional handling and can't be deleted.
func (a *API) IsSingleConfigurationApi() bool {
	return a.isSingleConfigurationApi
}

// Non unique name APIs are those APIs that don't work with an environment wide unique id.
// For such APIs, the name attribute can't be used as a id (Monaco default behavior), hence
// such APIs require additional handling.
func (a *API) IsNonUniqueNameApi() bool {
	return a.isNonUniqueNameApi
}

func (a *API) DeprecatedBy() string {
	return a.deprecatedBy
}

// ShouldSkipDownload indicates whether an API should be downloaded or not.
//
// Some APIs are not re-uploadable by design, either as they require hidden credentials,
// or if they require a special format, e.g. a zip file.
//
// Those configs include all configs handling credentials, as well as the extension-API.
func (a *API) ShouldSkipDownload() bool {
	return a.skipDownload
}

// newAPI create new API as a copy of template with a given name
func newAPI(name string, template *API) *API {
	n := *template // if API structure is contains pointers or substructure, perform deep copy!!!
	n.id = name
	n.applyRules(nameOfGetAllResponse, singleConfigurationApi) //order of applying roles is important!!!
	return &n
}

type rule func(*API)

func (a *API) applyRules(rules ...rule) *API {
	//NOTE: make sure that API structure is composed of only simple/prime data types
	// api is already a copy!!! (call by value)
	for _, r := range rules {
		r(a)
	}
	return a
}

func singleConfigurationApi(api *API) {
	if api.isSingleConfigurationApi == true {
		api.propertyNameOfGetAllResponse = ""
		api.isNonUniqueNameApi = false
	}
}

func nameOfGetAllResponse(api *API) {
	if api.propertyNameOfGetAllResponse == "" {
		api.propertyNameOfGetAllResponse = standardApiPropertyNameOfGetAllResponse
	}
}
