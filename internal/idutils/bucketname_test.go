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

// TestSanitizeBucketName tests that bucket names are sanitized as expected.
func TestSanitizeBucketName(t *testing.T) {
	tests := []struct {
		name               string
		inputName          string
		expectedOutputName string
	}{
		{
			name:               "Invalid first character is removed",
			inputName:          "0abc",
			expectedOutputName: "abc",
		},
		{
			name:               "Multiple invalid first characters are removed",
			inputName:          "0_abc",
			expectedOutputName: "abc",
		},
		{
			name:               "Invalid second character is removed",
			inputName:          "a_abc",
			expectedOutputName: "aabc",
		},
		{
			name:               "Multiple invalid second characters are removed",
			inputName:          "a_-abc",
			expectedOutputName: "aabc",
		},
		{
			name:               "Invalid first and second characters are removed",
			inputName:          "0_a_bc",
			expectedOutputName: "abc",
		},
		{
			name:               "Length is limited to 100 characters",
			inputName:          "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz",
			expectedOutputName: "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuv",
		},
		{
			name:               "Length is limited to 100 characters after removing invalid characters",
			inputName:          "0_abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz",
			expectedOutputName: "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuv",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedOutputName, idutils.SanitizeBucketName(tt.inputName))
		})
	}
}
