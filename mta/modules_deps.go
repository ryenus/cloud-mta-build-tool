package mta

import (
	"github.com/deckarep/golang-set"
	"github.com/pkg/errors"
)

type graphNode struct {
	module string
	deps   mapset.Set
	index  int
}

// New graphs node
func newGn(module *string, deps mapset.Set, index int) *graphNode {
	return &graphNode{module: *module, deps: deps, index: index}
}

// graphs - graph map
type graphs map[string]*graphNode

// getModulesOrder - Provides Modules ordered according to build-parameters' dependencies
func (mta *MTA) getModulesOrder() ([]string, error) {
	var graph = make(graphs)
	for index, module := range mta.Modules {
		deps := mapset.NewSet()
		if module.BuildParams.Requires != nil {
			for _, req := range module.BuildParams.Requires {
				deps.Add(req.Name)
			}
		}
		graph[module.Name] = newGn(&module.Name, deps, index)
	}
	return resolveGraph(&graph, mta)
}

// Resolves the dependency graphs
// For resolving cyclic dependencies Kahn’s algorithm of topological sorting is used.
// https://en.wikipedia.org/wiki/Topological_sorting
func resolveGraph(graph *graphs, mta *MTA) ([]string, error) {
	overleft := *graph

	// Iteratively find and remove nodes from the graphs which have no dependencies.
	// If at some point there are still nodes in the graphs and we cannot find
	// nodes without dependencies, that means we have a circular dependency
	var resolved []string
	for len(overleft) != 0 {
		// Get all nodes from the graphs which have no dependencies
		readyNodesSet := mapset.NewSet()
		readyModulesSet := mapset.NewSet()
		for _, node := range overleft {
			if node.deps.Cardinality() == 0 {
				readyNodesSet.Add(node)
				readyModulesSet.Add(node.module)
			}
		}

		// If there aren't any ready nodes, then we have a circular dependency
		if readyNodesSet.Cardinality() == 0 {
			module1, module2 := provideCyclicModules(&overleft)
			return nil, errors.Errorf("Circular dependency found. Check modules %v and %v", module1, module2)
		}

		// Remove the ready nodes and add them to the resolved graphs
		readyModulesIndexes := mapset.NewSet()
		for node := range readyNodesSet.Iter() {
			delete(overleft, node.(*graphNode).module)
			readyModulesIndexes.Add(node.(*graphNode).index)
		}

		for index, module := range mta.Modules {
			if readyModulesIndexes.Contains(index) {
				resolved = append(resolved, module.Name)
			}
		}

		// remove the ready nodes from the remaining node dependencies as well
		for _, node := range overleft {
			node.deps = node.deps.Difference(readyModulesSet)
		}
	}

	return resolved, nil
}

// provideCyclicModules - provide some of modules having cyclic dependencies
func provideCyclicModules(overleft *graphs) (string, string) {
	module1 := ""
	module2 := ""
	index := 0
	for _, node := range *overleft {
		if index == 0 {
			module1 = node.module
			index = 1
		} else {
			module2 = node.module
			break
		}
	}
	return module1, module2
}
