// Copyright 2025 Julien Bisconti. All rights reserved.
// Derived from rsc.io/script and github.com/rogpeppe/go-internal/testscript.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package diff

import (
	"bytes"
	"path/filepath"
	"testing"

	"golang.org/x/tools/txtar"
)

func clean(text []byte) []byte {
	text = bytes.ReplaceAll(text, []byte("$\n"), []byte("\n"))
	text = bytes.TrimSuffix(text, []byte("^D\n"))
	return text
}

// FuzzDiff verifies that Diff never panics on arbitrary input and that
// identical inputs always produce nil output.
func FuzzDiff(f *testing.F) {
	f.Add([]byte("hello\n"), []byte("world\n"))
	f.Add([]byte(""), []byte(""))
	f.Add([]byte("same\n"), []byte("same\n"))
	f.Add([]byte("line1\nline2\n"), []byte("line1\nline3\n"))
	f.Add([]byte("a\nb\nc\n"), []byte("a\nc\n"))
	f.Add([]byte("a\n"), []byte("a\nb\n"))

	f.Fuzz(func(t *testing.T, old, new []byte) {
		result := Diff("old", old, "new", new)

		// Identical inputs must produce nil.
		if string(old) == string(new) && result != nil {
			t.Errorf("Diff of identical inputs produced non-nil output: %q", result)
		}
	})
}

func Test(t *testing.T) {
	files, _ := filepath.Glob("testdata/*.txt")
	if len(files) == 0 {
		t.Fatalf("no testdata")
	}

	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			a, err := txtar.ParseFile(file)
			if err != nil {
				t.Fatal(err)
			}
			if len(a.Files) != 3 || a.Files[2].Name != "diff" {
				t.Fatalf("%s: want three files, third named \"diff\"", file)
			}
			diffs := Diff(a.Files[0].Name, clean(a.Files[0].Data), a.Files[1].Name, clean(a.Files[1].Data))
			want := clean(a.Files[2].Data)
			if !bytes.Equal(diffs, want) {
				t.Fatalf("%s: have:\n%s\nwant:\n%s\n%s", file,
					diffs, want, Diff("have", diffs, "want", want))
			}
		})
	}
}
