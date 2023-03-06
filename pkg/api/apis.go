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

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/maps"
	"strings"
)

type APIs map[string]*API

func NewApis() APIs {
	return getAPIs(configEndpoints)
}

func NewV1Apis() APIs {
	return getAPIs(configEndpointsV1)
}

func getAPIs(fromApiInputs APIs) APIs {
	apis := make(APIs)
	for id, a := range fromApiInputs {
		apis[id] = newAPI(id, a)
		//a.copy().setID(id).applyRules(nameOfGetAllResponse, singleConfigurationApi)
		//newAPI(*a, setId(id), nameOfGetAllResponse, singleConfigurationApi) //order of applying roles is important!!!
	}

	return apis
}

// Contains return true iff requested API is part APIs set
func (m APIs) Contains(api string) bool {
	_, ok := m[api]
	return ok
}

// ContainsApiName tests if part of project folder path contains an API
// folders with API in path are not valid projects
func (m APIs) ContainsApiName(path string) bool {
	for api := range m {
		if strings.Contains(path, api) {
			return true
		}
	}

	return false
}

// Filter apply all passed filters and return new filtered array
func (m APIs) Filter(filters ...Filter) APIs {
	apis := make(APIs)
	for k, v := range m {
		var keep = true
		for _, f := range filters {
			if f(v) && keep {
				keep = false
				break
			}
		}
		if keep {
			apis[k] = v
		}
	}
	return apis
}

// Filter return true iff specific api needs to be filtered/ removed from list
type Filter func(api *API) bool

// NoFilter is dummy filter that do nothing.
func NoFilter(*API) bool {
	return false
}

// RetainByName leave ony given apis. If api is not provided, nothing is removed.
func RetainByName(APIs []string) Filter {
	if len(APIs) == 0 {
		return NoFilter
	}

	return func(api *API) bool {
		for _, v := range APIs {
			if v == api.GetId() {
				return false
			}
		}
		return true
	}
}

func (apis APIs) GetApiNames() []string {
	return maps.Keys(apis)
}

func (apis APIs) GetApiNameLookup() map[string]struct{} {
	lookup := make(map[string]struct{}, len(apis))

	for k := range apis {
		lookup[k] = struct{}{}
	}

	return lookup
}
