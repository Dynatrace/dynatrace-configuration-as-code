//go:build unit

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
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestVersion_GreaterThan(t *testing.T) {
	tests := []struct {
		testVersion    Version
		currentVersion Version
		want           bool
	}{
		{
			Version{1, 236, 0},
			Version{1, 234, 0},
			false,
		},
		{
			Version{1, 236, 0},
			Version{1, 236, 5},
			true,
		},
		{
			Version{1, 236, 0},
			Version{2, 234, 0},
			true,
		},
		{
			Version{2, 236, 0},
			Version{1, 234, 0},
			false,
		},
		{
			Version{2, 236, 0},
			Version{2, 234, 75},
			false,
		},
		{
			Version{1, 236, 0},
			Version{1, 236, 65},
			true,
		},
		{
			Version{1, 236, 65},
			Version{1, 236, 65},
			false,
		},
		{
			Version{1, 236, 65},
			Version{1, 236, 0},
			false,
		},
	}
	for _, tt := range tests {
		tName := fmt.Sprintf("%s>%s==%v", tt.currentVersion, tt.testVersion, tt.want)
		t.Run(tName, func(t *testing.T) {
			if got := tt.currentVersion.GreaterThan(tt.testVersion); got != tt.want {
				t.Errorf("MinimumDynatraceVersionReached() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVersion_SmallerThan(t *testing.T) {
	tests := []struct {
		testVersion    Version
		currentVersion Version
		want           bool
	}{
		{
			Version{1, 236, 0},
			Version{1, 234, 0},
			true,
		},
		{
			Version{1, 236, 0},
			Version{1, 236, 5},
			false,
		},
		{
			Version{1, 236, 0},
			Version{2, 234, 0},
			false,
		},
		{
			Version{2, 236, 0},
			Version{1, 234, 0},
			true,
		},
		{
			Version{2, 236, 0},
			Version{2, 234, 75},
			true,
		},
		{
			Version{1, 236, 0},
			Version{1, 236, 65},
			false,
		},
		{
			Version{1, 236, 65},
			Version{1, 236, 65},
			false,
		},
		{
			Version{1, 236, 65},
			Version{1, 236, 0},
			true,
		},
	}
	for _, tt := range tests {
		tName := fmt.Sprintf("%s<%s==%v", tt.currentVersion, tt.testVersion, tt.want)
		t.Run(tName, func(t *testing.T) {
			if got := tt.currentVersion.SmallerThan(tt.testVersion); got != tt.want {
				t.Errorf("MinimumDynatraceVersionReached() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVersion_Valid(t *testing.T) {
	assert.True(t, Version{0, 0, 0}.Invalid())
	assert.True(t, Version{-1, 1, 1}.Invalid())
	assert.True(t, Version{0, -1, 1}.Invalid())
	assert.True(t, Version{0, 1, -1}.Invalid())
	assert.False(t, Version{0, 1, 0}.Invalid())

}

func Test_ParseVersion(t *testing.T) {
	tests := []struct {
		versionString string
		wantVersion   Version
		wantErr       bool
	}{
		{
			"1.236.0",
			Version{1, 236, 0},
			false,
		},
		{
			"1313.236.5",
			Version{1313, 236, 5},
			false,
		},
		{
			"2.234.0",
			Version{2, 234, 0},
			false,
		},
		{
			"1.5",
			Version{1, 5, 0},
			false,
		},
		{
			"2.0",
			Version{2, 0, 0},
			false,
		},
		{
			"1",
			Version{1, 0, 0},
			false,
		},
		{
			"236.0.20220203-192004",
			Version{},
			true,
		},
		{
			"1.2.236.0.20220203-192004",
			Version{},
			true,
		},
		{
			"hello.236.0.20220203-192004",
			Version{},
			true,
		},
		{
			"version 42",
			Version{},
			true,
		},
		{
			"1.",
			Version{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run("parseVersion("+tt.versionString+")", func(t *testing.T) {
			gotVersion, err := ParseVersion(tt.versionString)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotVersion, tt.wantVersion) {
				t.Errorf("parseVersion() gotVersion = %v, want %v", gotVersion, tt.wantVersion)
			}
		})
	}
}
