// Copyright 2021 The Ebiten Authors
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

package readerdriver

import (
	"errors"
	"io"
	"reflect"
	"runtime"
	"syscall/js"
	"unsafe"

	"github.com/hajimehoshi/ebiten/v2/audio/internal/go2cpp"
	"github.com/hajimehoshi/ebiten/v2/internal/jsutil"
)

func IsAvailable() bool {
	return true
}

type contextImpl struct {
	audioContext js.Value
	ready        bool
	callbacks    map[string]js.Func

	sampleRate      int
	channelNum      int
	bitDepthInBytes int
}

func NewContext(sampleRate int, channelNum int, bitDepthInBytes int, onReady func()) (Context, error) {
	if js.Global().Get("go2cpp").Truthy() {
		defer onReady()
		return &go2cppDriverWrapper{go2cpp.NewContext(sampleRate, channelNum, bitDepthInBytes)}, nil
	}

	class := js.Global().Get("AudioContext")
	if !class.Truthy() {
		class = js.Global().Get("webkitAudioContext")
	}
	if !class.Truthy() {
		return nil, errors.New("readerdriver: AudioContext or webkitAudioContext was not found")
	}
	options := js.Global().Get("Object").New()
	options.Set("sampleRate", sampleRate)

	d := &contextImpl{
		audioContext:    class.New(options),
		sampleRate:      sampleRate,
		channelNum:      channelNum,
		bitDepthInBytes: bitDepthInBytes,
	}

	setCallback := func(event string) js.Func {
		var f js.Func
		f = js.FuncOf(func(this js.Value, arguments []js.Value) interface{} {
			if !d.ready {
				d.audioContext.Call("resume")
				d.ready = true
				onReady()
			}
			js.Global().Get("document").Call("removeEventListener", event, f)
			return nil
		})
		js.Global().Get("document").Call("addEventListener", event, f)
		d.callbacks[event] = f
		return f
	}

	// Browsers require user interaction to start the audio.
	// https://developers.google.com/web/updates/2017/09/autoplay-policy-changes#webaudio
	d.callbacks = map[string]js.Func{}
	setCallback("touchend")
	setCallback("keyup")
	setCallback("mouseup")

	return d, nil
}

type playerImpl struct {
	context *contextImpl
	src     io.Reader
	eof     bool
	state   playerState
	gain    js.Value
	err     error
	buf     []byte

	nextPos           float64
	bufferSourceNodes []js.Value
	appendBufferFunc  js.Func
}

func (c *contextImpl) NewPlayer(src io.Reader) Player {
	p := &playerImpl{
		context: c,
		src:     src,
		gain:    c.audioContext.Call("createGain"),
	}
	p.appendBufferFunc = js.FuncOf(p.appendBuffer)
	p.gain.Call("connect", c.audioContext.Get("destination"))
	runtime.SetFinalizer(p, (*playerImpl).Close)
	return p
}

func (c *contextImpl) Close() error {
	// TODO: Implement this
	return nil
}

// TODO: The term 'buffer' is confusing. Name each buffer with good terms.

// oneBufferSize returns the size of one buffer in the player implementation.
func (c *contextImpl) oneBufferSize() int {
	bytesPerSample := c.channelNum * c.bitDepthInBytes
	s := c.sampleRate * bytesPerSample / 4

	// Align s in multiples of bytes per sample, or a buffer could have extra bytes.
	return s / bytesPerSample * bytesPerSample
}

// maxBufferSize returns the maximum size of the buffer for the audio source.
// This buffer is used when unreading on pausing the player.
func (c *contextImpl) MaxBufferSize() int {
	// The number of underlying buffers should be 2.
	return c.oneBufferSize() * 2
}

func (c *contextImpl) Suspend() {
	c.audioContext.Call("suspend")
}

func (c *contextImpl) Resume() {
	c.audioContext.Call("resume")
}

func (p *playerImpl) Pause() {
	if p.state != playerPlay {
		return
	}

	// Change the state first. appendBuffer is called as an 'ended' callback.
	for _, n := range p.bufferSourceNodes {
		n.Set("onended", nil)
		n.Call("stop")
		n.Call("disconnect")
	}
	p.state = playerPaused
	p.bufferSourceNodes = p.bufferSourceNodes[:0]
	p.nextPos = 0
}

func (p *playerImpl) appendBuffer(this js.Value, args []js.Value) interface{} {
	// appendBuffer is called as the 'ended' callback of a buffer.
	// 'this' is an AudioBufferSourceNode that already finishes its playing.
	for i, n := range p.bufferSourceNodes {
		if jsutil.Equal(n, this) {
			p.bufferSourceNodes = append(p.bufferSourceNodes[:i], p.bufferSourceNodes[i+1:]...)
			break
		}
	}

	if p.state != playerPlay {
		return nil
	}

	if p.eof {
		if len(p.bufferSourceNodes) == 0 {
			p.Pause()
		}
		return nil
	}

	c := p.context.audioContext.Get("currentTime").Float()
	if p.nextPos < c {
		// The exact current time might be too early. Add some delay on purpose to avoid buffer overlapping.
		p.nextPos = c + 1.0/60.0
	}

	tmp := make([]byte, 4096)
	bs := make([]byte, 0, p.context.oneBufferSize())
	for cap(bs)-len(bs) > 0 {
		if len(p.buf) > 0 {
			n := len(p.buf)
			if need := cap(bs) - len(bs); n > need {
				n = need
			}
			bs = append(bs, p.buf[:n]...)
			p.buf = p.buf[n:]
			continue
		}
		n, err := p.src.Read(tmp)
		if err != nil && err != io.EOF {
			p.err = err
			p.Pause()
			return nil
		}
		if need := cap(bs) - len(bs); n > need {
			p.buf = append(p.buf, tmp[need:]...)
			n = need
		}
		bs = append(bs, tmp[:n]...)
		if err == io.EOF {
			p.eof = true
			break
		}
	}

	if len(bs) == 0 {
		return nil
	}

	l, r := toLR(bs)
	tl, tr := float32SliceToTypedArray(l), float32SliceToTypedArray(r)

	buf := p.context.audioContext.Call("createBuffer", p.context.channelNum, len(bs)/p.context.channelNum/p.context.bitDepthInBytes, p.context.sampleRate)
	if buf.Get("copyToChannel").Truthy() {
		buf.Call("copyToChannel", tl, 0, 0)
		buf.Call("copyToChannel", tr, 1, 0)
	} else {
		// copyToChannel is not defined on Safari 11.
		buf.Call("getChannelData", 0).Call("set", tl)
		buf.Call("getChannelData", 1).Call("set", tr)
	}

	s := p.context.audioContext.Call("createBufferSource")
	s.Set("buffer", buf)
	s.Set("onended", p.appendBufferFunc)
	s.Call("connect", p.gain)
	s.Call("start", p.nextPos)
	p.nextPos += buf.Get("duration").Float()
	p.bufferSourceNodes = append(p.bufferSourceNodes, s)

	return nil
}

func (p *playerImpl) Play() {
	if p.state != playerPaused {
		return
	}
	p.state = playerPlay
	p.appendBuffer(js.Undefined(), nil)
	p.appendBuffer(js.Undefined(), nil)
}

func (p *playerImpl) IsPlaying() bool {
	return p.state == playerPlay
}

func (p *playerImpl) Reset() {
	if p.state == playerClosed {
		return
	}

	p.Pause()
	p.eof = false
	p.buf = p.buf[:0]
}

func (p *playerImpl) Volume() float64 {
	return p.gain.Get("gain").Get("value").Float()
}

func (p *playerImpl) SetVolume(volume float64) {
	p.gain.Get("gain").Set("value", volume)
}

func (p *playerImpl) UnplayedBufferSize() int64 {
	// This is not an accurate buffer size as part of the buffers might already be consumed.
	var sec float64
	for _, n := range p.bufferSourceNodes {
		sec += n.Get("buffer").Get("duration").Float()
	}
	return int64(sec * float64(p.context.sampleRate*p.context.channelNum*p.context.bitDepthInBytes))
}

func (p *playerImpl) Err() error {
	return p.err
}

func (p *playerImpl) Close() error {
	runtime.SetFinalizer(p, nil)
	p.Reset()
	p.state = playerClosed
	p.appendBufferFunc.Release()
	return nil
}

type go2cppDriverWrapper struct {
	c *go2cpp.Context
}

func (w *go2cppDriverWrapper) NewPlayer(r io.Reader) Player {
	return w.c.NewPlayer(r)
}

func (w *go2cppDriverWrapper) MaxBufferSize() int {
	return w.c.MaxBufferSize()
}

func (w *go2cppDriverWrapper) Suspend() {
	// Do nothing so far.
}

func (w *go2cppDriverWrapper) Resume() {
	// Do nothing so far.
}

func (w *go2cppDriverWrapper) Close() error {
	return w.c.Close()
}

func toLR(data []byte) ([]float32, []float32) {
	const max = 1 << 15

	l := make([]float32, len(data)/4)
	r := make([]float32, len(data)/4)
	for i := 0; i < len(data)/4; i++ {
		l[i] = float32(int16(data[4*i])|int16(data[4*i+1])<<8) / max
		r[i] = float32(int16(data[4*i+2])|int16(data[4*i+3])<<8) / max
	}
	return l, r
}

func float32SliceToTypedArray(s []float32) js.Value {
	h := (*reflect.SliceHeader)(unsafe.Pointer(&s))
	h.Len *= 4
	h.Cap *= 4
	bs := *(*[]byte)(unsafe.Pointer(h))

	a := js.Global().Get("Uint8Array").New(len(bs))
	js.CopyBytesToJS(a, bs)
	runtime.KeepAlive(s)
	buf := a.Get("buffer")
	return js.Global().Get("Float32Array").New(buf, a.Get("byteOffset"), a.Get("byteLength").Int()/4)
}
