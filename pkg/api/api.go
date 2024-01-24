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

import "strings"

// API structure present definition of config endpoints
type API struct {
	ID string
	//URLPath defines default path
	URLPath                      string
	PropertyNameOfGetAllResponse string
	// SingleConfiguration are those APIs that configure an environment global setting.
	// Such settings require additional handling and can't be deleted.
	SingleConfiguration bool
	// NonUniqueName name APIs are those APIs that don't work with an environment wide unique ID.
	// For such APIs, the name attribute can't be used as a ID (Monaco default behavior), hence
	// such APIs require additional handling.
	NonUniqueName bool
	DeprecatedBy  string
	// SkipDownload indicates whether an API should be downloaded or not.
	//
	// Some APIs are not re-uploadable by design, either as they require hidden credentials,
	// or if they require a special format, e.g. a zip file.
	//
	// Those configs include all configs handling credentials, as well as the extension-API.
	SkipDownload bool
	// TweakResponseFunc can be optionally registered to add custom code that changes the
	// payload of the downloaded api content (e.g. to exclude unwanted/unnecessary fields)
	TweakResponseFunc func(map[string]any)
}

func (a API) CreateURL(environmentURL string) string {
	return environmentURL + a.URLPath
}

func (a API) IsStandardAPI() bool {
	return a.PropertyNameOfGetAllResponse == StandardApiPropertyNameOfGetAllResponse
}

func (a API) Resolve(value string) API {
	newA := a
	newA.URLPath = strings.ReplaceAll(a.URLPath, "{SCOPE}", value)
	return newA
}

func (a API) IsSubPathAPI() bool {
	return strings.Contains(a.URLPath, "{SCOPE}")
}
