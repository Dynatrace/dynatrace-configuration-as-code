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

package idutils

import (
	"encoding/base64"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/coordinate"
)

// GenerateExternalID generates a string that serves as an external ID for a Settings 2.0 object.
// It requires a [[coordinate.Coordinate]] as input and produces a string in the format "monaco:<BASE64_ENCODED_STR>"
// If Type or ConfigId of the passed [[coordinate.Coordinate]] is empty, an error is returned
func GenerateExternalID(c coordinate.Coordinate) (string, error) {
	const prefix = "monaco:"
	const externalIDMaxLength = 500

	if c.Type == "" || c.ConfigId == "" {
		return "", fmt.Errorf("schema id and config id needs to be set to generate an external id for a settings 2.0 object")
	}

	var formattedID string
	if c.Project == "" {
		formattedID = fmt.Sprintf("%s$%s", c.Type, c.ConfigId)
	} else {
		formattedID = fmt.Sprintf("%s$%s$%s", c.Project, c.Type, c.ConfigId)
	}

	encodedID := base64.StdEncoding.EncodeToString([]byte(formattedID))
	encodedIDMaxLength := externalIDMaxLength - len(prefix)
	if len(encodedID) > encodedIDMaxLength {
		encodedID = encodedID[encodedIDMaxLength:]
	}

	return fmt.Sprintf("%s%s", prefix, encodedID), nil
}

type ExternalIDGenerator func(coordinate.Coordinate) (string, error)
