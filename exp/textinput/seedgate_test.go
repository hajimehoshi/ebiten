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

package textinput_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2/exp/textinput"
)

func TestSeedGateAdmitsCurrentGeneration(t *testing.T) {
	var g textinput.SeedGate
	gen := g.Start("abc")

	// The first admitted state of the applied seed switches the baseline.
	reset, ok := g.Admit(gen)
	if !ok || !reset {
		t.Errorf("Admit(%d) = %t, %t, want true, true", gen, ok, reset)
	}
	if got, want := g.PendingValue(), "abc"; got != want {
		t.Errorf("PendingValue() = %q, want %q", got, want)
	}

	// Later states of the same generation do not switch it again.
	reset, ok = g.Admit(gen)
	if !ok || reset {
		t.Errorf("Admit(%d) = %t, %t, want true, false", gen, ok, reset)
	}
}

// TestSeedGateAdmitsPreviousGenerationUntilApplied simulates rapid text input:
// a state of the previous generation reported after Start but before the
// platform applies the new seed is a genuine edit of the previous target and
// must stay accepted, without switching the baseline.
func TestSeedGateAdmitsPreviousGenerationUntilApplied(t *testing.T) {
	var g textinput.SeedGate
	gen1 := g.Start("abc")
	if _, ok := g.Admit(gen1); !ok {
		t.Fatalf("Admit(%d) = false, want true", gen1)
	}

	// A commit restarts the session; the next seed is posted but not applied.
	gen2 := g.Start("abcd")

	// The rapid follow-up edit still carries the previous generation.
	reset, ok := g.Admit(gen1)
	if !ok || reset {
		t.Errorf("Admit(%d) = %t, %t, want true, false", gen1, ok, reset)
	}

	// The first state of the applied seed switches the baseline.
	reset, ok = g.Admit(gen2)
	if !ok || !reset {
		t.Errorf("Admit(%d) = %t, %t, want true, true", gen2, ok, reset)
	}
}

func TestSeedGateDropsAbandonedGenerations(t *testing.T) {
	var g textinput.SeedGate
	gen1 := g.Start("abc")
	g.Abandon()

	// States of the abandoned target are dropped, even from before newer
	// starts.
	if _, ok := g.Admit(gen1); ok {
		t.Errorf("Admit(%d) = true, want false", gen1)
	}
	gen2 := g.Start("def")
	if _, ok := g.Admit(gen1); ok {
		t.Errorf("Admit(%d) = true, want false", gen1)
	}

	// The next target's states are admitted.
	if reset, ok := g.Admit(gen2); !ok || !reset {
		t.Errorf("Admit(%d) = %t, %t, want true, true", gen2, ok, reset)
	}
}

func TestSeedGateDropsUnknownFutureGeneration(t *testing.T) {
	var g textinput.SeedGate
	gen := g.Start("abc")
	if _, ok := g.Admit(gen + 1); ok {
		t.Errorf("Admit(%d) = true, want false", gen+1)
	}
}
