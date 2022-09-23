//go:build unit
// +build unit

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

package v2

import (
	config "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/coordinate"
	"testing"

	"gotest.tools/assert"
)

func Test_checkDuplicatedId(t *testing.T) {
	assert.Equal(t, len(getDuplicatedId(nil)), 0)
	assert.Equal(t, len(getDuplicatedId(singleElementList())), 0)
	assert.Equal(t, len(getDuplicatedId(listOfDifferentElements())), 0)
	assert.Equal(t, len(getDuplicatedId(oneDuplicatedElement())), 1)
	assert.Equal(t, getDuplicatedId(oneDuplicatedElement())[0], "id")
}

func Test_reportsOneDuplicateId(t *testing.T) {
	assert.Equal(t, len(getDuplicatedId(twiceDuplicatedElement())), 1)
}

func Test_notADuplicateIfFullCoordinateIsDifferent(t *testing.T) {
	assert.Equal(t, len(getDuplicatedId(duplicatedIdInDifferentProjects())), 0)
	assert.Equal(t, len(getDuplicatedId(duplicatedIdInDifferentApis())), 0)
}

func singleElementList() []config.Config {
	return []config.Config{{Coordinate: coordinate.Coordinate{Config: "id"}}}
}

func listOfDifferentElements() []config.Config {
	return []config.Config{
		{Coordinate: coordinate.Coordinate{Project: "project", Api: "api", Config: "id"}},
		{Coordinate: coordinate.Coordinate{Project: "project1", Api: "api1", Config: "id1"}}}
}

func oneDuplicatedElement() []config.Config {
	return []config.Config{
		{Coordinate: coordinate.Coordinate{Project: "project", Api: "api", Config: "id"}},
		{Coordinate: coordinate.Coordinate{Project: "project", Api: "api", Config: "id"}},
		{Coordinate: coordinate.Coordinate{Project: "project", Api: "api", Config: "id1"}}}
}

func twiceDuplicatedElement() []config.Config {
	return []config.Config{
		{Coordinate: coordinate.Coordinate{Project: "project", Api: "api", Config: "id"}},
		{Coordinate: coordinate.Coordinate{Project: "project", Api: "api", Config: "id"}},
		{Coordinate: coordinate.Coordinate{Project: "project", Api: "api", Config: "id"}},
		{Coordinate: coordinate.Coordinate{Project: "project", Api: "api", Config: "id1"}}}
}

func duplicatedIdInDifferentProjects() []config.Config {
	return []config.Config{
		{Coordinate: coordinate.Coordinate{Project: "project", Api: "api", Config: "id"}},
		{Coordinate: coordinate.Coordinate{Project: "project1", Api: "api", Config: "id"}}}
}

func duplicatedIdInDifferentApis() []config.Config {
	return []config.Config{
		{Coordinate: coordinate.Coordinate{Project: "project", Api: "aws-credentials", Config: "credential"}},
		{Coordinate: coordinate.Coordinate{Project: "project", Api: "azure-credentials", Config: "credential"}}}
}
