// Copyright 2025 Julien Bisconti. All rights reserved.
// Derived from rsc.io/script and github.com/rogpeppe/go-internal/testscript.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package script

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNewState(t *testing.T) {
	dir := t.TempDir()
	s, err := NewState(context.Background(), dir, []string{"FOO=bar", "BAZ=qux"})
	if err != nil {
		t.Fatal(err)
	}

	// Getwd should return the absolute working directory.
	if got := s.Getwd(); got != dir {
		t.Errorf("Getwd() = %q, want %q", got, dir)
	}

	// Passed vars should be accessible.
	if v, ok := s.LookupEnv("FOO"); !ok || v != "bar" {
		t.Errorf("LookupEnv(FOO) = (%q, %v), want (bar, true)", v, ok)
	}
	if v, ok := s.LookupEnv("BAZ"); !ok || v != "qux" {
		t.Errorf("LookupEnv(BAZ) = (%q, %v), want (qux, true)", v, ok)
	}

	// Pseudo-vars should exist.
	if v, ok := s.LookupEnv("/"); !ok || v != string(os.PathSeparator) {
		t.Errorf("LookupEnv(/) = (%q, %v), want (%q, true)", v, ok, string(os.PathSeparator))
	}
	if v, ok := s.LookupEnv(":"); !ok || v != string(os.PathListSeparator) {
		t.Errorf("LookupEnv(:) = (%q, %v), want (%q, true)", v, ok, string(os.PathListSeparator))
	}
}

func TestStateSetenv(t *testing.T) {
	dir := t.TempDir()
	s, err := NewState(context.Background(), dir, []string{})
	if err != nil {
		t.Fatal(err)
	}

	// Set a new variable.
	if err := s.Setenv("MYVAR", "hello"); err != nil {
		t.Fatal(err)
	}
	if v, ok := s.LookupEnv("MYVAR"); !ok || v != "hello" {
		t.Errorf("LookupEnv(MYVAR) = (%q, %v), want (hello, true)", v, ok)
	}

	// Overwrite the variable.
	if err := s.Setenv("MYVAR", "world"); err != nil {
		t.Fatal(err)
	}
	if v, ok := s.LookupEnv("MYVAR"); !ok || v != "world" {
		t.Errorf("LookupEnv(MYVAR) = (%q, %v), want (world, true)", v, ok)
	}
}

func TestStateChdir(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "subdir")
	if err := os.Mkdir(sub, 0o777); err != nil {
		t.Fatal(err)
	}

	s, err := NewState(context.Background(), dir, []string{})
	if err != nil {
		t.Fatal(err)
	}

	// Change to subdir.
	if err := s.Chdir("subdir"); err != nil {
		t.Fatal(err)
	}
	if got := s.Getwd(); got != sub {
		t.Errorf("Getwd() = %q, want %q", got, sub)
	}

	// Nonexistent directory should fail.
	if err := s.Chdir("nonexistent"); err == nil {
		t.Error("expected error for nonexistent directory")
	}
}

func TestStateExpandEnv(t *testing.T) {
	dir := t.TempDir()
	s, err := NewState(context.Background(), dir, []string{"NAME=world"})
	if err != nil {
		t.Fatal(err)
	}

	// Normal expansion.
	got := s.ExpandEnv("hello $NAME", false)
	if got != "hello world" {
		t.Errorf("ExpandEnv normal = %q, want %q", got, "hello world")
	}

	// Regexp mode should quote meta characters.
	if err := s.Setenv("PAT", "a.b"); err != nil {
		t.Fatal(err)
	}
	got = s.ExpandEnv("$PAT", true)
	if got != `a\.b` {
		t.Errorf("ExpandEnv regexp = %q, want %q", got, `a\.b`)
	}
}

func TestStatePath(t *testing.T) {
	dir := t.TempDir()
	s, err := NewState(context.Background(), dir, []string{})
	if err != nil {
		t.Fatal(err)
	}

	// Relative path should be joined with pwd.
	got := s.Path("foo")
	want := filepath.Join(dir, "foo")
	if got != want {
		t.Errorf("Path(foo) = %q, want %q", got, want)
	}

	// Absolute path should be cleaned only.
	got = s.Path("/absolute/path")
	want = filepath.Clean("/absolute/path")
	if got != want {
		t.Errorf("Path(/absolute/path) = %q, want %q", got, want)
	}
}
