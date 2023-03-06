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

package api

// Deprecated: NewApi, which was introduced for testing, exposes unwanted internal details and makes possible to create unsupported api
func NewApi(
	id string,
	apiPath string,
	propertyNameOfGetAllResponse string,
	isSingleConfigurationApi bool,
	isNonUniqueNameApi bool,
	isDeprecatedBy string,
	skipDownload bool,
) *API {
	return &API{
		id:                           id,
		apiPath:                      apiPath,
		propertyNameOfGetAllResponse: propertyNameOfGetAllResponse,
		isSingleConfigurationApi:     isSingleConfigurationApi,
		isNonUniqueNameApi:           isNonUniqueNameApi,
		deprecatedBy:                 isDeprecatedBy,
		skipDownload:                 skipDownload,
	}
}

// NewStandardApi creates an API with propertyNameOfGetAllResponse set to "values"
// Deprecated: NewStandardApi, which was introduced for testing, exposes unwanted internal details and makes possible to create unsupported api
func NewStandardApi(
	id string,
	apiPath string,
	isNonUniqueNameApi bool,
	isDeprecatedBy string,
	skipDownload bool,
) *API {
	return NewApi(id, apiPath, standardApiPropertyNameOfGetAllResponse, false, isNonUniqueNameApi, isDeprecatedBy, skipDownload)
}

// NewSingleConfigurationApi creates an API with isSingleConfigurationApi set to true
// Deprecated: NewSingleConfigurationApi, which was introduced for testing, exposes unwanted internal details and makes possible to create unsupported api
func NewSingleConfigurationApi(
	id string,
	apiPath string,
	isDeprecatedBy string,
	skipDownload bool,
) *API {
	return NewApi(id, apiPath, "", true, false, isDeprecatedBy, skipDownload)
}
