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

func TestComposerCallbackContract(t *testing.T) {
	for _, tc := range []struct {
		name        string
		prepare     func(*textinput.ComposerDriver)
		run         func(*textinput.ComposerDriver) error
		callback    string
		wantCommits []string
	}{
		{
			name:        "commit",
			prepare:     func(d *textinput.ComposerDriver) { d.Send(commitState("text")) },
			callback:    "commit",
			wantCommits: []string{"text"},
		},
		{
			name:        "closed",
			prepare:     func(d *textinput.ComposerDriver) { d.Send(compositionState("text")); d.EndByUser() },
			callback:    "commit",
			wantCommits: []string{"text"},
		},
		{
			name:        "confirm",
			prepare:     func(d *textinput.ComposerDriver) { d.Send(compositionState("text")) },
			run:         func(d *textinput.ComposerDriver) error { d.Composer.Confirm(); return nil },
			callback:    "commit",
			wantCommits: []string{"text"},
		},
		{
			name:     "composition",
			prepare:  func(d *textinput.ComposerDriver) { d.Send(compositionState("text")) },
			callback: "composition",
		},
		{
			name:        "empty composition",
			prepare:     func(d *textinput.ComposerDriver) { d.Send(commitState("text")) },
			callback:    "empty",
			wantCommits: []string{"text"},
		},
		{
			name:     "end by user",
			prepare:  func(d *textinput.ComposerDriver) { d.EndByUser() },
			callback: "end",
		},
		{
			name:     "new session",
			prepare:  func(d *textinput.ComposerDriver) { d.Composer.Cancel() },
			callback: "new",
		},
	} {
		for _, action := range []struct {
			name      string
			call      func(*textinput.Composer) error
			wantError bool
		}{
			{
				name: "Cancel",
				call: func(c *textinput.Composer) error { c.Cancel(); return nil },
			},
			{
				name: "Confirm",
				call: func(c *textinput.Composer) error { c.Confirm(); return nil },
			},
			{
				name:      "Update",
				call:      func(c *textinput.Composer) error { _, err := c.Update(); return err },
				wantError: true,
			},
		} {
			t.Run(tc.name+"/"+action.name, func(t *testing.T) {
				d := textinput.NewComposerDriver("", "")
				tc.prepare(d)
				calls := 0
				invoke := func() {
					calls++
					if calls != 1 {
						return
					}
					err := action.call(&d.Composer)
					if (err != nil) != action.wantError {
						t.Errorf("error = %v, wantError %t", err, action.wantError)
					}
				}
				d.Composer.OnNewSession = func() *textinput.SessionOptions {
					if tc.callback == "new" {
						invoke()
					}
					return nil
				}
				var commits []string
				d.Composer.OnCommit = func(c *textinput.Commit) {
					if d.InputOpen() {
						t.Error("input channel still open during OnCommit")
					}
					commits = append(commits, c.Text())
					if tc.callback == "commit" {
						invoke()
					}
				}
				var compositions []string
				d.Composer.OnComposition = func(c *textinput.Composition) {
					compositions = append(compositions, c.Text())
					if (tc.callback == "composition" && c.Text() != "") || (tc.callback == "empty" && c.Text() == "") {
						invoke()
					}
				}
				d.Composer.OnEndByUser = func() {
					if tc.callback == "end" {
						invoke()
					}
				}
				if tc.run != nil {
					if err := tc.run(d); err != nil {
						t.Fatal(err)
					}
				} else if _, err := d.Composer.Update(); err != nil {
					t.Fatal(err)
				}
				wantCommits := tc.wantCommits
				if tc.callback == "composition" && action.name == "Confirm" {
					wantCommits = []string{"text"}
				}
				if !slices.Equal(commits, wantCommits) {
					t.Errorf("commits = %q, want %q", commits, wantCommits)
				}
				if calls != 1 {
					t.Errorf("callback calls = %d, want 1", calls)
				}
				wantOpen := tc.callback == "composition" && action.wantError
				if d.SessionOpen() != wantOpen {
					t.Errorf("session open = %t, want %t", d.SessionOpen(), wantOpen)
				}
				if !wantOpen && tc.callback != "new" && (len(compositions) == 0 || compositions[len(compositions)-1] != "") {
					t.Errorf("composition not cleared: %q", compositions)
				}
			})
		}
	}
}
