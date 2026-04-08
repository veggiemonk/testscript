# Script Language Reference

## Overview

Scripts are line-based, one command per line. Lines beginning with `#` are comments that delimit log sections. Blank lines are ignored. Scripts are typically embedded in [txtar](https://pkg.go.dev/golang.org/x/tools/txtar) archives alongside their input files.

## Line syntax

Each line is parsed as a sequence of space-separated words. A complete line has the form:

```
[!] [?] [cond]... command args... [&] [&name&]
```

| Element | Meaning |
|---------|---------|
| `!` | Expect the command to fail. The script stops if it succeeds. |
| `?` | Accept either success or failure; continue regardless. |
| `[cond]` | Run the rest of the line only if *cond* is true. |
| `[!cond]` | Run the rest of the line only if *cond* is false. |
| `&` (trailing) | Run the command in the background (anonymous). |
| `&name&` (trailing) | Run the command in the background with the given name. |

Multiple condition guards on one line are AND-ed -- the command runs only when all conditions are satisfied.

```
[linux] [amd64] exec ./mybin
! exec false
? exec might-not-exist
```

## Quoting and expansion

Words are split on spaces and tabs. Additional rules:

- **Single quotes** disable both splitting and variable expansion. `'hello world'` is one argument.
- **Doubled single quote** produces a literal quote: `'Don''t'` becomes `Don't`.
- **`$VAR`** and **`${VAR}`** expand environment variables. Undefined variables expand to the empty string.
- **`${/}`** expands to the OS path separator (`/` or `\`).
- **`${:}`** expands to the OS list separator (`:` or `;`).
- **`#`** at any unquoted position ends the line (comment).

```
env GREETING='hello world'
echo $GREETING
exec ./prog -path=src${/}main.go
echo 'it''s a test' # this part is a comment
```

## Background commands

A trailing `&` or `&name&` runs a command in the background. Only commands marked `Async` in their usage (currently `exec`) support this.

```
exec server &
exec worker &build&
```

- `wait` -- wait for **all** background commands to finish.
- `wait build` -- wait only for the command started with `&build&`.

When a command is sent to the background, the stdout and stderr buffers are cleared immediately (they no longer correspond to the last foreground command). After `wait` returns, stdout and stderr contain the concatenated output of the collected background commands in start order.

```
exec slow-server &srv&
exec curl http://localhost:8080
stdout 'OK'
wait srv
```

## Commands

### Process execution

#### exec

```
exec program [args...] [&] [&name&]
```

Run an external program found via the script's `$PATH`. Does not terminate the script on its own (unlike a Unix shell). Can be run in the background.

```
exec go version
stdout 'go version'
```

#### wait

```
wait [name]
```

Wait for background commands to complete. With no arguments, waits for all. With a name, waits for only the background command started with `&name&`.

```
exec server &srv&
exec client
wait srv
```

#### stop

```
stop [msg]
```

Halt script execution immediately. The test passes (no error is reported). The optional message is written to the log.

```
[!exec:git] stop 'git not found, skipping'
exec git status
```

#### skip

```
skip [msg]
```

Skip the current test. Available only via the `scripttest` package. Equivalent to calling `t.Skip()`.

```
[short] skip 'skipping in short mode'
exec long-running-test
```

### Output

#### echo

```
echo string...
```

Write arguments to the stdout buffer, separated by spaces, followed by a newline.

```
echo hello world
stdout 'hello world'
```

#### cat

```
cat files...
```

Read the named files and write their concatenated contents to the stdout buffer.

```
cat config.json
stdout '"debug": true'
```

#### env

```
env [key[=value]...]
```

With no arguments, print the entire script environment to the log. With `key=value` arguments, set variables. With bare `key` arguments, print `key=value` to stdout.

```
env HOME=/tmp/test
env HOME
stdout 'HOME=/tmp/test'
```

### Assertions

#### stdout

```
stdout [-count=N] [-q] 'pattern'
```

Assert that the stdout buffer from the last command matches the given Go regexp. With `-count=N`, require exactly N matches. With `-q`, suppress printing matched lines.

```
exec echo hello world
stdout 'hello'
stdout -count=1 'hello'
```

#### stderr

```
stderr [-count=N] [-q] 'pattern'
```

Assert that the stderr buffer from the last command matches the given Go regexp. Same flags as `stdout`.

```
! exec false-cmd
stderr 'error'
```

#### grep

```
grep [-count=N] [-q] 'pattern' file
```

Assert that the contents of *file* match the given Go regexp. Does not modify stdout or stderr buffers.

```
grep 'package main' main.go
grep -count=2 'import' main.go
```

#### cmp

```
cmp [-q] file1 file2
```

Compare two files for exact byte equality. *file1* can be the literal `stdout` or `stderr` to compare against the corresponding buffer. On mismatch, prints a unified diff unless `-q` is given.

```
exec generate-config
cmp stdout expected.json
```

#### cmpenv

```
cmpenv [-q] file1 file2
```

Like `cmp`, but expands environment variables in both files before comparing.

```
env USER=testuser
cmpenv stdout expected.txt
```

#### exists

```
exists [-readonly] [-exec] file...
```

Check that the named files exist. With `-readonly`, also verify they are not writable. With `-exec`, verify they are executable.

```
exists go.mod go.sum
exists -exec ./bin/mytool
```

### Filesystem

#### cd

```
cd dir
```

Change the script's working directory.

```
mkdir subdir
cd subdir
exec pwd
```

#### mkdir

```
mkdir path...
```

Create directories, including any necessary parents (like `mkdir -p`).

```
mkdir -p src/pkg/util
```

#### cp

```
cp src... dst
```

Copy files. *src* can be the literal `stdout` or `stderr` to copy from the corresponding buffer. When copying multiple sources, *dst* must be a directory.

```
exec generate-config
cp stdout config.json
cp a.txt b.txt outdir/
```

#### mv

```
mv old new
```

Rename a file or directory.

```
mv draft.txt final.txt
```

#### rm

```
rm path...
```

Remove files or directories recursively.

```
rm tmp/
rm old.log
```

### Meta

#### help

```
help [-v] name...
```

Print documentation for commands and conditions. Enclose conditions in brackets: `help [GOOS]`. Pass `-v` for full detail.

```
help exec
help [exec]
help -v
```

#### update

```
update file
```

Write the current stdout buffer into the named file within the txtar archive. Available only via the `scripttest` package. This is a no-op unless the test is run with `-testscript.update`.

```
exec generate-output
update expected.txt
```

## Conditions

Conditions appear in square brackets before a command and control whether the line executes.

| Condition | True when |
|-----------|-----------|
| `[GOOS:linux]` | `runtime.GOOS == "linux"` |
| `[GOOS:darwin]` | `runtime.GOOS == "darwin"` |
| `[GOARCH:amd64]` | `runtime.GOARCH == "amd64"` |
| `[GOARCH:arm64]` | `runtime.GOARCH == "arm64"` |
| `[exec:prog]` | Executable `prog` is in the test binary's PATH (cached) |
| `[short]` | `testing.Short()` is true (scripttest only) |
| `[verbose]` | `testing.Verbose()` is true (scripttest only) |
| `[root]` | Running as root (`os.Geteuid() == 0`) |

Negation: prefix the condition name with `!`.

```
[!GOOS:windows] exec ./unix-only.sh
[exec:docker] exec docker ps
[!short] exec slow-integration-test
```

Multiple conditions on one line are AND-ed:

```
[GOOS:linux] [GOARCH:amd64] exec ./linux-amd64-binary
```

## Custom commands

Register custom commands by adding to the engine's `Cmds` map:

```go
import (
    script "github.com/veggiemonk/testscript"
    "github.com/veggiemonk/testscript/scripttest"
)

engine := scripttest.DefaultEngine()
engine.Cmds["greet"] = script.Command(
    script.CmdUsage{
        Summary: "print a greeting",
        Args:    "name",
    },
    func(s *script.State, args ...string) (script.WaitFunc, error) {
        if len(args) != 1 {
            return nil, script.ErrUsage
        }
        return func(*script.State) (string, string, error) {
            return "hello, " + args[0] + "\n", "", nil
        }, nil
    })
```

Then use it in scripts:

```
greet world
stdout 'hello, world'
```

## Custom conditions

Three constructors are available for registering conditions.

### BoolCondition

A static true/false condition. Does not accept a suffix.

```go
engine.Conds["ci"] = script.BoolCondition(
    "running in CI",
    os.Getenv("CI") == "true",
)
```

```
[ci] env VERBOSE=1
```

### CachedCondition

Evaluated once per unique suffix, then cached. The function does not receive a `*State` because results are shared across all script states.

```go
engine.Conds["exec"] = script.CachedCondition(
    "<suffix> names an executable in the test binary's PATH",
    func(name string) (bool, error) {
        _, err := exec.LookPath(name)
        return err == nil, nil
    })
```

```
[exec:curl] exec curl http://example.com
```

### PrefixCondition

Evaluated each time it appears. Receives both the `*State` and the suffix, so it can vary based on script state.

```go
engine.Conds["env"] = script.PrefixCondition(
    "environment variable <suffix> is set",
    func(s *script.State, suffix string) (bool, error) {
        _, ok := s.LookupEnv(suffix)
        return ok, nil
    })
```

```
[env:DEBUG] echo debug mode enabled
```

## Section-based logging

Comment lines starting with `#` delimit log sections. Each section is timed independently.

```
# Set up the environment
env HOME=/tmp/test
mkdir work

# Run the build
exec go build ./...
```

When the engine's `Quiet` mode is enabled, log output for successful sections is discarded -- only the section header and elapsed time are kept. If a command in a section fails, the full log (including every command and its output) is preserved for that section. This keeps test output concise while still providing full detail on failure.

## Credits and further reading

- [Test Scripts in Go](https://bitfieldconsulting.com/posts/test-scripts) -- John Arundel
- [Testscript: A Hidden Testing Gem](https://encore.dev/blog/testscript-hidden-testing-gem) -- Encore
- [Discussion: go-internal#297](https://github.com/rogpeppe/go-internal/issues/297)
