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

package memory

import (
	"github.com/stretchr/testify/assert"
	"math"
	"testing"
)

func TestDefaultLimitBytes(t *testing.T) {
	twoGibi := int64(2147483648)
	assert.Equal(t, twoGibi, defaultLimit)
}

func TestSetDefaultLimit(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want bool
	}{
		{
			"sets default limit",
			map[string]string{},
			true,
		},
		{
			"sets default limit - wrong env var",
			map[string]string{
				"GOMELIMT": "42GiB",
			},
			true,
		},
		{
			"doesn't set default limit if GOMEMLIMIT is defined",
			map[string]string{
				"GOMEMLIMIT": "42GiB",
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			assert.Equal(t, tt.want, SetDefaultLimit())
		})
	}
}

func TestSetLimit_CalculatesCorrectRelativeLimit(t *testing.T) {
	tests := []struct {
		name   string
		sysMem getSystemMemoryF
		want   int64
	}{
		{
			"sets expected limit",
			func() uint64 {
				return uint64(4 * gibibyte)
			},
			3 * gibibyte, //0.75 * 4
		},
		{
			"truncates to max int64 if needed",
			func() uint64 {
				return math.MaxUint64
			},
			int64(float64(math.MaxInt64) * 0.75),
		},
		{
			"defaults if sys mem return is invalid",
			func() uint64 {
				return 0
			},
			805306368, // 1 GiB * 0.75
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := setLimit(tt.sysMem)
			assert.Equal(t, tt.want, got)
		})
	}
}
