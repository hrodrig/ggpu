package ggmat

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseCSV(t *testing.T) {
	t.Parallel()
	m, err := ParseCSV(strings.NewReader("1,2\n3,4\n"), ',')
	if err != nil {
		t.Fatal(err)
	}
	if m.Rows() != 2 || m.Cols() != 2 || m[0][0] != 1 || m[1][1] != 4 {
		t.Fatalf("got %#v", m)
	}
}

func TestParseCSVSkipEmpty(t *testing.T) {
	t.Parallel()
	m, err := ParseCSV(strings.NewReader("\n1, 2\n\n"), ',')
	if err != nil {
		t.Fatal(err)
	}
	if m.Rows() != 1 || m[0][0] != 1 || m[0][1] != 2 {
		t.Fatalf("got %#v", m)
	}
}

func TestParseCSVRagged(t *testing.T) {
	t.Parallel()
	_, err := ParseCSV(strings.NewReader("1,2\n3\n"), ',')
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseCSVNoRows(t *testing.T) {
	t.Parallel()
	_, err := ParseCSV(strings.NewReader("\n\n"), ',')
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestReadMatrixPathMissing(t *testing.T) {
	t.Parallel()
	_, err := ReadMatrixPath("", ',')
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWriteCSV(t *testing.T) {
	t.Parallel()
	m := Matrix{{1, 2}, {3, 4}}
	var b strings.Builder
	if err := WriteCSV(&b, m, ','); err != nil {
		t.Fatal(err)
	}
	if b.String() != "1,2\n3,4" {
		t.Fatalf("got %q", b.String())
	}
}

func TestReadMatrixPathFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	p := filepath.Join(dir, "m.csv")
	if err := os.WriteFile(p, []byte("1,2,3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	m, err := ReadMatrixPath(p, ',')
	if err != nil {
		t.Fatal(err)
	}
	if m.Rows() != 1 || m.Cols() != 3 {
		t.Fatalf("got %dx%d", m.Rows(), m.Cols())
	}
}
