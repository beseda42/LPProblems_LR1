package tests

import (
	"LPProblems/graphs"
	"LPProblems/mcp"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMCP_ParseDIMACSFile(t *testing.T) {
	cases := []struct {
		name      string
		file      string
		wantN     int
		wantEdges int
	}{
		{name: "johnson", file: "johnson8-2-4.clq", wantN: 28, wantEdges: 210},
		{name: "C125", file: "C125.9.clq", wantN: 125, wantEdges: 6963},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join("files", "mcp", tc.file)
			graph, err := graphs.ParseDIMACSFile(path)
			if err != nil {
				t.Fatalf("ParseDIMACSFile(%s): %v", path, err)
			}
			if graph.N != tc.wantN {
				t.Fatalf("graph.N = %d, want %d", graph.N, tc.wantN)
			}
			if got := graph.EdgeCount(); got != tc.wantEdges {
				t.Fatalf("graph.EdgeCount() = %d, want %d", got, tc.wantEdges)
			}
		})
	}
}

func TestMCP_SolveMaximumCliqueBnB_DIMACS_Easy(t *testing.T) {
	cases := []struct {
		file string
		want int
	}{
		{file: "C125.9.clq", want: 34},
		{file: "johnson8-2-4.clq", want: 4},
		{file: "johnson16-2-4.clq", want: 8},
		{file: "MANN_a9.clq", want: 16},
		{file: "keller4.clq", want: 11},
		{file: "hamming8-4.clq", want: 16},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.file, func(t *testing.T) {
			runDIMACSInstanceWithTimeLimit(t, tc.file, tc.want, 11*time.Minute)
		})
	}
}

func TestMCP_SolveMaximumCliqueBnB_DIMACS_Moderate(t *testing.T) {
	cases := []struct {
		file string
		want int
	}{
		{file: "brock200_1.clq", want: 21},
		{file: "brock200_2.clq", want: 12},
		{file: "brock200_3.clq", want: 15},
		{file: "brock200_4.clq", want: 17},
		{file: "gen200_p0.9_44.clq", want: 44},
		{file: "gen200_p0.9_55.clq", want: 55},
		{file: "MANN_a27.clq", want: 126},
		{file: "p_hat1000-1.clq", want: 10},
		{file: "p_hat1000-2.clq", want: 46},
		{file: "p_hat300-3.clq", want: 36},
		{file: "p_hat500-3.clq", want: 50},
		{file: "sanr200_0.9.clq", want: 42},
		{file: "sanr400_0.7.clq", want: 21},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.file, func(t *testing.T) {
			runDIMACSInstanceWithTimeLimit(t, tc.file, tc.want, 30*time.Minute)
		})
	}
}

func runDIMACSInstanceWithTimeLimit(t *testing.T, file string, wantCliqueSize int, limit time.Duration) {
	t.Helper()
	path := filepath.Join("files", "mcp", file)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			t.Skipf("DIMACS file is missing: %s", path)
		}
		t.Fatalf("cannot access %s: %v", path, err)
	}

	done := make(chan struct{})
	var (
		gotRes mcp.SolveResult
		gotErr error
	)
	start := time.Now()
	go func() {
		defer close(done)
		graph, err := graphs.ParseDIMACSFile(path)
		if err != nil {
			gotErr = err
			return
		}
		gotRes, gotErr = mcp.SolveMaximumCliqueBnB(graph)
		if gotErr != nil {
			return
		}
		if err := mcp.ValidateClique(graph, gotRes.Clique); err != nil {
			gotErr = err
			return
		}
	}()

	select {
	case <-done:
	case <-time.After(limit):
		t.Fatalf("time limit exceeded for %s (> %s)", file, limit)
	}

	if gotErr != nil {
		t.Fatalf("failed for %s: %v", file, gotErr)
	}
	if gotRes.Size != wantCliqueSize {
		t.Fatalf("%s: clique size = %d, want %d", file, gotRes.Size, wantCliqueSize)
	}
	elapsed := time.Since(start)
	if elapsed > limit {
		t.Fatalf("%s: elapsed %s exceeds time limit %s", file, elapsed, limit)
	}
	if err := writeMCPOutput(file, elapsed, gotRes); err != nil {
		t.Fatalf("write output for %s: %v", file, err)
	}
}

func writeMCPOutput(inputFile string, elapsed time.Duration, result mcp.SolveResult) error {
	if err := os.MkdirAll(filepath.Join("output", "mcp"), 0o755); err != nil {
		return err
	}
	base := strings.TrimSuffix(filepath.Base(inputFile), filepath.Ext(inputFile))
	path := filepath.Join("output", "mcp", base+".txt")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	verticesOneBased := make([]int, len(result.Clique))
	for i, v := range result.Clique {
		verticesOneBased[i] = v + 1
	}

	_, err = fmt.Fprintf(
		f,
		"Время выполнения: %s\nРазмер максимальной клики: %d\nВершины клики: %v\n",
		elapsed.Truncate(time.Millisecond),
		result.Size,
		verticesOneBased,
	)
	return err
}
