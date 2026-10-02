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

// testReadback is a pixel read-back that a testReadPixelsDriver returns.
// The test decides when the read-back finishes.
type testReadback struct {
	driver *testReadPixelsDriver

	// done represents whether the read pixels are available.
	done bool

	// err is the error reported by Poll.
	err error

	// copies is the number of the calls to Copy.
	copies int

	// discards is the number of the calls to Discard.
	discards int
}

func (r *testReadback) Poll() (bool, error) {
	r.driver.polls++
	return r.done, r.err
}

func (r *testReadback) Copy(args []graphicsdriver.PixelsArgs) error {
	r.copies++
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

// testReadPixelsImage is an image of testReadPixelsDriver.
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

// testReadPixelsAsyncImage additionally supports graphicsdriver.AsyncPixelsReader.
type testReadPixelsAsyncImage struct {
	testReadPixelsImage
}

func (i *testReadPixelsAsyncImage) ReadPixelsAsync(args []graphicsdriver.PixelsArgs) (graphicsdriver.PixelsReadback, error) {
	i.driver.reads++
	i.driver.asyncs++
	if i.driver.asyncErr != nil {
		return nil, i.driver.asyncErr
	}
	r := &testReadback{driver: i.driver}
	i.driver.readbacks = append(i.driver.readbacks, r)
	return r, nil
}

// testReadPixelsDriver is a graphics driver for testing the asynchronous pixel read-back.
type testReadPixelsDriver struct {
	graphicsdriver.Graphics

	// nextID is the next image ID.
	nextID graphicsdriver.ImageID

	// async represents whether the images support graphicsdriver.AsyncPixelsReader.
	async bool

	// readValue is the value written into the read pixels.
	readValue byte

	// readErr is the error returned by a synchronous read.
	readErr error

	// asyncErr is the error returned when starting an asynchronous read.
	asyncErr error

	// endErr is the error returned by End.
	endErr   error
	beginErr error

	// reads is the number of the read-backs started, both synchronous and asynchronous.
	reads int

	// asyncs is the number of the asynchronous read-backs started.
	asyncs int

	// polls is the number of the polls of the read-backs.
	polls int

	// readbacks are the asynchronous read-backs that have not been finished.
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

// completeAll finishes all the pending read-backs successfully.
func (g *testReadPixelsDriver) completeAll() {
	for _, r := range g.readbacks {
		r.done = true
	}
}

// testReadPixelsSetup is the environment of a pixel read-back test.
type testReadPixelsSetup struct {
	img     *graphicscommand.Image
	driver  *testReadPixelsDriver
	manager *graphicscommand.CommandQueueManagerForTesting

	// sync waits until the render thread finishes the tasks queued so far. The driver state must
	// only be read or written after sync, as the render thread is the only user of the driver.
	sync func()
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
	restore := graphicscommand.SetRenderThreadForTesting(renderThread)
	graphicscommand.SetVsyncEnabled(false)

	s := &testReadPixelsSetup{
		driver: &testReadPixelsDriver{async: async},
		sync:   func() { thread.Call(renderThread, func() {}) },
	}
	s.manager = &graphicscommand.CommandQueueManagerForTesting{}
	s.img = graphicscommand.NewImageForTesting(s.manager, 4, 4, false, "")
	// Execute the newImageCommand so that the driver image is available.
	if err := s.manager.FlushForTesting(s.driver, graphicsdriver.FlushModeIntermediate); err != nil {
		t.Fatal(err)
	}
	s.sync()

	t.Cleanup(func() {
		restore()
		graphicscommand.SetVsyncEnabled(true)
		cancel()
		<-done
	})
	return s
}

// flush flushes the command queue and waits until the render thread finishes the flush.
func (s *testReadPixelsSetup) flush(mode graphicsdriver.FlushMode) error {
	if err := s.manager.FlushForTesting(s.driver, mode); err != nil {
		return err
	}
	s.sync()
	return nil
}

// readPixelsAsync starts a read-back of the whole image.
func (s *testReadPixelsSetup) readPixelsAsync(pixels []byte) <-chan error {
	return s.manager.ReadPixelsAsyncForTesting(s.img, []graphicsdriver.PixelsArgs{{
		Pixels: pixels,
		Region: image.Rect(0, 0, 4, 4),
	}})
}

// requireOneValue receives one value from ch and requires that the channel is not closed and has
// no more values.
func requireOneValue(t *testing.T, ch <-chan error) error {
	t.Helper()
	var err error
	var ok bool
	select {
	case err, ok = <-ch:
	case <-time.After(5 * time.Second):
		t.Fatal("read-back did not publish a result")
	}
	if !ok {
		t.Fatal("the channel must not be closed")
	}
	select {
	case _, ok := <-ch:
		if !ok {
			t.Fatal("the channel must not be closed after the value is received")
		}
		t.Fatal("the channel must not receive a second value")
	default:
	}
	return err
}

func TestReadPixelsAsyncDoesNotFlush(t *testing.T) {
	s := newTestReadPixelsSetup(t, true)

	pixels := make([]byte, 4*4*4)
	ch := s.readPixelsAsync(pixels)
	s.sync()

	// The read-back must not have been started before a flush. That would mean the call waited for
	// the render thread, which is exactly what an asynchronous read-back must avoid.
	if s.driver.reads != 0 {
		t.Errorf("the read-back was started before a flush: reads = %d, want 0", s.driver.reads)
	}
	select {
	case err := <-ch:
		t.Fatalf("the result was published before a flush: %v", err)
	default:
	}

	if err := s.flush(graphicsdriver.FlushModeEndFrame); err != nil {
		t.Fatal(err)
	}
	if s.driver.reads != 1 || s.driver.asyncs != 1 {
		t.Fatalf("the read-back was not started at the flush: reads = %d, asyncs = %d", s.driver.reads, s.driver.asyncs)
	}
	select {
	case err := <-ch:
		t.Fatalf("the result was published although the read-back is not finished: %v", err)
	default:
	}

	s.driver.completeAll()
	if err := s.flush(graphicsdriver.FlushModeEndFrame); err != nil {
		t.Fatal(err)
	}

	if s.driver.reads != 1 || s.driver.asyncs != 1 {
		t.Errorf("reads = %d, asyncs = %d; want 1, 1", s.driver.reads, s.driver.asyncs)
	}
	if err := requireOneValue(t, ch); err != nil {
		t.Fatal(err)
	}
	for i, p := range pixels {
		if p != s.driver.readValue {
			t.Errorf("pixels[%d] = %#x, want %#x", i, p, s.driver.readValue)
		}
	}
	for _, r := range s.driver.readbacks {
		if r.copies != 1 {
			t.Errorf("copies = %d, want 1", r.copies)
		}
		if r.discards != 1 {
			t.Errorf("discards = %d, want 1", r.discards)
		}
	}
	if n := s.manager.PendingReadPixelsForTesting(); n != 0 {
		t.Errorf("pending read-backs = %d, want 0", n)
	}
}

// The result must not be published before the read-back is finished, so that a caller can rely on
// the pixels being the captured ones.
func TestReadPixelsAsyncWaitsForTheReadback(t *testing.T) {
	s := newTestReadPixelsSetup(t, true)

	// The capture must see the drawing operations before it and not the ones after it.
	s.driver.readValue = 0x11
	pixels := make([]byte, 4*4*4)
	ch := s.readPixelsAsync(pixels)

	// Flushing without finishing the read-back must not publish the result.
	for range 3 {
		if err := s.flush(graphicsdriver.FlushModeEndFrame); err != nil {
			t.Fatal(err)
		}
	}
	select {
	case err := <-ch:
		t.Fatalf("the result was published before the read-back finished: %v", err)
	default:
	}
	if s.driver.polls == 0 {
		t.Error("the pending read-back was not polled")
	}

	s.driver.completeAll()
	if err := s.flush(graphicsdriver.FlushModeEndFrame); err != nil {
		t.Fatal(err)
	}

	if err := requireOneValue(t, ch); err != nil {
		t.Fatal(err)
	}
	for i, p := range pixels {
		if p != 0x11 {
			t.Fatalf("pixels[%d] = %#x, want %#x: the read-back must capture the value at submission", i, p, 0x11)
		}
	}
}

// A graphics driver that does not implement AsyncPixelsReader must still complete the read-back.
func TestReadPixelsAsyncSynchronousFallback(t *testing.T) {
	s := newTestReadPixelsSetup(t, false)
	s.driver.readValue = 0x33

	pixels := make([]byte, 4*4*4)
	ch := s.readPixelsAsync(pixels)

	if err := s.flush(graphicsdriver.FlushModeEndFrame); err != nil {
		t.Fatal(err)
	}

	if s.driver.reads != 1 || s.driver.asyncs != 0 {
		t.Errorf("reads = %d, asyncs = %d; want 1, 0", s.driver.reads, s.driver.asyncs)
	}
	if err := requireOneValue(t, ch); err != nil {
		t.Fatal(err)
	}
	for i, p := range pixels {
		if p != 0x33 {
			t.Fatalf("pixels[%d] = %#x, want %#x", i, p, 0x33)
		}
	}
	if n := s.manager.PendingReadPixelsForTesting(); n != 0 {
		t.Errorf("pending read-backs = %d, want 0", n)
	}
}

// Ignoring the result channel must not prevent the read-back from completing, as publishing the
// result must never block.
func TestReadPixelsAsyncIgnoredResult(t *testing.T) {
	s := newTestReadPixelsSetup(t, true)
	s.driver.readValue = 0x44

	const count = 16
	chans := make([]<-chan error, count)
	for i := range chans {
		chans[i] = s.readPixelsAsync(make([]byte, 4*4*4))
	}
	if err := s.flush(graphicsdriver.FlushModeEndFrame); err != nil {
		t.Fatal(err)
	}
	s.driver.completeAll()
	if err := s.flush(graphicsdriver.FlushModeEndFrame); err != nil {
		t.Fatal(err)
	}

	// Every result must have been published even though no channel was received yet.
	for i, ch := range chans {
		select {
		case err := <-ch:
			if err != nil {
				t.Errorf("read-back %d: %v", i, err)
			}
		default:
			t.Errorf("read-back %d was not completed although its result channel was ignored", i)
		}
	}
}

// A pending read-back must always receive a result, even when the graphics driver is broken.
func TestReadPixelsAsyncAbortedByFlushError(t *testing.T) {
	s := newTestReadPixelsSetup(t, true)

	ch := s.readPixelsAsync(make([]byte, 4*4*4))

	// Submit the read-back, then break the driver.
	if err := s.flush(graphicsdriver.FlushModeIntermediate); err != nil {
		t.Fatal(err)
	}
	if len(s.driver.readbacks) != 1 {
		t.Fatalf("read-backs = %d, want 1", len(s.driver.readbacks))
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
		t.Fatal("the flush of a broken driver did not report an error")
	}

	if rerr := requireOneValue(t, ch); rerr == nil {
		t.Error("an aborted read-back must report an error")
	}
	if n := s.manager.PendingReadPixelsForTesting(); n != 0 {
		t.Errorf("pending read-backs = %d, want 0", n)
	}
	for _, r := range s.driver.readbacks {
		if r.discards != 1 {
			t.Errorf("discards = %d, want 1: the read-back must release its resources", r.discards)
		}
	}
}

// An error reported by the read-back itself must be published.
func TestReadPixelsAsyncReadbackError(t *testing.T) {
	s := newTestReadPixelsSetup(t, true)

	ch := s.readPixelsAsync(make([]byte, 4*4*4))
	if err := s.flush(graphicsdriver.FlushModeIntermediate); err != nil {
		t.Fatal(err)
	}

	for _, r := range s.driver.readbacks {
		r.err = errors.New("test: read-back failed")
		r.done = true
	}
	if err := s.flush(graphicsdriver.FlushModeEndFrame); err != nil {
		t.Fatal(err)
	}

	if err := requireOneValue(t, ch); err == nil {
		t.Error("a failed read-back must report an error")
	}
}

// An error starting the read-back must be published instead of being reported as a flush error, as
// the caller receives the result via the channel.
func TestReadPixelsAsyncSubmitError(t *testing.T) {
	s := newTestReadPixelsSetup(t, true)
	s.driver.asyncErr = errors.New("test: cannot start a read-back")

	ch := s.readPixelsAsync(make([]byte, 4*4*4))
	if err := s.flush(graphicsdriver.FlushModeEndFrame); err != nil {
		t.Fatal(err)
	}

	if err := requireOneValue(t, ch); err == nil {
		t.Error("a read-back that cannot be started must report an error")
	}
}

// Several read-backs must not share their pixel buffers.
func TestReadPixelsAsyncMultipleReadbacks(t *testing.T) {
	s := newTestReadPixelsSetup(t, true)
	s.driver.readValue = 0x55

	const count = 8
	chans := make([]<-chan error, count)
	pixels := make([][]byte, count)
	for i := range count {
		pixels[i] = make([]byte, 4*4*4)
		chans[i] = s.readPixelsAsync(pixels[i])
	}
	if err := s.flush(graphicsdriver.FlushModeEndFrame); err != nil {
		t.Fatal(err)
	}
	s.driver.completeAll()
	if err := s.flush(graphicsdriver.FlushModeEndFrame); err != nil {
		t.Fatal(err)
	}

	for i, ch := range chans {
		if err := requireOneValue(t, ch); err != nil {
			t.Fatal(err)
		}
		for j, p := range pixels[i] {
			if p != s.driver.readValue {
				t.Fatalf("read-back %d: pixels[%d] = %#x, want %#x", i, j, p, s.driver.readValue)
			}
		}
	}
	if len(s.driver.readbacks) != count {
		t.Errorf("read-backs = %d, want %d", len(s.driver.readbacks), count)
	}
}

// A command that fails before the read-back command is a problem for the read-back: the command
// never runs, so the read-back must still report a result.
func TestReadPixelsAsyncAbortedByEarlierCommand(t *testing.T) {
	s := newTestReadPixelsSetup(t, true)

	// The failing command is enqueued first, so that the read-back command is left over when the
	// flush fails.
	s.manager.EnqueueCommandForTesting(&failingCommand{})
	ch := s.readPixelsAsync(make([]byte, 4*4*4))
	if err := s.manager.FlushForTesting(s.driver, graphicsdriver.FlushModeEndFrame); err != nil {
		t.Fatal(err)
	}
	s.sync()

	if rerr := requireOneValue(t, ch); rerr == nil {
		t.Error("a read-back that never ran must report an error")
	}
	if s.driver.reads != 0 {
		t.Errorf("the read-back ran although the flush failed earlier: reads = %d, want 0", s.driver.reads)
	}
	if n := s.manager.PendingReadPixelsForTesting(); n != 0 {
		t.Errorf("pending read-backs = %d, want 0", n)
	}
}

func TestReadPixelsAsyncAbortedByBeginError(t *testing.T) {
	s := newTestReadPixelsSetup(t, true)
	ch := s.readPixelsAsync(make([]byte, 4*4*4))
	s.driver.beginErr = errors.New("test: Begin failed")
	_ = s.flush(graphicsdriver.FlushModeEndFrame)
	if err := requireOneValue(t, ch); !errors.Is(err, s.driver.beginErr) {
		t.Fatalf("got %v, want %v", err, s.driver.beginErr)
	}
}

func TestReadPixelsAsyncQueuedAfterFlushError(t *testing.T) {
	s := newTestReadPixelsSetup(t, true)
	s.driver.endErr = errors.New("test: End failed")
	_ = s.flush(graphicsdriver.FlushModeEndFrame)
	ch := s.readPixelsAsync(make([]byte, 4*4*4))
	if err := s.flush(graphicsdriver.FlushModeEndFrame); err == nil {
		t.Fatal("expected flush error")
	}
	if err := requireOneValue(t, ch); !errors.Is(err, s.driver.endErr) {
		t.Fatalf("got %v, want %v", err, s.driver.endErr)
	}
}
