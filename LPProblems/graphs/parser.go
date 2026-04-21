package graphs

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// ParseDIMACSFile reads a DIMACS-like graph file with lines c/p/e.
func ParseDIMACSFile(path string) (*Graph, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer func() {
		_ = file.Close()
	}()
	return ParseDIMACS(file)
}

// ParseDIMACS parses DIMACS-like graph data from a scanner-compatible stream.
func ParseDIMACS(reader io.Reader) (*Graph, error) {
	scanner := bufio.NewScanner(reader)
	lineN := 0

	var graph *Graph
	var declaredEdges int
	actualEdges := 0

	for scanner.Scan() {
		lineN++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		switch fields[0] {
		case "c":
			continue
		case "p":
			if graph != nil {
				return nil, fmt.Errorf("line %d: multiple problem lines", lineN)
			}
			if len(fields) != 4 {
				return nil, fmt.Errorf("line %d: invalid problem line", lineN)
			}
			if fields[1] != "edge" && fields[1] != "col" {
				return nil, fmt.Errorf("line %d: unsupported problem type %q", lineN, fields[1])
			}
			n, err := strconv.Atoi(fields[2])
			if err != nil {
				return nil, fmt.Errorf("line %d: parse vertex count: %w", lineN, err)
			}
			declaredEdges, err = strconv.Atoi(fields[3])
			if err != nil {
				return nil, fmt.Errorf("line %d: parse edge count: %w", lineN, err)
			}
			graph, err = NewGraph(n)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineN, err)
			}
		case "e":
			if graph == nil {
				return nil, fmt.Errorf("line %d: edge before problem line", lineN)
			}
			if len(fields) != 3 {
				return nil, fmt.Errorf("line %d: invalid edge line", lineN)
			}
			u, err := strconv.Atoi(fields[1])
			if err != nil {
				return nil, fmt.Errorf("line %d: parse edge endpoint: %w", lineN, err)
			}
			v, err := strconv.Atoi(fields[2])
			if err != nil {
				return nil, fmt.Errorf("line %d: parse edge endpoint: %w", lineN, err)
			}
			u--
			v--
			if u == v {
				return nil, fmt.Errorf("line %d: self loops are not allowed", lineN)
			}
			if u < 0 || v < 0 || u >= graph.N || v >= graph.N {
				return nil, fmt.Errorf("line %d: edge endpoint out of range", lineN)
			}
			if !graph.Adj[u][v] {
				actualEdges++
			}
			if err := graph.AddEdge(u, v); err != nil {
				return nil, fmt.Errorf("line %d: add edge: %w", lineN, err)
			}
		default:
			return nil, fmt.Errorf("line %d: unsupported directive %q", lineN, fields[0])
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan error: %w", err)
	}
	if graph == nil {
		return nil, fmt.Errorf("missing problem line")
	}
	if err := graph.Validate(); err != nil {
		return nil, err
	}
	if declaredEdges != actualEdges {
		return nil, fmt.Errorf("declared %d edges but parsed %d unique edges", declaredEdges, actualEdges)
	}
	return graph, nil
}
