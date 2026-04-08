// Copyright 2025 Julien Bisconti. All rights reserved.
// Derived from rsc.io/script and github.com/rogpeppe/go-internal/testscript.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scripttest_test

import (
	"testing"

	"github.com/veggiemonk/testscript/scripttest"
)

func TestScripts(t *testing.T) {
	scripttest.Test(t, "testdata/*.txt")
}
