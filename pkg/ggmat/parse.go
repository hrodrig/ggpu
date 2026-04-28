package ggmat

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// ReadMatrixPath loads a CSV matrix from path. Use "-" for stdin.
func ReadMatrixPath(path string, sep rune) (Matrix, error) {
	if path == "" {
		return nil, fmt.Errorf("missing matrix path")
	}
	if path == "-" {
		return ParseCSV(os.Stdin, sep)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ParseCSV(f, sep)
}

// ParseCSV reads row-major floats separated by sep on each line.
// Empty lines are skipped.
func ParseCSV(r io.Reader, sep rune) (Matrix, error) {
	sepStr := string(sep)
	sc := bufio.NewScanner(r)
	var rows [][]float64
	lineNum := 0
	for sc.Scan() {
		lineNum++
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		parts := strings.Split(line, sepStr)
		row := make([]float64, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			v, err := strconv.ParseFloat(p, 64)
			if err != nil {
				return nil, fmt.Errorf("line %d: parse %q: %w", lineNum, p, err)
			}
			row = append(row, v)
		}
		if len(row) == 0 {
			continue
		}
		rows = append(rows, row)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("no data rows")
	}
	m := Matrix(rows)
	if err := ValidateDims(m); err != nil {
		return nil, err
	}
	return m, nil
}

// WriteCSV writes m in row-major order using sep between columns.
func WriteCSV(w io.Writer, m Matrix, sep rune) error {
	if err := ValidateDims(m); err != nil {
		return err
	}
	sepStr := string(sep)
	for ri, row := range m {
		for ci, v := range row {
			if ci > 0 {
				if _, err := io.WriteString(w, sepStr); err != nil {
					return err
				}
			}
			if _, err := io.WriteString(w, strconv.FormatFloat(v, 'g', -1, 64)); err != nil {
				return err
			}
		}
		if ri < len(m)-1 {
			if _, err := io.WriteString(w, "\n"); err != nil {
				return err
			}
		}
	}
	return nil
}
