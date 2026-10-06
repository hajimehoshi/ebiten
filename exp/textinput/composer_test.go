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

func TestComposerDispatchesClearedComposition(t *testing.T) {
	d := textinput.NewComposerDriver("", "")
	var compositions []string
	d.Composer.OnComposition = func(c *textinput.Composition) {
		compositions = append(compositions, c.Text())
	}

	d.Send(compositionState("にほ"))
	if _, err := d.Composer.Update(); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got, want := compositions, []string{"にほ"}; !slices.Equal(got, want) {
		t.Fatalf("compositions after Update = %q, want %q", got, want)
	}

	d.Send(compositionState(""))
	handled, err := d.Composer.Update()
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !handled {
		t.Error("Update() = false, want true")
	}
	if got, want := compositions, []string{"にほ", ""}; !slices.Equal(got, want) {
		t.Errorf("compositions after Update = %q, want %q", got, want)
	}
	if !d.SessionOpen() {
		t.Error("SessionOpen() = false, want true")
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

func TestComposerConfirmFromCallbacks(t *testing.T) {
	for _, tc := range []struct {
		name        string
		prepare     func(*textinput.ComposerDriver)
		onCommit    bool
		confirm     bool
		wantCommits []string
	}{
		{
			name:        "commit",
			prepare:     func(d *textinput.ComposerDriver) { d.Send(compositionState("te")); d.Send(commitState("text")) },
			onCommit:    true,
			wantCommits: []string{"text"},
		},
		{
			name:        "closed",
			prepare:     func(d *textinput.ComposerDriver) { d.Send(compositionState("text")); d.EndByUser() },
			onCommit:    true,
			wantCommits: []string{"text"},
		},
		{
			name:        "confirm",
			prepare:     func(d *textinput.ComposerDriver) { d.Send(compositionState("text")) },
			onCommit:    true,
			confirm:     true,
			wantCommits: []string{"text"},
		},
		{
			name:        "composition",
			prepare:     func(d *textinput.ComposerDriver) { d.Send(compositionState("text")) },
			wantCommits: []string{"text"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := textinput.NewComposerDriver("", "")
			d.Composer.OnNewSession = func() *textinput.SessionOptions { return nil }
			tc.prepare(d)
			calls := 0
			invoke := func() {
				calls++
				if calls == 1 {
					d.Composer.Confirm()
				}
			}
			var commits, compositions []string
			d.Composer.OnCommit = func(c *textinput.Commit) {
				commits = append(commits, c.Text())
				if tc.onCommit {
					invoke()
				}
			}
			d.Composer.OnComposition = func(c *textinput.Composition) {
				compositions = append(compositions, c.Text())
				if !tc.onCommit && c.Text() != "" {
					invoke()
				}
			}
			if tc.confirm {
				d.Composer.Confirm()
			} else if _, err := d.Composer.Update(); err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(commits, tc.wantCommits) {
				t.Errorf("commits = %q, want %q", commits, tc.wantCommits)
			}
			if calls != 1 {
				t.Errorf("callback calls = %d, want 1", calls)
			}
			if d.SessionOpen() {
				t.Error("session remains open")
			}
			if len(compositions) == 0 || compositions[len(compositions)-1] != "" {
				t.Errorf("composition not cleared: %q", compositions)
			}
		})
	}
}

func TestComposerUpdateFromCallbacks(t *testing.T) {
	for _, tc := range []struct {
		name    string
		setup   func(*textinput.ComposerDriver, func())
		confirm bool
	}{
		{
			name: "commit",
			setup: func(d *textinput.ComposerDriver, invoke func()) {
				d.Send(commitState("text"))
				d.Composer.OnCommit = func(*textinput.Commit) { invoke() }
			},
		},
		{
			name: "composition",
			setup: func(d *textinput.ComposerDriver, invoke func()) {
				d.Send(compositionState("text"))
				d.Composer.OnComposition = func(*textinput.Composition) { invoke() }
			},
		},
		{
			name: "end by user",
			setup: func(d *textinput.ComposerDriver, invoke func()) {
				d.EndByUser()
				d.Composer.OnEndByUser = invoke
			},
		},
		{
			name: "new session",
			setup: func(d *textinput.ComposerDriver, invoke func()) {
				d.Composer.Cancel()
				d.Composer.OnNewSession = func() *textinput.SessionOptions { invoke(); return nil }
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := textinput.NewComposerDriver("", "")
			d.Composer.OnNewSession = func() *textinput.SessionOptions { return nil }
			calls := 0
			tc.setup(d, func() {
				calls++
				if calls > 1 {
					t.Error("callback called more than once")
					return
				}
				open := d.SessionOpen()
				handled, err := d.Composer.Update()
				if err == nil {
					t.Error("callback Update returned nil error")
				}
				if handled {
					t.Error("callback Update handled input")
				}
				if d.SessionOpen() != open {
					t.Error("callback Update changed session")
				}
			})
			if tc.confirm {
				d.Composer.Confirm()
			} else if _, err := d.Composer.Update(); err != nil {
				t.Fatal(err)
			}
			if calls != 1 {
				t.Errorf("callback calls = %d, want 1", calls)
			}
		})
	}
}
