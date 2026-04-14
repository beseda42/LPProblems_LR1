package solver

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"

	"github.com/bartolsthoorn/gohighs/highs"
)

type Matrix struct {
	Data [][]float64
}

// Initialize reads a .CSV file.
// Sets m Matrix Data - matrix[row][col].
func (m *Matrix) Initialize(filename string, comma rune) error {
	f, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("open file %s error: %w", filename, err)
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	r := csv.NewReader(f)
	r.Comma = comma

	records, err := r.ReadAll()
	if err != nil {
		return fmt.Errorf("read csv: %w", err)
	}

	if len(records) == 0 {
		return fmt.Errorf("empty csv file")
	}

	numCols := len(records[0])
	matrix := make([][]float64, len(records))

	for i, row := range records {
		if len(row) != numCols {
			return fmt.Errorf("row %d: expected %d columns, got %d", i, numCols, len(row))
		}
		matrix[i] = make([]float64, numCols)
		for j, cell := range row {
			elem, err := strconv.ParseFloat(cell, 64)
			if err != nil {
				return fmt.Errorf("row %d col %d: parse %q: %w", i, j, cell, err)
			}
			matrix[i][j] = float64(elem)
		}
	}

	m.Data = matrix

	return nil
}

// LPProblem holds the HiGHS model and the shift applied to the matrix.
type LPProblem struct {
	Model highs.Model
	Shift float64
}

// ToLPProblem builds the game-theory LP from the matrix: m variables (one per row), n constraints (one per column).
// If min(matrix) <= 0, the matrix is shifted by -min+1 so all entries will be positive.
func (m *Matrix) ToLPProblem() (LPProblem, error) {
	if len(m.Data) == 0 || len(m.Data[0]) == 0 {
		return LPProblem{}, fmt.Errorf("empty matrix")
	}
	rows, cols := len(m.Data), len(m.Data[0])

	// find min value
	minVal := m.Data[0][0]
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			if m.Data[i][j] < minVal {
				minVal = m.Data[i][j]
			}
		}
	}

	// set shift
	var shift float64 = 0
	if minVal <= 0 {
		shift = -minVal + 1
	}

	// m variables: lower 0, upper +inf, cost 1.0 each
	colCosts := make([]float64, rows)
	colLower := make([]float64, rows)
	colUpper := make([]float64, rows)
	for i := 0; i < rows; i++ {
		colCosts[i] = 1
		colLower[i] = 0
		colUpper[i] = highs.Inf()
	}

	model := highs.Model{
		Maximize: false, // minimize sum(x) for second player's LP
		ColCosts: colCosts,
		ColLower: colLower,
		ColUpper: colUpper,
	}

	// constraints: for each column j, 1 <= m[1][j]*x_1 + m[2][j]x_2 + ... + m[i][j]*x_i (<= +inf)
	coeffs := make([]float64, rows)
	for j := 0; j < cols; j++ {
		for i := 0; i < rows; i++ {
			coeffs[i] = m.Data[i][j] + shift
		}
		model.AddDenseRow(1, coeffs, highs.Inf())
	}

	return LPProblem{
		Model: model,
		Shift: shift,
	}, nil
}

// GameSolution holds the game value and mixed strategies for both players.
type GameSolution struct {
	GameValue float64   // price of the game
	PFirst    []float64 // first player's strategy
	PSecond   []float64 // second player's strategy
}

// Solve runs the LP and returns the game solution.
func (lp *LPProblem) Solve() (GameSolution, error) {
	solution, err := lp.Model.Solve(
		highs.WithOutput(false),
		// highs.WithStringOption("solver", "ipm"), // for A_10000
	)
	if err != nil {
		return GameSolution{}, err
	}
	if !solution.HasSolution() {
		return GameSolution{}, fmt.Errorf("solver did not find a solution: status %s", solution.Status)
	}

	x := solution.ColValues
	var V float64
	for _, elem := range x {
		V += elem
	}
	if V == 0 {
		return GameSolution{}, fmt.Errorf("sum of primal solution is zero")
	}

	gameValue := 1/V - lp.Shift

	pFirst := make([]float64, len(x))
	for i, elem := range x {
		fmt.Printf("pFirst[%d] = %f; V = %v\n", i, elem, V)
		pFirst[i] = math.Abs(elem / V)
	}

	duals := solution.RowDuals
	pSecond := make([]float64, len(duals))
	for i, elem := range duals {
		pSecond[i] = math.Abs(elem / V)
	}

	return GameSolution{
		GameValue: gameValue,
		PFirst:    pFirst,
		PSecond:   pSecond,
	}, nil
}

func sumFloat64(a []float64) float64 {
	var s float64
	for _, v := range a {
		s += v
	}
	return s
}

// Fprint writes the solution summary to io.Writer.
func (s *GameSolution) Fprint(w io.Writer) {
	fmt.Fprintf(w, "Сумма вероятностей первого игрока = %g\n", sumFloat64(s.PFirst))
	fmt.Fprintf(w, "Сумма вероятностей второго игрока = %g\n", sumFloat64(s.PSecond))
	fmt.Fprintf(w, "Цена игры: %g\n", s.GameValue)
	fmt.Fprintf(w, "Стратегия первого игрока: %v\n", s.PFirst)
	fmt.Fprintf(w, "Стратегия второго игрока: %v\n", s.PSecond)
}
