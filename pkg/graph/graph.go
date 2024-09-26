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

package graph

import (
	"errors"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/encoding/dot"
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"
	"gonum.org/v1/gonum/graph/traverse"
)

// coordinateToNodeIDMap is a lookup map from a configuration's coordinate.Coordinate to the int64 ID of its graph node.
type coordinateToNodeIDMap map[coordinate.Coordinate]int64

// referencesLookup is a double lookup map to check dependencies between configs using their coordinates.
type referencesLookup map[coordinate.Coordinate]map[coordinate.Coordinate]struct{}

// ConfigGraph is a directed graph containing ConfigNode s
type ConfigGraph interface {
	graph.Directed
	graph.NodeRemover
}

// ConfigNode implements the gonum graph.Node interface and contains a pointer to its respective config.Config in addition to the unique ID required.
type ConfigNode struct {
	NodeID      int64
	Config      *config.Config
	DOTEncoding string
}

// ID returns the node's integer ID by which it is referenced in the graph.
func (n ConfigNode) ID() int64 {
	return n.NodeID
}

// DOTID returns the node's identifier when printed to a DOT file. For readability of files this is the coordinate.Coordinate of the Config, instead of the node's ID integer.
func (n ConfigNode) DOTID() string {
	if n.DOTEncoding != "" {
		return n.DOTEncoding
	}
	return n.Config.Coordinate.String()
}

func (n ConfigNode) String() string {
	return fmt.Sprintf("ConfigNode{ id=%d, configCoordinate=%v }", n.NodeID, n.Config.Coordinate)
}

// ConfigGraphPerEnvironment is a map of directed dependency graphs per environment name.
type ConfigGraphPerEnvironment map[string]*simple.DirectedGraph

// EncodeToDOT returns a DOT string represenation of the dependency graph for the given environment.
func (graphs ConfigGraphPerEnvironment) EncodeToDOT(environment string) ([]byte, error) {
	g, ok := graphs[environment]
	if !ok {
		return nil, missingDependencyGraphForEnvironmentError(environment)
	}
	return dot.Marshal(g, environment+"_dependency_graph", "", "  ")
}

// SortConfigs returns a slice of config.Config for the given environment sorted according to their dependencies.
func (graphs ConfigGraphPerEnvironment) SortConfigs(environment string) ([]config.Config, error) {
	g, ok := graphs[environment]
	if !ok {
		return nil, missingDependencyGraphForEnvironmentError(environment)
	}

	sortedNodes, err := topo.Sort(g)
	if err != nil {
		sortErr := topo.Unorderable{}
		if ok := errors.As(err, &sortErr); ok {
			return []config.Config{}, newCyclicDependencyError(environment, sortErr)
		}
	}
	sortedCfgs := make([]config.Config, len(sortedNodes))
	for i, n := range sortedNodes {
		sortedCfgs[i] = *n.(ConfigNode).Config
	}

	return sortedCfgs, nil
}

// SortedComponent represents a weakly connected component found in a graph.
type SortedComponent struct {
	// Graph is a directed graph representation of the weakly connected component/sub-graph found in another graph.
	Graph *simple.DirectedGraph
	// SortedNodes are a topologically sorted slice of graph.Node s, which can be deployed in order.
	// This exists for convenience, so callers of GetIndependentlySortedConfigs can work with the component without implementing graph algorithms.
	SortedNodes []graph.Node
}

// GetIndependentlySortedConfigs returns sorted slices of SortedComponent.
// Dependent configurations are returned as a sub-graph as well as a slice, sorted in the correct order to deploy them sequentially.
func (graphs ConfigGraphPerEnvironment) GetIndependentlySortedConfigs(environment string) ([]SortedComponent, error) {
	g, ok := graphs[environment]
	if !ok {
		return nil, missingDependencyGraphForEnvironmentError(environment)
	}

	components := findConnectedComponents(g)
	errs := make(SortingErrors, 0, len(components))
	sortedComponents := make([]SortedComponent, len(components))
	for i, subGraph := range components {
		nodes, err := topo.Sort(subGraph)
		if err != nil {
			sortErr := topo.Unorderable{}
			if ok := errors.As(err, &sortErr); ok {
				errs = append(errs, newCyclicDependencyError(environment, sortErr))
			} else {
				errs = append(errs, fmt.Errorf("failed to sort dependency graph: %w", err))
			}
			continue
		}

		sortedComponents[i] = SortedComponent{
			Graph:       components[i],
			SortedNodes: nodes,
		}
	}
	if len(errs) > 0 {
		return []SortedComponent{}, errs
	}

	return sortedComponents, nil
}

func missingDependencyGraphForEnvironmentError(environment string) error {
	return fmt.Errorf("no dependency graph exists for environment %s", environment)
}

func findConnectedComponents(d *simple.DirectedGraph) []*simple.DirectedGraph {
	u := buildUndirectedGraph(d)

	var graphs []*simple.DirectedGraph

	w := traverse.DepthFirst{
		Traverse: func(edge graph.Edge) bool {
			sub := graphs[len(graphs)-1]
			if d.HasEdgeFromTo(edge.From().ID(), edge.To().ID()) {
				sub.SetEdge(sub.NewEdge(edge.From(), edge.To()))
			} else {
				sub.SetEdge(sub.NewEdge(edge.To(), edge.From()))
			}
			return true
		},
	}

	before := func() {
		sub := simple.NewDirectedGraph()
		graphs = append(graphs, sub)
	}

	during := func(n graph.Node) {
		sub := graphs[len(graphs)-1]
		if sub.Node(n.ID()) == nil {
			// add nodes that where not added via edge traversal already
			sub.AddNode(n)
		}
	}

	w.WalkAll(u, before, nil, during)

	return graphs
}

func buildUndirectedGraph(d *simple.DirectedGraph) *simple.UndirectedGraph {
	u := simple.NewUndirectedGraph()
	nodeIter := d.Nodes()
	for nodeIter.Next() {
		u.AddNode(nodeIter.Node())
	}

	edgeIter := d.Edges()
	for edgeIter.Next() {
		u.SetEdge(u.NewEdge(edgeIter.Edge().From(), edgeIter.Edge().To()))
	}
	return u
}

type NodeOption func(n *ConfigNode)

// New creates a new ConfigGraphPerEnvironment based on the given projects and environments.
func New(projects []project.Project, environments []string, nodeOptions ...NodeOption) ConfigGraphPerEnvironment {
	graphs := make(ConfigGraphPerEnvironment)
	for _, environment := range environments {
		cfgGraph := buildDependencyGraph(projects, environment, nodeOptions)
		graphs[environment] = cfgGraph
	}
	return graphs
}

func buildDependencyGraph(projects []project.Project, environment string, nodeOptions []NodeOption) *simple.DirectedGraph {
	log.Debug("Creating dependency graph for %s", environment)
	g := simple.NewDirectedGraph()
	coordinateToNodeIDs := make(coordinateToNodeIDMap)
	configReferences := make(referencesLookup)

	var configs []config.Config

	for _, p := range projects {
		for _, cfgs := range p.Configs[environment] {
			configs = append(configs, cfgs...)
		}
	}

	for i, c := range configs {
		if c.Skip && featureflags.Temporary[featureflags.IgnoreSkippedConfigs].Enabled() {
			log.Debug("Excluding config %s from dependency graph", c.Coordinate)
			continue
		}
		c := c
		n := ConfigNode{
			NodeID: int64(i),
			Config: &c,
		}
		for _, o := range nodeOptions {
			o(&n)
		}
		g.AddNode(n)

		coordinateToNodeIDs[c.Coordinate] = n.ID()

		configReferences[c.Coordinate] = map[coordinate.Coordinate]struct{}{}

		for _, ref := range c.References() {
			configReferences[c.Coordinate][ref] = struct{}{}
		}
	}
	log.Debug("Added %d config nodes to graph", g.Nodes().Len())
	log.Debug("Adding edges between dependent config nodes...")
	for c, refs := range configReferences {
		for other, _ := range refs {
			if c == other {
				continue // configs may have references between their own parameters, but self-edges must not be added to the dependency graph
			}
			cNode := coordinateToNodeIDs[c]
			if otherNode, ok := coordinateToNodeIDs[other]; ok {
				logDependency(c, other)
				g.SetEdge(g.NewEdge(g.Node(otherNode), g.Node(cNode)))
			} else {
				//TODO: to comply with the current 'continue-on-error' behaviour we can not recognize invalid references at this point but must return a dependency graph even if we know things will fail later on
				log.Warn("Configuration %q references unknown configuration %q", c, other)
			}

		}
	}

	return g
}

func logDependency(depending, dependedOn coordinate.Coordinate) {
	log.Debug("Configuration: %s has dependency on %s", depending, dependedOn)
}

// Roots returns all nodes that do not have incoming edges
func Roots(g graph.Directed) []graph.Node {
	var roots []graph.Node
	nodes := g.Nodes()

	for nodes.Next() {
		if g.To(nodes.Node().ID()).Len() == 0 {
			roots = append(roots, nodes.Node())
		}
	}

	return roots
}
