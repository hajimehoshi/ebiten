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

func TestComposerReentrantCallbacks(t *testing.T) {
	for _, state := range []string{"commit", "closed", "confirm", "composition", "empty composition"} {
		for _, action := range []string{"Cancel", "Confirm", "Update"} {
			t.Run(state+"/"+action, func(t *testing.T) {
				d := textinput.NewComposerDriver("", "")
				d.Composer.OnNewSession = func() *textinput.SessionOptions { return nil }
				var commits []string
				calls := 0
				reenter := func() {
					calls++
					if calls != 1 {
						return
					}
					switch action {
					case "Cancel":
						d.Composer.Cancel()
					case "Confirm":
						d.Composer.Confirm()
					case "Update":
						if _, err := d.Composer.Update(); err != nil {
							t.Fatal(err)
						}
					}
				}
				d.Composer.OnCommit = func(c *textinput.Commit) {
					commits = append(commits, c.Text())
					if state != "composition" && state != "empty composition" {
						reenter()
					}
				}
				d.Composer.OnComposition = func(c *textinput.Composition) {
					if (state == "composition" && c.Text() != "") || (state == "empty composition" && c.Text() == "") {
						reenter()
					}
				}
				var wantCommits []string
				switch state {
				case "commit", "empty composition":
					d.Send(commitState("にほんご"))
					wantCommits = []string{"にほんご"}
				case "closed":
					d.Send(compositionState("にほんご"))
					d.EndByUser()
					wantCommits = []string{"にほんご"}
				case "confirm":
					d.Send(compositionState("にほんご"))
					wantCommits = []string{"にほんご"}
				case "composition":
					d.Send(compositionState("にほんご"))
					if action == "Confirm" {
						wantCommits = []string{"にほんご"}
					}
				}
				if state == "confirm" {
					d.Composer.Confirm()
				} else if _, err := d.Composer.Update(); err != nil {
					t.Fatal(err)
				}
				if got, want := commits, wantCommits; !slices.Equal(got, want) {
					t.Errorf("commits = %q, want %q", got, want)
				}
				if calls != 1 {
					t.Errorf("callback calls = %d, want 1", calls)
				}
				if got, want := d.SessionOpen(), state == "composition" && action == "Update"; got != want {
					t.Errorf("SessionOpen() = %t, want %t", got, want)
				}
			})
		}
	}
}

func TestComposerEndReplacementDispatchesPendingCommit(t *testing.T) {
	for _, confirm := range []bool{false, true} {
		t.Run(map[bool]string{false: "Cancel", true: "Confirm"}[confirm], func(t *testing.T) {
			d := textinput.NewComposerDriver("", "")
			d.Composer.OnNewSession = func() *textinput.SessionOptions { return nil }
			var commits []string
			d.Composer.OnCommit = func(c *textinput.Commit) {
				commits = append(commits, c.Text())
				if c.Text() != "first" {
					return
				}
				d.StartNextSession()
				if confirm {
					d.Composer.Confirm()
				} else {
					d.Composer.Cancel()
				}
			}
			d.Send(commitState("first"))
			d.Send(commitState("second"))
			if _, err := d.Composer.Update(); err != nil {
				t.Fatal(err)
			}
			if want := []string{"first", "second"}; !slices.Equal(commits, want) {
				t.Errorf("commits = %q, want %q", commits, want)
			}
			if d.SessionOpen() {
				t.Error("replacement session remains open")
			}
		})
	}
}

func TestComposerConfirmPreservesReplacementComposition(t *testing.T) {
	d := textinput.NewComposerDriver("", "")
	d.Composer.OnNewSession = func() *textinput.SessionOptions { return nil }
	var compositions []string
	d.Composer.OnComposition = func(c *textinput.Composition) {
		compositions = append(compositions, c.Text())
	}
	d.Composer.OnCommit = func(c *textinput.Commit) {
		d.StartNextSession()
		d.Send(compositionState("replacement"))
		if _, err := d.Composer.Update(); err != nil {
			t.Fatal(err)
		}
	}
	d.Send(compositionState("old"))
	d.Composer.Confirm()
	if got := compositions[len(compositions)-1]; got != "replacement" {
		t.Errorf("last composition = %q, want replacement", got)
	}
	if !d.SessionOpen() {
		t.Error("replacement session is not open")
	}
}
