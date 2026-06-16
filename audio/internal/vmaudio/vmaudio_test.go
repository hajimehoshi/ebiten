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

package vmaudio_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"runtime"
	"slices"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio/internal/vmaudio"
)

func newContext(t *testing.T, sampleRate int) *vmaudio.Context {
	t.Helper()
	return vmaudio.NewContext(sampleRate)
}

func pcmBytes(comps ...float32) []byte {
	b := make([]byte, 4*len(comps))
	for i, v := range comps {
		binary.LittleEndian.PutUint32(b[4*i:], math.Float32bits(v))
	}
	return b
}

func pcmComps(b []byte) []float32 {
	comps := make([]float32, len(b)/4)
	for i := range comps {
		comps[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[4*i:]))
	}
	return comps
}

func ramp(from, n int) []float32 {
	comps := make([]float32, n)
	for i := range comps {
		comps[i] = float32(from + i)
	}
	return comps
}

// onlyControlID takes the pending control push, requires it to report exactly one player, and returns
// that player's id.
func onlyControlID(t *testing.T, c *vmaudio.Context) int64 {
	t.Helper()
	controls := c.TakeControlChangesForTesting(nil)
	if len(controls) != 1 {
		t.Fatalf("got %d controls; want 1", len(controls))
	}
	return controls[0].ID
}

func TestControlReportsCreationAndChanges(t *testing.T) {
	c := newContext(t, 48000)
	p := c.NewPlayer(bytes.NewReader(pcmBytes(ramp(1, 8)...)))

	// A new player is reported once, not playing, at full volume.
	controls := c.TakeControlChangesForTesting(nil)
	if len(controls) != 1 {
		t.Fatalf("got %d controls; want 1 (creation)", len(controls))
	}
	if controls[0].Playing || controls[0].Volume != 1 {
		t.Fatalf("creation control = %+v; want not playing, volume 1", controls[0])
	}

	// No change: nothing reported.
	if controls := c.TakeControlChangesForTesting(nil); len(controls) != 0 {
		t.Fatalf("got %d controls with no change; want 0", len(controls))
	}

	// A play and a volume change since the last push are reported together, once.
	p.Play()
	p.SetVolume(0.5)
	controls = c.TakeControlChangesForTesting(nil)
	if len(controls) != 1 {
		t.Fatalf("got %d controls; want 1", len(controls))
	}
	if !controls[0].Playing || controls[0].Volume != 0.5 {
		t.Fatalf("control = %+v; want playing, volume 0.5", controls[0])
	}
}

func TestCloseRemovesPlayerAndReportsRemoval(t *testing.T) {
	c := newContext(t, 8)
	p := c.NewPlayer(bytes.NewReader(pcmBytes(ramp(1, 8)...)))
	p.Play()
	id := onlyControlID(t, c)

	// Closing the player before its source ends removes it and reports the removal once.
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
	controls := c.TakeControlChangesForTesting(nil)
	if len(controls) != 1 {
		t.Fatalf("got %d controls after close; want 1", len(controls))
	}
	if controls[0].ID != id || !controls[0].Closed {
		t.Fatalf("close control = %+v; want {ID:%d, Closed:true}", controls[0], id)
	}

	// The player is gone: a read reports end-of-stream with no samples.
	if pcm, eof := c.ReadForTesting(id, 16); len(pcm) != 0 || !eof {
		t.Errorf("read after close = (%d bytes, eof=%v); want (0, true)", len(pcm), eof)
	}

	// Close is idempotent and the removal is reported only once: a second close pushes nothing.
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
	if controls := c.TakeControlChangesForTesting(nil); len(controls) != 0 {
		t.Fatalf("got %d controls after a second close; want 0", len(controls))
	}
}

func TestReadStreamsSamples(t *testing.T) {
	c := newContext(t, 8)
	// 8 components = 4 stereo frames = 32 bytes.
	p := c.NewPlayer(bytes.NewReader(pcmBytes(ramp(1, 8)...)))
	p.Play()
	id := onlyControlID(t, c)

	var got []float32
	for {
		// Two frames (16 bytes) at a time.
		pcm, eof := c.ReadForTesting(id, 16)
		got = append(got, pcmComps(pcm)...)
		if eof {
			break
		}
	}
	if want := ramp(1, 8); !slices.Equal(got, want) {
		t.Errorf("samples %v; want %v", got, want)
	}

	// The finished player is kept (not removed) but produces no more samples: a further read reports
	// end-of-stream with no samples.
	if pcm, eof := c.ReadForTesting(id, 16); len(pcm) != 0 || !eof {
		t.Errorf("read after EOF = (%d bytes, eof=%v); want (0, true)", len(pcm), eof)
	}
}

func TestAbandonedPlayerIsReclaimed(t *testing.T) {
	c := newContext(t, 8)
	// Create and play a player, then drop every reference to it (it stays in a sub-scope).
	id := func() int64 {
		p := c.NewPlayer(bytes.NewReader(pcmBytes(ramp(1, 8)...)))
		p.Play()
		return onlyControlID(t, c) // consumes the creation control; p falls out of scope here
	}()

	// The map holds the player only weakly, so GC reclaims it; the registered cleanup then removes it and
	// reports the removal to the host on the next control push.
	var reclaimed bool
	for i := 0; i < 100 && !reclaimed; i++ {
		runtime.GC()
		for _, ctrl := range c.TakeControlChangesForTesting(nil) {
			if ctrl.ID == id && ctrl.Closed {
				reclaimed = true
			}
		}
		if !reclaimed {
			time.Sleep(10 * time.Millisecond)
		}
	}
	if !reclaimed {
		t.Fatal("an abandoned player was not reclaimed by GC")
	}
}

func TestEOFKeepsPlayerAndAllowsReplay(t *testing.T) {
	c := newContext(t, 8)
	p := c.NewPlayer(bytes.NewReader(pcmBytes(ramp(1, 8)...)))
	p.Play()
	id := onlyControlID(t, c)

	// Drain the source to EOF.
	for {
		if _, eof := c.ReadForTesting(id, 16); eof {
			break
		}
	}

	// Reaching EOF does not remove the player; it now reports not playing (like audio.Player at its end).
	controls := c.TakeControlChangesForTesting(nil)
	if len(controls) != 1 {
		t.Fatalf("after EOF, got %d controls; want 1 (the playing→false change)", len(controls))
	}
	if controls[0].ID != id || controls[0].Playing || controls[0].Closed {
		t.Fatalf("after EOF, control = %+v; want {ID:%d, Playing:false}", controls[0], id)
	}

	// Seeking the source back and reading again replays it: the player was not dropped.
	if _, err := p.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	pcm, _ := c.ReadForTesting(id, 16)
	if got, want := pcmComps(pcm), ramp(1, 4); !slices.Equal(got, want) {
		t.Errorf("after replay %v; want %v", got, want)
	}
	// The replayed player reports playing again.
	if controls := c.TakeControlChangesForTesting(nil); len(controls) != 1 || !controls[0].Playing {
		t.Fatalf("after replay, controls = %+v; want one playing", controls)
	}
}

func TestPlayersAreNotMixed(t *testing.T) {
	c := newContext(t, 8)
	p := c.NewPlayer(bytes.NewReader(pcmBytes(1, 1, 1, 1)))
	q := c.NewPlayer(bytes.NewReader(pcmBytes(2, 2, 2, 2)))
	q.SetVolume(0.25)
	p.Play()
	q.Play()

	// Controls are sorted by ID, so the first player created comes first.
	controls := c.TakeControlChangesForTesting(nil)
	if len(controls) != 2 {
		t.Fatalf("got %d controls; want 2", len(controls))
	}
	if controls[1].Volume != 0.25 {
		t.Errorf("player 2 volume = %v; want 0.25 (reported)", controls[1].Volume)
	}

	pPCM, _ := c.ReadForTesting(controls[0].ID, 16)
	qPCM, _ := c.ReadForTesting(controls[1].ID, 16)
	if got, want := pcmComps(pPCM), []float32{1, 1, 1, 1}; !slices.Equal(got, want) {
		t.Errorf("player 1 samples %v; want %v", got, want)
	}
	// The volume is reported but NOT applied to the samples.
	if got, want := pcmComps(qPCM), []float32{2, 2, 2, 2}; !slices.Equal(got, want) {
		t.Errorf("player 2 samples %v; want %v (volume must not be applied)", got, want)
	}
}

func TestPausedReadsNothing(t *testing.T) {
	c := newContext(t, 8)
	p := c.NewPlayer(bytes.NewReader(pcmBytes(ramp(1, 8)...)))
	id := onlyControlID(t, c)

	// Not playing: a read yields no samples and is not finished.
	if pcm, eof := c.ReadForTesting(id, 16); len(pcm) != 0 || eof {
		t.Fatalf("not-playing read = (%d bytes, eof=%v); want (0, false)", len(pcm), eof)
	}
	p.Play()
	pcm, _ := c.ReadForTesting(id, 16)
	if got, want := pcmComps(pcm), ramp(1, 4); !slices.Equal(got, want) {
		t.Errorf("after play %v; want %v", got, want)
	}
}

func TestSuspendDoesNotConsume(t *testing.T) {
	c := newContext(t, 8)
	p := c.NewPlayer(bytes.NewReader(pcmBytes(ramp(1, 8)...)))
	p.Play()
	id := onlyControlID(t, c)

	if err := c.Suspend(); err != nil {
		t.Fatal(err)
	}
	// A suspended device produces no samples and is not finished.
	if pcm, eof := c.ReadForTesting(id, 16); len(pcm) != 0 || eof {
		t.Fatalf("suspended read = (%d bytes, eof=%v); want (0, false)", len(pcm), eof)
	}
	if err := c.Resume(); err != nil {
		t.Fatal(err)
	}
	pcm, _ := c.ReadForTesting(id, 16)
	if got, want := pcmComps(pcm), ramp(1, 4); !slices.Equal(got, want) {
		t.Errorf("after resume %v; want %v (suspend must not consume the source)", got, want)
	}
}

func TestSeek(t *testing.T) {
	c := newContext(t, 8)
	p := c.NewPlayer(bytes.NewReader(pcmBytes(ramp(1, 8)...)))
	p.Play()
	id := onlyControlID(t, c)

	c.ReadForTesting(id, 16) // ramp(1,4)
	if _, err := p.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	pcm, _ := c.ReadForTesting(id, 16)
	if got, want := pcmComps(pcm), ramp(1, 4); !slices.Equal(got, want) {
		t.Errorf("samples after seek %v; want %v", got, want)
	}
}

// chunkReader yields its data at most chunk bytes per Read, exercising short and sample-unaligned
// reads.
type chunkReader struct {
	data  []byte
	chunk int
}

func (r *chunkReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	n := min(len(p), r.chunk, len(r.data))
	copy(p, r.data[:n])
	r.data = r.data[n:]
	return n, nil
}

func TestUnalignedReads(t *testing.T) {
	c := newContext(t, 8)
	p := c.NewPlayer(&chunkReader{data: pcmBytes(ramp(1, 8)...), chunk: 5})
	p.Play()
	id := onlyControlID(t, c)

	var got []float32
	for {
		pcm, eof := c.ReadForTesting(id, 16)
		got = append(got, pcmComps(pcm)...)
		if eof {
			break
		}
	}
	if want := ramp(1, 8); !slices.Equal(got, want) {
		t.Errorf("samples %v; want %v", got, want)
	}
}

// stallReader yields its data on the first Read and then keeps returning no data, like a real-time
// stream that is not producing.
type stallReader struct {
	data []byte
}

func (r *stallReader) Read(p []byte) (int, error) {
	n := copy(p, r.data)
	r.data = r.data[n:]
	return n, nil
}

func TestStallingSource(t *testing.T) {
	c := newContext(t, 8)
	// Two frames are available; the player keeps playing rather than finishing.
	p := c.NewPlayer(&stallReader{data: pcmBytes(1, 2, 3, 4)})
	p.Play()
	id := onlyControlID(t, c)

	// Ask for four frames; only the two available come back, and the player is not finished.
	pcm, eof := c.ReadForTesting(id, 32)
	if eof {
		t.Error("a stalled source must not finish the player")
	}
	if got, want := pcmComps(pcm), []float32{1, 2, 3, 4}; !slices.Equal(got, want) {
		t.Errorf("stalled read = %v; want %v (the available frames)", got, want)
	}
}

type failReader struct {
	err error
}

func (r *failReader) Read(p []byte) (int, error) {
	return 0, r.err
}

func TestFailingSource(t *testing.T) {
	c := newContext(t, 8)
	wantErr := errors.New("source failed")
	p := c.NewPlayer(&failReader{err: wantErr})
	p.Play()
	id := onlyControlID(t, c)

	if _, eof := c.ReadForTesting(id, 16); !eof {
		t.Error("a failed source did not finish the player")
	}
	if err := p.Err(); !errors.Is(err, wantErr) {
		t.Errorf("Err() = %v; want %v", err, wantErr)
	}
}
