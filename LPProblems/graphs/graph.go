package graphs

import (
	"fmt"
	"sort"
)

// Graph is an undirected graph with
// N vertices,
// Adj as a boolean adjacency matrix.
type Graph struct {
	N   int
	Adj [][]bool
}

// NewGraph creates an empty graph with n vertices.
func NewGraph(n int) (*Graph, error) {
	if n <= 0 {
		return nil, fmt.Errorf("graph must have positive number of vertices, got %d", n)
	}
	adj := make([][]bool, n)
	for i := 0; i < n; i++ {
		adj[i] = make([]bool, n)
	}
	return &Graph{N: n, Adj: adj}, nil
}

// AddEdge adds an undirected edge (u, v).
func (g *Graph) AddEdge(u, v int) error {
	if err := g.validateVertex(u); err != nil {
		return err
	}
	if err := g.validateVertex(v); err != nil {
		return err
	}
	if u == v {
		return fmt.Errorf("self loops are not allowed: %d", u)
	}
	g.Adj[u][v] = true
	g.Adj[v][u] = true
	return nil
}

// HasEdge returns true if u and v are adjacent.
func (g *Graph) HasEdge(u, v int) bool {
	if u < 0 || v < 0 || u >= g.N || v >= g.N || u == v {
		return false
	}
	return g.Adj[u][v]
}

// EdgeCount returns the number of all edges.
func (g *Graph) EdgeCount() int {
	edges := 0
	for i := 0; i < g.N; i++ {
		for j := i + 1; j < g.N; j++ {
			if g.Adj[i][j] {
				edges++
			}
		}
	}
	return edges
}

func (g *Graph) validateVertex(v int) error {
	if v < 0 || v >= g.N {
		return fmt.Errorf("vertex %d is out of range [0, %d)", v, g.N)
	}
	return nil
}

// Validate checks graph for number of vertices and symmetry.
func (g *Graph) Validate() error {
	if g.N <= 0 {
		return fmt.Errorf("graph must have positive number of vertices")
	}
	if len(g.Adj) != g.N {
		return fmt.Errorf("adjacency row count %d does not match N=%d", len(g.Adj), g.N)
	}
	for i := 0; i < g.N; i++ {
		if len(g.Adj[i]) != g.N {
			return fmt.Errorf("adjacency row %d has length %d, want %d", i, len(g.Adj[i]), g.N)
		}
		if g.Adj[i][i] {
			return fmt.Errorf("self loop at vertex %d", i)
		}
		for j := i + 1; j < g.N; j++ {
			if g.Adj[i][j] != g.Adj[j][i] {
				return fmt.Errorf("adjacency matrix is not symmetric for (%d, %d)", i, j)
			}
		}
	}
	return nil
}

// SortVerticesList returns sorted vertices list.
func SortVerticesList(vertices []int) []int {
	if len(vertices) == 0 {
		return nil
	}
	cp := append([]int(nil), vertices...)
	sort.Ints(cp)
	return cp
}

// BuildIndependentSets build independent sets for all non-edge pairs.
// For restrictions in LP Model.
func (g *Graph) BuildIndependentSets() [][]int {
	type pair struct {
		a int
		b int
	}
	covered := make(map[pair]struct{})
	sets := make([][]int, 0)

	for i := 0; i < g.N; i++ {
		for j := i + 1; j < g.N; j++ {
			if g.HasEdge(i, j) {
				continue
			}
			p := pair{a: i, b: j}
			if _, ok := covered[p]; ok {
				continue
			}
			current := []int{i, j}
			for candidate := 0; candidate < g.N; candidate++ {
				if candidate == i || candidate == j {
					continue
				}
				ok := true
				for _, existing := range current {
					if g.HasEdge(candidate, existing) || candidate == existing {
						ok = false
						break
					}
				}
				if ok {
					current = append(current, candidate)
				}
			}
			for a := 0; a < len(current); a++ {
				for b := a + 1; b < len(current); b++ {
					x := current[a]
					y := current[b]
					if x > y {
						x, y = y, x
					}
					covered[pair{a: x, b: y}] = struct{}{}
				}
			}
			sets = append(sets, current)
		}
	}
	return sets
}
