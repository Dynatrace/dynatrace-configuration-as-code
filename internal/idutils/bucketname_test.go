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

package idutils_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
)

// TestGenerateBucketName tests that bucket names are generated as expected both when the SanitizeBucketNames feature flag is enabled and disabled.
func TestGenerateBucketName(t *testing.T) {
	type args struct {
		c coordinate.Coordinate
	}
	tests := []struct {
		name                       string
		sanitizeBucketNamesEnabled bool
		coordinate                 coordinate.Coordinate
		expectedBucketName         string
	}{
		{
			name:               "Project name and config id are simply concatenated when feature flag disabled",
			coordinate:         coordinate.Coordinate{Project: "project", Type: "bucket", ConfigId: "bucket"},
			expectedBucketName: "project_bucket",
		},
		{
			name:               "Short project name and config id are simply concatenated when feature flag disabled",
			coordinate:         coordinate.Coordinate{Project: "p", Type: "bucket", ConfigId: "b"},
			expectedBucketName: "p_b",
		},
		{
			name:                       "Project name and config id are simply concatenated when feature flag enabled",
			sanitizeBucketNamesEnabled: true,
			coordinate:                 coordinate.Coordinate{Project: "project", Type: "bucket", ConfigId: "bucket"},
			expectedBucketName:         "project_bucket",
		},
		{
			name:                       "Bucket name sanitized when feature flag enabed",
			sanitizeBucketNamesEnabled: true,
			coordinate:                 coordinate.Coordinate{Project: "p", Type: "bucket", ConfigId: "b"},
			expectedBucketName:         "pb",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(featureflags.SanitizeBucketNames.EnvName(), strconv.FormatBool(tt.sanitizeBucketNamesEnabled))
			assert.Equal(t, tt.expectedBucketName, idutils.GenerateBucketName(tt.coordinate))
		})
	}
}

// TestGenerateBucketNameSanitizes tests that generated bucket names are sanitized as expected.
func TestGenerateBucketNameSanitizes(t *testing.T) {
	t.Setenv(featureflags.SanitizeBucketNames.EnvName(), "true")
	tests := []struct {
		name               string
		inputCoordinate    coordinate.Coordinate
		expectedOutputName string
	}{
		{
			name:               "Invalid first character is removed",
			inputCoordinate:    coordinate.Coordinate{Project: "0abc", Type: "bucket", ConfigId: "bucket"},
			expectedOutputName: "abc_bucket",
		},
		{
			name:               "Multiple invalid first characters are removed",
			inputCoordinate:    coordinate.Coordinate{Project: "0_abc", Type: "bucket", ConfigId: "bucket"},
			expectedOutputName: "abc_bucket",
		},
		{
			name:               "Invalid second character is removed",
			inputCoordinate:    coordinate.Coordinate{Project: "p_1", Type: "bucket", ConfigId: "bucket"},
			expectedOutputName: "p1_bucket",
		},
		{
			name:               "Multiple invalid second characters are removed",
			inputCoordinate:    coordinate.Coordinate{Project: "p__1", Type: "bucket", ConfigId: "bucket"},
			expectedOutputName: "p1_bucket",
		},
		{
			name:               "Invalid first and second characters are removed",
			inputCoordinate:    coordinate.Coordinate{Project: "_p_1", Type: "bucket", ConfigId: "bucket"},
			expectedOutputName: "p1_bucket",
		},
		{
			name:               "Length is limited to 100 characters",
			inputCoordinate:    coordinate.Coordinate{Project: "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuv", Type: "bucket", ConfigId: "bucket"},
			expectedOutputName: "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuv",
		},
		{
			name:               "Length is limited to 100 characters after removing invalid characters",
			inputCoordinate:    coordinate.Coordinate{Project: "_p_abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuv", Type: "bucket", ConfigId: "bucket"},
			expectedOutputName: "pabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstu",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedOutputName, idutils.GenerateBucketName(tt.inputCoordinate))
		})
	}
}
