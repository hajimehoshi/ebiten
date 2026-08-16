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

package textinput

// seedGate arbitrates asynchronous reseeding of a platform text buffer. Start
// posts a seed the platform applies later, and every state the platform
// reports carries the generation of the seed its buffer was on. Until the new
// seed is applied, the buffer still holds the previous target, so states of
// the previous generation are genuine edits and stay accepted; states of an
// abandoned target are dropped.
type seedGate struct {
	generation    int
	abandoned     int
	pendingValue  string
	baselineReset bool
}

// start records value as the next seed and returns its generation.
func (g *seedGate) start(value string) int {
	g.generation++
	g.pendingValue = value
	g.baselineReset = false
	return g.generation
}

// abandon marks the current generation's target abandoned, dropping its
// states from now on.
func (g *seedGate) abandon() {
	g.abandoned = g.generation
}

// admit reports whether the caller must reset the diff baseline to the
// pending seed, and whether a state of generation is delivered. The first
// admitted state of the current generation is the first one reported for the
// applied seed, so the baseline switches there.
func (g *seedGate) admit(generation int) (resetBaseline bool, ok bool) {
	if generation <= g.abandoned || generation > g.generation {
		return false, false
	}
	if generation < g.generation {
		// The platform has not applied the latest seed yet; the state
		// continues the previous target.
		return false, true
	}
	if !g.baselineReset {
		g.baselineReset = true
		return true, true
	}
	return false, true
}
