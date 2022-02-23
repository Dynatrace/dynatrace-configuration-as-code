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
	uuidLib "github.com/google/uuid"
)

// UUID v3 (MD5 hash based) for "dynatrace.com" in the "URL" namespace
const dynatraceNamespaceUuid = "a2673303-5d44-3a6e-999e-9a9d83487e64"

// GenerateUuidFromName generates a fixed UUID from a given configuration name.
// This is used when dealing with select Dynatrace APIs that do not/or no longer support unique name properties.
// As a convention between monaco and such APIs, both monaco and Dynatrace will generate the same name-based UUID
// using UUID v3 (MD5 hash based) with a "dynatrace.com" URL namespace UUID.
func GenerateUuidFromName(name string) (string, error) {
	namespaceUuid, err := uuidLib.Parse(dynatraceNamespaceUuid)
	if err != nil {
		return "", err
	}
	uuid := uuidLib.NewMD5(namespaceUuid, []byte(name)).String()
	return uuid, nil
}
