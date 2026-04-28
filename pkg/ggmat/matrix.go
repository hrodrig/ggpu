package ggmat

import "fmt"

// Matrix is row-major: each row is one slice; A[i][j] is row i, column j.
type Matrix [][]float64

// Rows returns the number of rows.
func (m Matrix) Rows() int {
	if len(m) == 0 {
		return 0
	}
	return len(m)
}

// Cols returns the number of columns (width of row 0).
func (m Matrix) Cols() int {
	if len(m) == 0 {
		return 0
	}
	return len(m[0])
}

// Clone returns a deep copy.
func (m Matrix) Clone() Matrix {
	if m == nil {
		return nil
	}
	out := make(Matrix, len(m))
	for i := range m {
		out[i] = make([]float64, len(m[i]))
		copy(out[i], m[i])
	}
	return out
}

// SameShape reports whether a and b have identical dimensions.
func SameShape(a, b Matrix) bool {
	return a.Rows() == b.Rows() && a.Cols() == b.Cols()
}

// ValidateDims checks non-empty rectangular data.
func ValidateDims(m Matrix) error {
	if len(m) == 0 {
		return fmt.Errorf("empty matrix")
	}
	c0 := len(m[0])
	for i, row := range m {
		if len(row) != c0 {
			return fmt.Errorf("row %d: length %d, expected %d", i, len(row), c0)
		}
	}
	return nil
}
