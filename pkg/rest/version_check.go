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

package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type Version struct {
	Major int
	Minor int
	Patch int
}

func (v *Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

type ApiVersionObject struct {
	Version string `json:"version"`
}

const versionPath = "/api/v1/config/clusterversion"

func GetDynatraceVersion(client *http.Client, environmentUrl string, apiToken string) (Version, error) {
	versionUrl := environmentUrl + versionPath
	resp, err := get(client, versionUrl, apiToken)
	if err != nil {
		return Version{}, fmt.Errorf("failed to query version of Dynatrace environment: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return Version{}, fmt.Errorf("failed to query version of Dynatrace environment: (HTTP %v) %v", resp.StatusCode, string(resp.Body))
	}

	var jsonResp ApiVersionObject
	if err := json.Unmarshal(resp.Body, &jsonResp); err != nil {
		return Version{}, fmt.Errorf("failed to parse Dynatrace version JSON: %w", err)
	}

	return parseVersion(jsonResp.Version)
}

func MinimumDynatraceVersionReached(expectedMinVersion Version, currentVersion Version) bool {
	if currentVersion.Major < expectedMinVersion.Major {
		return false
	}
	if currentVersion.Major == expectedMinVersion.Major &&
		currentVersion.Minor < expectedMinVersion.Minor {
		return false
	}
	if currentVersion.Major == expectedMinVersion.Major &&
		currentVersion.Minor == expectedMinVersion.Minor &&
		currentVersion.Patch < expectedMinVersion.Patch {
		return false
	}
	return true
}

// parseVersion turns a Dynatrace version string in the format MAJOR.MINOR.PATCH.DATE into a Version object
// for the version check purposes of monaco the build date part is ignored, assuming correct semantic versioning and
// not needing to check anything but >= feature versions for our compatibility usecases
func parseVersion(versionString string) (version Version, err error) {
	split := strings.Split(versionString, ".")
	if len(split) != 4 {
		return version, fmt.Errorf("failed to parse Dynatrace version: format did not meet expected MAJOR.MINOR.PATCH.DATE pattern: %v", versionString)
	}

	version.Major, err = strconv.Atoi(split[0])
	if err != nil {
		return version, fmt.Errorf("failed to parse Dynatrace version: major %v is not a number", split[0])
	}
	version.Minor, err = strconv.Atoi(split[1])
	if err != nil {
		return version, fmt.Errorf("failed to parse Dynatrace version: minor %v is not a number", split[1])
	}
	version.Patch, err = strconv.Atoi(split[2])
	if err != nil {
		return version, fmt.Errorf("failed to parse Dynatrace version: patch %v is not a number", split[2])
	}

	return
}
