package mcp

import (
	"LPProblems/graphs"
	"fmt"
)

func ValidateClique(graph *graphs.Graph, vertices []int) error {
	if graph == nil {
		return fmt.Errorf("graph is nil")
	}
	seen := make(map[int]struct{}, len(vertices))
	for _, v := range vertices {
		if v < 0 || v >= graph.N {
			return fmt.Errorf("vertex %d is out of range [0,%d)", v, graph.N)
		}
		if _, exists := seen[v]; exists {
			return fmt.Errorf("vertex %d appears more than once", v)
		}
		seen[v] = struct{}{}
	}
	for i := 0; i < len(vertices); i++ {
		for j := i + 1; j < len(vertices); j++ {
			u := vertices[i]
			v := vertices[j]
			if !graph.HasEdge(u, v) {
				return fmt.Errorf("vertices %d and %d are not adjacent", u, v)
			}
		}
	}
	return nil
}
