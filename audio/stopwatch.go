// Copyright 2024 The Ebitengine Authors
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

package audio

import (
	"sync"
	"time"
)

type stopwatch struct {
	duration    time.Duration
	lastStarted time.Time
	running     bool

	m sync.Mutex
}

func (s *stopwatch) start() {
	s.m.Lock()
	defer s.m.Unlock()

	if s.running {
		return
	}
	s.lastStarted = time.Now()
	s.running = true
}

func (s *stopwatch) stop() {
	s.m.Lock()
	defer s.m.Unlock()

	if !s.running {
		return
	}
	s.duration += time.Since(s.lastStarted)
	s.running = false
}

func (s *stopwatch) current() time.Duration {
	s.m.Lock()
	defer s.m.Unlock()

	d := s.duration
	if s.running {
		d += time.Since(s.lastStarted)
	}
	return d
}

func (s *stopwatch) reset() {
	s.m.Lock()
	defer s.m.Unlock()

	s.duration = 0
	s.lastStarted = time.Time{}
	s.running = false
}
