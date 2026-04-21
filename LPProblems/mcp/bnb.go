package mcp

import (
	"LPProblems/graphs"
	"fmt"
	"math"
	"runtime"
	"sync"
	"sync/atomic"

	"LPProblems/solver"
)

const (
	defaultE             = 1e-7
	defaultParallelDepth = 2
)

type nodeState int8

const (
	nodeFree nodeState = iota
	nodeForceZero
	nodeForceOne
)

// SolveResult stores best found clique and stats of branches.
type SolveResult struct {
	Clique []int
	Size   int
}

type bnbSolver struct {
	graph         *graphs.Graph
	lp            solver.CliqueLP
	e             float64
	parallelDepth int
	parallelSem   chan struct{}

	currentResMu   sync.Mutex
	currentRes     []int
	currentResSize atomic.Int64
}

// SolveMaximumCliqueBnB solves maximum clique with branch and bound.
func SolveMaximumCliqueBnB(g *graphs.Graph) (SolveResult, error) {
	// Validation
	if g == nil {
		return SolveResult{}, fmt.Errorf("graph is nil")
	}
	if err := g.Validate(); err != nil {
		return SolveResult{}, err
	}
	maxParallel := runtime.GOMAXPROCS(0)

	// Branches init
	bnb := &bnbSolver{
		graph:         g,
		e:             defaultE,
		parallelDepth: defaultParallelDepth,
		parallelSem:   make(chan struct{}, maxParallel),
	}

	// LP model with restrictions for all branches (independent sets).
	bnb.lp = solver.NewCliqueLP(g.N, g.BuildIndependentSets())

	// Start bounds: for each x_i: 0 <= x_i <= 1.
	fixed := make([]nodeState, g.N)
	colLower := make([]float64, g.N)
	colUpper := make([]float64, g.N)
	for i := range colUpper {
		colUpper[i] = 1
	}
	if err := bnb.branch(fixed, colLower, colUpper, 0); err != nil {
		return SolveResult{}, err
	}

	// Get currentRes
	bnb.currentResMu.Lock()
	currentRes := append([]int(nil), bnb.currentRes...)
	bnb.currentResMu.Unlock()
	currentResSize := int(bnb.currentResSize.Load())

	return SolveResult{
		Clique: graphs.SortVerticesList(currentRes),
		Size:   currentResSize,
	}, nil
}

// branch processes one node of the tree:
// does pruning on estimates,
// updates currentRes for int solution
// or branches on a float solution.
func (s *bnbSolver) branch(fixed []nodeState, colLower, colUpper []float64, depth int) error {
	// prune by coloring
	shouldPrune := s.pruneByColoringBound(fixed)
	if shouldPrune {
		return nil
	}

	// prune by lp
	colValues, shouldPrune, err := s.pruneByLPBound(colLower, colUpper)
	if err != nil {
		return err
	}
	if shouldPrune {
		return nil
	}

	if isIntegralSolution(colValues, s.e) {
		currentRes := make([]int, 0, len(colValues))
		for i, value := range colValues {
			if value >= 1-s.e {
				currentRes = append(currentRes, i)
			}
		}

		if tErr := ValidateClique(s.graph, currentRes); tErr != nil {
			return fmt.Errorf("invalid integral solution: %w", tErr)
		}

		// try to add solution
		s.updateCurrentRes(currentRes)
		return nil
	}
	// else:
	return s.branchOnFractional(colValues, fixed, colLower, colUpper, depth)
}

// pruneByColoringBound perform only greedy coloring pruning
// by already fixed vertices.
func (s *bnbSolver) pruneByColoringBound(fixed []nodeState) bool {
	coloringUpperBound, infeasible := s.greedyColoringUpperBound(fixed)
	if infeasible {
		return true
	}

	if coloringUpperBound <= int(s.currentResSize.Load()) {
		return true
	}
	return false
}

// greedyColoringUpperBound calculates upper bound for node:
// forced ones + number of color classes
func (s *bnbSolver) greedyColoringUpperBound(fixed []nodeState) (int, bool) {
	// check if all fixed vertices are adjacent
	forcedOnes := make([]int, 0, s.graph.N)
	for v, state := range fixed {
		if state != nodeForceOne {
			continue
		}
		for _, u := range forcedOnes {
			if !s.graph.Adj[v][u] {
				return 0, true
			}
		}
		forcedOnes = append(forcedOnes, v)
	}

	// free vertices that can get in a current clique (adjacent to current clique)
	candidates := make([]int, 0, s.graph.N-len(forcedOnes))
fixedLoop:
	for v, state := range fixed {
		if state != nodeFree {
			continue
		}
		for _, u := range forcedOnes {
			if !s.graph.Adj[v][u] {
				continue fixedLoop
			}
		}
		candidates = append(candidates, v)
	}

	// independent sets in candidates
	colorClasses := make([][]int, 0, len(candidates))
	for _, v := range candidates {
		placed := false
		for i := range colorClasses {
			compatible := true
			for _, u := range colorClasses[i] {
				if s.graph.Adj[v][u] {
					compatible = false
					break
				}
			}

			if compatible {
				colorClasses[i] = append(colorClasses[i], v)
				placed = true
				break
			}
		}

		if !placed {
			colorClasses = append(colorClasses, []int{v})
		}
	}

	return len(forcedOnes) + len(colorClasses), false
}

// pruneByLPBound solve LP for current bounds and prune if
// there is no solution
// or upper bound is worse than currentRes.
func (s *bnbSolver) pruneByLPBound(colLower, colUpper []float64) ([]float64, bool, error) {
	colValues, hasSolution, err := s.lp.Solve(colLower, colUpper, 1)
	if err != nil {
		return nil, false, err
	}
	if !hasSolution {
		return nil, true, nil
	}

	upperBound := 0.0
	for _, value := range colValues {
		upperBound += value
	}
	if upperBound <= float64(s.currentResSize.Load())+s.e {
		return nil, true, nil
	}
	return colValues, false, nil
}

// isIntegralSolution checks if all values are close to 0 or 1.
func isIntegralSolution(values []float64, e float64) bool {
	for _, value := range values {
		if value > e && value < 1-e {
			return false
		}
	}
	return true
}

// branchOnFractional choose vertice v close to 0.5 (uncertain) and creates to branches:
// x_v=0 (not include) и x_v=1 (include).
func (s *bnbSolver) branchOnFractional(colValues []float64, fixed []nodeState, colLower, colUpper []float64, depth int) error {
	branchV := pickBranchVariable(colValues, s.e)
	if branchV < 0 {
		return fmt.Errorf("fractional solution does not contain branching candidate")
	}

	if depth < s.parallelDepth { // do parallel
		done, err := s.runParallelBranches(branchV, fixed, colLower, colUpper, depth)
		if err != nil {
			return err
		}
		if done {
			return nil
		}
	}

	prevLower := colLower[branchV]
	prevUpper := colUpper[branchV]

	fixed[branchV] = nodeForceZero
	colLower[branchV] = 0
	colUpper[branchV] = 0
	if err := s.branch(fixed, colLower, colUpper, depth+1); err != nil {
		return err
	}

	fixed[branchV] = nodeForceOne
	colLower[branchV] = 1
	colUpper[branchV] = 1
	if err := s.branch(fixed, colLower, colUpper, depth+1); err != nil {
		return err
	}

	fixed[branchV] = nodeFree
	colLower[branchV] = prevLower
	colUpper[branchV] = prevUpper
	return nil
}

// updateCurrentRes updates best result and prints it to console.
func (s *bnbSolver) updateCurrentRes(candidate []int) {
	size := int64(len(candidate))
	if size <= s.currentResSize.Load() {
		return
	}
	s.currentResMu.Lock()
	defer s.currentResMu.Unlock()
	if size > s.currentResSize.Load() {
		s.currentRes = append([]int(nil), candidate...)
		s.currentResSize.Store(size)
		vertices := graphs.SortVerticesList(candidate)
		verticesOneBased := make([]int, len(vertices))
		for i, v := range vertices {
			verticesOneBased[i] = v + 1
		}
		fmt.Printf("[TL] current best clique size=%d vertices=%v\n", size, verticesOneBased)
	}
}

// pickBranchVariable choose float value that is closest to 0.5 (undefined).
func pickBranchVariable(values []float64, e float64) int {
	bestIdx := -1
	bestDiff := math.Inf(1)
	for i, value := range values {
		if value <= e || value >= 1-e { // 0 or 1
			continue
		}
		diff := math.Abs(0.5 - value)
		if diff < bestDiff {
			bestDiff = diff
			bestIdx = i
		}
	}
	return bestIdx
}
