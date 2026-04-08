# testscript

A Go library for filesystem-based test scripting.

## What is testscript?

testscript lets you test CLI tools by writing scripts in a simple, shell-like language. Scripts live alongside your Go tests in txtar files. Each script runs commands, checks output, and manages files -- all without writing Go boilerplate for every test case.

## Quickstart

Create a trivial CLI, a test harness, and a test script.

**main.go**

```go
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	name := flag.String("name", "", "name to greet")
	flag.Parse()
	if *name == "" {
		fmt.Fprintln(os.Stderr, "usage: hello --name <name>")
		os.Exit(1)
	}
	fmt.Printf("Hello, %s!\n", *name)
}
```

**main_test.go**

```go
package main_test

import (
	"context"
	"os"
	"testing"

	"github.com/veggiemonk/testscript/scripttest"
)

func TestMain(m *testing.M) {
	scripttest.Main(m, map[string]func(){
		"hello": main,
	})
}

func TestScripts(t *testing.T) {
	engine := scripttest.DefaultEngine()
	scripttest.Test(t, context.Background(), engine, os.Environ(), "testdata/*.txt")
}
```

**testdata/hello.txt**

```
# success case
exec hello --name World
stdout 'Hello, World!'

# error case: missing --name flag
! exec hello
stderr 'usage: hello --name <name>'
```

Run:

```
go test -v
```

## How it works

Test scripts use the [txtar format](https://pkg.go.dev/golang.org/x/tools/txtar). The comment section (before any file entries) is the script itself. File entries below the `--` markers become files in the test's working directory.

The engine parses and executes the script line by line. Each line is a command with optional condition guards and prefix modifiers (`!` for expected failure, `?` for don't-care). Lines starting with `#` are section comments that appear in the test log with elapsed time.

`scripttest.Main` compiles your CLI functions into executables available on `$PATH` during test execution. `scripttest.Test` creates a parallel subtest for each matched txtar file.

## Built-in commands

| Command  | Description |
|----------|-------------|
| `cat`    | Concatenate files and print to stdout buffer |
| `cd`     | Change the working directory |
| `cmp`    | Compare two files (or stdout/stderr with a file) for differences |
| `cmpenv` | Compare files with environment variable expansion |
| `cp`     | Copy files to a target file or directory |
| `echo`   | Display a line of text |
| `env`    | Set or print environment variables |
| `exec`   | Run an executable program with arguments |
| `exists` | Check that files exist (with optional `-readonly`, `-exec` flags) |
| `grep`   | Find lines in a file matching a Go regexp |
| `help`   | Log help text for commands and conditions |
| `mkdir`  | Create directories (parents created automatically) |
| `mv`     | Rename a file or directory |
| `rm`     | Remove a file or directory recursively |
| `stderr` | Match a Go regexp against the stderr buffer |
| `stdout` | Match a Go regexp against the stdout buffer |
| `stop`   | Stop script execution (success) |
| `wait`   | Wait for background commands to complete |
| `skip`   | Skip the current test (from scripttest) |
| `update` | Update a golden file in the txtar archive with current stdout (from scripttest) |

Use `-testscript.update` flag to automatically update golden files:

```
go test -testscript.update
```

## Built-in conditions

| Condition      | Description |
|----------------|-------------|
| `[GOOS:os]`   | True when `runtime.GOOS` matches (e.g. `[GOOS:linux]`) |
| `[GOARCH:arch]`| True when `runtime.GOARCH` matches (e.g. `[GOARCH:amd64]`) |
| `[exec:prog]` | True when `prog` is found in the test binary's PATH |
| `[short]`     | True when `go test -short` is set |
| `[verbose]`   | True when `go test -v` is set |
| `[root]`      | True when running as root (euid == 0) |
| `[!cond]`     | Negate any condition (e.g. `[!GOOS:windows]`) |

## Tutorials

- [Golden file testing](docs/tutorial-golden-files.md)
- [Script language reference](docs/script-language-reference.md)

## Credits

Derived from [rsc.io/script](https://pkg.go.dev/rsc.io/script) and [github.com/rogpeppe/go-internal/testscript](https://pkg.go.dev/github.com/rogpeppe/go-internal/testscript).

Recommended reading:

- [Test scripts in Go](https://bitfieldconsulting.com/posts/test-scripts) -- Bitfield Consulting
- [testscript: hidden testing gem](https://encore.dev/blog/testscript-hidden-testing-gem) -- Encore
- [go-internal/testscript discussion](https://github.com/rogpeppe/go-internal/issues/297)

## License

BSD 3-Clause. See [LICENSE](LICENSE).
