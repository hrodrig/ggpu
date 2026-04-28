package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunUsage(t *testing.T) {
	t.Parallel()
	if c := run([]string{}); c != 2 {
		t.Fatalf("exit %d", c)
	}
}

func TestRunUnknownAction(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	pa := filepath.Join(dir, "a.csv")
	if err := os.WriteFile(pa, []byte("1,2,3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	pb := filepath.Join(dir, "b.csv")
	if err := os.WriteFile(pb, []byte("1,2,3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if c := run([]string{"-a", pa, "-b", pb, "-action", "nope"}); c != 2 {
		t.Fatalf("exit %d", c)
	}
}

func TestRunAdd(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	pa := filepath.Join(dir, "a.csv")
	pb := filepath.Join(dir, "b.csv")
	if err := os.WriteFile(pa, []byte("1,2\n3,4\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pb, []byte("10,20\n30,40\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "o.csv")
	if c := run([]string{"-a", pa, "-b", pb, "-action", "add", "-out", out}); c != 0 {
		t.Fatalf("exit %d", c)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(b)) != "11,22\n33,44" {
		t.Fatalf("got %q", b)
	}
}

func TestRunMatMul(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	pa := filepath.Join(dir, "a.csv")
	pb := filepath.Join(dir, "b.csv")
	// 2x2 * 2x2 identity
	if err := os.WriteFile(pa, []byte("1,2\n3,4\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pb, []byte("1,0\n0,1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "o.csv")
	if c := run([]string{"-a", pa, "-b", pb, "-action", "matmul", "-out", out}); c != 0 {
		t.Fatalf("exit %d", c)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(b)) != "1,2\n3,4" {
		t.Fatalf("got %q", b)
	}
}

func TestRunMissingB(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	pa := filepath.Join(dir, "a.csv")
	if err := os.WriteFile(pa, []byte("1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if c := run([]string{"-a", pa, "-action", "add"}); c != 2 {
		t.Fatalf("exit %d", c)
	}
}

func TestRunMulEW(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	pa := filepath.Join(dir, "a.csv")
	pb := filepath.Join(dir, "b.csv")
	if err := os.WriteFile(pa, []byte("1,2,3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pb, []byte("4,5,6\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "o.csv")
	if c := run([]string{"-a", pa, "-b", pb, "-action", "mul-ew", "-out", out, "-vcores", "2"}); c != 0 {
		t.Fatalf("exit %d", c)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(b)) != "4,10,18" {
		t.Fatalf("got %q", b)
	}
}

func TestRunIdentity(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	pa := filepath.Join(dir, "a.csv")
	out := filepath.Join(dir, "o.csv")
	if err := os.WriteFile(pa, []byte("7\n8\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if c := run([]string{"-a", pa, "-action", "identity", "-out", out}); c != 0 {
		t.Fatalf("exit %d", c)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(b)) != "7\n8" {
		t.Fatalf("got %q", b)
	}
}

func TestRunDot(t *testing.T) {
	// mutates os.Stdout; keep serial
	dir := t.TempDir()
	pa := filepath.Join(dir, "a.csv")
	pb := filepath.Join(dir, "b.csv")
	if err := os.WriteFile(pa, []byte("1,2,3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pb, []byte("1\n1\n1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	// capture stdout — run uses fmt.Printf for dot
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	code := run([]string{"-a", pa, "-b", pb, "-action", "dot", "-vcores", "2"})
	_ = w.Close()
	os.Stdout = old
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	out, err := io.ReadAll(r)
	_ = r.Close()
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(out)) != "6" {
		t.Fatalf("got %q", out)
	}
}

func TestSepInvalid(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	pa := filepath.Join(dir, "a.csv")
	if err := os.WriteFile(pa, []byte("1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if c := run([]string{"-a", pa, "-b", pa, "-action", "add", "-sep", ",,"}); c != 2 {
		t.Fatalf("exit %d", c)
	}
}
