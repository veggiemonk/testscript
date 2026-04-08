# testscript Library Design Spec

**Module:** `github.com/veggiemonk/testscript`  
**Date:** 2026-04-08  
**Status:** Draft  

## Goal

Build a reusable Go testscript library combining rsc-script's clean architecture (Engine/State separation, interface-based commands, idiomatic error handling) with goint-script's battle-tested feature set (Main subprocess mechanism, named background processes, golden file support). Linux and Darwin only.

First consumer: gojur (`~/perso/gojur`) — CLI integration tests and golden file evolution.

## Architectural Decisions

- **rsc-script is the default architectural choice.** When both projects offer different approaches, follow rsc-script unless goint-script's feature explicitly requires otherwise.
- **Error returns, not panic-for-control-flow.** Commands return `(WaitFunc, error)`. The engine handles negation (`!`, `?`) centrally via `checkStatus`. No `failNow` panic sentinel.
- **Engine is immutable, State is mutable.** Multiple scripts share one Engine concurrently. State is per-script.
- **Explicit golden file updates.** An `update` command (not a magic `UpdateScripts` flag) writes stdout to a named txtar archive file. Controlled by a `-testscript.update` test flag; no-op when the flag is off.

## Copyright

Every source file carries:

```go
// Copyright 2025 Julien Bisconti. All rights reserved.
// Derived from rsc.io/script and github.com/rogpeppe/go-internal/testscript.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
```

A `LICENSE` file (BSD 3-clause) is included at the repo root.

## Dependencies

- `golang.org/x/tools/txtar` — txtar archive format
- Standard library only otherwise

`github.com/google/go-cmp` is NOT used. We port rsc-script's internal anchored diff algorithm.

## Package Structure

```
github.com/veggiemonk/testscript/
├── go.mod
├── go.sum
├── LICENSE
├── TODO.md                        # Deferred features tracker
├── engine.go                      # Engine, Execute, parse, expandArgs, checkStatus
├── engine_test.go                 # Parser unit tests, execution loop tests
├── state.go                       # State, NewState, CloseAndWait, Chdir, Getenv/Setenv
├── state_test.go                  # Environment handling, working directory tests
├── errors.go                      # CommandError, UsageError, ErrUnexpectedSuccess
├── cmds.go                        # Built-in commands, Command() constructor, DefaultCmds()
├── cmds_test.go                   # Command unit tests
├── conds.go                       # Condition constructors, DefaultConds()
├── conds_test.go                  # Condition constructor tests
├── exe.go                         # copyBinary logic
├── exe_darwin.go                  # Darwin clonefile syscall
├── exe_linux.go                   # Linux hard link strategy
├── internal/
│   └── diff/
│       ├── diff.go                # Anchored (patience) diff, O(n log n)
│       └── diff_test.go           # Table-driven tests
├── scripttest/
│   ├── scripttest.go              # Test(), Run(), Main(), DefaultEngine(), Skip
│   ├── scripttest_test.go         # Meta-tests using fakeT + script files
│   └── testdata/
│       ├── basic.txt
│       ├── exec.txt
│       ├── background.txt
│       ├── conditions.txt
│       ├── cmp.txt
│       ├── update.txt
│       └── ...
└── testdata/
    └── ...                        # Core engine test scripts
```

## Core Types

### Engine (`engine.go`)

```go
type Engine struct {
    Cmds  map[string]Cmd
    Conds map[string]Cond
    Quiet bool
}

func (e *Engine) Execute(s *State, file string, script *bufio.Reader, log io.Writer) error
func (e *Engine) ListCmds(s *State, verbose bool) (string, error)
func (e *Engine) ListConds(s *State, verbose bool) (string, error)
```

Immutable after construction. Shared across concurrent script executions. `Quiet` suppresses successful section logs (only failing sections emit detail).

### Cmd (`engine.go`)

```go
type Cmd interface {
    Run(s *State, args ...string) (WaitFunc, error)
    Usage() *CmdUsage
}

type WaitFunc func(*State) (stdout, stderr string, err error)

type CmdUsage struct {
    Summary    string
    Args       string
    Detail     []string
    Async      bool                              // true if Run may return non-nil WaitFunc
    RegexpArgs func(rawArgs ...string) []int     // which arg positions are regexp patterns
}
```

Commands never see negation (`!`/`?`). They return errors; the engine applies `checkStatus` uniformly.

`Command(usage CmdUsage, run func(*State, ...string) (WaitFunc, error)) Cmd` — factory for simple commands.

### Cond (`conds.go`)

```go
type Cond interface {
    Eval(s *State, suffix string) (bool, error)
    Usage() *CondUsage
}

type CondUsage struct {
    Summary string
    Prefix  bool   // true for [name:suffix] form, false for [name] form
}
```

Three constructors:

- `BoolCondition(summary string, v bool) Cond` — static truth value, computed once before registration.
- `CachedCondition(summary string, eval func(string) (bool, error)) Cond` — per-suffix cache using `sync.Map` with channel-as-inflight-marker pattern for concurrent safety.
- `PrefixCondition(summary string, eval func(*State, string) (bool, error)) Cond` — prefix form, called every eval.

### State (`state.go`)

```go
type State struct {
    // unexported fields
}

func NewState(ctx context.Context, workdir string, env []string) (*State, error)
func (s *State) CloseAndWait(log io.Writer) error
func (s *State) Chdir(dir string) error
func (s *State) Getenv(key string) string
func (s *State) Setenv(key, value string) error
func (s *State) Environ() []string
func (s *State) Context() context.Context
func (s *State) Stdout() string
func (s *State) Stderr() string
func (s *State) Path() string                              // current working directory
func (s *State) MkAbs(path string) string                  // resolve relative to virtual cwd
func (s *State) ExtractFiles(ar *txtar.Archive) error      // extract txtar files to workdir
func (s *State) LookPath(name string) (string, error)      // search script's $PATH
```

Internal fields:
- `ctx`, `cancel` — context for subprocess lifecycle
- `pwd` — virtual working directory (not host cwd)
- `env []string`, `envMap map[string]string` — dual storage for O(1) lookup + exec.Cmd compatibility
- `stdout`, `stderr string` — output from most recent foreground command
- `background []backgroundCmd` — pending background processes (supports named: `&name&`)
- `log bytes.Buffer` — section-based log buffer
- `engine *Engine` — set by `Execute`, used by `help`

### Error Types (`errors.go`)

```go
var ErrUnexpectedSuccess = errors.New("unexpected success")

type CommandError struct {
    File string
    Line int
    Op   string
    Args []string
    Err  error
}

type UsageError struct {
    Cmd  Cmd
    Msg  string
}
```

Sentinels (unexported):
- `stopError` — clean halt, test passes
- `skipError` — test skip (caught by scripttest)
- `waitError` — wraps multiple background CommandErrors; `Unwrap()` returns inner error only when exactly one (rsc-style)

## Script Language

### Syntax

```
# Comment — starts a new log section
[condition] command args...      # condition guard
[!condition] command args...     # negated condition
! command args...                # expect failure
? command args...                # accept success or failure

exec myapp &                     # anonymous background
exec myapp &build&               # named background

'single quoted' with '' escaping
$VAR ${VAR} expansion
${/} path separator  ${:} list separator
```

### Parsing

Single-pass character scanner (rsc-style). No regex, no lexer struct. Rules:
1. `#` at any position (when not quoted) ends the line
2. Prefixes parsed before command name: `!` (expect failure), `?` (either outcome), `[cond]`/`[!cond]` (guards, AND-ed)
3. Command name: first non-prefix token
4. Arguments: remaining tokens, space-separated
5. Background: trailing `&` (anonymous) or `&name&` (named), removed from args
6. Single-quoted strings disable splitting and env expansion; `''` = literal `'`

### Variable Expansion

`os.Expand` with the script's env map. Pseudo-variables `${/}` (filepath separator) and `${:}` (list separator). `CmdUsage.RegexpArgs` designates which argument positions are regexp patterns — the engine applies `regexp.QuoteMeta` to expanded values in those positions.

### Execution Flow

1. Read line by line via `bufio.Reader`
2. Check `ctx.Err()` at top of each iteration
3. `#` lines trigger `endSection` (flush log, record timing) and start new section
4. `parse()` produces a command struct; blank/comment lines produce nil
5. `conditionsActive()` evaluates all condition guards; skip if any false
6. `expandArgs()` expands env vars (with RegexpArgs handling)
7. `runCommand()`: validate, call `impl.Run(s, args...)`, handle background vs foreground, call `checkStatus`
8. `checkStatus`: success (default) + error = fail; `!` + no error = `ErrUnexpectedSuccess`; `?` = always pass
9. `stopError` triggers clean halt; `skipError` caught by scripttest

### Section-Based Logging

`#` comment lines delimit log sections. In quiet mode (`Engine.Quiet`), successful sections are discarded. Failed sections dump their full log. Timestamps appended at section boundaries. This means test output only shows failing sections in detail.

## Built-in Commands (v1)

| Command | Async | Behavior |
|---|---|---|
| `exec prog args...` | Yes | Subprocess via script's `$PATH`. `&`/`&name&` background. ETXTBSY retry on Linux. |
| `cat files...` | No | Read files to stdout buffer |
| `echo args...` | No | Write args to stdout buffer |
| `cd dir` | No | Change virtual working directory |
| `mkdir dirs...` | No | `os.MkdirAll` |
| `cp src... dst` | No | `stdout`/`stderr` as virtual sources |
| `mv old new` | No | `os.Rename` |
| `rm paths...` | No | Recursive, chmod writable first |
| `env [key=val...]` | No | No args: print all. With args: set or display |
| `stdout [-count=N] pattern` | No | Regex match on stdout buffer |
| `stderr [-count=N] pattern` | No | Regex match on stderr buffer |
| `grep [-count=N] pattern file` | No | Regex match on file contents |
| `cmp [-q] f1 f2` | No | Exact byte comparison. Shows anchored diff on failure. `f1` can be `stdout`/`stderr`. |
| `cmpenv [-q] f1 f2` | No | Like `cmp` but expands env vars first |
| `update file` | No | Write current stdout to named file in txtar archive. No-op unless `-testscript.update` flag is set. |
| `exists [-readonly] [-exec] files...` | No | Stat checks with optional permission assertions |
| `skip [msg]` | No | Skip test (sentinel error caught by scripttest) |
| `stop [msg]` | No | Clean halt, test passes |
| `wait [name]` | No | Collect background processes by name or all |
| `help [-v] name...` | No | Introspect engine commands/conditions |

## Built-in Conditions (v1)

| Condition | Type | Behavior |
|---|---|---|
| `[darwin]`, `[linux]` | Bool | `runtime.GOOS == cond` |
| `[amd64]`, `[arm64]` | Bool | `runtime.GOARCH == cond` |
| `[exec:prog]` | Cached | `LookPath(prog)` on host PATH, result cached per name |
| `[short]` | Bool | `testing.Short()` — registered by scripttest, not core |
| `[verbose]` | Bool | `testing.Verbose()` — registered by scripttest, not core |

## The `scripttest` Package

### Public API

```go
// Test discovers and runs txtar script files as subtests.
func Test(t *testing.T, ctx context.Context, engine *script.Engine, env []string, pattern string)

// Run executes a single script against a State, reporting results to t.
func Run(t *testing.T, engine *script.Engine, s *script.State, filename string, script *bufio.Reader)

// Main sets up subprocess dispatch for TestMain.
func Main(m *testing.M, commands map[string]func())

// DefaultEngine returns an Engine with DefaultCmds + DefaultConds.
func DefaultEngine() *script.Engine

// DefaultCmds returns the core DefaultCmds (no test-specific additions).
func DefaultCmds() map[string]script.Cmd

// DefaultConds returns DefaultConds + test-specific conditions (short, verbose, exec:prog).
func DefaultConds() map[string]script.Cond
```

### Test() Flow

1. Glob txtar files matching `pattern`
2. Each file becomes a `t.Run` subtest (parallel by default)
3. Create temp dir as `$WORK`, set `$TMPDIR`
4. Parse txtar archive: comment = script, files = extracted to `$WORK`
5. Call `Run(t, engine, state, file, scriptReader)`
6. Propagate deadline from `t.Deadline()` with grace period for subprocess cleanup

### Main() / Subprocess Mechanism

1. Check `os.Args[0]` — if it matches a registered command, dispatch and `os.Exit(0)`
2. Otherwise: create temp `bin/` dir, copy test binary per registered command name, prepend to `$PATH`, call `m.Run()`
3. Binary copying: Darwin uses `clonefile` syscall, fallback hard link, fallback copy. Linux uses hard link, fallback copy.

### The `update` Command

- `update golden.txt` writes current `stdout` to `golden.txt` in the txtar archive
- Controlled by `-testscript.update` test flag
- When flag is off, `update` is a silent no-op — scripts can contain both `cmp` and `update` permanently
- After the script finishes, modified archives are written back to disk
- Only works for files that came from the txtar archive (not arbitrary filesystem files)
- The `update` command is registered by `scripttest`, not the core engine. If `Engine.Execute` is used directly (outside of `scripttest.Test`), the `update` command is not available unless explicitly added.

### Error Flow

- `skipError` sentinel -> `t.Skip(msg)`
- `stopError` sentinel -> return nil (test passes)
- `*CommandError` -> `t.Errorf` with file:line context

## Internal Diff (`internal/diff`)

Ported from rsc-script's anchored (patience) diff algorithm.

```go
func Diff(oldName string, old []byte, newName string, new []byte) []byte
```

Returns unified diff format. Returns nil if inputs are identical. O(n log n) via Szymanski LCS-of-unique-lines. Used by `cmp`/`cmpenv` to display textual differences on failure.

## Testing Strategy

1. **Self-hosting meta-tests** — the library's own tests use script files in `testdata/`. A `fakeT` implementation lets us run nested script sessions in-process.
2. **Parser unit tests** — table-driven for `parse()`: quoted strings, prefixes, conditions, background, edge cases.
3. **Condition constructor tests** — `BoolCondition`, `CachedCondition`, `PrefixCondition` each tested.
4. **Integration script tests** — one `.txt` per command or logical group, covering: all built-in commands, background lifecycle, negation/success-or-failure, condition evaluation, env expansion, update command, Main mechanism, help output.
5. **Diff tests** — table-driven via txtar testdata files.
6. **TDD throughout** — red/green/refactor per CLAUDE.md instructions.

## Deferred Features (tracked in TODO.md)

- `replace old new... file` — string replacement in files
- `chmod perm paths...` — file permissions
- `symlink path -> target` — symbolic links
- `kill [-signal] [name]` — signal background processes
- `sleep duration` — wait for duration
- `stdin file` — set stdin for next exec
- `ttyin`/`ttyout` — PTY support
- `unquote files...` — txtar quoting convention
- `unix2dos paths...` — line ending conversion
- `ContinueOnError` mode
- Typo suggestions via edit distance
