// Command ggmat runs small dense-matrix ops with optional parallelism (teaching / CLI).
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/hrodrig/ggpu/pkg/ggmat"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

var binOps = map[string]matBinOp{
	"add":    ggmat.Add,
	"mul-ew": ggmat.MulEW,
	"matmul": ggmat.MatMul,
}

func run(args []string) int {
	fs := flag.NewFlagSet("ggmat", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	aPath := fs.String("a", "", "matrix A: file path or - for stdin (CSV, row-major)")
	bPath := fs.String("b", "", "matrix B (not used for identity)")
	action := fs.String("action", "", "add | mul-ew | matmul | identity | dot")
	vcores := fs.Int("vcores", 0, "worker goroutines (0 means min(64, NumCPU))")
	sep := fs.String("sep", ",", "CSV column separator (single character)")
	outPath := fs.String("out", "", "write matrix result here (default stdout); not used for dot")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *action == "" || *aPath == "" {
		printUsage()
		return 2
	}
	workers := workerCount(*vcores)
	sepR, err := sepRune(*sep)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ggmat: %v\n", err)
		return 2
	}
	act := strings.ToLower(strings.TrimSpace(*action))
	a, err := ggmat.ReadMatrixPath(*aPath, sepR)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ggmat: read A: %v\n", err)
		return 1
	}
	b, code := loadB(act, *bPath, sepR)
	if code != 0 {
		return code
	}
	err = dispatch(act, *action, a, b, *outPath, sepR, workers)
	if err != nil {
		if err != errUsage {
			fmt.Fprintf(os.Stderr, "ggmat: %v\n", err)
		}
		return exitCode(err)
	}
	return 0
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "usage: ggmat -a <path|-> -action <name> [-b <path|->] [-vcores N] [-sep char] [-out path]\n")
	fmt.Fprintf(os.Stderr, "actions: add, mul-ew, matmul, identity, dot\n")
}

func workerCount(vcores int) int {
	w := vcores
	if w < 1 {
		w = min(64, runtime.NumCPU())
		if w < 1 {
			w = 1
		}
	}
	return w
}

func loadB(act, bPath string, sepR rune) (ggmat.Matrix, int) {
	if act == "identity" {
		return nil, 0
	}
	if bPath == "" {
		fmt.Fprintf(os.Stderr, "ggmat: -b required for action %q\n", act)
		return nil, 2
	}
	b, err := ggmat.ReadMatrixPath(bPath, sepR)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ggmat: read B: %v\n", err)
		return nil, 1
	}
	return b, 0
}

func dispatch(act, actionFlag string, a, b ggmat.Matrix, outPath string, sep rune, workers int) error {
	if op, ok := binOps[act]; ok {
		return writeMatrixResult(outPath, a, b, sep, workers, op)
	}
	switch act {
	case "identity":
		out, err := ggmat.Identity(a)
		if err != nil {
			return err
		}
		return writeMatrixOut(outPath, out, sep)
	case "dot":
		s, err := ggmat.Dot(a, b, workers)
		if err != nil {
			return err
		}
		fmt.Printf("%g\n", s)
		return nil
	default:
		fmt.Fprintf(os.Stderr, "ggmat: unknown action %q (use add, mul-ew, matmul, identity, dot)\n", actionFlag)
		return errUsage
	}
}

// errUsage: invalid CLI action (message already printed).
var errUsage = fmt.Errorf("invalid action")

func exitCode(err error) int {
	if err == errUsage {
		return 2
	}
	return 1
}

type matBinOp func(a, b ggmat.Matrix, workers int) (ggmat.Matrix, error)

func writeMatrixResult(outPath string, a, b ggmat.Matrix, sep rune, workers int, op matBinOp) error {
	out, err := op(a, b, workers)
	if err != nil {
		return err
	}
	return writeMatrixOut(outPath, out, sep)
}

func writeMatrixOut(outPath string, m ggmat.Matrix, sep rune) error {
	var w io.Writer = os.Stdout
	if outPath != "" {
		f, err := os.Create(outPath)
		if err != nil {
			return err
		}
		defer f.Close()
		w = f
	}
	return ggmat.WriteCSV(w, m, sep)
}

func sepRune(s string) (rune, error) {
	runes := []rune(strings.TrimSpace(s))
	if len(runes) != 1 {
		return 0, fmt.Errorf("-sep must be exactly one character")
	}
	return runes[0], nil
}
