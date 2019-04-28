// Copyright 2019 The Ebiten Authors
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
	"io"
	"runtime"
	"sync"
)

type mux struct {
	ps map[*playerImpl]struct{}
	m  sync.RWMutex
}

const (
	channelNum     = 2
	bytesPerSample = 2 * channelNum

	// TODO: This assumes that bytesPerSample is a power of 2.
	mask = ^(bytesPerSample - 1)
)

func newMux() *mux {
	return &mux{
		ps: map[*playerImpl]struct{}{},
	}
}

func (m *mux) Read(b []byte) (int, error) {
	m.m.Lock()
	defer m.m.Unlock()

	if len(m.ps) == 0 {
		l := len(b)
		l &= mask
		copy(b, make([]byte, l))
		return l, nil
	}

	l := len(b)
	l &= mask

	allSkipped := true

	// TODO: Now a player is not locked. Should we lock it?

	for p := range m.ps {
		if p.shouldSkip() {
			continue
		}
		allSkipped = false
		s := p.bufferSizeInBytes()
		if l > s {
			l = s
			l &= mask
		}
	}

	if allSkipped {
		l = 0
	}

	if l == 0 {
		// If l is 0, all the ps might reach EOF at the next update.
		// However, this Read might block forever and never causes context switch
		// on single-thread environment (e.g. browser).
		// Call Gosched to cause context switch on purpose.
		runtime.Gosched()
	}

	b16s := [][]int16{}
	for p := range m.ps {
		buf, err := p.bufferToInt16(l)
		if err != nil {
			return 0, err
		}
		if l > len(buf)*2 {
			l = len(buf) * 2
		}
		b16s = append(b16s, buf)
	}

	for i := 0; i < l/2; i++ {
		x := 0
		for _, b16 := range b16s {
			x += int(b16[i])
		}
		if x > (1<<15)-1 {
			x = (1 << 15) - 1
		}
		if x < -(1 << 15) {
			x = -(1 << 15)
		}
		b[2*i] = byte(x)
		b[2*i+1] = byte(x >> 8)
	}

	closed := []*playerImpl{}
	for p := range m.ps {
		if p.eof() {
			closed = append(closed, p)
		}
	}
	for _, p := range closed {
		if p.isFinalized() {
			p.closeImpl()
		}
		delete(m.ps, p)
	}

	return l, nil
}

func (m *mux) addPlayer(player *playerImpl) {
	m.m.Lock()
	m.ps[player] = struct{}{}
	m.m.Unlock()
}

func (m *mux) removePlayer(player *playerImpl) {
	m.m.Lock()
	delete(m.ps, player)
	m.m.Unlock()
}

func (m *mux) hasPlayer(player *playerImpl) bool {
	m.m.RLock()
	_, ok := m.ps[player]
	m.m.RUnlock()
	return ok
}

func (m *mux) hasSource(src io.ReadCloser) bool {
	m.m.RLock()
	defer m.m.RUnlock()
	for p := range m.ps {
		if p.src == src {
			return true
		}
	}
	return false
}
