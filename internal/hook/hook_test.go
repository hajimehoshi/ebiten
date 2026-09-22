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

package hook_test

import (
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2/internal/hook"
)

func runWithTimeout(t *testing.T, name string, f func() error) {
	t.Helper()
	done := make(chan error, 1)
	go func() {
		done <- f()
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("%s: %v", name, err)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("%s did not return", name)
	}
}

func TestRunBeforeUpdateHooksReentrantAppend(t *testing.T) {
	hook.AppendHookOnBeforeUpdate(func() error {
		hook.AppendHookOnBeforeUpdate(func() error {
			return nil
		})
		return nil
	})
	runWithTimeout(t, "RunBeforeUpdateHooks", hook.RunBeforeUpdateHooks)
}

func TestRunBeforeUpdateHooksWithVMGuestInfoReentrantAppend(t *testing.T) {
	hook.AppendHookOnBeforeUpdateWithVMGuestInfo(func(vmGuest bool) error {
		hook.AppendHookOnBeforeUpdateWithVMGuestInfo(func(vmGuest bool) error {
			return nil
		})
		return nil
	})
	runWithTimeout(t, "RunBeforeUpdateHooksWithVMGuestInfo", func() error {
		return hook.RunBeforeUpdateHooksWithVMGuestInfo(true)
	})
}

func TestSuspendAudioReentrantOnSuspendAudio(t *testing.T) {
	runWithTimeout(t, "ResumeAudio", hook.ResumeAudio)
	hook.OnSuspendAudio(func() error {
		hook.OnSuspendAudio(func() error {
			return nil
		})
		return nil
	})
	runWithTimeout(t, "SuspendAudio", hook.SuspendAudio)
}

func TestResumeAudioReentrantOnResumeAudio(t *testing.T) {
	runWithTimeout(t, "SuspendAudio", hook.SuspendAudio)
	hook.OnResumeAudio(func() error {
		hook.OnResumeAudio(func() error {
			return nil
		})
		return nil
	})
	runWithTimeout(t, "ResumeAudio", hook.ResumeAudio)
}
