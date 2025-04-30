/*
 * @license
 * Copyright 2025 Dynatrace LLC
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

package resolver_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	refParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/reference"
	valueParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/dependency_resolution/resolver"
)

func TestAhoCorasickResolver(t *testing.T) {
	tests := []struct {
		name            string
		allConfigs      map[string]config.Config
		validatedConfig *config.Config
		expected        *config.Config
	}{
		{
			"single config works",
			map[string]config.Config{
				"id": {
					Type:       config.ClassicApiType{Api: "api"},
					Template:   template.NewInMemoryTemplate("id", "content"),
					Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id"},
				},
			},
			&config.Config{
				Type:       config.ClassicApiType{Api: "api"},
				Template:   template.NewInMemoryTemplate("id", "content"),
				Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id"},
			},
			&config.Config{
				Type:       config.ClassicApiType{Api: "api"},
				Template:   template.NewInMemoryTemplate("id", "content"),
				Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id"},
			},
		},
		{
			"disjunctive config works",
			map[string]config.Config{
				"id": {
					Type:       config.ClassicApiType{Api: "api"},
					Template:   template.NewInMemoryTemplate("id", "content"),
					Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id"},
				},
				"id2": {
					Type:       config.ClassicApiType{Api: "api"},
					Template:   template.NewInMemoryTemplate("id2", "content2"),
					Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id2"},
				},
			},
			&config.Config{
				Type:       config.ClassicApiType{Api: "api"},
				Template:   template.NewInMemoryTemplate("id", "content"),
				Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id"},
			},
			&config.Config{
				Type:       config.ClassicApiType{Api: "api"},
				Template:   template.NewInMemoryTemplate("id", "content"),
				Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id"},
			},
		},
		{
			"referencing a config works",
			map[string]config.Config{
				"c1-id": {
					Type:       config.ClassicApiType{Api: "api"},
					Template:   template.NewInMemoryTemplate("c1-id", "content"),
					Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "c1-id"},
					Parameters: config.Parameters{},
				},
				"c2-id": {
					Type:       config.ClassicApiType{Api: "api"},
					Template:   template.NewInMemoryTemplate("c2-id", "something something c1-id something something"),
					Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "c2-id"},
					Parameters: config.Parameters{},
				},
			},
			&config.Config{
				Type:       config.ClassicApiType{Api: "api"},
				Template:   template.NewInMemoryTemplate("c2-id", "something something c1-id something something"),
				Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "c2-id"},
				Parameters: config.Parameters{},
			},
			&config.Config{
				Type:       config.ClassicApiType{Api: "api"},
				Template:   template.NewInMemoryTemplate("c2-id", makeTemplateString("something something %s something something", "api", "c1-id")),
				Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "c2-id"},
				Parameters: config.Parameters{
					resolver.CreateParameterName("api", "c1-id"): refParam.New("project", "api", "c1-id", "id"),
				},
			},
		},
		{
			"no parameter created for false positive",
			map[string]config.Config{
				"c1-id": {
					Type:       config.ClassicApiType{Api: "api"},
					Template:   template.NewInMemoryTemplate("c1-id", "content"),
					Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "c1-id"},
					Parameters: config.Parameters{},
				},
				"c2-id": {
					Type:       config.ClassicApiType{Api: "api"},
					Template:   template.NewInMemoryTemplate("c2-id", "something somethingc1-idsomething something"),
					Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "c2-id"},
					Parameters: config.Parameters{},
				},
			},
			&config.Config{
				Type:       config.ClassicApiType{Api: "api"},
				Template:   template.NewInMemoryTemplate("c2-id", "something somethingc1-idsomething something"),
				Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "c2-id"},
				Parameters: config.Parameters{},
			},
			&config.Config{
				Type:       config.ClassicApiType{Api: "api"},
				Template:   template.NewInMemoryTemplate("c2-id", "something somethingc1-idsomething something"),
				Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "c2-id"},
				Parameters: config.Parameters{},
			},
		},
		{
			"cyclic reference works",
			map[string]config.Config{
				"c1-id": {
					Type:       config.ClassicApiType{Api: "api"},
					Template:   template.NewInMemoryTemplate("c1-id", `"template of config 1 references config 2: c2-id"`),
					Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "c1-id"},
					Parameters: config.Parameters{
						resolver.CreateParameterName("api", "c2-id"): refParam.New("project", "api", "c2-id", "id"),
					},
				},
				"c2-id": {
					Type:       config.ClassicApiType{Api: "api"},
					Template:   template.NewInMemoryTemplate("c2-id", `"template of config 2 references config 1: c1-id"`),
					Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "c2-id"},
					Parameters: config.Parameters{},
				},
			},
			&config.Config{
				Type:       config.ClassicApiType{Api: "api"},
				Template:   template.NewInMemoryTemplate("c2-id", `"template of config 2 references config 1: c1-id"`),
				Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "c2-id"},
				Parameters: config.Parameters{},
			},
			&config.Config{
				Type:       config.ClassicApiType{Api: "api"},
				Template:   template.NewInMemoryTemplate("c2-id", makeTemplateString(`"template of config 2 references config 1: %s"`, "api", "c1-id")),
				Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "c2-id"},
				Parameters: config.Parameters{
					resolver.CreateParameterName("api", "c1-id"): refParam.New("project", "api", "c1-id", "id"),
				},
			},
		},
		{
			"Scope is replaced in dependency resolution",
			map[string]config.Config{
				"id1": {
					Template:   template.NewInMemoryTemplate("id1", ""),
					Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id1"},
					Type:       config.SettingsType{SchemaId: "api"},
					Parameters: config.Parameters{
						config.ScopeParameter: &valueParam.ValueParameter{Value: "id2"},
					},
				},
				"id2": {
					Template:   template.NewInMemoryTemplate("id2", ""),
					Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id2"},
					Type:       config.SettingsType{SchemaId: "api"},
					Parameters: config.Parameters{
						config.ScopeParameter: &valueParam.ValueParameter{Value: "tenant"},
					},
				},
			},
			&config.Config{
				Template:   template.NewInMemoryTemplate("id1", ""),
				Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id1"},
				Type:       config.SettingsType{SchemaId: "api"},
				Parameters: config.Parameters{
					config.ScopeParameter: &valueParam.ValueParameter{Value: "id2"},
				},
			},
			&config.Config{
				Template:   template.NewInMemoryTemplate("id1", ""),
				Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id1"},
				Type:       config.SettingsType{SchemaId: "api"},
				Parameters: config.Parameters{
					config.ScopeParameter: refParam.New("project", "api", "id2", "id"),
				},
			},
		},
		{
			"Scope is not replaced if no dependency is present",
			map[string]config.Config{
				"id1": {
					Template:   template.NewInMemoryTemplate("id1", ""),
					Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id1"},
					Type:       config.SettingsType{SchemaId: "api"},
					Parameters: config.Parameters{
						config.ScopeParameter: &valueParam.ValueParameter{Value: "id2"},
					},
				},
				"id2": {
					Template:   template.NewInMemoryTemplate("id2", ""),
					Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id2"},
					Type:       config.SettingsType{SchemaId: "api"},
					Parameters: config.Parameters{
						config.ScopeParameter: &valueParam.ValueParameter{Value: "tenant"},
					},
				},
			},
			&config.Config{
				Template:   template.NewInMemoryTemplate("id2", ""),
				Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id2"},
				Type:       config.SettingsType{SchemaId: "api"},
				Parameters: config.Parameters{
					config.ScopeParameter: &valueParam.ValueParameter{Value: "tenant"},
				},
			},
			&config.Config{
				Template:   template.NewInMemoryTemplate("id2", ""),
				Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id2"},
				Type:       config.SettingsType{SchemaId: "api"},
				Parameters: config.Parameters{
					config.ScopeParameter: &valueParam.ValueParameter{Value: "tenant"},
				},
			},
		},
		{
			"Dashboards should not be able to reference a dashboard-share-setting, even if it's the dashboard's share setting",
			map[string]config.Config{
				"dashboard-id": {
					Template:   template.NewInMemoryTemplate("t1", "dashboard-id-share"),
					Coordinate: coordinate.Coordinate{Project: "project", Type: "dashboard", ConfigId: "dashboard-id"},
					Type:       config.ClassicApiType{Api: "dashboard"},
					Parameters: config.Parameters{},
				},
				"dashboard-id-share": {
					Template:   template.NewInMemoryTemplate("t2", ""),
					Coordinate: coordinate.Coordinate{Project: "project", Type: "dashboard-share-setting", ConfigId: "dashboard-id-share"},
					Type:       config.ClassicApiType{Api: "dashboard-share-setting"},
					Parameters: config.Parameters{
						config.ScopeParameter: refParam.New("project", "dashboard", "dashboard-id", "id"),
					},
				},
			},
			&config.Config{
				Template:   template.NewInMemoryTemplate("t1", "dashboard-id-share"),
				Coordinate: coordinate.Coordinate{Project: "project", Type: "dashboard", ConfigId: "dashboard-id"},
				Type:       config.ClassicApiType{Api: "dashboard"},
				Parameters: config.Parameters{},
			},
			&config.Config{
				Template:   template.NewInMemoryTemplate("t1", "dashboard-id-share"),
				Coordinate: coordinate.Coordinate{Project: "project", Type: "dashboard", ConfigId: "dashboard-id"},
				Type:       config.ClassicApiType{Api: "dashboard"},
				Parameters: config.Parameters{},
			},
		},
	}
	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {
			res, err := resolver.AhoCorasickResolver(test.allConfigs)
			require.NoError(t, err)

			err = res.ResolveDependencyReferences(test.validatedConfig)
			require.NoError(t, err)

			assert.Equal(t, test.expected, test.validatedConfig)
		})
	}
}

func TestAhoCorasickResolver_ErrorOnEmpty(t *testing.T) {
	_, err := resolver.AhoCorasickResolver(map[string]config.Config{})
	require.Error(t, err)
}

func makeTemplateString(template, api, configId string) string {
	return fmt.Sprintf(template, "{{."+resolver.CreateParameterName(api, configId)+"}}")
}
