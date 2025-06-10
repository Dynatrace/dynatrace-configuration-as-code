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

package entities

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
)

func TestEntityMap_PutResolved(t *testing.T) {

	t.Run("EntityMap - PutResolved", func(t *testing.T) {
		c1 := coordinate.Coordinate{
			Project:  "project",
			Type:     "type",
			ConfigId: "configID",
		}

		r1 := ResolvedEntity{
			Coordinate: c1,
		}

		entityMap := New()
		entityMap.Put(r1)
		assert.Equal(t, entityMap.Get(), map[coordinate.Coordinate]ResolvedEntity{
			c1: r1,
		})
	})

	t.Run("EntityMap - PutResolved - skipped", func(t *testing.T) {
		c1 := coordinate.Coordinate{
			Project:  "project",
			Type:     "type",
			ConfigId: "configID",
		}

		r1 := ResolvedEntity{
			Coordinate: c1,
			Skip:       true,
		}

		entityMap := New()
		entityMap.Put(r1)
		assert.Equal(t, entityMap.Get(), map[coordinate.Coordinate]ResolvedEntity{
			c1: r1,
		})
	})

	t.Run("EntityMap - PutResolved - No entity name", func(t *testing.T) {
		c1 := coordinate.Coordinate{
			Project:  "project",
			Type:     "type",
			ConfigId: "configID",
		}

		r1 := ResolvedEntity{Coordinate: c1}

		entityMap := New()
		entityMap.Put(r1)
		assert.Equal(t, entityMap.Get(), map[coordinate.Coordinate]ResolvedEntity{
			c1: r1,
		})
	})

}
