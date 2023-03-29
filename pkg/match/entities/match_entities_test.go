// @license
// Copyright 2023 Dynatrace LLC
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

//go:build unit

package entities

import (
	"reflect"
	"testing"

	"github.com/spf13/afero"
	"gotest.tools/assert"

	config "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/match"
	project "github.com/dynatrace/dynatrace-configuration-as-code/pkg/project/v2"
)

func TestMatchEntities(t *testing.T) {

	tests := []struct {
		name                string
		matchParameters     match.MatchParameters
		entityPerTypeSource project.ConfigsPerType
		entityPerTypeTarget project.ConfigsPerType
		wantStats           []string
		wantNbEntSource     int
		wantNbEntTarget     int
	}{
		{
			name:            "MatchEntities",
			matchParameters: genTestMatchParameters(),
			entityPerTypeSource: project.ConfigsPerType{
				"AZURE_VM": []config.Config{
					config.Config{
						Template: template.NewDownloadTemplate("AZURE_VM", "AZURE_VM", entityListJsonSorted),
						Type: config.EntityType{
							EntitiesType: "AZURE_VM",
							From:         "1",
							To:           "2",
						},
					},
				},
			},
			entityPerTypeTarget: project.ConfigsPerType{
				"AZURE_VM": []config.Config{
					config.Config{
						Template: template.NewDownloadTemplate("AZURE_VM", "AZURE_VM", entityListJson),
						Type: config.EntityType{
							EntitiesType: "AZURE_VM",
							From:         "2",
							To:           "3",
						},
					},
				},
			},
			wantStats: []string{
				"                                                             Type    Matched MultiMatched  UnMatched      Total     Source",
				"                                                         AZURE_VM          3            0          0          3          3",
			},
			wantNbEntSource: 3,
			wantNbEntTarget: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			gotStats, gotNbEntitiesSource, gotNbEntitiesTarget, err := MatchEntities(fs, tt.matchParameters, tt.entityPerTypeSource, tt.entityPerTypeTarget)

			assert.NilError(t, err)
			if !reflect.DeepEqual(gotStats, tt.wantStats) {
				t.Errorf("LoadMatchingParameters() \ngotStats  = %v\nwantStats = %v", gotStats, tt.wantStats)
			}
			assert.Equal(t, gotNbEntitiesSource, tt.wantNbEntSource)
			assert.Equal(t, gotNbEntitiesTarget, tt.wantNbEntTarget)
		})
	}
}
