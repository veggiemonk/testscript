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
			return suffix == runtime.GOOS, nil
		})

	conds["GOARCH"] = PrefixCondition(
		"runtime.GOARCH == <suffix>",
		func(_ *State, suffix string) (bool, error) {
			return suffix == runtime.GOARCH, nil
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

// Condition returns a Cond with the given summary and evaluation function.
// The Cond rejects the use of condition suffixes.
// For conditions that accept a suffix, use PrefixCondition.
func Condition(summary string, evalFn func(*State) (bool, error)) Cond {
	return &simpleCond{evalFn: evalFn, usage: CondUsage{Summary: summary}}
}

type simpleCond struct {
	evalFn func(*State) (bool, error)
	usage  CondUsage
}

func (c *simpleCond) Usage() *CondUsage { return &c.usage }

func (c *simpleCond) Eval(s *State, suffix string) (bool, error) {
	if suffix != "" {
		return false, ErrUsage
	}
	return c.evalFn(s)
}

// BoolCondition returns a Cond with the given truth value and summary.
// The Cond rejects the use of condition suffixes.
func BoolCondition(summary string, v bool) Cond {
	return Condition(summary, func(_ *State) (bool, error) { return v, nil })
}

// OnceCondition returns a Cond that calls fn the first time the condition
// is evaluated. Future calls reuse the same result. The Cond rejects suffixes.
//
// The fn function is not passed a *State because the condition is cached
// across all execution states and must not vary by state.
func OnceCondition(summary string, fn func() (bool, error)) Cond {
	return &onceCond{fn: fn, usage: CondUsage{Summary: summary}}
}

type onceCond struct {
	once  sync.Once
	v     bool
	err   error
	fn    func() (bool, error)
	usage CondUsage
}

func (c *onceCond) Usage() *CondUsage { return &c.usage }

func (c *onceCond) Eval(_ *State, suffix string) (bool, error) {
	if suffix != "" {
		return false, ErrUsage
	}
	c.once.Do(func() { c.v, c.err = c.fn() })
	return c.v, c.err
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
