// Copyright 2026 The Ebiten Authors
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

package hook

import (
	"errors"
	"testing"
)

func TestAudioHookErrorDoesNotChangeState(t *testing.T) {
	m.Lock()
	oldAudioSuspended := audioSuspended
	oldOnSuspendAudio := onSuspendAudio
	oldOnResumeAudio := onResumeAudio
	audioSuspended = false
	onSuspendAudio = nil
	onResumeAudio = nil
	m.Unlock()
	t.Cleanup(func() {
		m.Lock()
		audioSuspended = oldAudioSuspended
		onSuspendAudio = oldOnSuspendAudio
		onResumeAudio = oldOnResumeAudio
		m.Unlock()
	})

	errHook := errors.New("hook failed")
	suspendCalls := 0
	resumeCalls := 0
	OnSuspendAudio(func() error {
		suspendCalls++
		if suspendCalls == 1 {
			return errHook
		}
		return nil
	})
	OnResumeAudio(func() error {
		resumeCalls++
		if resumeCalls == 1 {
			return errHook
		}
		return nil
	})

	if err := SuspendAudio(); !errors.Is(err, errHook) {
		t.Errorf("SuspendAudio() error = %v, want %v", err, errHook)
	}
	if err := ResumeAudio(); err != nil {
		t.Errorf("ResumeAudio() after failed suspend: %v", err)
	}
	if resumeCalls != 0 {
		t.Errorf("resume hook calls after failed suspend = %d, want 0", resumeCalls)
	}
	if err := SuspendAudio(); err != nil {
		t.Errorf("retrying SuspendAudio(): %v", err)
	}
	if suspendCalls != 2 {
		t.Errorf("suspend hook calls = %d, want 2", suspendCalls)
	}

	if err := ResumeAudio(); !errors.Is(err, errHook) {
		t.Errorf("ResumeAudio() error = %v, want %v", err, errHook)
	}
	if err := SuspendAudio(); err != nil {
		t.Errorf("SuspendAudio() after failed resume: %v", err)
	}
	if suspendCalls != 2 {
		t.Errorf("suspend hook calls after failed resume = %d, want 2", suspendCalls)
	}
	if err := ResumeAudio(); err != nil {
		t.Errorf("retrying ResumeAudio(): %v", err)
	}
	if resumeCalls != 2 {
		t.Errorf("resume hook calls = %d, want 2", resumeCalls)
	}
}
