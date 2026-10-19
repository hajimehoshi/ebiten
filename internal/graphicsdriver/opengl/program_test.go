// Copyright 2026 The Ebitengine Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build !playstation5

package opengl

import (
	"testing"
)

func TestUniformCacheKeepsValuesForEachProgram(t *testing.T) {
	var c uniformCache

	c.use(1)
	c.set(uniformSlot(0), []uint32{1, 2})
	c.use(2)
	if c.isSame(uniformSlot(0), []uint32{1, 2}) {
		t.Errorf("program 2 must not have the value of program 1")
	}
	c.set(uniformSlot(0), []uint32{3, 4})

	c.use(1)
	if !c.isSame(uniformSlot(0), []uint32{1, 2}) {
		t.Errorf("program 1 must keep its value after a switch of programs")
	}
	if c.isSame(uniformSlot(0), []uint32{3, 4}) {
		t.Errorf("program 1 must not have the value of program 2")
	}
}

func TestUniformCacheOwnsValues(t *testing.T) {
	var c uniformCache
	c.use(1)

	v := []uint32{1, 2}
	c.set(uniformSlot(0), v)
	// The caller reuses its slice for the next values.
	v[0] = 5
	if !c.isSame(uniformSlot(0), []uint32{1, 2}) {
		t.Errorf("the cache must keep a copy of the value")
	}
	if c.isSame(uniformSlot(0), v) {
		t.Errorf("the cache must not follow a change of the caller's slice")
	}
}

// Issue #2517
func TestUniformCacheInvalidate(t *testing.T) {
	var c uniformCache
	c.use(1)
	c.set(uniformSlot(0), []uint32{1})
	c.set(samplerSlot(0), []uint32{0})
	c.use(2)
	c.set(uniformSlot(0), []uint32{2})

	// The values must be sent again after each present.
	c.invalidate()
	if c.isSame(uniformSlot(0), []uint32{2}) {
		t.Errorf("program 2: the value must not be the same after invalidate")
	}
	c.use(1)
	if c.isSame(uniformSlot(0), []uint32{1}) {
		t.Errorf("program 1: the value must not be the same after invalidate")
	}
	if c.isSame(samplerSlot(0), []uint32{0}) {
		t.Errorf("program 1: the sampler value must not be the same after invalidate")
	}

	c.set(uniformSlot(0), []uint32{1})
	if !c.isSame(uniformSlot(0), []uint32{1}) {
		t.Errorf("program 1: the value must be the same after set")
	}
}

func TestUniformCacheInvalidateAllocs(t *testing.T) {
	var c uniformCache
	c.use(1)
	v := []uint32{1}
	c.set(uniformSlot(0), v)
	if n := testing.AllocsPerRun(10, func() {
		c.invalidate()
		c.set(uniformSlot(0), v)
	}); n != 0 {
		t.Errorf("allocations: got: %v, want: 0", n)
	}
}

func TestUniformCacheDeleteProgram(t *testing.T) {
	var c uniformCache
	c.use(1)
	c.set(uniformSlot(0), []uint32{1})
	c.use(2)
	c.set(uniformSlot(0), []uint32{2})

	// A new program can reuse the name of a deleted program.
	c.deleteProgram(1)
	c.use(1)
	if c.isSame(uniformSlot(0), []uint32{1}) {
		t.Errorf("a program with the name of a deleted program must not have its values")
	}

	// Delete the current program.
	c.set(uniformSlot(0), []uint32{1})
	c.deleteProgram(1)
	if c.isSame(uniformSlot(0), []uint32{1}) {
		t.Errorf("the deleted current program must not have its values")
	}

	c.use(2)
	if !c.isSame(uniformSlot(0), []uint32{2}) {
		t.Errorf("program 2 must keep its value after program 1 is deleted")
	}
}

func TestUniformCacheClear(t *testing.T) {
	var c uniformCache
	c.use(1)
	c.set(uniformSlot(0), []uint32{1})

	// A new context has no programs.
	c.clear()
	if c.isSame(uniformSlot(0), []uint32{1}) {
		t.Errorf("the value must not be the same after clear")
	}
	c.use(1)
	if c.isSame(uniformSlot(0), []uint32{1}) {
		t.Errorf("program 1: the value must not be the same after clear")
	}
}
