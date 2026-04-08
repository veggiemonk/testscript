// Copyright 2025 Julien Bisconti. All rights reserved.
// Derived from rsc.io/script and github.com/rogpeppe/go-internal/testscript.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package script

import (
	"runtime"
	"sync/atomic"
	"testing"
)

func newTestState(t *testing.T) *State {
	t.Helper()
	dir := t.TempDir()
	s, err := NewState(t.Context(), dir, []string{"PATH=/usr/bin", "HOME=/tmp"})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestBoolCondition(t *testing.T) {
	c := BoolCondition("always true", true)
	s := newTestState(t)

	ok, err := c.Eval(s, "")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("expected true, got false")
	}

	// Suffix should be rejected.
	_, err = c.Eval(s, "something")
	if err == nil {
		t.Error("expected error for suffix, got nil")
	}
}

func TestBoolConditionFalse(t *testing.T) {
	c := BoolCondition("always false", false)
	s := newTestState(t)

	ok, err := c.Eval(s, "")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("expected false, got true")
	}
}

func TestCachedCondition(t *testing.T) {
	var callCount atomic.Int32
	c := CachedCondition("cached", func(suffix string) (bool, error) {
		callCount.Add(1)
		return suffix == "yes", nil
	})
	s := newTestState(t)

	// First call with "yes" — should evaluate.
	ok, err := c.Eval(s, "yes")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("expected true for 'yes'")
	}
	if n := callCount.Load(); n != 1 {
		t.Errorf("expected 1 call, got %d", n)
	}

	// Second call with same suffix — should use cache.
	ok, err = c.Eval(s, "yes")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("expected true for cached 'yes'")
	}
	if n := callCount.Load(); n != 1 {
		t.Errorf("expected still 1 call, got %d", n)
	}

	// Different suffix — should evaluate again.
	ok, err = c.Eval(s, "no")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("expected false for 'no'")
	}
	if n := callCount.Load(); n != 2 {
		t.Errorf("expected 2 calls, got %d", n)
	}
}

func TestPrefixCondition(t *testing.T) {
	c := PrefixCondition("runtime.GOOS == <suffix>", func(_ *State, suffix string) (bool, error) {
		return suffix == runtime.GOOS, nil
	})
	s := newTestState(t)

	ok, err := c.Eval(s, runtime.GOOS)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Errorf("expected true for GOOS=%s", runtime.GOOS)
	}

	ok, err = c.Eval(s, "plan9")
	if err != nil {
		t.Fatal(err)
	}
	if ok && runtime.GOOS != "plan9" {
		t.Error("expected false for plan9")
	}
}

func TestCondition(t *testing.T) {
	s := newTestState(t)

	c := Condition("always true from state", func(_ *State) (bool, error) {
		return true, nil
	})

	ok, err := c.Eval(s, "")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("expected true, got false")
	}

	// Suffix should be rejected.
	_, err = c.Eval(s, "something")
	if err == nil {
		t.Error("expected error for suffix, got nil")
	}
}

func TestOnceCondition(t *testing.T) {
	var callCount atomic.Int32
	c := OnceCondition("computed once", func() (bool, error) {
		callCount.Add(1)
		return true, nil
	})
	s := newTestState(t)

	// First call should evaluate.
	ok, err := c.Eval(s, "")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("expected true")
	}
	if n := callCount.Load(); n != 1 {
		t.Errorf("expected 1 call, got %d", n)
	}

	// Second call should reuse cached result.
	ok, err = c.Eval(s, "")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("expected true on second call")
	}
	if n := callCount.Load(); n != 1 {
		t.Errorf("expected still 1 call, got %d", n)
	}

	// Suffix should be rejected.
	_, err = c.Eval(s, "something")
	if err == nil {
		t.Error("expected error for suffix, got nil")
	}
}

func TestDefaultConds(t *testing.T) {
	conds := DefaultConds()
	s := newTestState(t)

	// GOOS should exist and match runtime.GOOS.
	goos, ok := conds["GOOS"]
	if !ok {
		t.Fatal("GOOS condition not found")
	}
	match, err := goos.Eval(s, runtime.GOOS)
	if err != nil {
		t.Fatal(err)
	}
	if !match {
		t.Error("GOOS condition did not match runtime.GOOS")
	}

	// GOARCH should exist.
	goarch, ok := conds["GOARCH"]
	if !ok {
		t.Fatal("GOARCH condition not found")
	}
	match, err = goarch.Eval(s, runtime.GOARCH)
	if err != nil {
		t.Fatal(err)
	}
	if !match {
		t.Error("GOARCH condition did not match runtime.GOARCH")
	}

	// root should exist.
	if _, ok := conds["root"]; !ok {
		t.Fatal("root condition not found")
	}
}
