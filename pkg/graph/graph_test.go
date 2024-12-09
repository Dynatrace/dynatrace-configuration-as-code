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

package graph_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	graph2 "gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/graph"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
)

func TestConfigGraphPerEnvironment_GetConnectedConfigs(t *testing.T) {
	projectId := "project1"
	referencedProjectId := "project2"
	environmentName := "dev"

	dashboardApiId := "dashboard"
	dashboardConfigCoordinate := coordinate.Coordinate{
		Project:  projectId,
		Type:     dashboardApiId,
		ConfigId: "sample dashboard",
	}

	autoTagApiId := "auto-tag"
	autoTagConfigId := "tag"
	autoTagCoordinates := coordinate.Coordinate{
		Project:  referencedProjectId,
		Type:     autoTagApiId,
		ConfigId: autoTagConfigId,
	}

	referencedPropertyName := "tagId"

	individualConfigCoordinate := coordinate.Coordinate{
		Project:  projectId,
		Type:     dashboardApiId,
		ConfigId: "Random Dashboard",
	}

	projects := []project.Project{
		{
			Id: projectId,
			Configs: project.ConfigsPerTypePerEnvironments{
				environmentName: {
					dashboardApiId: []config.Config{
						{
							Coordinate:  dashboardConfigCoordinate,
							Environment: environmentName,
							Parameters: map[string]parameter.Parameter{
								"autoTagId": &parameter.DummyParameter{
									References: []parameter.ParameterReference{
										{
											Config:   autoTagCoordinates,
											Property: referencedPropertyName,
										},
									},
								},
							},
						},
						{
							Coordinate:  individualConfigCoordinate,
							Environment: environmentName,
							Parameters: map[string]parameter.Parameter{
								"name": &parameter.DummyParameter{
									Value: "sample",
								},
							},
						},
					},
				},
			},
		},
		{
			Id: referencedProjectId,
			Configs: project.ConfigsPerTypePerEnvironments{
				environmentName: {
					autoTagApiId: []config.Config{
						{
							Coordinate:  autoTagCoordinates,
							Environment: environmentName,
							Parameters: map[string]parameter.Parameter{
								referencedPropertyName: &parameter.DummyParameter{
									Value: "10",
								},
							},
						},
					},
				},
			},
		},
	}

	environments := []string{
		environmentName,
	}

	graphs := graph.New(projects, environments)
	components, errs := graphs.GetIndependentlySortedConfigs(environmentName)
	assert.NoError(t, errs)
	assert.Len(t, components, 2)

	for _, comp := range components {
		cfgs := comp.SortedNodes
		if len(cfgs) > 1 {
			assert.Len(t, cfgs, 2)
			assert.Equal(t, autoTagCoordinates, cfgs[0].(graph.ConfigNode).Config.Coordinate, "expected auto-tag to be sorted first")
			assert.Equal(t, dashboardConfigCoordinate, cfgs[1].(graph.ConfigNode).Config.Coordinate, "expected dashboard sorted after auto-tag it depends on")
		} else {
			assert.Len(t, cfgs, 1)
			assert.Equal(t, individualConfigCoordinate, cfgs[0].(graph.ConfigNode).Config.Coordinate)
		}
	}
}

func TestGraphExport(t *testing.T) {
	projectId := "project1"
	referencedProjectId := "project2"
	environmentName := "dev"

	dashboardApiId := "dashboard"
	dashboardConfigCoordinate := coordinate.Coordinate{
		Project:  projectId,
		Type:     dashboardApiId,
		ConfigId: "sample dashboard",
	}

	autoTagApiId := "auto-tag"
	autoTagConfigId := "tag"
	autoTagCoordinates := coordinate.Coordinate{
		Project:  referencedProjectId,
		Type:     autoTagApiId,
		ConfigId: autoTagConfigId,
	}

	referencedPropertyName := "tagId"

	individualConfigCoordinate := coordinate.Coordinate{
		Project:  projectId,
		Type:     dashboardApiId,
		ConfigId: "Random Dashboard",
	}

	projects := []project.Project{
		{
			Id: projectId,
			Configs: project.ConfigsPerTypePerEnvironments{
				environmentName: {
					dashboardApiId: []config.Config{
						{
							Coordinate:  dashboardConfigCoordinate,
							Environment: environmentName,
							Parameters: map[string]parameter.Parameter{
								"autoTagId": &parameter.DummyParameter{
									References: []parameter.ParameterReference{
										{
											Config:   autoTagCoordinates,
											Property: referencedPropertyName,
										},
									},
								},
							},
						},
						{
							Coordinate:  individualConfigCoordinate,
							Environment: environmentName,
							Parameters: map[string]parameter.Parameter{
								"name": &parameter.DummyParameter{
									Value: "sample",
								},
							},
						},
					},
				},
			},
		},
		{
			Id: referencedProjectId,
			Configs: project.ConfigsPerTypePerEnvironments{
				environmentName: {
					autoTagApiId: []config.Config{
						{
							Coordinate:  autoTagCoordinates,
							Environment: environmentName,
							Parameters: map[string]parameter.Parameter{
								referencedPropertyName: &parameter.DummyParameter{
									Value: "10",
								},
							},
						},
					},
				},
			},
		},
	}

	environments := []string{
		environmentName,
	}

	graphs := graph.New(projects, environments)
	dot, err := graphs.EncodeToDOT(environmentName)
	assert.NoError(t, err)
	assert.Equal(t, string(dot), "strict digraph dev_dependency_graph {\n  // Node definitions.\n  \"project1:dashboard:sample dashboard\";\n  \"project1:dashboard:Random Dashboard\";\n  \"project2:auto-tag:tag\";\n\n  // Edge definitions.\n  \"project2:auto-tag:tag\" -> \"project1:dashboard:sample dashboard\";\n}")
}

func TestGraphCycleErrors(t *testing.T) {
	projectId := "project1"
	referencedProjectId := "project2"
	environmentName := "dev"

	dashboardApiId := "dashboard"
	dashboardConfigID := "dashboard"
	dashboardConfigCoordinate := coordinate.Coordinate{
		Project:  projectId,
		Type:     dashboardApiId,
		ConfigId: dashboardConfigID,
	}

	autoTagApiId := "auto-tag"
	autoTagConfigId := "tag"
	autoTagCoordinates := coordinate.Coordinate{
		Project:  referencedProjectId,
		Type:     autoTagApiId,
		ConfigId: autoTagConfigId,
	}

	secondTagConfigID := "tag2"
	secondTagConfigCoordinate := coordinate.Coordinate{
		Project:  referencedProjectId,
		Type:     autoTagApiId,
		ConfigId: secondTagConfigID,
	}

	individualConfigCoordinate := coordinate.Coordinate{
		Project:  projectId,
		Type:     dashboardApiId,
		ConfigId: "Random Dashboard",
	}

	dashCycleCoordinate1 := coordinate.Coordinate{
		Project:  projectId,
		Type:     dashboardApiId,
		ConfigId: "Dash cycle 1",
	}
	dashCycleCoordinate2 := coordinate.Coordinate{
		Project:  projectId,
		Type:     dashboardApiId,
		ConfigId: "Dash cycle 2",
	}

	dash1 := config.Config{
		Coordinate:  dashboardConfigCoordinate,
		Environment: environmentName,
		Parameters: map[string]parameter.Parameter{
			"autoTagName": &parameter.DummyParameter{
				References: []parameter.ParameterReference{
					{
						Config:   autoTagCoordinates,
						Property: "name",
					},
				},
			},
			"name": &parameter.DummyParameter{
				Value: "Dashboard #1 - Referenced by Tag #2",
			},
		},
	}
	dash2 := config.Config{
		Coordinate:  individualConfigCoordinate,
		Environment: environmentName,
		Parameters: map[string]parameter.Parameter{
			"name": &parameter.DummyParameter{
				Value: "Dashboard #2 - On it's own",
			},
		},
	}
	dash3 := config.Config{
		Coordinate:  dashCycleCoordinate1,
		Environment: environmentName,
		Parameters: map[string]parameter.Parameter{
			"name": &parameter.DummyParameter{
				Value: "Dashboard #3 - References dash #4",
			},
			"dash4": &parameter.DummyParameter{
				References: []parameter.ParameterReference{
					{
						Config:   dashCycleCoordinate2,
						Property: "name",
					},
				},
			},
		},
	}
	dash4 := config.Config{
		Coordinate:  dashCycleCoordinate2,
		Environment: environmentName,
		Parameters: map[string]parameter.Parameter{
			"name": &parameter.DummyParameter{
				Value: "Dashboard #4 - References dash #3",
			},
			"dash3": &parameter.DummyParameter{
				References: []parameter.ParameterReference{
					{
						Config:   dashCycleCoordinate1,
						Property: "name",
					},
				},
			},
		},
	}

	tag1 := config.Config{
		Coordinate:  autoTagCoordinates,
		Environment: environmentName,
		Parameters: map[string]parameter.Parameter{
			"name": &parameter.DummyParameter{
				Value: "Tag #1 - Referenced by Dashboard #1",
			},
			"otherTag": &parameter.DummyParameter{
				References: []parameter.ParameterReference{
					{
						Config:   secondTagConfigCoordinate,
						Property: "name",
					},
				},
			},
		},
	}
	tag2 := config.Config{
		Coordinate:  secondTagConfigCoordinate,
		Environment: environmentName,
		Parameters: map[string]parameter.Parameter{
			"name": &parameter.DummyParameter{
				Value: "Tag #2 - Referenced by Tag #1, Referencing Dashboard #1 (cycle via Tag #1)",
			},
			"dashboard": &parameter.DummyParameter{
				References: []parameter.ParameterReference{
					{
						Config:   dashboardConfigCoordinate,
						Property: "name",
					},
				},
			},
		},
	}

	projects := []project.Project{
		{
			Id: projectId,
			Configs: project.ConfigsPerTypePerEnvironments{
				environmentName: {
					dashboardApiId: []config.Config{
						dash1,
						dash2,
						dash3,
						dash4,
					},
				},
			},
		},
		{
			Id: referencedProjectId,
			Configs: project.ConfigsPerTypePerEnvironments{
				environmentName: {
					autoTagApiId: []config.Config{
						tag1,
						tag2,
					},
				},
			},
		},
	}

	environments := []string{
		environmentName,
	}

	graphs := graph.New(projects, environments)

	_, errs := graphs.GetIndependentlySortedConfigs(environmentName)
	assert.Error(t, errs)
	assert.Len(t, errs, 2, "expected cyclic dependency errors for two components")
	assert.ElementsMatch(t, errs, []graph.CyclicDependencyError{
		{
			Environment: environmentName,
			ConfigsInDependencyCycle: [][]graph.DependencyLocation{
				{
					{Coordinate: dash1.Coordinate},
					{Coordinate: tag1.Coordinate},
					{Coordinate: tag2.Coordinate},
				},
			},
		},
		{
			Environment: environmentName,
			ConfigsInDependencyCycle: [][]graph.DependencyLocation{
				{
					{Coordinate: dash3.Coordinate},
					{Coordinate: dash4.Coordinate},
				},
			},
		},
	})
}

func TestRoots(t *testing.T) {
	const (
		n1 simple.Node = iota
		n2
		n3
		n4
		n5
	)

	tests := []struct {
		name     string
		setup    func() graph2.Directed
		expected []graph2.Node
	}{
		{
			name: "Empty {}",
			setup: func() graph2.Directed {
				return simple.NewDirectedGraph()
			},
			expected: []graph2.Node{},
		},
		{
			name: "One node {1}",
			setup: func() graph2.Directed {
				g := simple.NewDirectedGraph()
				g.AddNode(n1)
				return g
			},
			expected: []graph2.Node{n1},
		},
		{
			name: "Two independent nodes {1, 2}",
			setup: func() graph2.Directed {
				g := simple.NewDirectedGraph()
				g.AddNode(n1)
				g.AddNode(n2)
				return g
			},
			expected: []graph2.Node{n1, n2},
		},
		{
			name: "Two dependent nodes {1->2}",
			setup: func() graph2.Directed {
				g := simple.NewDirectedGraph()
				g.SetEdge(g.NewEdge(n1, n2))
				return g
			},
			expected: []graph2.Node{n1},
		},
		{
			name: "Two cyclic nodes {1->2->1}",
			setup: func() graph2.Directed {
				g := simple.NewDirectedGraph()
				g.SetEdge(g.NewEdge(n1, n2))
				g.SetEdge(g.NewEdge(n2, n1))
				return g
			},
			expected: []graph2.Node{},
		},
		{
			name: "Five nodes, two independent components {1->2, 3->4->5}",
			setup: func() graph2.Directed {
				g := simple.NewDirectedGraph()
				g.SetEdge(g.NewEdge(n1, n2))
				g.SetEdge(g.NewEdge(n3, n4))
				g.SetEdge(g.NewEdge(n4, n5))

				return g
			},
			expected: []graph2.Node{n1, n3},
		},
		{
			name: "3 roots, 2 shared children {1->2, 3->2, 3->4, 5->4}",
			setup: func() graph2.Directed {
				g := simple.NewDirectedGraph()
				g.SetEdge(g.NewEdge(n1, n2))
				g.SetEdge(g.NewEdge(n3, n2))
				g.SetEdge(g.NewEdge(n3, n4))
				g.SetEdge(g.NewEdge(n5, n4))

				return g
			},
			expected: []graph2.Node{n1, n3, n5},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			g := test.setup()
			r := graph.Roots(g)

			assert.ElementsMatch(t, r, test.expected)
		})
	}
}
