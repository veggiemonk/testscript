// Copyright 2025 Julien Bisconti. All rights reserved.
// Derived from rsc.io/script and github.com/rogpeppe/go-internal/testscript.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package script

import (
	"io"
	"os"
)

// CopyBinary makes a copy of a binary to a new location.
// It tries cloneFile first (hard link on Linux, clonefile on macOS),
// falling back to a full copy.
//
// It does not use symlinks because tools like cmd/go's -toolexec
// dereference symlinks and use the target for executing the program.
func CopyBinary(from, to string) error {
	if err := cloneFile(from, to); err == nil {
		return nil
	}
	writer, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE, 0o777)
	if err != nil {
		return err
	}
	defer writer.Close()

	reader, err := os.Open(from)
	if err != nil {
		return err
	}
	defer reader.Close()

	_, err = io.Copy(writer, reader)
	return err
}
