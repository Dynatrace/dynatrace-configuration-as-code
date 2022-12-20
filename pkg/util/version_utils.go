// @license
// Copyright 2022 Dynatrace LLC
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"fmt"
	"strconv"
	"strings"
)

type Version struct {
	Major int
	Minor int
	Patch int
}

func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

func (v Version) GreaterThan(other Version) bool {
	if v == other {
		return false
	}
	if v.Major < other.Major {
		return false
	}
	if v.Major == other.Major &&
		v.Minor < other.Minor {
		return false
	}
	if v.Major == other.Major &&
		v.Minor == other.Minor &&
		v.Patch < other.Patch {
		return false
	}
	return true
}

func (v Version) SmallerThan(other Version) bool {
	return other.GreaterThan(v)
}

func ParseVersion(versionString string) (Version, error) {
	split := strings.Split(versionString, ".")
	if !(len(split) == 2 || len(split) == 3) {
		return Version{}, fmt.Errorf("failed to parse version: format did not meet expected MAJOR.MINOR or MAJOR.MINOR.PATCH pattern: %v", versionString)
	}

	majorVersion, err := strconv.Atoi(split[0])
	if err != nil {
		return Version{}, fmt.Errorf("failed to parse version: major %v is not a number", split[0])
	}
	minorVersion, err := strconv.Atoi(split[1])
	if err != nil {
		return Version{}, fmt.Errorf("failed to parse version: minor %v is not a number", split[1])
	}
	patchVersion := 0
	if len(split) == 3 {
		patchVersion, err = strconv.Atoi(split[2])
		if err != nil {
			return Version{}, fmt.Errorf("failed to parse version: patch %v is not a number", split[2])
		}
	}

	return Version{
		Major: majorVersion,
		Minor: minorVersion,
		Patch: patchVersion,
	}, nil
}
