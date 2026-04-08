# testscript Library Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a reusable Go testscript library at `github.com/veggiemonk/testscript` combining rsc-script's architecture with goint-script's features.

**Architecture:** Three layers -- core engine (no test dependency), internal diff, and scripttest adapter. Engine/State separation with immutable Engine and mutable per-script State. Commands use interface-based design with `Run() (WaitFunc, error)`. Error returns, no panic-for-control-flow.

**Tech Stack:** Go 1.26, `golang.org/x/tools/txtar`

**Source material:** Copy and adapt from `rsc-script/` (primary architecture) and `goint-script/` (features like Main, named backgrounds). Both are in the same repo at `/Users/julien/perso/testscript/`.

---

## File Structure

| File | Responsibility | Primary Source |
|---|---|---|
| `LICENSE` | BSD 3-clause license | New |
| `TODO.md` | Deferred feature tracker | New |
| `internal/diff/diff.go` | Anchored patience diff algorithm | Copy from `rsc-script/internal/diff/diff.go` |
| `internal/diff/diff_test.go` | Diff tests with txtar testdata | Copy from `rsc-script/internal/diff/diff_test.go` |
| `internal/diff/testdata/*.txt` | Diff test fixtures | Copy from `rsc-script/internal/diff/testdata/` |
| `errors.go` | CommandError, UsageError, ErrUnexpectedSuccess, sentinels | Copy from `rsc-script/errors.go` |
| `conds.go` | Cond interface, BoolCondition, CachedCondition, PrefixCondition, DefaultConds | Adapt from `rsc-script/conds.go` |
| `conds_test.go` | Condition constructor tests | New (TDD) |
| `state.go` | State type, NewState, env handling, background processes | Adapt from `rsc-script/state.go` |
| `state_test.go` | State unit tests | New (TDD) |
| `engine.go` | Engine, Execute, parse, expandArgs, checkStatus, ListCmds/ListConds | Adapt from `rsc-script/engine.go` |
| `engine_test.go` | Parser unit tests | New (TDD) |
| `cmds.go` | All built-in commands, Command() constructor, DefaultCmds | Adapt from `rsc-script/cmds.go` |
| `cmds_unix.go` | isETXTBSY for unix | Adapt from `rsc-script/cmds_posix.go` |
| `exe.go` | copyBinary, Main-related binary copying | Adapt from `goint-script/exe.go` |
| `exe_darwin.go` | Darwin clonefile | Copy from `goint-script/clonefile_darwin.go` |
| `exe_linux.go` | Linux hard link | Copy from `goint-script/clonefile.go` |
| `scripttest/scripttest.go` | Test(), Run(), Main(), DefaultEngine, Skip, update cmd | Adapt from `rsc-script/scripttest/scripttest.go` + `goint-script/exe.go` |
| `scripttest/scripttest_test.go` | Integration tests using fakeT and script files | New (TDD) |
| `scripttest/testdata/*.txt` | Script test fixtures | New |

---

### Task 1: Project Scaffolding -- LICENSE, TODO.md, go.mod cleanup

**Files:**
- Create: `LICENSE`
- Create: `TODO.md`
- Modify: `go.mod`

- [ ] **Step 1: Create the BSD 3-clause LICENSE file**

Write to `LICENSE` the standard BSD 3-clause text with:
- Copyright 2025 Julien Bisconti
- "Derived from rsc.io/script and github.com/rogpeppe/go-internal/testscript."

- [ ] **Step 2: Create TODO.md with deferred features**

```markdown
# TODO

## Deferred Commands
- [ ] `replace old new... file` -- string replacement in files
- [ ] `chmod perm paths...` -- file permissions
- [ ] `symlink path -> target` -- symbolic links
- [ ] `kill [-signal] [name]` -- signal background processes
- [ ] `sleep duration` -- wait for duration

## Deferred Features
- [ ] `stdin file` -- set stdin for next exec
- [ ] `ttyin`/`ttyout` -- PTY support
- [ ] `unquote files...` -- txtar quoting convention
- [ ] `unix2dos paths...` -- line ending conversion
- [ ] `ContinueOnError` mode
- [ ] Typo suggestions via edit distance
```

Write to `TODO.md`.

- [ ] **Step 3: Clean up go.mod**

Remove `github.com/google/go-cmp` from `go.mod` since we don't need it.

Run: `cd /Users/julien/perso/testscript && go mod edit -droprequire github.com/google/go-cmp`

- [ ] **Step 4: Commit**

```bash
git add LICENSE TODO.md go.mod go.sum
git commit -S -m "chore: add LICENSE, TODO.md, clean go.mod"
```

---

### Task 2: Internal Diff Package

**Files:**
- Create: `internal/diff/diff.go`
- Create: `internal/diff/diff_test.go`
- Create: `internal/diff/testdata/*.txt` (12 files)

- [ ] **Step 1: Copy diff testdata files**

Copy all 12 `.txt` files from `rsc-script/internal/diff/testdata/` to `internal/diff/testdata/`.

Run: `mkdir -p internal/diff/testdata && cp rsc-script/internal/diff/testdata/*.txt internal/diff/testdata/`

- [ ] **Step 2: Create diff_test.go (RED)**

Copy from `rsc-script/internal/diff/diff_test.go`. Change only the copyright header to our standard:

```go
// Copyright 2025 Julien Bisconti. All rights reserved.
// Derived from rsc.io/script and github.com/rogpeppe/go-internal/testscript.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
```

Keep the rest verbatim (imports `golang.org/x/tools/txtar`, the `clean` function, and `Test` function).

- [ ] **Step 3: Run test to verify it fails (RED)**

Run: `cd /Users/julien/perso/testscript && go test ./internal/diff/`
Expected: compilation failure -- `Diff` not defined.

- [ ] **Step 4: Create diff.go (GREEN)**

Copy from `rsc-script/internal/diff/diff.go`. Change only the copyright header. Keep the `Diff`, `lines`, and `tgs` functions verbatim.

- [ ] **Step 5: Run tests to verify they pass (GREEN)**

Run: `cd /Users/julien/perso/testscript && go test ./internal/diff/`
Expected: PASS (12 subtests)

- [ ] **Step 6: Run vet**

Run: `cd /Users/julien/perso/testscript && go vet ./internal/diff/`
Expected: no issues

- [ ] **Step 7: Commit**

```bash
git add internal/diff/
git commit -S -m "feat: add internal diff package (anchored patience diff)"
```

---

### Task 3: Error Types

**Files:**
- Create: `errors.go`

- [ ] **Step 1: Create errors.go**

Copy from `rsc-script/errors.go`. Changes:
1. Update copyright header
2. Keep all types exactly as-is: `CommandError`, `UsageError`, `ErrUnexpectedSuccess`, `ErrUsage`, `cmdError`
3. Move `stopError` here (from `rsc-script/cmds.go` lines 1012-1022)
4. Move `waitError` here (from `rsc-script/cmds.go` lines 1104-1125)
5. Add `skipError` type (from `rsc-script/scripttest/scripttest.go` lines 130-139)
6. Export `SkipError` as a type alias and add `MakeSkipError` factory so `scripttest` package can create and check for skip errors

Note: This file references `command` and `quoteArgs` from `engine.go` and `Cmd` interface -- that's OK since they're in the same package. It won't compile until engine.go exists (Task 6).

- [ ] **Step 2: Commit**

```bash
git add errors.go
git commit -S -m "feat: add error types (CommandError, UsageError, sentinels)"
```

---

### Task 4: Conditions

**Files:**
- Create: `conds.go`
- Create: `conds_test.go`

- [ ] **Step 1: Write conds_test.go (RED)**

Test each condition constructor:
- `TestBoolCondition` -- true/false values, suffix rejection
- `TestBoolConditionFalse` -- false case
- `TestCachedCondition` -- caching behavior (call count verification), different suffixes
- `TestPrefixCondition` -- prefix matching with runtime.GOOS
- `TestDefaultConds` -- verifies GOOS, GOARCH, root are present and GOOS matches current platform

See full test code in the source files analyzed earlier. Each test creates a condition, calls `Eval`, and checks the result.

- [ ] **Step 2: Run tests to verify they fail (RED)**

Run: `cd /Users/julien/perso/testscript && go test -run TestBoolCondition -count=1 .`
Expected: compilation failure -- types not defined.

- [ ] **Step 3: Create conds.go (GREEN)**

Copy from `rsc-script/conds.go`. Changes:
1. Update copyright header
2. Remove `OnceCondition` (not needed -- degenerate case of `CachedCondition`)
3. Remove `Condition` constructor (only need Bool, Cached, Prefix)
4. Remove `compiler` condition from `DefaultConds` (not relevant)
5. Keep `GOOS`, `GOARCH`, `root` in `DefaultConds`
6. Keep `BoolCondition`, `CachedCondition`, `PrefixCondition` with their concrete types (`boolCond`, `cachedCond`, `prefixCond`) verbatim

Won't compile until `State`, `Cond`, `CondUsage`, `ErrUsage` are defined (Tasks 5-6).

- [ ] **Step 4: Commit**

```bash
git add conds.go conds_test.go
git commit -S -m "feat: add condition types (BoolCondition, CachedCondition, PrefixCondition)"
```

---

### Task 5: State

**Files:**
- Create: `state.go`
- Create: `state_test.go`

- [ ] **Step 1: Write state_test.go (RED)**

Tests:
- `TestNewState` -- creates state, checks Getwd, LookupEnv for passed vars and pseudo-vars (`/`, `:`)
- `TestStateSetenv` -- set, overwrite, verify
- `TestStateChdir` -- change to subdirectory, verify Getwd, test nonexistent dir fails
- `TestStateExpandEnv` -- normal expansion and regexp mode
- `TestStatePath` -- relative and absolute path resolution

- [ ] **Step 2: Create state.go (GREEN)**

Copy from `rsc-script/state.go`. Changes:
1. Update copyright header
2. Add `name string` field to `backgroundCmd` struct for named background support
3. Keep everything else verbatim: `State` struct, `NewState`, `CloseAndWait`, `Chdir`, `Context`, `Environ`, `ExpandEnv`, `ExtractFiles`, `Getwd`, `Logf`, `flushLog`, `LookupEnv`, `Path`, `Setenv`, `Stdout`, `Stderr`, `cleanEnv`

Won't compile until `Engine`, `WaitFunc`, `Wait()`, `command` are defined (Tasks 6-7).

- [ ] **Step 3: Commit**

```bash
git add state.go state_test.go
git commit -S -m "feat: add State type with env handling and working directory"
```

---

### Task 6: Engine -- Types, Parsing, and Execution

**Files:**
- Create: `engine.go`
- Create: `engine_test.go`

This is the largest task. Copy `rsc-script/engine.go` and adapt it.

- [ ] **Step 1: Write engine_test.go with parser tests (RED)**

Tests for the `parse` function:
- `TestParseBlankLine` -- blank line returns nil, nil
- `TestParseComment` -- `# comment` returns nil, nil
- `TestParseSimpleCommand` -- `echo hello world` -> name="echo", 2 rawArgs
- `TestParseNegation` -- `! exec false` -> want=failure
- `TestParseSuccessOrFailure` -- `? exec maybe` -> want=successOrFailure
- `TestParseCondition` -- `[linux] exec uname` -> 1 cond with want=true, tag="linux"
- `TestParseNegatedCondition` -- `[!windows] exec ls` -> want=false
- `TestParseMultipleConditions` -- `[linux] [amd64] exec uname` -> 2 conds
- `TestParseBackground` -- `exec sleep 10 &` -> background=true
- `TestParseNamedBackground` -- `exec sleep 10 &sleeper&` -> background=true, bgName="sleeper"
- `TestParseQuotedString` -- `echo 'hello world'` -> 1 rawArg, quoted=true
- `TestParseQuotedEscape` -- `echo 'Don''t'` -> fragments for Don and t
- `TestParseInlineComment` -- `echo hello # ignored` -> 1 rawArg
- `TestParseUnterminatedQuote` -- error
- `TestParseDuplicatedPrefix` -- `! ! exec false` -> error
- `TestParseMissingCommand` -- `!` alone -> error
- `TestQuoteArgs` -- round-trip tests for quoteArgs

- [ ] **Step 2: Create engine.go (GREEN)**

Copy from `rsc-script/engine.go`. Changes:
1. Update copyright header
2. Add `bgName string` field to the `command` struct
3. Modify the background detection at the end of `parse()` to also handle `&name&` syntax:
   - Check if last token matches `&` (anonymous) or `&word&` where word is `[a-zA-Z_0-9]+`
   - Set `cmd.bgName` for named backgrounds
4. In `runCommand()`, pass `cmd.bgName` to `backgroundCmd.name`
5. Keep everything else verbatim: `Engine`, `NewEngine`, `Cmd`, `WaitFunc`, `CmdUsage`, `Cond`, `CondUsage`, `Execute`, `expandArgs`, `quoteArgs`, `conditionsActive`, `checkStatus`, `ListCmds`, `ListConds`, `wrapLine`

- [ ] **Step 3: Run tests to verify parser tests pass (GREEN)**

Run: `cd /Users/julien/perso/testscript && go test -run TestParse -count=1 -v .`
Expected: all TestParse* tests PASS.

- [ ] **Step 4: Run all package tests**

Run: `cd /Users/julien/perso/testscript && go test -count=1 -v .`
Expected: all tests pass (parser, state, conditions).

- [ ] **Step 5: Run vet**

Run: `cd /Users/julien/perso/testscript && go vet .`

- [ ] **Step 6: Commit**

```bash
git add engine.go engine_test.go
git commit -S -m "feat: add Engine with parser, execution loop, and named background support"
```

---

### Task 7: Built-in Commands

**Files:**
- Create: `cmds.go`
- Create: `cmds_unix.go`

- [ ] **Step 1: Create cmds_unix.go**

Build tag `//go:build unix`. Contains `isETXTBSY` checking `syscall.ETXTBSY`. Copied from `rsc-script/cmds_posix.go` with updated copyright and simplified build tag.

- [ ] **Step 2: Create cmds.go**

Copy from `rsc-script/cmds.go`. Changes:
1. Update copyright header
2. Update diff import: `"github.com/veggiemonk/testscript/internal/diff"`
3. Remove from `DefaultCmds()`: `chmod`, `replace`, `sleep`, `symlink` (deferred to TODO.md)
4. Keep: `cat`, `cd`, `cmp`, `cmpenv`, `cp`, `echo`, `env`, `exec`, `exists`, `grep`, `help`, `mkdir`, `mv`, `rm`, `stderr`, `stdout`, `stop`, `wait`
5. Simplify `lookPath()` -- remove Windows/Plan9 code. Only unix logic: check executable bit, darwin case-insensitive. Hardcode `"PATH"` instead of `pathEnvName()`.
6. Modify `Wait()` to support named backgrounds:
   - Accept optional `[name]` argument
   - If name given, only wait for backgrounds matching that name, keep others
   - If no name, wait for all (original behavior)
   - Pass `cmd.bgName` to `backgroundCmd.name` in `runCommand()`
7. Keep all helper functions: `Command()`, `firstNonFlag`, `match`, `doCompare`, `startCommand`, `removeAll`
8. Remove `stopError` from this file (moved to `errors.go` in Task 3)
9. Remove `waitError` from this file (moved to `errors.go` in Task 3)

- [ ] **Step 3: Build the package**

Run: `cd /Users/julien/perso/testscript && go build ./...`
Expected: PASS

- [ ] **Step 4: Run all tests**

Run: `cd /Users/julien/perso/testscript && go test -count=1 -v .`
Expected: all tests pass.

- [ ] **Step 5: Run vet**

Run: `cd /Users/julien/perso/testscript && go vet ./...`

- [ ] **Step 6: Commit**

```bash
git add cmds.go cmds_unix.go
git commit -S -m "feat: add built-in commands with named background wait support"
```

---

### Task 8: Binary Copying (exe files)

**Files:**
- Create: `exe.go`
- Create: `exe_darwin.go`
- Create: `exe_linux.go`

- [ ] **Step 1: Create exe_darwin.go**

Build tag: implicit (filename). Uses `golang.org/x/sys/unix` for `Clonefile`. One function: `cloneFile(from, to string) error`.

- [ ] **Step 2: Create exe_linux.go**

Build tag: `//go:build linux`. Uses `os.Link` for hard linking. One function: `cloneFile(from, to string) error`.

- [ ] **Step 3: Create exe.go**

Contains exported `CopyBinary(from, to string) error`:
1. Try `cloneFile(from, to)` first
2. Fall back to full copy via `io.Copy`

Adapted from `goint-script/exe.go` `copyBinary` function, renamed to exported `CopyBinary`.

- [ ] **Step 4: Add golang.org/x/sys dependency**

Run: `cd /Users/julien/perso/testscript && go get golang.org/x/sys`

- [ ] **Step 5: Build**

Run: `cd /Users/julien/perso/testscript && go build ./...`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add exe.go exe_darwin.go exe_linux.go go.mod go.sum
git commit -S -m "feat: add binary copying with platform-specific cloneFile"
```

---

### Task 9: scripttest Package -- Test(), Run(), Main(), Skip, Update

**Files:**
- Create: `scripttest/scripttest.go`
- Modify: `errors.go` (export skip helpers)

- [ ] **Step 1: Update errors.go -- export skip error helpers**

Add to `errors.go`:
- `SkipError` as a type alias for `skipError` (so scripttest can `errors.As` on it)
- `MakeSkipError(msg string) error` factory function

- [ ] **Step 2: Create scripttest.go**

This merges `rsc-script/scripttest/scripttest.go` with `goint-script/exe.go`'s `Main` logic:

1. **`Main(m TestingM, commands map[string]func())`** -- adapted from goint-script:
   - Check `os.Args[0]` for command dispatch
   - Create temp `bin/` dir, copy test binary per command via `script.CopyBinary`
   - Prepend `bin/` to `$PATH`, call `m.Run()`

2. **`Test(t, ctx, engine, env, pattern)`** -- from rsc-script's scripttest:
   - Glob files, create subtests (parallel)
   - Create temp workdir, parse txtar, extract files
   - Clone the engine to inject per-test `update` command
   - After script, apply updates if `-testscript.update` flag is set

3. **`Run(t, engine, state, filename, reader)`** -- from rsc-script's scripttest:
   - Log env in verbose mode
   - Execute script, handle skip/stop/error

4. **`DefaultEngine()`**, **`DefaultCmds()`**, **`DefaultConds()`** -- bridge functions

5. **`Skip()`** -- command that returns `MakeSkipError`

6. **`makeUpdateCmd(file, archive)`** -- the explicit `update` command:
   - Takes one arg (filename in archive)
   - If `-testscript.update` flag is off: no-op
   - If on: writes current `s.Stdout()` to the named file in the archive

7. **`cloneEngine(e)`** -- shallow copy of Engine for per-test command injection

8. **`applyUpdates(t, file, archive)`** -- writes modified archive back to disk

9. **`initScriptDirs(t, s)`** -- sets `$WORK`, `$TMPDIR`

10. Flag: `var updateFlag = flag.Bool("testscript.update", false, "update testscript golden files")`

- [ ] **Step 3: Build everything**

Run: `cd /Users/julien/perso/testscript && go build ./...`
Expected: PASS

- [ ] **Step 4: Run all tests**

Run: `cd /Users/julien/perso/testscript && go test ./... -count=1`
Expected: all tests pass (core package tests from previous tasks).

- [ ] **Step 5: Run vet**

Run: `cd /Users/julien/perso/testscript && go vet ./...`

- [ ] **Step 6: Commit**

```bash
git add scripttest/ errors.go
git commit -S -m "feat: add scripttest package with Test, Run, Main, Skip, and Update commands"
```

---

### Task 10: Integration Tests -- scripttest test suite

**Files:**
- Create: `scripttest/scripttest_test.go`
- Create: `scripttest/testdata/basic.txt`
- Create: `scripttest/testdata/exec.txt`
- Create: `scripttest/testdata/cmp.txt`
- Create: `scripttest/testdata/conditions.txt`
- Create: `scripttest/testdata/env.txt`
- Create: `scripttest/testdata/background.txt`

- [ ] **Step 1: Create basic.txt test script**

```txtar
# Test basic echo and stdout matching.
echo hello world
stdout 'hello world'
! stderr .
```

- [ ] **Step 2: Create exec.txt test script**

```txtar
# Test exec with a simple command.
exec echo hello from exec
stdout 'hello from exec'

# Test negated exec.
! exec false
```

- [ ] **Step 3: Create cmp.txt test script**

```txtar
# Test file comparison.
echo hello
cp stdout actual.txt
cmp actual.txt expected.txt

-- expected.txt --
hello
```

- [ ] **Step 4: Create conditions.txt test script**

```txtar
# Test GOOS condition.
[GOOS:linux] echo running on linux
[GOOS:darwin] echo running on darwin

# Test negated condition.
[!GOOS:windows] echo not windows
stdout 'not windows'

# Test exec condition.
[exec:echo] echo echo exists
stdout 'echo exists'
```

- [ ] **Step 5: Create env.txt test script**

```txtar
# Test environment variable setting and expansion.
env GREETING=hello
echo $GREETING
stdout 'hello'

# Test env display.
env GREETING
stdout 'GREETING=hello'
```

- [ ] **Step 6: Create background.txt test script**

```txtar
# Test anonymous background.
exec echo bg1 &
exec echo bg2 &
wait
stdout 'bg1'
stdout 'bg2'
```

- [ ] **Step 7: Create scripttest_test.go**

```go
// Copyright 2025 Julien Bisconti. All rights reserved.
// Derived from rsc.io/script and github.com/rogpeppe/go-internal/testscript.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scripttest_test

import (
	"context"
	"os"
	"testing"

	"github.com/veggiemonk/testscript/scripttest"
)

func TestScripts(t *testing.T) {
	engine := scripttest.DefaultEngine()
	scripttest.Test(t, context.Background(), engine, os.Environ(), "testdata/*.txt")
}
```

- [ ] **Step 8: Run integration tests**

Run: `cd /Users/julien/perso/testscript && go test ./scripttest/ -count=1 -v`
Expected: PASS -- all 6 subtests (basic, exec, cmp, conditions, env, background).

- [ ] **Step 9: Run full test suite**

Run: `cd /Users/julien/perso/testscript && go test ./... -count=1`
Expected: all tests pass across all packages.

- [ ] **Step 10: Run vet**

Run: `cd /Users/julien/perso/testscript && go vet ./...`

- [ ] **Step 11: Commit**

```bash
git add scripttest/scripttest_test.go scripttest/testdata/
git commit -S -m "feat: add integration tests for scripttest package"
```

---

### Task 11: go mod tidy and Final Verification

**Files:**
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: Run go mod tidy**

Run: `cd /Users/julien/perso/testscript && go mod tidy`

- [ ] **Step 2: Build everything**

Run: `cd /Users/julien/perso/testscript && go build ./...`
Expected: PASS

- [ ] **Step 3: Run all tests**

Run: `cd /Users/julien/perso/testscript && go test ./... -count=1 -v`
Expected: all tests pass.

- [ ] **Step 4: Run vet**

Run: `cd /Users/julien/perso/testscript && go vet ./...`

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum
git commit -S -m "chore: go mod tidy"
```

---

### Task 12: Clean up source directories

**Files:**
- Remove: `rsc-script/` directory
- Remove: `goint-script/` directory

- [ ] **Step 1: Verify the new library works independently**

Run: `cd /Users/julien/perso/testscript && go test ./... -count=1`
Expected: PASS (no dependency on the source directories).

- [ ] **Step 2: Remove source directories**

Ask the user before removing:
> "The library is complete and all tests pass. Should I remove the `rsc-script/` and `goint-script/` source directories that we copied from?"

If yes:
```bash
rm -rf rsc-script/ goint-script/
git add -A
git commit -S -m "chore: remove source reference directories"
```

---

## Summary

| Task | What | Files |
|---|---|---|
| 1 | Project scaffolding | LICENSE, TODO.md, go.mod |
| 2 | Internal diff package | internal/diff/* |
| 3 | Error types | errors.go |
| 4 | Conditions | conds.go, conds_test.go |
| 5 | State | state.go, state_test.go |
| 6 | Engine (parser + execution) | engine.go, engine_test.go |
| 7 | Built-in commands | cmds.go, cmds_unix.go |
| 8 | Binary copying | exe.go, exe_darwin.go, exe_linux.go |
| 9 | scripttest package | scripttest/scripttest.go |
| 10 | Integration tests | scripttest/scripttest_test.go, testdata/*.txt |
| 11 | go mod tidy | go.mod, go.sum |
| 12 | Cleanup | remove rsc-script/, goint-script/ |

Tasks 1-7 build from the bottom up. Each task compiles independently after Task 6 (engine.go provides the types everything depends on). Tasks 3-5 may have compilation failures until Task 6 is done -- that's expected and noted in each task. Task 8-9 add the scripttest layer. Task 10-12 are integration and cleanup.
