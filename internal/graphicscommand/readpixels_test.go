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

package graphicscommand_test

import (
	"context"
	"errors"
	"image"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicscommand"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/thread"
)

type testReadback struct {
	driver   *testReadPixelsDriver
	done     bool
	err      error
	discards int
}

func (r *testReadback) Poll() (bool, error) {
	return r.done, r.err
}

func (r *testReadback) Copy(args []graphicsdriver.PixelsArgs) error {
	for _, a := range args {
		for i := range a.Pixels {
			a.Pixels[i] = r.driver.readValue
		}
	}
	return nil
}

func (r *testReadback) Discard() {
	r.discards++
}

type testReadPixelsImage struct {
	graphicsdriver.Image
	driver *testReadPixelsDriver
	id     graphicsdriver.ImageID
}

func (i *testReadPixelsImage) ID() graphicsdriver.ImageID { return i.id }

func (i *testReadPixelsImage) Dispose() {}

func (i *testReadPixelsImage) ReadPixels(args []graphicsdriver.PixelsArgs) error {
	i.driver.reads++
	if i.driver.readErr != nil {
		return i.driver.readErr
	}
	for _, a := range args {
		for j := range a.Pixels {
			a.Pixels[j] = i.driver.readValue
		}
	}
	return nil
}

func (i *testReadPixelsImage) WritePixels(args []graphicsdriver.PixelsArgs) error { return nil }

type testReadPixelsAsyncImage struct {
	testReadPixelsImage
}

func (i *testReadPixelsAsyncImage) ReadPixelsAsync(args []graphicsdriver.PixelsArgs) (graphicsdriver.PixelsReadback, error) {
	i.driver.reads++
	if i.driver.asyncErr != nil {
		return nil, i.driver.asyncErr
	}
	r := &testReadback{driver: i.driver}
	i.driver.readbacks = append(i.driver.readbacks, r)
	return r, nil
}

type testReadPixelsDriver struct {
	graphicsdriver.Graphics
	nextID    graphicsdriver.ImageID
	async     bool
	readValue byte
	readErr   error
	asyncErr  error
	endErr    error
	beginErr  error
	reads     int
	readbacks []*testReadback
}

func (g *testReadPixelsDriver) NewImage(width, height int) (graphicsdriver.Image, error) {
	g.nextID++
	i := &testReadPixelsImage{
		driver: g,
		id:     g.nextID,
	}
	if g.async {
		return &testReadPixelsAsyncImage{testReadPixelsImage: *i}, nil
	}
	return i, nil
}

func (g *testReadPixelsDriver) NewScreenFramebufferImage(width, height int) (graphicsdriver.Image, error) {
	return g.NewImage(width, height)
}

func (g *testReadPixelsDriver) Begin() error { return g.beginErr }

func (g *testReadPixelsDriver) End(mode graphicsdriver.FlushMode) error {
	return g.endErr
}

func (*testReadPixelsDriver) SetVsyncEnabled(enabled bool) {}

func (g *testReadPixelsDriver) completeAll() {
	for _, r := range g.readbacks {
		r.done = true
	}
}

type testReadPixelsSetup struct {
	img     *graphicscommand.Image
	driver  *testReadPixelsDriver
	manager *graphicscommand.CommandQueueManagerForTesting
	sync    func()
}

func newTestReadPixelsSetup(t *testing.T, async bool) *testReadPixelsSetup {
	t.Helper()
	renderThread := thread.NewOSThread()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = renderThread.LoopAndStop(ctx)
	}()
	manager, restoreManager := graphicscommand.ResetCommandQueueManagerForTesting()
	restore := graphicscommand.SetRenderThreadForTesting(renderThread)
	graphicscommand.SetVsyncEnabled(false)

	t.Cleanup(func() {
		restoreManager()
		restore()
		graphicscommand.SetVsyncEnabled(true)
		cancel()
		<-done
	})

	s := &testReadPixelsSetup{
		driver: &testReadPixelsDriver{async: async},
		sync:   func() { thread.Call(renderThread, func() {}) },
	}
	s.manager = manager
	s.img = graphicscommand.NewImage(4, 4, false, "")
	if err := s.manager.FlushForTesting(s.driver, graphicsdriver.FlushModeIntermediate); err != nil {
		t.Fatal(err)
	}
	s.sync()

	return s
}

func (s *testReadPixelsSetup) flush(mode graphicsdriver.FlushMode) error {
	if err := s.manager.FlushForTesting(s.driver, mode); err != nil {
		return err
	}
	s.sync()
	return nil
}

func (s *testReadPixelsSetup) readPixelsAsync(pixels []byte) <-chan error {
	return s.img.ReadPixelsAsync([]graphicsdriver.PixelsArgs{{
		Pixels: pixels,
		Region: image.Rect(0, 0, 4, 4),
	}})
}

func requireOneValue(t *testing.T, ch <-chan error) error {
	t.Helper()
	var err error
	var ok bool
	select {
	case err, ok = <-ch:
	case <-time.After(5 * time.Second):
		t.Error("read-back did not publish a result")
		return errors.New("test: read-back timed out")
	}
	if !ok {
		t.Error("the channel must not be closed")
	}
	select {
	case _, ok := <-ch:
		if !ok {
			t.Error("the channel must not be closed after the value is received")
		}
		t.Error("the channel must not receive a second value")
	default:
	}
	return err
}

func TestReadPixelsAsyncDoesNotFlush(t *testing.T) {
	s := newTestReadPixelsSetup(t, true)

	pixels := make([]byte, 4*4*4)
	ch := s.readPixelsAsync(pixels)
	s.sync()
	if s.driver.reads != 0 {
		t.Errorf("the read-back was started before a flush: reads = %d, want 0", s.driver.reads)
	}
	select {
	case err := <-ch:
		t.Errorf("the result was published before a flush: %v", err)
	default:
	}

	if err := s.flush(graphicsdriver.FlushModeEndFrame); err != nil {
		t.Error(err)
		return
	}
	select {
	case err := <-ch:
		t.Errorf("the result was published although the read-back is not finished: %v", err)
	default:
	}

	s.driver.completeAll()
	if err := s.flush(graphicsdriver.FlushModeEndFrame); err != nil {
		t.Error(err)
		return
	}

	if err := requireOneValue(t, ch); err != nil {
		t.Error(err)
		return
	}
	for i, p := range pixels {
		if p != s.driver.readValue {
			t.Errorf("pixels[%d] = %#x, want %#x", i, p, s.driver.readValue)
		}
	}
	for _, r := range s.driver.readbacks {
		if r.discards != 1 {
			t.Errorf("discards = %d, want 1", r.discards)
		}
	}
}

func TestReadPixelsAsyncWaitsForTheReadback(t *testing.T) {
	s := newTestReadPixelsSetup(t, true)
	pixels := make([]byte, 4*4*4)
	ch := s.readPixelsAsync(pixels)
	for range 3 {
		if err := s.flush(graphicsdriver.FlushModeEndFrame); err != nil {
			t.Error(err)
			return
		}
	}
	select {
	case err := <-ch:
		t.Errorf("the result was published before the read-back finished: %v", err)
	default:
	}

	s.driver.completeAll()
	if err := s.flush(graphicsdriver.FlushModeEndFrame); err != nil {
		t.Error(err)
		return
	}

	if err := requireOneValue(t, ch); err != nil {
		t.Error(err)
		return
	}
}

func TestReadPixelsAsyncSynchronousFallback(t *testing.T) {
	s := newTestReadPixelsSetup(t, false)
	s.driver.readValue = 0x33

	pixels := make([]byte, 4*4*4)
	ch := s.readPixelsAsync(pixels)

	if err := s.flush(graphicsdriver.FlushModeEndFrame); err != nil {
		t.Error(err)
		return
	}

	if err := requireOneValue(t, ch); err != nil {
		t.Error(err)
		return
	}
	for i, p := range pixels {
		if p != 0x33 {
			t.Errorf("pixels[%d] = %#x, want %#x", i, p, 0x33)
		}
	}
}

func TestReadPixelsAsyncAbortedByFlushError(t *testing.T) {
	s := newTestReadPixelsSetup(t, true)

	ch := s.readPixelsAsync(make([]byte, 4*4*4))
	if err := s.flush(graphicsdriver.FlushModeIntermediate); err != nil {
		t.Error(err)
		return
	}
	s.driver.endErr = errors.New("test: broken driver")

	var err error
	for range 10 {
		if ferr := s.flush(graphicsdriver.FlushModeEndFrame); ferr != nil {
			err = ferr
			break
		}
	}
	if err == nil {
		t.Error("the flush of a broken driver did not report an error")
	}

	if rerr := requireOneValue(t, ch); rerr == nil {
		t.Error("an aborted read-back must report an error")
	}
	for _, r := range s.driver.readbacks {
		if r.discards != 1 {
			t.Errorf("discards = %d, want 1: the read-back must release its resources", r.discards)
		}
	}
}

func TestReadPixelsAsyncReadbackError(t *testing.T) {
	s := newTestReadPixelsSetup(t, true)

	ch := s.readPixelsAsync(make([]byte, 4*4*4))
	if err := s.flush(graphicsdriver.FlushModeIntermediate); err != nil {
		t.Error(err)
		return
	}

	for _, r := range s.driver.readbacks {
		r.err = errors.New("test: read-back failed")
		r.done = true
	}
	if err := s.flush(graphicsdriver.FlushModeEndFrame); err != nil {
		t.Error(err)
		return
	}

	if err := requireOneValue(t, ch); err == nil {
		t.Error("a failed read-back must report an error")
	}
}

func TestReadPixelsAsyncSubmitError(t *testing.T) {
	s := newTestReadPixelsSetup(t, true)
	s.driver.asyncErr = errors.New("test: cannot start a read-back")

	ch := s.readPixelsAsync(make([]byte, 4*4*4))
	if err := s.flush(graphicsdriver.FlushModeEndFrame); err != nil {
		t.Error(err)
		return
	}

	if err := requireOneValue(t, ch); err == nil {
		t.Error("a read-back that cannot be started must report an error")
	}
}

func TestReadPixelsAsyncAbortedByEarlierCommand(t *testing.T) {
	s := newTestReadPixelsSetup(t, true)
	s.manager.EnqueueCommandForTesting(&failingCommand{})
	ch := s.readPixelsAsync(make([]byte, 4*4*4))
	if err := s.manager.FlushForTesting(s.driver, graphicsdriver.FlushModeEndFrame); err != nil {
		t.Error(err)
		return
	}
	s.sync()

	if rerr := requireOneValue(t, ch); rerr == nil {
		t.Error("a read-back that never ran must report an error")
	}
}

func TestReadPixelsAsyncAbortedByBeginError(t *testing.T) {
	s := newTestReadPixelsSetup(t, true)
	ch := s.readPixelsAsync(make([]byte, 4*4*4))
	s.driver.beginErr = errors.New("test: Begin failed")
	_ = s.flush(graphicsdriver.FlushModeEndFrame)
	if err := requireOneValue(t, ch); !errors.Is(err, s.driver.beginErr) {
		t.Errorf("got %v, want %v", err, s.driver.beginErr)
	}
}

func TestReadPixelsAsyncQueuedAfterFlushError(t *testing.T) {
	s := newTestReadPixelsSetup(t, true)
	s.driver.endErr = errors.New("test: End failed")
	_ = s.flush(graphicsdriver.FlushModeEndFrame)
	ch := s.readPixelsAsync(make([]byte, 4*4*4))
	if err := s.flush(graphicsdriver.FlushModeEndFrame); err == nil {
		t.Error("expected flush error")
	}
	if err := requireOneValue(t, ch); !errors.Is(err, s.driver.endErr) {
		t.Errorf("got %v, want %v", err, s.driver.endErr)
	}
}
