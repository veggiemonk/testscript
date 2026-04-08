// Copyright 2025 Julien Bisconti. All rights reserved.
// Derived from rsc.io/script and github.com/rogpeppe/go-internal/testscript.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package script

import (
	"errors"
	"io/fs"
	"testing"
)

func TestWaitErrorUnwrapSingle(t *testing.T) {
	inner := &CommandError{File: "test.txt", Line: 1, Op: "exec", Err: fs.ErrNotExist}
	w := waitError{errs: []*CommandError{inner}}

	if !errors.Is(w, fs.ErrNotExist) {
		t.Error("errors.Is failed to find wrapped fs.ErrNotExist in single-error waitError")
	}

	var ce *CommandError
	if !errors.As(w, &ce) {
		t.Error("errors.As failed to find *CommandError in single-error waitError")
	}
}

func TestWaitErrorUnwrapMultiple(t *testing.T) {
	err1 := &CommandError{File: "test.txt", Line: 1, Op: "exec", Err: fs.ErrNotExist}
	err2 := &CommandError{File: "test.txt", Line: 2, Op: "cat", Err: fs.ErrPermission}
	w := waitError{errs: []*CommandError{err1, err2}}

	// Both wrapped errors should be reachable.
	if !errors.Is(w, fs.ErrNotExist) {
		t.Error("errors.Is failed to find fs.ErrNotExist in multi-error waitError")
	}
	if !errors.Is(w, fs.ErrPermission) {
		t.Error("errors.Is failed to find fs.ErrPermission in multi-error waitError")
	}

	var ce *CommandError
	if !errors.As(w, &ce) {
		t.Error("errors.As failed to find *CommandError in multi-error waitError")
	}
}
