// Copyright 2018 The Ebiten Authors
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
	"sync"
)

// mu guards the registries below. It is never held while a hook runs, so a hook may call any of the
// registration functions in this package.
var mu sync.Mutex

var onBeforeUpdateHooks []func() error

// AppendHookOnBeforeUpdate appends a hook function that is run before the main update function every
// tick.
func AppendHookOnBeforeUpdate(f func() error) {
	mu.Lock()
	defer mu.Unlock()
	onBeforeUpdateHooks = append(onBeforeUpdateHooks, f)
}

// beforeUpdateHooks returns a snapshot of the registered hooks that is safe to iterate without the lock.
func beforeUpdateHooks() []func() error {
	mu.Lock()
	defer mu.Unlock()
	// Appends never overwrite the elements within the current length, so the slice header alone is a
	// stable snapshot.
	return onBeforeUpdateHooks
}

func RunBeforeUpdateHooks() error {
	for _, f := range beforeUpdateHooks() {
		if err := f(); err != nil {
			return err
		}
	}
	return nil
}

var onBeforeUpdateWithVMGuestInfoHooks []func(vmGuest bool) error

// AppendHookOnBeforeUpdateWithVMGuestInfo appends a hook function that runs before the main update
// function every tick, the same as [AppendHookOnBeforeUpdate], but is passed whether the process is
// running as a virtualization guest.
func AppendHookOnBeforeUpdateWithVMGuestInfo(f func(vmGuest bool) error) {
	mu.Lock()
	defer mu.Unlock()
	onBeforeUpdateWithVMGuestInfoHooks = append(onBeforeUpdateWithVMGuestInfoHooks, f)
}

// beforeUpdateWithVMGuestInfoHooks returns a snapshot of the registered hooks that is safe to iterate
// without the lock.
func beforeUpdateWithVMGuestInfoHooks() []func(vmGuest bool) error {
	mu.Lock()
	defer mu.Unlock()
	// Appends never overwrite the elements within the current length, so the slice header alone is a
	// stable snapshot.
	return onBeforeUpdateWithVMGuestInfoHooks
}

func RunBeforeUpdateHooksWithVMGuestInfo(vmGuest bool) error {
	for _, f := range beforeUpdateWithVMGuestInfoHooks() {
		if err := f(vmGuest); err != nil {
			return err
		}
	}
	return nil
}

var (
	// audioMu serializes the audio suspend and resume transitions and guards audioSuspended. Unlike mu,
	// it is held while the suspend or resume hook runs, so those hooks must not call [SuspendAudio] or
	// [ResumeAudio].
	audioMu sync.Mutex

	audioSuspended bool
	onSuspendAudio func() error
	onResumeAudio  func() error
)

func OnSuspendAudio(f func() error) {
	mu.Lock()
	defer mu.Unlock()
	onSuspendAudio = f
}

func OnResumeAudio(f func() error) {
	mu.Lock()
	defer mu.Unlock()
	onResumeAudio = f
}

func suspendAudioHook() func() error {
	mu.Lock()
	defer mu.Unlock()
	return onSuspendAudio
}

func resumeAudioHook() func() error {
	mu.Lock()
	defer mu.Unlock()
	return onResumeAudio
}

// SuspendAudio runs the suspend hook unless the audio is already suspended.
//
// The audio is recorded as suspended only when the hook succeeds, so a failed call leaves the state
// unchanged and the next call runs the hook again. Concurrent calls are serialized: a call that finds
// its transition already made by another call returns nil.
func SuspendAudio() error {
	audioMu.Lock()
	defer audioMu.Unlock()

	if audioSuspended {
		return nil
	}
	if f := suspendAudioHook(); f != nil {
		if err := f(); err != nil {
			return err
		}
	}
	audioSuspended = true
	return nil
}

// ResumeAudio runs the resume hook unless the audio is not suspended.
//
// The audio is recorded as resumed only when the hook succeeds, so a failed call leaves the state
// unchanged and the next call runs the hook again. Concurrent calls are serialized: a call that finds
// its transition already made by another call returns nil.
func ResumeAudio() error {
	audioMu.Lock()
	defer audioMu.Unlock()

	if !audioSuspended {
		return nil
	}
	if f := resumeAudioHook(); f != nil {
		if err := f(); err != nil {
			return err
		}
	}
	audioSuspended = false
	return nil
}
