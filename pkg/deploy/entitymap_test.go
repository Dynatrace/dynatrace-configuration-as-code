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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/v2/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/v2/parameter"
	"gotest.tools/assert"
	"reflect"
	"testing"
)

func TestNewEntityMap(t *testing.T) {
	type args struct {
		apis api.APIs
	}
	tests := []struct {
		name string
		args args
		want *entityMap
	}{
		{
			name: "Test crate entity map",
			args: args{api.APIs{"dashboard": api.API{ID: "dashboard", URLPath: "dashboard", DeprecatedBy: "dashboard-v2"}}},
			want: &entityMap{
				resolvedEntities: parameter.ResolvedEntities{},
				knownEntityNames: map[string]map[string]struct{}{"dashboard": {}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newEntityMap(tt.args.apis); !reflect.DeepEqual(got, tt.want) {
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

		entityMap := newEntityMap(api.APIs{"dashboard": api.API{ID: "dashboard", URLPath: "dashboard", DeprecatedBy: "dashboard-v2"}})
		entityMap.put(r1)
		assert.Equal(t, entityMap.contains("type", "entityName"), true)
		assert.DeepEqual(t, entityMap.get(), parameter.ResolvedEntities{
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

		entityMap := newEntityMap(api.APIs{"dashboard": api.API{ID: "dashboard", URLPath: "dashboard", DeprecatedBy: "dashboard-v2"}})
		entityMap.put(r1)
		assert.Equal(t, entityMap.contains("type", "entityName"), false)
		assert.DeepEqual(t, entityMap.get(), parameter.ResolvedEntities{
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

		entityMap := newEntityMap(api.APIs{"dashboard": api.API{ID: "dashboard", URLPath: "dashboard", DeprecatedBy: "dashboard-v2"}})
		entityMap.put(r1)
		assert.Equal(t, entityMap.contains("type", ""), false)
		assert.DeepEqual(t, entityMap.get(), parameter.ResolvedEntities{
			c1: r1,
		})
	})

}
