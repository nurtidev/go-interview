// Package runner executes user-submitted Go code against a hidden test file in
// a throwaway temp directory and reports whether it compiled, passed or timed
// out.
//
// SECURITY: this is a local-development MVP. It relies only on OS process
// isolation (a fresh temp HOME/GOCACHE, no network via GOPROXY=off, a timeout
// and process-group kill). That is NOT a real sandbox: user code runs with the
// server's privileges and can read the filesystem, exhaust CPU/memory, etc.
// Before any public/multi-tenant deployment this MUST run inside a real
// sandbox (gVisor, nsjail, a container with seccomp/cgroups, or a dedicated
// judge service).
package runner

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// Result classifies the outcome of a run.
type Result string

const (
	ResultPassed       Result = "passed"
	ResultTestsFailed  Result = "tests_failed"
	ResultCompileError Result = "compile_error"
	ResultTimeout      Result = "timeout"
)

const (
	// MaxCodeBytes is the largest accepted user submission.
	MaxCodeBytes = 64 * 1024
	// maxOutputBytes caps the captured combined output.
	maxOutputBytes = 64 * 1024
	// waitGrace is added to the task time limit for compilation/startup.
	waitGrace = 2 * time.Second
	// defaultTimeLimit is used when a task specifies no limit.
	defaultTimeLimit = 10 * time.Second
)

// Sentinel errors surfaced to the caller.
var (
	// ErrBusy means the concurrency limit was hit; retry later (HTTP 503).
	ErrBusy = errors.New("runner busy")
	// ErrCodeTooLarge means the submission exceeded MaxCodeBytes (HTTP 400).
	ErrCodeTooLarge = errors.New("code too large")
)

// Request is a single run request.
type Request struct {
	Code         string // user's main.go
	TestCode     string // hidden main_test.go
	TimeLimitSec int    // per-task limit; <= 0 falls back to defaultTimeLimit
	Race         bool   // run go test with -race
}

// Response is the outcome of a run.
type Response struct {
	Result Result
	Output string
}

// Runner executes submissions with a global concurrency limit and a shared,
// reused build cache.
type Runner struct {
	sem      chan struct{}
	cacheDir string
}

// New builds a Runner allowing at most maxConcurrent simultaneous runs. The
// build cache is shared across runs (under os.TempDir) so repeated runs are
// fast.
func New(maxConcurrent int) (*Runner, error) {
	if maxConcurrent < 1 {
		maxConcurrent = 1
	}
	cacheDir := filepath.Join(os.TempDir(), "goprep-gocache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, err
	}
	return &Runner{
		sem:      make(chan struct{}, maxConcurrent),
		cacheDir: cacheDir,
	}, nil
}

// Run compiles and tests the submission. It returns ErrBusy if the concurrency
// limit is saturated and ErrCodeTooLarge if the code is too big; otherwise the
// error is nil and the outcome is carried in Response.Result.
func (r *Runner) Run(ctx context.Context, req Request) (Response, error) {
	if len(req.Code) > MaxCodeBytes {
		return Response{}, ErrCodeTooLarge
	}

	// Non-blocking acquire: reject rather than queue when saturated.
	select {
	case r.sem <- struct{}{}:
		defer func() { <-r.sem }()
	default:
		return Response{}, ErrBusy
	}

	dir, err := os.MkdirTemp("", "goprep-run-*")
	if err != nil {
		return Response{}, err
	}
	defer os.RemoveAll(dir)

	files := map[string]string{
		"go.mod":       "module sandbox\n\ngo 1.24\n",
		"main.go":      req.Code,
		"main_test.go": req.TestCode,
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			return Response{}, err
		}
	}

	limit := defaultTimeLimit
	if req.TimeLimitSec > 0 {
		limit = time.Duration(req.TimeLimitSec) * time.Second
	}

	runCtx, cancel := context.WithTimeout(ctx, limit+waitGrace)
	defer cancel()

	args := []string{"test", "-count=1", "-run", "."}
	if req.Race {
		args = append(args, "-race")
	}

	cmd := exec.CommandContext(runCtx, "go", args...)
	cmd.Dir = dir
	cmd.Env = minimalEnv(dir, r.cacheDir)
	// Put the child in its own process group so we can kill any grandchildren
	// (e.g. the compiled test binary) it may have spawned.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		// Negative pid => signal the whole process group.
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	// If the killed process leaves a pipe-holding child, give up waiting on
	// the output pipes after this grace period.
	cmd.WaitDelay = waitGrace

	out, runErr := cmd.CombinedOutput()
	output := truncate(out)

	if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
		return Response{Result: ResultTimeout, Output: output}, nil
	}
	if runErr == nil {
		return Response{Result: ResultPassed, Output: output}, nil
	}
	return Response{Result: classify(output), Output: output}, nil
}

// classify distinguishes a compile/build failure from a plain test failure.
// `go test` prints "[build failed]" (or "[setup failed]") when the package
// does not compile; otherwise a non-zero exit means the tests ran and failed.
func classify(output string) Result {
	if strings.Contains(output, "[build failed]") || strings.Contains(output, "[setup failed]") {
		return ResultCompileError
	}
	return ResultTestsFailed
}

// minimalEnv builds a deliberately small environment: a temp HOME, a shared
// build cache, module mode without any network access.
func minimalEnv(home, gocache string) []string {
	return []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + home,
		"GOCACHE=" + gocache,
		"GOPATH=" + filepath.Join(home, "go"),
		"GOFLAGS=-mod=mod",
		"GOPROXY=off",
		"GOTOOLCHAIN=local", // never fetch a toolchain
	}
}

func truncate(b []byte) string {
	if len(b) > maxOutputBytes {
		return string(b[:maxOutputBytes]) + "\n... [output truncated]"
	}
	return string(b)
}
