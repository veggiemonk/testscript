// Copyright 2025 Julien Bisconti. All rights reserved.
// Derived from rsc.io/script and github.com/rogpeppe/go-internal/testscript.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package script

import (
	"bufio"
	"strings"
	"testing"
)

func TestParseBlankLine(t *testing.T) {
	cmd, err := parse("test.txt", 1, "")
	if err != nil {
		t.Fatal(err)
	}
	if cmd != nil {
		t.Error("expected nil cmd for blank line")
	}
}

func TestParseComment(t *testing.T) {
	cmd, err := parse("test.txt", 1, "# this is a comment")
	if err != nil {
		t.Fatal(err)
	}
	if cmd != nil {
		t.Error("expected nil cmd for comment-only line")
	}
}

func TestParseSimpleCommand(t *testing.T) {
	cmd, err := parse("test.txt", 1, "echo hello world")
	if err != nil {
		t.Fatal(err)
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	if cmd.name != "echo" {
		t.Errorf("name = %q, want %q", cmd.name, "echo")
	}
	if len(cmd.rawArgs) != 2 {
		t.Fatalf("rawArgs count = %d, want 2", len(cmd.rawArgs))
	}
	if cmd.rawArgs[0][0].s != "hello" {
		t.Errorf("arg[0] = %q, want %q", cmd.rawArgs[0][0].s, "hello")
	}
	if cmd.rawArgs[1][0].s != "world" {
		t.Errorf("arg[1] = %q, want %q", cmd.rawArgs[1][0].s, "world")
	}
}

func TestParseNegation(t *testing.T) {
	cmd, err := parse("test.txt", 1, "! exec false")
	if err != nil {
		t.Fatal(err)
	}
	if cmd.want != failure {
		t.Errorf("want = %q, expected %q", cmd.want, failure)
	}
	if cmd.name != "exec" {
		t.Errorf("name = %q, want %q", cmd.name, "exec")
	}
}

func TestParseSuccessOrFailure(t *testing.T) {
	cmd, err := parse("test.txt", 1, "? exec maybe")
	if err != nil {
		t.Fatal(err)
	}
	if cmd.want != successOrFailure {
		t.Errorf("want = %q, expected %q", cmd.want, successOrFailure)
	}
}

func TestParseCondition(t *testing.T) {
	cmd, err := parse("test.txt", 1, "[root] exec whoami")
	if err != nil {
		t.Fatal(err)
	}
	if len(cmd.conds) != 1 {
		t.Fatalf("conds count = %d, want 1", len(cmd.conds))
	}
	if cmd.conds[0].tag != "root" {
		t.Errorf("cond tag = %q, want %q", cmd.conds[0].tag, "root")
	}
	if !cmd.conds[0].want {
		t.Error("expected cond.want = true")
	}
}

func TestParseNegatedCondition(t *testing.T) {
	cmd, err := parse("test.txt", 1, "[!root] skip")
	if err != nil {
		t.Fatal(err)
	}
	if len(cmd.conds) != 1 {
		t.Fatalf("conds count = %d, want 1", len(cmd.conds))
	}
	if cmd.conds[0].want {
		t.Error("expected cond.want = false for negated condition")
	}
	if cmd.conds[0].tag != "root" {
		t.Errorf("cond tag = %q, want %q", cmd.conds[0].tag, "root")
	}
}

func TestParseMultipleConditions(t *testing.T) {
	cmd, err := parse("test.txt", 1, "[GOOS:linux] [GOARCH:amd64] exec uname")
	if err != nil {
		t.Fatal(err)
	}
	if len(cmd.conds) != 2 {
		t.Fatalf("conds count = %d, want 2", len(cmd.conds))
	}
	if cmd.conds[0].tag != "GOOS:linux" {
		t.Errorf("cond[0] tag = %q, want %q", cmd.conds[0].tag, "GOOS:linux")
	}
	if cmd.conds[1].tag != "GOARCH:amd64" {
		t.Errorf("cond[1] tag = %q, want %q", cmd.conds[1].tag, "GOARCH:amd64")
	}
}

func TestParseBackground(t *testing.T) {
	cmd, err := parse("test.txt", 1, "exec sleep 1 &")
	if err != nil {
		t.Fatal(err)
	}
	if !cmd.background {
		t.Error("expected background = true")
	}
	if cmd.bgName != "" {
		t.Errorf("bgName = %q, want empty", cmd.bgName)
	}
	if cmd.name != "exec" {
		t.Errorf("name = %q, want %q", cmd.name, "exec")
	}
}

func TestParseNamedBackground(t *testing.T) {
	cmd, err := parse("test.txt", 1, "exec sleep 10 &sleeper&")
	if err != nil {
		t.Fatal(err)
	}
	if !cmd.background {
		t.Error("expected background = true")
	}
	if cmd.bgName != "sleeper" {
		t.Errorf("bgName = %q, want %q", cmd.bgName, "sleeper")
	}
}

func TestParseQuotedString(t *testing.T) {
	cmd, err := parse("test.txt", 1, "echo 'hello world'")
	if err != nil {
		t.Fatal(err)
	}
	if len(cmd.rawArgs) != 1 {
		t.Fatalf("rawArgs count = %d, want 1", len(cmd.rawArgs))
	}
	if cmd.rawArgs[0][0].s != "hello world" {
		t.Errorf("arg = %q, want %q", cmd.rawArgs[0][0].s, "hello world")
	}
	if !cmd.rawArgs[0][0].quoted {
		t.Error("expected quoted = true")
	}
}

func TestParseQuotedEscape(t *testing.T) {
	cmd, err := parse("test.txt", 1, "echo 'Don''t'")
	if err != nil {
		t.Fatal(err)
	}
	if len(cmd.rawArgs) != 1 {
		t.Fatalf("rawArgs count = %d, want 1", len(cmd.rawArgs))
	}
	// The raw fragments should be "Don" (quoted) + "'" (quoted) + "t" (quoted)
	// When expanded, this becomes Don't
	frags := cmd.rawArgs[0]
	var result strings.Builder
	for _, f := range frags {
		result.WriteString(f.s)
	}
	if result.String() != "Don't" {
		t.Errorf("combined fragments = %q, want %q", result.String(), "Don't")
	}
}

func TestParseInlineComment(t *testing.T) {
	cmd, err := parse("test.txt", 1, "echo hello # this is a comment")
	if err != nil {
		t.Fatal(err)
	}
	if cmd.name != "echo" {
		t.Errorf("name = %q, want %q", cmd.name, "echo")
	}
	if len(cmd.rawArgs) != 1 {
		t.Fatalf("rawArgs count = %d, want 1", len(cmd.rawArgs))
	}
	if cmd.rawArgs[0][0].s != "hello" {
		t.Errorf("arg = %q, want %q", cmd.rawArgs[0][0].s, "hello")
	}
}

func TestParseUnterminatedQuote(t *testing.T) {
	_, err := parse("test.txt", 1, "echo 'unterminated")
	if err == nil {
		t.Error("expected error for unterminated quote")
	}
}

func TestParseDuplicatedPrefix(t *testing.T) {
	_, err := parse("test.txt", 1, "! ? exec false")
	if err == nil {
		t.Error("expected error for duplicated prefix")
	}
}

func TestParseMissingCommand(t *testing.T) {
	_, err := parse("test.txt", 1, "!")
	if err == nil {
		t.Error("expected error for missing command after prefix")
	}
}

func TestContinueOnError(t *testing.T) {
	e := NewEngine()
	e.SetContinueOnError(true)

	s, err := NewState(t.Context(), t.TempDir(), []string{"PATH=/usr/bin"})
	if err != nil {
		t.Fatal(err)
	}

	// Script where the first command fails but the second succeeds.
	script := "! exec true\nexec true\n"
	var log strings.Builder
	err = e.Execute(s, "test.txt", bufio.NewReader(strings.NewReader(script)), &log)
	if err == nil {
		t.Fatal("expected error from ContinueOnError execution, got nil")
	}
	// The error should mention the failing command.
	if !strings.Contains(err.Error(), "unexpected") {
		t.Errorf("error = %q, expected it to mention 'unexpected'", err.Error())
	}
}

func TestContinueOnErrorAllPass(t *testing.T) {
	e := NewEngine()
	e.SetContinueOnError(true)

	s, err := NewState(t.Context(), t.TempDir(), []string{"PATH=/usr/bin"})
	if err != nil {
		t.Fatal(err)
	}

	script := "exec true\nexec true\n"
	var log strings.Builder
	err = e.Execute(s, "test.txt", bufio.NewReader(strings.NewReader(script)), &log)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestQuoteArgs(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"hello"}, "hello"},
		{[]string{"hello", "world"}, "hello world"},
		{[]string{"hello world"}, "'hello world'"},
		{[]string{"Don't"}, "'Don''t'"},
		{[]string{}, ""},
		{[]string{"a", "b c", "d"}, "a 'b c' d"},
	}

	for _, tt := range tests {
		got := quoteArgs(tt.args)
		if got != tt.want {
			t.Errorf("quoteArgs(%v) = %q, want %q", tt.args, got, tt.want)
		}
	}
}
