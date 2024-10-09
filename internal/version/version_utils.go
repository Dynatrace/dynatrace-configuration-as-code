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
	"fmt"
	"strconv"
	"strings"
)

// UnknownVersion is just a version that is not set
var UnknownVersion = Version{}

// Version represents a software version composed of
type Version struct {
	Major int
	Minor int
	Patch int
}

// String returns the version in a printable format, e.g.: 1.2.3
func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// GreaterThan determines whether this version is greater than the other version
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

// SmallerThan determines whether this version is smaller than the given version
func (v Version) SmallerThan(other Version) bool {
	return other.GreaterThan(v)
}

// Invalid returns whether this version is valid or not.
// A version is considered to be "invalid" if it is equal to "0.0.0" or
// contains a negative number, e.e. "0.-1.2"
func (v Version) Invalid() bool {
	return (v.Major <= 0 && v.Minor <= 0 && v.Patch <= 0) ||
		v.Major < 0 || v.Minor < 0 || v.Patch < 0

}

// ParseVersion takes a version as string and tries to parse it to convert it to a Version value.
// It returns the Version value and possibly an error if the string could not be parsed.
// Expected formats are "MAJOR", "MAJOR.MINOR" or "MAJOR.MINOR.PATCH" with each component being a non-negative number.
// Omitted minor or patch versions are interpreted as 0, so "2" is interpreted as 2.0.0 and "2.1" is interpreted as 2.1.0.
func ParseVersion(versionString string) (Version, error) {
	split := strings.Split(versionString, ".")
	if len(split) < 1 || len(split) > 3 {
		return Version{}, fmt.Errorf("failed to parse version: format did not meet expected MAJOR.MINOR or MAJOR.MINOR.PATCH pattern: %v", versionString)
	}

	majorVersion, err := strconv.Atoi(split[0])
	if err != nil {
		return Version{}, fmt.Errorf("failed to parse version: major %v is not a number", split[0])
	}

	minorVersion := 0
	if len(split) >= 2 {
		minorVersion, err = strconv.Atoi(split[1])
		if err != nil {
			return Version{}, fmt.Errorf("failed to parse version: minor %v is not a number", split[1])
		}
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
