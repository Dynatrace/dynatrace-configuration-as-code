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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
)

// GenerateBucketName returns the "bucketName" identifier for a bucket based on the coordinate.
// As all buckets are of the same type and never overlap with configs of different types on the same API, the "type" is omitted.
// Since the bucket API does not support colons, we concatenate them using underscores.
func GenerateBucketName(c coordinate.Coordinate) string {
	return fmt.Sprintf("%s_%s", c.Project, c.ConfigId)
}
