package ggmat

import (
	"math"
	"testing"
)

func TestAddMulEW(t *testing.T) {
	t.Parallel()
	a := Matrix{{1, 2}, {3, 4}}
	b := Matrix{{10, 20}, {30, 40}}
	sum, err := Add(a, b, 2)
	if err != nil {
		t.Fatal(err)
	}
	if sum[0][0] != 11 || sum[1][1] != 44 {
		t.Fatalf("sum %#v", sum)
	}
	prod, err := MulEW(a, b, 2)
	if err != nil {
		t.Fatal(err)
	}
	if prod[0][0] != 10 || prod[1][1] != 160 {
		t.Fatalf("prod %#v", prod)
	}
}

func TestMatMul(t *testing.T) {
	t.Parallel()
	a := Matrix{{1, 2, 3}, {4, 5, 6}}   // 2x3
	b := Matrix{{1, 0}, {0, 1}, {1, 1}} // 3x2
	c, err := MatMul(a, b, 2)
	if err != nil {
		t.Fatal(err)
	}
	// row0: 1+0+3=4, 0+2+3=5; row1: 4+0+6=10, 0+5+6=11
	if c.Rows() != 2 || c.Cols() != 2 {
		t.Fatalf("shape %dx%d", c.Rows(), c.Cols())
	}
	if c[0][0] != 4 || c[0][1] != 5 || c[1][0] != 10 || c[1][1] != 11 {
		t.Fatalf("c %#v", c)
	}
}

func TestMatMulMismatch(t *testing.T) {
	t.Parallel()
	a := Matrix{{1, 2}}
	b := Matrix{{1, 2}}
	_, err := MatMul(a, b, 1)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestIdentity(t *testing.T) {
	t.Parallel()
	a := Matrix{{1, 2}}
	out, err := Identity(a)
	if err != nil {
		t.Fatal(err)
	}
	out[0][0] = 99
	if a[0][0] == 99 {
		t.Fatal("clone was shallow")
	}
}

func TestDotRowCol(t *testing.T) {
	t.Parallel()
	a := Matrix{{1, 2, 3}}
	b := Matrix{{2}, {3}, {4}}
	s, err := Dot(a, b, 3)
	if err != nil {
		t.Fatal(err)
	}
	// 2+6+12=20
	if math.Abs(s-20) > 1e-9 {
		t.Fatalf("dot %g", s)
	}
}

func TestDotMismatch(t *testing.T) {
	t.Parallel()
	a := Matrix{{1, 2}}
	b := Matrix{{1, 2, 3}}
	_, err := Dot(a, b, 1)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDotNotVector(t *testing.T) {
	t.Parallel()
	a := Matrix{{1, 2}, {3, 4}}
	b := Matrix{{1, 0}, {0, 1}}
	_, err := Dot(a, b, 1)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAddShapeErr(t *testing.T) {
	t.Parallel()
	_, err := Add(Matrix{{1}}, Matrix{{1, 2}}, 1)
	if err == nil {
		t.Fatal("expected error")
	}
}
