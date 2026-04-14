package tests

import (
	"LPProblems/solver"
	"bytes"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	matrix100 = "files/A_100.csv"
	comma     = ';'
)

var solveMatrices = []struct {
	name string
	path string
}{
	{"A_100", "files/A_100.csv"},
	{"A_1000", "files/A_1000.csv"},
	{"A_1mln_10", "files/A_1mln_10.csv"},
	{"A_1mln_100", "files/A_1mln_100.csv"}, // 140s
	{"A_10000", "files/A_10000.csv"},       // 2000s
}

// solutionOutputPath maps files/A_100.csv -> output/A_100.txt (fixed names, overwritten each run).
func solutionOutputPath(inputRelPath string) string {
	base := strings.TrimSuffix(filepath.Base(inputRelPath), filepath.Ext(inputRelPath))
	return filepath.Join("output", base+".txt")
}

func writeSolutionOutput(inputRelPath string, sol *solver.GameSolution) error {
	if err := os.MkdirAll("output", 0o755); err != nil {
		return err
	}
	path := solutionOutputPath(inputRelPath)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	sol.Fprint(f)
	return nil
}

func TestMatrix_Initialize(t *testing.T) {
	var m solver.Matrix
	err := m.Initialize(matrix100, comma)
	if err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	if len(m.Data) == 0 {
		t.Fatal("expected non-empty matrix")
	}

	rows, cols := len(m.Data), len(m.Data[0])
	t.Logf("matrix shape: %d rows, %d cols", rows, cols)

	if rows != 100 || cols != 100 {
		t.Errorf("expected 100x100, got %dx%d", rows, cols)
	}

	if m.Data[0][0] != -178 {
		t.Errorf("matrix[0][0] = %v, want -178", m.Data[0][0])
	}

	if m.Data[1][0] != 36 {
		t.Errorf("matrix[1][0] = %v, want 36", m.Data[1][0])
	}
}

func TestLPProblem_Solve(t *testing.T) {
	for _, tc := range solveMatrices {
		t.Run(tc.name, func(t *testing.T) {
			var m solver.Matrix
			err := m.Initialize(tc.path, comma)
			if err != nil {
				t.Fatalf("Initialize: %v", err)
			}

			lp, err := m.ToLPProblem()
			if err != nil {
				t.Fatalf("ToLPProblem: %v", err)
			}

			sol, err := lp.Solve()
			if err != nil {
				t.Fatalf("Solve: %v", err)
			}

			if err := writeSolutionOutput(tc.path, &sol); err != nil {
				t.Fatalf("write solution output: %v", err)
			}

			sumFirst := sumFloat64(sol.PFirst)
			sumSecond := sumFloat64(sol.PSecond)
			if math.Abs(sumFirst-1) > 1e-6 {
				t.Errorf("sum of first player probabilities = %g, want 1", sumFirst)
			}
			if math.Abs(sumSecond-1) > 1e-6 {
				t.Errorf("sum of second player probabilities = %g, want 1", sumSecond)
			}

			t.Logf("game value: %g", sol.GameValue)
			t.Logf("sum first: %g, sum second: %g", sumFirst, sumSecond)
		})
	}
}

func TestLPProblem_Solve_Print(t *testing.T) {
	for _, tc := range solveMatrices {
		t.Run(tc.name, func(t *testing.T) {
			var m solver.Matrix
			err := m.Initialize(tc.path, comma)
			if err != nil {
				t.Fatalf("Initialize: %v", err)
			}

			lp, err := m.ToLPProblem()
			if err != nil {
				t.Fatalf("ToLPProblem: %v", err)
			}

			sol, err := lp.Solve()
			if err != nil {
				t.Fatalf("Solve: %v", err)
			}

			if err := writeSolutionOutput(tc.path, &sol); err != nil {
				t.Fatalf("write solution output: %v", err)
			}

			var buf bytes.Buffer
			sol.Fprint(&buf)
			out := buf.String()

			want := []string{
				"Сумма вероятностей первого игрока =",
				"Сумма вероятностей второго игрока =",
				"Цена игры:",
				"Стратегия первого игрока:",
				"Стратегия второго игрока:",
			}
			for _, s := range want {
				if !strings.Contains(out, s) {
					t.Errorf("Fprint output missing %q;\n got:\n%s", s, out)
				}
			}

			t.Logf("Print output:\n%s", out)
		})
	}
}

func sumFloat64(a []float64) float64 {
	var s float64
	for _, v := range a {
		s += v
	}
	return s
}
