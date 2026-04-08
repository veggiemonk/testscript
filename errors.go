// Copyright 2025 Julien Bisconti. All rights reserved.
// Derived from rsc.io/script and github.com/rogpeppe/go-internal/testscript.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package script

import (
	"errors"
	"fmt"
	"strings"
)

// ErrUnexpectedSuccess indicates that a script command that was expected to
// fail (as indicated by a "!" prefix) instead completed successfully.
var ErrUnexpectedSuccess = errors.New("unexpected success")

// A CommandError describes an error resulting from attempting to execute a
// specific command.
type CommandError struct {
	File string
	Line int
	Op   string
	Args []string
	Err  error
}

func cmdError(cmd *command, err error) *CommandError {
	return &CommandError{
		File: cmd.file,
		Line: cmd.line,
		Op:   cmd.name,
		Args: cmd.args,
		Err:  err,
	}
}

func (e *CommandError) Error() string {
	if len(e.Args) == 0 {
		return fmt.Sprintf("%s:%d: %s: %v", e.File, e.Line, e.Op, e.Err)
	}
	return fmt.Sprintf("%s:%d: %s %s: %v", e.File, e.Line, e.Op, quoteArgs(e.Args), e.Err)
}

func (e *CommandError) Unwrap() error { return e.Err }

// A UsageError reports the valid arguments for a command.
//
// It may be returned in response to invalid arguments.
type UsageError struct {
	Name    string
	Command Cmd
}

func (e *UsageError) Error() string {
	usage := e.Command.Usage()
	suffix := ""
	if usage.Async {
		suffix = " [&]"
	}
	return fmt.Sprintf("usage: %s %s%s", e.Name, usage.Args, suffix)
}

// ErrUsage may be returned by a Command to indicate that it was called with
// invalid arguments; its Usage method may be called to obtain details.
var ErrUsage = errors.New("invalid usage")

// stopError is the sentinel error type returned by the Stop command.
type stopError struct {
	msg string
}

func (s stopError) Error() string {
	if s.msg == "" {
		return "stop"
	}
	return "stop: " + s.msg
}

// A waitError wraps one or more errors returned by background commands.
type waitError struct {
	errs []*CommandError
}

func (w waitError) Error() string {
	b := new(strings.Builder)
	for i, err := range w.errs {
		if i != 0 {
			b.WriteString("\n")
		}
		b.WriteString(err.Error())
	}
	return b.String()
}

func (w waitError) Unwrap() []error {
	errs := make([]error, len(w.errs))
	for i, e := range w.errs {
		errs[i] = e
	}
	return errs
}

// A SkipError indicates that a script invoked the "skip" command.
// It is used by the scripttest package to translate script skips into
// testing.T.Skip calls.
type SkipError struct {
	msg string
}

func (s SkipError) Error() string {
	if s.msg == "" {
		return "skip"
	}
	return s.msg
}

// MakeSkipError creates a new SkipError with the given message.
func MakeSkipError(msg string) error {
	return SkipError{msg: msg}
}
