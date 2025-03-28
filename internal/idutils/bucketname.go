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
	"fmt"
	"regexp"
	"strings"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
)

// GenerateBucketName returns the "bucketName" identifier for a bucket based on the coordinate.
// As all buckets are of the same type and never overlap with configs of different types on the same API, the "type" is omitted.
// Since the bucket API does not support colons, we concatenate them using underscores.
// If SanitizeBucketNames feature flag is enabled, the name will be sanizited to something matching `([a-z])([a-z0-9])([a-z0-9_-])+` with a max length of 100.
func GenerateBucketName(c coordinate.Coordinate) string {
	name := fmt.Sprintf("%s_%s", c.Project, c.ConfigId)

	if !featureflags.SanitizeBucketNames.Enabled() {
		return name
	}

	sanitizedName := sanitizeBucketName(name)
	if sanitizedName != name {
		log.Warn("Bucket name was changed to '%s' from '%s'", sanitizedName, name)
	}

	return sanitizedName
}

// sanitizeBucketName modifies the specified name to meet the requirements of the bucket-definitions API: pattern: `([a-z])([a-z0-9])([a-z0-9_-])+`,  maxLength: 100.
// It does this by deleting invalid characters and truncating the result if it is more than 100 characters long.
func sanitizeBucketName(name string) string {
	const maximumBucketNameLength = 100

	// make name lower case
	name = strings.ToLower(name)

	// delete any characters that are not in [a-z0-9_-]
	name = regexp.MustCompile(`[^a-z0-9_-]+`).ReplaceAllString(name, "")

	// delete first character while it is not [a-z]
	name = regexp.MustCompile(`^[0-9_-]+`).ReplaceAllString(name, "")

	// delete second character while it is not [a-z0-9]
	name = regexp.MustCompile(`^([a-z])([_-]+)`).ReplaceAllString(name, "$1")

	// truncate if longer that 100 characters. this only works because name only consists of characters [a-z0-9_-]: each is one byte in UTF-8.
	if len(name) > maximumBucketNameLength {
		name = name[0:maximumBucketNameLength]
	}
	return name
}
