//go:build unit

// @license
// Copyright 2021 Dynatrace LLC
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

package deploy

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/config/v2/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/config/v2/parameter"
	"gotest.tools/assert"
	"reflect"
	"testing"
)

func TestNewEntityMap(t *testing.T) {
	type args struct {
		apis api.ApiMap
	}
	tests := []struct {
		name string
		args args
		want *EntityMap
	}{
		{
			name: "Test crate entity map",
			args: args{api.ApiMap{"dashboard": api.NewStandardApi("dashboard", "dashboard", false, "dashboard-v2", false)}},
			want: &EntityMap{
				resolvedEntities: parameter.ResolvedEntities{},
				knownEntityNames: map[string]map[string]struct{}{"dashboard": {}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewEntityMap(tt.args.apis); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewEntityMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEntityMap_PutResolved(t *testing.T) {

	t.Run("EntityMap - PutResolved", func(t *testing.T) {
		c1 := coordinate.Coordinate{
			Project:  "project",
			Type:     "type",
			ConfigId: "configID",
		}

		r1 := parameter.ResolvedEntity{
			EntityName: "entityName",
			Coordinate: c1,
		}

		entityMap := NewEntityMap(api.ApiMap{"dashboard": api.NewStandardApi("dashboard", "dashboard", false, "dashboard-v2", false)})
		entityMap.PutResolved(c1, r1)
		assert.Equal(t, entityMap.Known("type", "entityName"), true)
		assert.DeepEqual(t, entityMap.Resolved(), parameter.ResolvedEntities{
			c1: r1,
		})
	})

	t.Run("EntityMap - PutResolved - skipped", func(t *testing.T) {
		c1 := coordinate.Coordinate{
			Project:  "project",
			Type:     "type",
			ConfigId: "configID",
		}

		r1 := parameter.ResolvedEntity{
			EntityName: "entityName",
			Coordinate: c1,
			Skip:       true,
		}

		entityMap := NewEntityMap(api.ApiMap{"dashboard": api.NewStandardApi("dashboard", "dashboard", false, "dashboard-v2", false)})
		entityMap.PutResolved(c1, r1)
		assert.Equal(t, entityMap.Known("type", "entityName"), false)
		assert.DeepEqual(t, entityMap.Resolved(), parameter.ResolvedEntities{
			c1: r1,
		})
	})

	t.Run("EntityMap - PutResolved - No entity name", func(t *testing.T) {
		c1 := coordinate.Coordinate{
			Project:  "project",
			Type:     "type",
			ConfigId: "configID",
		}

		r1 := parameter.ResolvedEntity{Coordinate: c1}

		entityMap := NewEntityMap(api.ApiMap{"dashboard": api.NewStandardApi("dashboard", "dashboard", false, "dashboard-v2", false)})
		entityMap.PutResolved(c1, r1)
		assert.Equal(t, entityMap.Known("type", ""), false)
		assert.DeepEqual(t, entityMap.Resolved(), parameter.ResolvedEntities{
			c1: r1,
		})
	})

}
