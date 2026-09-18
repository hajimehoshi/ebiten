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
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/exp/textinput"
)

func compositionState(text string) textinput.TextInputState {
	return textinput.TextInputState{
		Text:                             text,
		CompositionSelectionStartInBytes: len(text),
		CompositionSelectionEndInBytes:   len(text),
	}
}

func TestComposerEndDispatchesPendingStates(t *testing.T) {
	for _, tc := range []struct {
		name        string
		pending     textinput.TextInputState
		confirm     bool
		wantCommits []string
	}{
		{
			name:        "commit then Confirm",
			pending:     commitState("にほんご"),
			confirm:     true,
			wantCommits: []string{"にほんご"},
		},
		{
			name:        "commit then Cancel",
			pending:     commitState("にほんご"),
			confirm:     false,
			wantCommits: []string{"にほんご"},
		},
		{
			name:        "composition then Confirm",
			pending:     compositionState("にほんご"),
			confirm:     true,
			wantCommits: []string{"にほんご"},
		},
		{
			name:        "composition then Cancel",
			pending:     compositionState("にほんご"),
			confirm:     false,
			wantCommits: nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := textinput.NewComposerDriver("", "")
			var commits []string
			d.Composer.OnCommit = func(c *textinput.Commit) {
				commits = append(commits, c.Text())
			}
			var compositions []string
			d.Composer.OnComposition = func(c *textinput.Composition) {
				compositions = append(compositions, c.Text())
			}

			d.Send(compositionState("にほ"))
			handled, err := d.Composer.Update()
			if err != nil {
				t.Fatalf("Update: %v", err)
			}
			if !handled {
				t.Fatal("Update() = false, want true")
			}
			if got, want := compositions, []string{"にほ"}; !slices.Equal(got, want) {
				t.Fatalf("compositions after Update = %q, want %q", got, want)
			}

			d.Send(tc.pending)
			if tc.confirm {
				d.Composer.Confirm()
			} else {
				d.Composer.Cancel()
			}

			if got, want := commits, tc.wantCommits; !slices.Equal(got, want) {
				t.Errorf("commits = %q, want %q", got, want)
			}
			if got, want := compositions[len(compositions)-1], ""; got != want {
				t.Errorf("last composition = %q, want %q", got, want)
			}
			if d.SessionOpen() {
				t.Error("SessionOpen() = true, want false")
			}
		})
	}
}

func TestComposerEndDispatchesUserEnding(t *testing.T) {
	for _, tc := range []struct {
		name    string
		confirm bool
	}{
		{
			name:    "Confirm",
			confirm: true,
		},
		{
			name:    "Cancel",
			confirm: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := textinput.NewComposerDriver("", "")
			var commits []string
			d.Composer.OnCommit = func(c *textinput.Commit) {
				commits = append(commits, c.Text())
			}
			var endedByUser int
			d.Composer.OnEndByUser = func() {
				endedByUser++
			}

			d.Send(compositionState("にほ"))
			if _, err := d.Composer.Update(); err != nil {
				t.Fatalf("Update: %v", err)
			}

			d.EndByUser()
			if tc.confirm {
				d.Composer.Confirm()
			} else {
				d.Composer.Cancel()
			}

			if got, want := commits, []string{"にほ"}; !slices.Equal(got, want) {
				t.Errorf("commits = %q, want %q", got, want)
			}
			if got, want := endedByUser, 1; got != want {
				t.Errorf("OnEndByUser calls = %d, want %d", got, want)
			}
			if d.SessionOpen() {
				t.Error("SessionOpen() = true, want false")
			}
		})
	}
}
