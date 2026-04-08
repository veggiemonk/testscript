// Copyright 2025 Julien Bisconti. All rights reserved.
// Derived from rsc.io/script and github.com/rogpeppe/go-internal/testscript.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package script

import (
	"strings"
	"testing"
)

// FuzzParse verifies that parse never panics on arbitrary input.
func FuzzParse(f *testing.F) {
	f.Add("echo hello world")
	f.Add("")
	f.Add("# comment")
	f.Add("! exec false")
	f.Add("? exec maybe")
	f.Add("[root] exec whoami")
	f.Add("[!root] skip")
	f.Add("exec sleep 1 &")
	f.Add("exec sleep 10 &sleeper&")
	f.Add("echo 'hello world'")
	f.Add("echo 'Don''t'")
	f.Add("echo 'unterminated")
	f.Add("! ? exec false")
	f.Add("!")
	f.Add("[GOOS:linux] [GOARCH:amd64] exec uname")
	f.Add("echo $HOME ${PATH}")
	f.Add("echo '''' ")
	f.Add(strings.Repeat("a", 10000))

	f.Fuzz(func(t *testing.T, line string) {
		// parse must not panic on any input.
		parse("fuzz.txt", 1, line)
	})
}

// FuzzParseQuoteRoundtrip verifies the roundtrip property:
// for any arg, quoteArgs produces a string that parse reconstructs
// back to the original arg.
//
// This addresses the TODO at engine.go:573.
func FuzzParseQuoteRoundtrip(f *testing.F) {
	f.Add("hello")
	f.Add("hello world")
	f.Add("Don't")
	f.Add("'single quotes'")
	f.Add("")
	f.Add("$HOME")
	f.Add("[cond]")
	f.Add("!")
	f.Add("?")
	f.Add("&")
	f.Add("# comment")
	f.Add("with\ttab")
	f.Add("a b c")
	f.Add("&sleeper&")
	f.Add("''")

	f.Fuzz(func(t *testing.T, arg string) {
		// parse works on single lines; newlines can't roundtrip.
		if strings.ContainsAny(arg, "\n\r") {
			return
		}

		quoted := quoteArgs([]string{arg})
		line := "cmd " + quoted
		cmd, err := parse("fuzz.txt", 1, line)
		if err != nil {
			t.Fatalf("parse(%q) failed: %v", line, err)
		}
		if cmd == nil {
			t.Fatalf("parse(%q) returned nil cmd", line)
		}
		if cmd.name != "cmd" {
			t.Errorf("name = %q, want %q (line: %q)", cmd.name, "cmd", line)
		}
		if len(cmd.rawArgs) != 1 {
			t.Fatalf("got %d rawArgs, want 1 (line: %q)", len(cmd.rawArgs), line)
		}

		var got strings.Builder
		for _, frag := range cmd.rawArgs[0] {
			got.WriteString(frag.s)
		}
		if got.String() != arg {
			t.Errorf("roundtrip: input=%q, quoted=%q, parsed=%q",
				arg, quoted, got.String())
		}
	})
}

// FuzzTxtarUnquote verifies that txtarUnquote never panics on arbitrary input.
func FuzzTxtarUnquote(f *testing.F) {
	f.Add([]byte(">hello\n"))
	f.Add([]byte(">line1\n>line2\n"))
	f.Add([]byte(""))
	f.Add([]byte("no prefix\n"))
	f.Add([]byte(">missing newline"))
	f.Add([]byte(">"))
	f.Add([]byte("\n"))

	f.Fuzz(func(t *testing.T, data []byte) {
		// txtarUnquote must not panic on any input.
		txtarUnquote(data)
	})
}
