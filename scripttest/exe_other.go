// Copyright 2025 Julien Bisconti. All rights reserved.
// Derived from rsc.io/script and github.com/rogpeppe/go-internal/testscript.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !darwin && !linux

package scripttest

import "errors"

// cloneFile is not supported on this platform; copyBinary will fall back to a full copy.
func cloneFile(_, _ string) error {
	return errors.New("cloneFile not supported on this platform")
}
