package solver

import (
	"fmt"

	"github.com/bartolsthoorn/gohighs/highs"
)

// CliqueLP stores base LP Model for BnB.
type CliqueLP struct {
	base highs.Model
}

// NewCliqueLP builds LP Model:
// max sum x_i,
// 0<=x_i<=1,
// for each independent set: sum of elements <= 1.
func NewCliqueLP(numVertices int, independentSets [][]int) CliqueLP {
	colCosts := make([]float64, numVertices)
	colLower := make([]float64, numVertices)
	colUpper := make([]float64, numVertices)
	for i := 0; i < numVertices; i++ {
		colCosts[i] = 1
		colLower[i] = 0
		colUpper[i] = 1
	}

	model := highs.Model{
		Maximize: true,
		ColCosts: colCosts,
		ColLower: colLower,
		ColUpper: colUpper,
	}
	for _, set := range independentSets {
		vals := make([]float64, len(set))
		for i := range vals {
			vals[i] = 1
		}
		model.AddSparseRow(-highs.Inf(), set, vals, 1)
	}
	return CliqueLP{base: model}
}

// Solve applies current node's bounds to LP and solve.
// Returns a flag of the existence of a solution
// and colValues.
func (lp *CliqueLP) Solve(colLower, colUpper []float64, threads int) ([]float64, bool, error) {
	model := lp.base
	model.ColLower = colLower
	model.ColUpper = colUpper

	solution, err := model.Solve(
		highs.WithOutput(false),
		highs.WithThreads(threads),
		highs.WithStringOption("solver", "ipm"), // faster
	)
	if err != nil {
		return nil, false, fmt.Errorf("solve LP: %w", err)
	}
	if !solution.HasSolution() {
		return nil, false, nil
	}
	return solution.ColValues, true, nil
}
