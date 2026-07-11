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

package vibrate

import (
	"sync"
	"sync/atomic"
	"time"
)

// Vibration is a device vibration request: how long it lasts and its magnitude, in 0..1.
type Vibration struct {
	Duration  time.Duration
	Magnitude float64
}

// Vibrate vibrates the device for duration at magnitude, in 0..1. When recording is enabled it also
// records the request so [AppendPendingVibrations] can drain it.
func Vibrate(duration time.Duration, magnitude float64) {
	theRecorder.record(duration, magnitude)
	vibrate(duration, magnitude)
}

// EnableRecording makes [Vibrate] record the latest requested vibration for [AppendPendingVibrations]
// to drain. The virtualization guest enables it to forward device vibration to its host; other builds
// never call it, so recording stays off and Vibrate behaves exactly as before.
func EnableRecording() {
	theRecorder.enabled.Store(true)
}

// AppendPendingVibrations appends the device vibration requested since the last call to dst and returns
// the extended slice, clearing it so each request is reported once. It appends at most one entry: a
// device has a single vibration state, so a later request replaces an earlier undrained one.
func AppendPendingVibrations(dst []Vibration) []Vibration {
	return theRecorder.appendPending(dst)
}

var theRecorder recorder

// recorder captures device vibration requests so a virtualization guest can forward them. It stays
// inert until [EnableRecording] is called.
type recorder struct {
	// enabled gates recording so builds that never enable it pay only an atomic load per Vibrate call.
	enabled atomic.Bool

	m          sync.Mutex
	pending    Vibration
	hasPending bool
}

func (r *recorder) record(duration time.Duration, magnitude float64) {
	if !r.enabled.Load() {
		return
	}
	r.m.Lock()
	defer r.m.Unlock()
	r.pending = Vibration{Duration: duration, Magnitude: magnitude}
	r.hasPending = true
}

func (r *recorder) appendPending(dst []Vibration) []Vibration {
	r.m.Lock()
	defer r.m.Unlock()
	if r.hasPending {
		dst = append(dst, r.pending)
		r.hasPending = false
	}
	return dst
}
