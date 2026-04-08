# Golden File Testing with testscript

This tutorial walks through golden file testing using the `github.com/veggiemonk/testscript` library. By the end, you will know how to embed expected output in txtar archives, compare it against actual program output, and auto-update those files when the output format changes intentionally.

## What are golden files?

A golden file is a file that contains the expected output of a program. The testing pattern is straightforward:

1. Run the program and capture its output.
2. Compare the output against the golden file.
3. If they match, the test passes. If they differ, the test fails and shows a diff.
4. When the output format changes intentionally, update the golden file.

This pattern is common in compilers (expected AST dumps), formatters (expected reformatted code), CLI tools (expected help text or structured output), and code generators. The golden file acts as a snapshot of correct behavior -- any unintended change in output is caught immediately.

## The txtar format

testscript uses the txtar format from `golang.org/x/tools/txtar`. A txtar file has two parts:

1. A **comment section** at the top -- this is the script that testscript executes.
2. **File sections** delimited by `-- filename --` markers -- these are extracted to a temporary working directory before the script runs.

Here is what a minimal txtar file looks like:

```
# This is the script section.
# Lines starting with # are comments (and section delimiters).
echo hello
cmp stdout expected.txt

-- expected.txt --
hello
```

When testscript runs this file, it:

1. Creates a temporary working directory.
2. Extracts `expected.txt` into that directory with the content `hello\n`.
3. Executes the script line by line: runs `echo hello`, then compares stdout against `expected.txt`.

The `-- filename --` syntax supports subdirectories too. `-- sub/dir/file.txt --` creates the necessary parent directories automatically.

## The problem with manual golden files

Maintaining expected output by hand works when the output is a single line. It breaks down quickly:

- **Large outputs are tedious to hand-write.** A CLI that emits 50 lines of formatted JSON requires copying that output carefully into the test file.
- **Copy-paste mistakes are silent.** A trailing space, a missing newline, a swapped field -- these produce confusing diffs that waste debugging time.
- **Format changes cascade.** If you update the output format of your program, you must find and update every golden file that references it. Miss one and CI fails.

testscript solves this with the `update` command, which we will cover after building a working example.

## Writing your first golden test

We will build a tiny CLI called `jsonfmt` that reads JSON from stdin and pretty-prints it with indentation. Then we will test it with a golden file.

### The CLI

Create `cmd/jsonfmt/main.go`:

```go
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

func main() {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "reading stdin: %v\n", err)
		os.Exit(1)
	}

	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		fmt.Fprintf(os.Stderr, "parsing json: %v\n", err)
		os.Exit(1)
	}

	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "formatting json: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(out))
}
```

### The test setup

Create `cmd/jsonfmt/script_test.go`:

```go
package main

import (
	"context"
	"os"
	"testing"

	"github.com/veggiemonk/testscript/scripttest"
)

func TestMain(m *testing.M) {
	scripttest.Main(m, map[string]func(){
		"jsonfmt": main,
	})
}

func TestScripts(t *testing.T) {
	engine := scripttest.DefaultEngine()
	scripttest.Test(t, context.Background(), engine, os.Environ(), "testdata/*.txt")
}
```

Two things to note:

- `TestMain` registers the `jsonfmt` binary by mapping its name to the package's `main` function. When testscript runs, it compiles the test binary and places a symlink named `jsonfmt` on the PATH. Script commands like `exec jsonfmt` will invoke it.
- `TestScripts` creates an engine with the default commands (`cmp`, `exec`, `echo`, etc.) and runs every `.txt` file in `testdata/`.

### The test script

Create `cmd/jsonfmt/testdata/format-object.txt`:

```
# Test that jsonfmt pretty-prints a JSON object.
exec jsonfmt < input.json
cmp stdout expected.json

-- input.json --
{"name":"Alice","age":30,"active":true}
-- expected.json --
{
  "active": true,
  "age": 30,
  "name": "Alice"
}
```

The script does three things:

1. Runs `jsonfmt` with `input.json` piped to stdin via `<`.
2. Compares the program's stdout against the embedded `expected.json` file.
3. If they match, the test passes.

### When the test passes

Run the test:

```
$ go test ./cmd/jsonfmt/
ok  	example.com/myproject/cmd/jsonfmt	0.3s
```

No output beyond the standard `ok` line. The golden file matched.

### When the golden file is wrong

Suppose the golden file has a typo -- `"age": 31` instead of `"age": 30`. The test fails with a unified diff:

```
$ go test ./cmd/jsonfmt/
--- FAIL: TestScripts/format-object (0.04s)
    script_test.go:18: # Test that jsonfmt pretty-prints a JSON object.
        > exec jsonfmt < input.json
        [stdout]
        {
          "active": true,
          "age": 30,
          "name": "Alice"
        }
        > cmp stdout expected.json
        diff stdout expected.json
        --- stdout
        +++ expected.json
        @@ -1,5 +1,5 @@
         {
           "active": true,
        -  "age": 30,
        +  "age": 31,
           "name": "Alice"
         }

FAIL
```

The diff shows exactly which line differs, with `-` for the actual output and `+` for the expected content. Fix the golden file and the test passes again.

## The `update` command

Manually editing golden files is fine for small changes. For larger outputs or bulk updates across many test scripts, testscript provides the `update` command.

### How it works

The `update` command writes the current stdout buffer to a named file inside the txtar archive. It is controlled by the `-testscript.update` flag:

- **When the flag is off (the default):** `update` is a no-op. It does nothing.
- **When the flag is on:** `update` captures stdout and overwrites the named file entry in the txtar archive on disk.

This means you can commit scripts that contain both `cmp` (for assertion) and `update` (for refresh). In normal test runs, `update` is invisible.

### The pattern

Here is the updated test script with both commands:

```
# Test that jsonfmt pretty-prints a JSON object.
exec jsonfmt < input.json
cmp stdout expected.json
update expected.json

-- input.json --
{"name":"Alice","age":30,"active":true}
-- expected.json --
{
  "active": true,
  "age": 30,
  "name": "Alice"
}
```

The order matters: `cmp` runs first so the test still fails if the output is wrong. `update` runs after, but only writes when the flag is set.

### Updating golden files

When you intentionally change the output format, run:

```
$ go test ./cmd/jsonfmt/ -testscript.update
```

testscript re-runs every script. For each one, the `update` command captures the current stdout and writes it back into the txtar file's `expected.json` section. The file on disk is modified in place.

After running the update, inspect the diff:

```
$ git diff cmd/jsonfmt/testdata/
```

Review the changes, confirm they are intentional, and commit.

### Important: review before committing

The `-testscript.update` flag is a power tool. It overwrites golden files with whatever the program currently produces. If the program has a bug, the update will bake that bug into the golden file. Always review the diff before committing updated golden files.

## `cmpenv` for dynamic values

Sometimes your program's output contains values that change between runs -- temporary directory paths, timestamps, or other environment-dependent strings. The `cmp` command does a literal byte comparison, so these dynamic values cause spurious failures.

The `cmpenv` command solves this. It expands environment variables in both the actual and expected content before comparing.

### Example

Suppose your program prints the working directory:

```
# Test that the tool prints the working directory.
exec mytool
cmpenv stdout expected.txt

-- expected.txt --
working directory: $WORK
```

`$WORK` is a variable that testscript sets to the temporary working directory. With `cmp`, this comparison would fail because the literal string `$WORK` would not match `/tmp/testscript12345`. With `cmpenv`, `$WORK` in `expected.txt` is expanded to the actual path before comparison.

You can use any variable from the script environment. The most commonly used are:

- `$WORK` -- the script's working directory
- `$TMPDIR` -- the script's temporary directory
- Any variable set with the `env` command (e.g., `env MY_VAR=hello`)

## Best practices

**Keep golden files small and focused.** Each test script should test one behavior. A script that tests JSON formatting of objects should not also test error handling for invalid input. Separate those into different `.txt` files.

**Use comment sections to document intent.** Lines starting with `#` are both comments and section delimiters in testscript. Use them to explain what each part of the script tests:

```
# Verify that nested objects are indented correctly.
exec jsonfmt < nested.json
cmp stdout expected-nested.json

# Verify that arrays are indented correctly.
exec jsonfmt < array.json
cmp stdout expected-array.json
```

**Review diffs from `-testscript.update` before committing.** This cannot be overstated. The update flag is a convenience, not a correctness guarantee. Run `git diff` and read every changed line.

**Use `cmpenv` sparingly.** Prefer making your output deterministic. If you can avoid printing paths or timestamps, do so. `cmpenv` introduces a layer of indirection that makes golden files harder to read. Reserve it for cases where dynamic values are unavoidable.

**Place `update` after `cmp`.** The `cmp` command is the assertion. If you put `update` first and it overwrites the golden file, `cmp` will always pass (because it now compares against what was just written). The correct order is always: assert first, then update.

## Credits and further reading

- John Arundel, [Test scripts in Go](https://bitfieldconsulting.com/posts/test-scripts) -- an excellent introduction to the testscript approach
- Encore, [testscript: a hidden Go testing gem](https://encore.dev/blog/testscript-hidden-testing-gem) -- practical examples and patterns
- [rogpeppe/go-internal#297](https://github.com/rogpeppe/go-internal/issues/297) -- discussion on the update mechanism and golden file workflows
