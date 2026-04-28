package ggmat

import (
	"fmt"
	"sync"
)

// Add returns C = A + B (element-wise). Shapes must match.
func Add(a, b Matrix, workers int) (Matrix, error) {
	if err := ValidateDims(a); err != nil {
		return nil, fmt.Errorf("add: A: %w", err)
	}
	if err := ValidateDims(b); err != nil {
		return nil, fmt.Errorf("add: B: %w", err)
	}
	if !SameShape(a, b) {
		return nil, fmt.Errorf("add: shape mismatch %dx%d vs %dx%d",
			a.Rows(), a.Cols(), b.Rows(), b.Cols())
	}
	r, c := a.Rows(), a.Cols()
	out := make(Matrix, r)
	for i := range out {
		out[i] = make([]float64, c)
	}
	n := r * c
	parFor(n, workers, func(i int) {
		ri, ci := i/c, i%c
		out[ri][ci] = a[ri][ci] + b[ri][ci]
	})
	return out, nil
}

// MulEW returns C = A * B element-wise. Shapes must match.
func MulEW(a, b Matrix, workers int) (Matrix, error) {
	if err := ValidateDims(a); err != nil {
		return nil, fmt.Errorf("mul-ew: A: %w", err)
	}
	if err := ValidateDims(b); err != nil {
		return nil, fmt.Errorf("mul-ew: B: %w", err)
	}
	if !SameShape(a, b) {
		return nil, fmt.Errorf("mul-ew: shape mismatch %dx%d vs %dx%d",
			a.Rows(), a.Cols(), b.Rows(), b.Cols())
	}
	r, c := a.Rows(), a.Cols()
	out := make(Matrix, r)
	for i := range out {
		out[i] = make([]float64, c)
	}
	n := r * c
	parFor(n, workers, func(i int) {
		ri, ci := i/c, i%c
		out[ri][ci] = a[ri][ci] * b[ri][ci]
	})
	return out, nil
}

// MatMul returns C = A B with A (M×K) and B (K×N) → C (M×N).
func MatMul(a, b Matrix, workers int) (Matrix, error) {
	if err := ValidateDims(a); err != nil {
		return nil, fmt.Errorf("matmul: A: %w", err)
	}
	if err := ValidateDims(b); err != nil {
		return nil, fmt.Errorf("matmul: B: %w", err)
	}
	m, k1 := a.Rows(), a.Cols()
	k2, n := b.Rows(), b.Cols()
	if k1 != k2 {
		return nil, fmt.Errorf("matmul: inner dimension mismatch A is %dx%d, B is %dx%d (need cols(A)==rows(B))",
			m, k1, k2, n)
	}
	k := k1
	out := make(Matrix, m)
	for i := range out {
		out[i] = make([]float64, n)
	}
	parFor(m, workers, func(i int) {
		for j := 0; j < n; j++ {
			var sum float64
			for kk := 0; kk < k; kk++ {
				sum += a[i][kk] * b[kk][j]
			}
			out[i][j] = sum
		}
	})
	return out, nil
}

// Identity returns a deep copy of A (ignores B). For CLI symmetry with other actions.
func Identity(a Matrix) (Matrix, error) {
	if err := ValidateDims(a); err != nil {
		return nil, fmt.Errorf("identity: %w", err)
	}
	return a.Clone(), nil
}

// Dot returns the scalar product of two vectors. Each operand must be 1×N or N×1.
func Dot(a, b Matrix, workers int) (float64, error) {
	if err := ValidateDims(a); err != nil {
		return 0, fmt.Errorf("dot: A: %w", err)
	}
	if err := ValidateDims(b); err != nil {
		return 0, fmt.Errorf("dot: B: %w", err)
	}
	va, err := flattenVec(a)
	if err != nil {
		return 0, fmt.Errorf("dot: %w", err)
	}
	vb, err := flattenVec(b)
	if err != nil {
		return 0, fmt.Errorf("dot: %w", err)
	}
	if len(va) != len(vb) {
		return 0, fmt.Errorf("dot: length mismatch %d vs %d", len(va), len(vb))
	}
	return dotParallel(va, vb, workers), nil
}

func flattenVec(m Matrix) ([]float64, error) {
	r, c := m.Rows(), m.Cols()
	switch {
	case r == 1:
		return append([]float64(nil), m[0]...), nil
	case c == 1:
		v := make([]float64, r)
		for i := range m {
			v[i] = m[i][0]
		}
		return v, nil
	default:
		return nil, fmt.Errorf("need 1×N or N×1 matrix, got %dx%d", r, c)
	}
}

func dotParallel(va, vb []float64, workers int) float64 {
	n := len(va)
	if workers < 1 {
		workers = 1
	}
	if workers > n {
		workers = n
	}
	if n == 0 {
		return 0
	}
	chunk := (n + workers - 1) / workers
	partials := make([]float64, workers)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		start := w * chunk
		if start >= n {
			break
		}
		end := start + chunk
		if end > n {
			end = n
		}
		wg.Add(1)
		go func(wid, s, e int) {
			defer wg.Done()
			var ssum float64
			for i := s; i < e; i++ {
				ssum += va[i] * vb[i]
			}
			partials[wid] = ssum
		}(w, start, end)
	}
	wg.Wait()
	var total float64
	for _, p := range partials {
		total += p
	}
	return total
}
