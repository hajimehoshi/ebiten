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
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2/internal/hook"
)

func callWithTimeout(t *testing.T, name string, f func() error) error {
	t.Helper()
	done := make(chan error, 1)
	go func() {
		done <- f()
	}()
	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
		t.Fatalf("%s did not return", name)
	}
	return nil
}

func runWithTimeout(t *testing.T, name string, f func() error) {
	t.Helper()
	if err := callWithTimeout(t, name, f); err != nil {
		t.Errorf("%s: %v", name, err)
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

var errHookFailed = errors.New("hook_test: hook failed")

func TestSuspendAudioRetriesAfterError(t *testing.T) {
	hook.OnResumeAudio(func() error {
		return nil
	})
	runWithTimeout(t, "ResumeAudio", hook.ResumeAudio)

	var suspendCalls, resumeCalls int
	hook.OnSuspendAudio(func() error {
		suspendCalls++
		if suspendCalls == 1 {
			return errHookFailed
		}
		return nil
	})
	hook.OnResumeAudio(func() error {
		resumeCalls++
		return nil
	})

	if err := callWithTimeout(t, "SuspendAudio", hook.SuspendAudio); !errors.Is(err, errHookFailed) {
		t.Errorf("SuspendAudio: got: %v, want: %v", err, errHookFailed)
	}
	runWithTimeout(t, "SuspendAudio", hook.SuspendAudio)
	if got, want := suspendCalls, 2; got != want {
		t.Errorf("suspend hook calls: got: %d, want: %d", got, want)
	}
	runWithTimeout(t, "ResumeAudio", hook.ResumeAudio)
	if got, want := resumeCalls, 1; got != want {
		t.Errorf("resume hook calls: got: %d, want: %d", got, want)
	}
}

func TestResumeAudioRetriesAfterError(t *testing.T) {
	hook.OnSuspendAudio(func() error {
		return nil
	})
	runWithTimeout(t, "SuspendAudio", hook.SuspendAudio)

	var suspendCalls, resumeCalls int
	hook.OnResumeAudio(func() error {
		resumeCalls++
		if resumeCalls == 1 {
			return errHookFailed
		}
		return nil
	})
	hook.OnSuspendAudio(func() error {
		suspendCalls++
		return nil
	})

	if err := callWithTimeout(t, "ResumeAudio", hook.ResumeAudio); !errors.Is(err, errHookFailed) {
		t.Errorf("ResumeAudio: got: %v, want: %v", err, errHookFailed)
	}
	runWithTimeout(t, "ResumeAudio", hook.ResumeAudio)
	if got, want := resumeCalls, 2; got != want {
		t.Errorf("resume hook calls: got: %d, want: %d", got, want)
	}
	runWithTimeout(t, "SuspendAudio", hook.SuspendAudio)
	if got, want := suspendCalls, 1; got != want {
		t.Errorf("suspend hook calls: got: %d, want: %d", got, want)
	}
}

func TestSuspendAudioConcurrentCallsRunHookOnce(t *testing.T) {
	hook.OnResumeAudio(func() error {
		return nil
	})
	runWithTimeout(t, "ResumeAudio", hook.ResumeAudio)

	var calls atomic.Int32
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	hook.OnSuspendAudio(func() error {
		calls.Add(1)
		entered <- struct{}{}
		<-release
		return nil
	})

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for range 2 {
		wg.Go(func() {
			errs <- hook.SuspendAudio()
		})
	}
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("the suspend hook did not run")
	}
	close(release)
	runWithTimeout(t, "SuspendAudio", func() error {
		wg.Wait()
		return nil
	})
	close(errs)
	for err := range errs {
		if err != nil {
			t.Errorf("SuspendAudio: %v", err)
		}
	}
	if got, want := calls.Load(), int32(1); got != want {
		t.Errorf("suspend hook calls: got: %d, want: %d", got, want)
	}
}
