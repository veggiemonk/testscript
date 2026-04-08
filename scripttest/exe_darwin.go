// Copyright 2025 Julien Bisconti. All rights reserved.
// Derived from rsc.io/script and github.com/rogpeppe/go-internal/testscript.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build darwin

package scripttest

import "golang.org/x/sys/unix"

// cloneFile makes a clone of a file via macOS's clonefile syscall.
func cloneFile(from, to string) error {
	return unix.Clonefile(from, to, 0)
}
