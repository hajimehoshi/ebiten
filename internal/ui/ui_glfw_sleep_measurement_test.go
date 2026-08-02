// Copyright 2026 The Ebitengine Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build ebitengine_sleep_measurement && !android && !ios && !js && !nintendosdk && !playstation5

package ui

import (
	"sort"
	"testing"
	"time"
)

func TestTimeSleepOvershoot(t *testing.T) {
	const (
		d          = time.Second / 60
		iterations = 500
	)

	overshoots := make([]time.Duration, 0, iterations)
	var total time.Duration
	for range iterations {
		start := time.Now()
		time.Sleep(d)
		overshoot := time.Since(start) - d
		overshoots = append(overshoots, overshoot)
		total += overshoot
	}

	sort.Slice(overshoots, func(i, j int) bool {
		return overshoots[i] < overshoots[j]
	})
	median := (overshoots[iterations/2-1] + overshoots[iterations/2]) / 2
	p95 := overshoots[(95*iterations+99)/100-1]
	t.Logf("duration=%s iterations=%d min=%s median=%s mean=%s p95=%s max=%s",
		d, iterations, overshoots[0], median, total/iterations, p95, overshoots[iterations-1])
}
