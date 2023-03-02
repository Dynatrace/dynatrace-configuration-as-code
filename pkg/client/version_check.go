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

package client

import (
	"encoding/json"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/version"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/rest"
	"net/http"
	"strings"
)

type ApiVersionObject struct {
	Version string `json:"version"`
}

const versionPath = "/api/v1/config/clusterversion"

func GetDynatraceVersion(client *http.Client, environmentUrl string) (version.Version, error) {
	versionUrl := environmentUrl + versionPath
	resp, err := rest.Get(client, versionUrl)
	if err != nil {
		return version.Version{}, fmt.Errorf("failed to query version of Dynatrace environment: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return version.Version{}, fmt.Errorf("failed to query version of Dynatrace environment: (HTTP %v) %v", resp.StatusCode, string(resp.Body))
	}

	var jsonResp ApiVersionObject
	if err := json.Unmarshal(resp.Body, &jsonResp); err != nil {
		return version.Version{}, fmt.Errorf("failed to parse Dynatrace version JSON: %w", err)
	}

	return parseDynatraceVersion(jsonResp.Version)
}

// parseDynatraceVersion turns a Dynatrace version string in the format MAJOR.MINOR.PATCH.DATE into a Version object
// for the version check purposes of monaco the build date part is ignored, assuming correct semantic versioning and
// not needing to check anything but >= feature versions for our compatibility usecases
func parseDynatraceVersion(versionString string) (v version.Version, err error) {
	if len(strings.Split(versionString, ".")) != 4 {
		return v, fmt.Errorf("failed to parse Dynatrace version: format did not meet expected MAJOR.MINOR.PATCH.DATE pattern: %v", versionString)
	}

	i := strings.LastIndex(versionString, ".")
	trimmed := versionString[:i] // remove trailing .DATE part

	return version.ParseVersion(trimmed)
}
