// Copyright 2025 Julien Bisconti. All rights reserved.
// Derived from rsc.io/script and github.com/rogpeppe/go-internal/testscript.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package script

import (
	"os"
	"runtime"
	"sync"
)

// DefaultConds returns a set of broadly useful script conditions.
//
// Run the 'help' command within a script engine to view a list of the available
// conditions.
func DefaultConds() map[string]Cond {
	conds := make(map[string]Cond)

	conds["GOOS"] = PrefixCondition(
		"runtime.GOOS == <suffix>",
		func(_ *State, suffix string) (bool, error) {
			if suffix == runtime.GOOS {
				return true, nil
			}
			return false, nil
		})

	conds["GOARCH"] = PrefixCondition(
		"runtime.GOARCH == <suffix>",
		func(_ *State, suffix string) (bool, error) {
			if suffix == runtime.GOARCH {
				return true, nil
			}
			return false, nil
		})

	conds["root"] = BoolCondition("os.Geteuid() == 0", os.Geteuid() == 0)

	return conds
}

// PrefixCondition returns a Cond with the given summary and evaluation function.
func PrefixCondition(summary string, evalFn func(*State, string) (bool, error)) Cond {
	return &prefixCond{evalFn: evalFn, usage: CondUsage{Summary: summary, Prefix: true}}
}

type prefixCond struct {
	evalFn func(*State, string) (bool, error)
	usage  CondUsage
}

func (c *prefixCond) Usage() *CondUsage { return &c.usage }

func (c *prefixCond) Eval(s *State, suffix string) (bool, error) {
	return c.evalFn(s, suffix)
}

// BoolCondition returns a Cond with the given truth value and summary.
// The Cond rejects the use of condition suffixes.
func BoolCondition(summary string, v bool) Cond {
	return &boolCond{v: v, usage: CondUsage{Summary: summary}}
}

type boolCond struct {
	v     bool
	usage CondUsage
}

func (b *boolCond) Usage() *CondUsage { return &b.usage }

func (b *boolCond) Eval(s *State, suffix string) (bool, error) {
	if suffix != "" {
		return false, ErrUsage
	}
	return b.v, nil
}

// CachedCondition is like Condition but only calls evalFn the first time the
// condition is evaluated for a given suffix.
// Future calls with the same suffix reuse the earlier result.
//
// The evalFn function is not passed a *State because the condition is cached
// across all execution states and must not vary by state.
func CachedCondition(summary string, evalFn func(string) (bool, error)) Cond {
	return &cachedCond{evalFn: evalFn, usage: CondUsage{Summary: summary, Prefix: true}}
}

type cachedCond struct {
	m      sync.Map
	evalFn func(string) (bool, error)
	usage  CondUsage
}

func (c *cachedCond) Usage() *CondUsage { return &c.usage }

func (c *cachedCond) Eval(_ *State, suffix string) (bool, error) {
	for {
		var ready chan struct{}

		v, loaded := c.m.Load(suffix)
		if !loaded {
			ready = make(chan struct{})
			v, loaded = c.m.LoadOrStore(suffix, (<-chan struct{})(ready))

			if !loaded {
				inPanic := true
				defer func() {
					if inPanic {
						c.m.Delete(suffix)
					}
					close(ready)
				}()

				b, err := c.evalFn(suffix)
				inPanic = false

				if err == nil {
					c.m.Store(suffix, b)
					return b, nil
				} else {
					c.m.Store(suffix, err)
					return false, err
				}
			}
		}

		switch v := v.(type) {
		case bool:
			return v, nil
		case error:
			return false, v
		case <-chan struct{}:
			<-v
		}
	}
}
