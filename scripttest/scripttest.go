// Copyright 2025 Julien Bisconti. All rights reserved.
// Derived from rsc.io/script and github.com/rogpeppe/go-internal/testscript.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package scripttest adapts the script engine for use in tests.
package scripttest

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"flag"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	script "github.com/veggiemonk/testscript"

	"golang.org/x/tools/txtar"
)

// updateFlag controls whether testscript golden files are updated in place.
var updateFlag = flag.Bool("testscript.update", false, "update testscript golden files")

// TestingM is implemented by *testing.M. It's defined as an interface
// to allow scripttest to co-exist with other testing frameworks
// that might also wish to call M.Run.
type TestingM interface {
	Run() int
}

// Main should be called within a TestMain function to allow
// subcommands to be run in the testscript context.
// Main always calls [os.Exit], so it does not return back to the caller.
//
// The commands map holds the set of command names, each
// with an associated run function which may call os.Exit.
//
// When Run is called, these commands are installed as regular commands in the
// shell path, so can be invoked with "exec".
func Main(m TestingM, commands map[string]func()) {
	cmdName := filepath.Base(os.Args[0])
	if mainf := commands[cmdName]; mainf != nil {
		os.Args[0] = cmdName
		mainf()
		os.Exit(0)
	}
	// Unknown command; this is just the top-level execution of the
	// test binary by "go test".
	os.Exit(testingMRun(m, commands))
}

// testingMRun sets up the command binaries in $PATH and runs the tests.
func testingMRun(m TestingM, commands map[string]func()) int {
	tmpdir, err := os.MkdirTemp("", "testscript-main")
	if err != nil {
		log.Fatalf("could not set up temporary directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpdir); err != nil {
			log.Fatalf("cannot delete temporary directory: %v", err)
		}
	}()
	bindir := filepath.Join(tmpdir, "bin")
	if err := os.MkdirAll(bindir, 0o777); err != nil {
		log.Fatalf("could not set up PATH binary directory: %v", err)
	}
	if err := os.Setenv("PATH", bindir+string(filepath.ListSeparator)+os.Getenv("PATH")); err != nil {
		log.Fatalf("could not set PATH: %v", err)
	}

	for name := range commands {
		binfile := filepath.Join(bindir, name)
		binpath, err := os.Executable()
		if err == nil {
			err = copyBinary(binpath, binfile)
		}
		if err != nil {
			log.Fatalf("could not set up %s in $PATH: %v", name, err)
		}
	}
	return m.Run()
}

// DefaultCmds returns a set of broadly useful script commands.
//
// This set includes all of the commands in [script.DefaultCmds],
// as well as a "skip" command that halts the script and causes the
// testing.TB passed to [Run] to be skipped.
func DefaultCmds() map[string]script.Cmd {
	cmds := script.DefaultCmds()
	cmds["skip"] = Skip()
	return cmds
}

// DefaultConds returns a set of broadly useful script conditions.
//
// This set includes all of the conditions in [script.DefaultConds],
// as well as:
//
//   - Conditions of the form "exec:foo" are active when the executable "foo" is
//     found in the test process's PATH, and inactive when the executable is
//     not found.
//
//   - "short" is active when testing.Short() is true.
//
//   - "verbose" is active when testing.Verbose() is true.
func DefaultConds() map[string]script.Cond {
	conds := script.DefaultConds()
	conds["exec"] = CachedExec()
	conds["short"] = script.BoolCondition("testing.Short()", testing.Short())
	conds["verbose"] = script.BoolCondition("testing.Verbose()", testing.Verbose())
	return conds
}

// DefaultEngine returns an [script.Engine] configured with [DefaultCmds] and [DefaultConds].
func DefaultEngine() *script.Engine {
	e := script.NewEngine()
	// Add scripttest-specific commands and conditions on top of the base set.
	e.AddCmd("skip", Skip())
	e.AddCond("exec", CachedExec())
	e.AddCond("short", script.BoolCondition("testing.Short()", testing.Short()))
	e.AddCond("verbose", script.BoolCondition("testing.Verbose()", testing.Verbose()))
	return e
}

// An Option configures how [Test] runs script tests.
// The set of available options is closed: only options provided by this
// package may be used.
type Option interface {
	apply(*config)
}

// option is the concrete adapter that implements [Option].
type option func(*config)

func (o option) apply(cfg *config) { o(cfg) }

type config struct {
	engine *script.Engine
	env    []string
	ctx    context.Context
}

// WithEngine sets the script engine used to run tests.
// If not provided, [DefaultEngine] is used.
// If called multiple times, the last call wins.
func WithEngine(e *script.Engine) Option {
	return option(func(c *config) { c.engine = e })
}

// WithEnv sets the environment variables for the script execution.
// If not provided, [os.Environ] is used.
// If called multiple times, the last call wins.
func WithEnv(env []string) Option {
	return option(func(c *config) { c.env = env })
}

// WithContext sets the base context for script execution.
// If not provided, t.Context() is used.
// If called multiple times, the last call wins.
func WithContext(ctx context.Context) Option {
	return option(func(c *config) { c.ctx = ctx })
}

// Test runs the test scripts matching the given pattern.
// It creates parallel subtests for each matched file, unpacks any
// txtar archive files, and runs the script engine on each.
//
// Options can be used to override the default engine, environment, and context.
// Without options, Test uses [DefaultEngine], [os.Environ], and t.Context().
func Test(t *testing.T, pattern string, opts ...Option) {
	cfg := &config{}
	for _, opt := range opts {
		opt.apply(cfg)
	}
	if cfg.engine == nil {
		cfg.engine = DefaultEngine()
	}
	if cfg.env == nil {
		cfg.env = os.Environ()
	}
	if cfg.ctx == nil {
		cfg.ctx = t.Context()
	}

	ctx := cfg.ctx
	gracePeriod := 100 * time.Millisecond
	if deadline, ok := t.Deadline(); ok {
		timeout := time.Until(deadline)

		// If time allows, increase the termination grace period to 5% of the
		// remaining time.
		if gp := timeout / 20; gp > gracePeriod {
			gracePeriod = gp
		}

		// When we run commands that execute subprocesses, we want to reserve two
		// grace periods to clean up. We will send the first termination signal when
		// the context expires, then wait one grace period for the process to
		// produce whatever useful output it can (such as a stack trace). After the
		// first grace period expires, we'll escalate to os.Kill, leaving the second
		// grace period for the test function to record its output before the test
		// process itself terminates.
		timeout -= 2 * gracePeriod

		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		t.Cleanup(cancel)
	}

	files, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("bad pattern %q: %v", pattern, err)
	}
	if len(files) == 0 {
		t.Fatal("no testdata")
	}
	for _, file := range files {
		name := strings.TrimSuffix(filepath.Base(file), ".txt")
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			workdir := t.TempDir()
			s, err := script.NewState(ctx, workdir, cfg.env)
			if err != nil {
				t.Fatal(err)
			}

			// Unpack archive.
			a, err := txtar.ParseFile(file)
			if err != nil {
				t.Fatal(err)
			}
			initScriptDirs(t, s)
			if err := s.ExtractFiles(a); err != nil {
				t.Fatal(err)
			}

			// Clone the engine to inject a per-test "update" command.
			e := cfg.engine.Clone()
			e.AddCmd("update", makeUpdateCmd(a))

			t.Log(time.Now().UTC().Format(time.RFC3339))
			work, _ := s.LookupEnv("WORK")
			t.Logf("$WORK=%s", work)

			// Note: Do not use filepath.Base(file) here:
			// editors that can jump to file:line references in the output
			// will work better seeing the full path relative to the project root.
			Run(t, e, s, file, bytes.NewReader(a.Comment))

			// After script runs, if -testscript.update flag is set, write
			// modified archive back.
			if *updateFlag {
				applyUpdates(t, file, a)
			}
		})
	}
}

// Run runs the script from the given filename starting at the given initial state.
// When the script completes, Run closes the state.
func Run(t testing.TB, e *script.Engine, s *script.State, filename string, testScript io.Reader) {
	t.Helper()
	err := func() (err error) {
		log := new(strings.Builder)
		log.WriteString("\n") // Start output on a new line for consistent indentation.

		// Defer writing to the test log in case the script engine panics during execution,
		// but write the log before we write the final "skip" or "FAIL" line.
		t.Helper()
		defer func() {
			t.Helper()

			if closeErr := s.CloseAndWait(log); err == nil {
				err = closeErr
			}

			if log.Len() > 0 {
				t.Log(strings.TrimSuffix(log.String(), "\n"))
			}
		}()

		if testing.Verbose() {
			// Add the environment to the start of the script log.
			wait, err := script.Env().Run(s)
			if err != nil {
				t.Fatal(err)
			}
			if wait != nil {
				stdout, stderr, err := wait(s)
				if err != nil {
					t.Fatalf("env: %v\n%s", err, stderr)
				}
				if len(stdout) > 0 {
					s.Logf("%s\n", stdout)
				}
			}
		}

		return e.Execute(s, filename, bufio.NewReader(testScript), log)
	}()

	if skip := (script.SkipError{}); errors.As(err, &skip) {
		if skip.Error() == "skip" {
			t.Skip("SKIP")
		} else {
			t.Skipf("SKIP: %v", skip.Error())
		}
	}
	if err != nil {
		t.Errorf("FAIL: %v", err)
	}
}

// Skip returns a [script.Cmd] that halts the script and causes the
// testing.TB passed to [Run] to be skipped.
func Skip() script.Cmd {
	return script.Command(
		script.CmdUsage{
			Summary: "skip the current test",
			Args:    "[msg]",
		},
		func(_ *script.State, args ...string) (script.WaitFunc, error) {
			if len(args) > 1 {
				return nil, script.ErrUsage
			}
			if len(args) == 0 {
				return nil, script.MakeSkipError("")
			}
			return nil, script.MakeSkipError(args[0])
		})
}

// CachedExec returns a [script.Cond] that reports whether the PATH of the test
// binary itself (not the script's current environment) contains the named
// executable.
func CachedExec() script.Cond {
	return script.CachedCondition(
		"<suffix> names an executable in the test binary's PATH",
		func(name string) (bool, error) {
			_, err := exec.LookPath(name)
			return err == nil, nil
		})
}

// initScriptDirs sets up $WORK and $TMPDIR for the script state.
func initScriptDirs(t testing.TB, s *script.State) {
	must := func(err error) {
		if err != nil {
			t.Helper()
			t.Fatal(err)
		}
	}

	work := s.Getwd()
	must(s.Setenv("WORK", work))
	must(os.MkdirAll(filepath.Join(work, "tmp"), 0o777))
	must(s.Setenv("TMPDIR", filepath.Join(work, "tmp")))
}

// makeUpdateCmd creates a per-test "update" command that captures stdout
// into a named file within the txtar archive. This is used with the
// -testscript.update flag to update golden files.
func makeUpdateCmd(ar *txtar.Archive) script.Cmd {
	return script.Command(
		script.CmdUsage{
			Summary: "update a file in the test archive with the current stdout",
			Args:    "filename",
		},
		func(s *script.State, args ...string) (script.WaitFunc, error) {
			if len(args) != 1 {
				return nil, script.ErrUsage
			}
			if !*updateFlag {
				return nil, nil
			}

			name := args[0]
			stdout := s.Stdout()
			data := []byte(stdout)

			// Find existing file in the archive and update it,
			// or append a new file entry.
			found := false
			for i := range ar.Files {
				if ar.Files[i].Name == name {
					ar.Files[i].Data = data
					found = true
					break
				}
			}
			if !found {
				ar.Files = append(ar.Files, txtar.File{
					Name: name,
					Data: data,
				})
			}

			return nil, nil
		})
}

// applyUpdates writes the modified archive back to disk.
func applyUpdates(t testing.TB, file string, ar *txtar.Archive) {
	t.Helper()
	data := txtar.Format(ar)
	if err := os.WriteFile(file, data, 0o666); err != nil {
		t.Fatal(err)
	}
}
