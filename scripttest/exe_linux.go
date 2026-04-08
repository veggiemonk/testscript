// Copyright 2025 Julien Bisconti. All rights reserved.
// Derived from rsc.io/script and github.com/rogpeppe/go-internal/testscript.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build linux

package scripttest

import "os"

// cloneFile makes a clone of a file via a hard link.
func cloneFile(from, to string) error {
	return os.Link(from, to)
}
