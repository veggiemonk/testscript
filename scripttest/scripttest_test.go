// Copyright 2025 Julien Bisconti. All rights reserved.
// Derived from rsc.io/script and github.com/rogpeppe/go-internal/testscript.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scripttest_test

import (
	"context"
	"os"
	"testing"

	"github.com/veggiemonk/testscript/scripttest"
)

func TestScripts(t *testing.T) {
	engine := scripttest.DefaultEngine()
	scripttest.Test(t, context.Background(), engine, os.Environ(), "testdata/*.txt")
}
