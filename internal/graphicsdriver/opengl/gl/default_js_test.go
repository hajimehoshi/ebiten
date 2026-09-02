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

package gl_test

import (
	"fmt"
	"syscall/js"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl/gl"
)

// fakeWebGLContextSource creates an object that behaves like a WebGL context enough for the tests.
// getUniformLocation returns a new object for each call, as a real WebGL implementation does.
const fakeWebGLContextSource = `(() => {
  const state = {locationCount: 0, uniform1iCalls: []};
  const nop = function () { return null; };
  const ctx = {
    state: state,
    createProgram: function () { return {}; },
    isProgram: function (program) { return program !== null && program !== undefined; },
    getUniformLocation: function (program, name) {
      const location = {index: state.locationCount};
      state.locationCount++;
      return location;
    },
    uniform1i: function (location, v0) {
      state.uniform1iCalls.push(location === null || location === undefined ? -1 : location.index);
    },
  };
  return new Proxy(ctx, {
    get(target, prop) {
      if (prop in target) {
        return target[prop];
      }
      return nop;
    },
  });
})()`

func TestGetUniformLocationManyUniforms(t *testing.T) {
	v := js.Global().Call("eval", fakeWebGLContextSource)
	c, err := gl.NewDefaultContext(v)
	if err != nil {
		t.Fatal(err)
	}

	p := c.CreateProgram()

	const uniformCount = 100
	var locations []int32
	for i := range uniformCount {
		locations = append(locations, c.GetUniformLocation(p, fmt.Sprintf("U%d", i)))
	}
	for _, l := range locations {
		c.Uniform1i(l, 0)
	}

	calls := v.Get("state").Get("uniform1iCalls")
	if got, want := calls.Length(), uniformCount; got != want {
		t.Fatalf("calls.Length(): got: %d, want: %d", got, want)
	}
	for i := range uniformCount {
		if got, want := calls.Index(i).Int(), i; got != want {
			t.Errorf("uniform %d: got location index %d, want %d", i, got, want)
		}
	}
}

func TestGetUniformLocationMultiplePrograms(t *testing.T) {
	v := js.Global().Call("eval", fakeWebGLContextSource)
	c, err := gl.NewDefaultContext(v)
	if err != nil {
		t.Fatal(err)
	}

	const (
		programCount = 3
		uniformCount = 40
	)

	var programs []uint32
	var locations [][]int32
	for range programCount {
		p := c.CreateProgram()
		programs = append(programs, p)
		var ls []int32
		for j := range uniformCount {
			ls = append(ls, c.GetUniformLocation(p, fmt.Sprintf("U%d", j)))
		}
		locations = append(locations, ls)
	}

	// A location ID must be unique in the whole context.
	ids := map[int32]struct{}{}
	for _, ls := range locations {
		for _, l := range ls {
			if _, ok := ids[l]; ok {
				t.Errorf("duplicated location ID: %d", l)
			}
			ids[l] = struct{}{}
		}
	}

	if got, want := gl.UniformLocationCount(c), programCount*uniformCount; got != want {
		t.Errorf("gl.UniformLocationCount(c): got: %d, want: %d", got, want)
	}

	for i, p := range programs {
		c.DeleteProgram(p)
		if got, want := gl.UniformLocationCount(c), (programCount-i-1)*uniformCount; got != want {
			t.Errorf("gl.UniformLocationCount(c) after deleting the program %d: got: %d, want: %d", i, got, want)
		}
	}
}
