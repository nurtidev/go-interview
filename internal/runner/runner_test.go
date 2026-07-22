package runner

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// passTest is a Go test that always passes.
const passTest = `package main

import "testing"

func TestOK(t *testing.T) {
	if Add(2, 3) != 5 {
		t.Fatal("wrong")
	}
}
`

const goodCode = `package main

func Add(a, b int) int { return a + b }

func main() {}
`

func newRunner(t *testing.T) *Runner {
	t.Helper()
	r, err := New(2)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return r
}

func TestRunPassed(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	res, err := r.Run(context.Background(), Request{
		Code:         goodCode,
		TestCode:     passTest,
		TimeLimitSec: 10,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Result != ResultPassed {
		t.Fatalf("result = %q, want passed; output:\n%s", res.Result, res.Output)
	}
}

func TestRunTestsFailed(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	badCode := `package main

func Add(a, b int) int { return a - b } // wrong on purpose

func main() {}
`
	res, err := r.Run(context.Background(), Request{
		Code:         badCode,
		TestCode:     passTest,
		TimeLimitSec: 10,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Result != ResultTestsFailed {
		t.Fatalf("result = %q, want tests_failed; output:\n%s", res.Result, res.Output)
	}
}

func TestRunCompileError(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	brokenCode := `package main

func Add(a, b int) int { return a + b + } // syntax error

func main() {}
`
	res, err := r.Run(context.Background(), Request{
		Code:         brokenCode,
		TestCode:     passTest,
		TimeLimitSec: 10,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Result != ResultCompileError {
		t.Fatalf("result = %q, want compile_error; output:\n%s", res.Result, res.Output)
	}
}

func TestRunTimeout(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	loopTest := `package main

import "testing"

func TestLoop(t *testing.T) {
	for {
	}
}
`
	res, err := r.Run(context.Background(), Request{
		Code:         goodCode,
		TestCode:     loopTest,
		TimeLimitSec: 1, // short limit; +2s grace
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Result != ResultTimeout {
		t.Fatalf("result = %q, want timeout; output:\n%s", res.Result, res.Output)
	}
}

func TestRunCodeTooLarge(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	_, err := r.Run(context.Background(), Request{
		Code:     strings.Repeat("x", MaxCodeBytes+1),
		TestCode: passTest,
	})
	if !errors.Is(err, ErrCodeTooLarge) {
		t.Fatalf("err = %v, want ErrCodeTooLarge", err)
	}
}

func TestRunBusyRejects(t *testing.T) {
	t.Parallel()
	// A runner with zero-capacity-after-fill: capacity 1, hold it, then a
	// second concurrent run must be rejected with ErrBusy.
	r := newRunner(t)
	// Saturate both slots synthetically.
	r.sem <- struct{}{}
	r.sem <- struct{}{}
	_, err := r.Run(context.Background(), Request{Code: goodCode, TestCode: passTest})
	if !errors.Is(err, ErrBusy) {
		t.Fatalf("err = %v, want ErrBusy", err)
	}
	<-r.sem
	<-r.sem
}
