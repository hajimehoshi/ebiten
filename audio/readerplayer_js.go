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

package audio

import (
	"errors"
	"io"
	"reflect"
	"runtime"
	"syscall/js"
	"unsafe"

	"github.com/hajimehoshi/ebiten/v2/audio/internal/go2cpp"
)

func isReaderContextAvailable() bool {
	return true
}

type readerDriverImpl struct {
	context      *Context
	audioContext js.Value
	ready        bool
	callbacks    map[string]js.Func
}

func newReaderDriverImpl(context *Context) (readerDriver, error) {
	if js.Global().Get("go2cpp").Truthy() {
		return &go2cppDriverWrapper{go2cpp.NewContext(context.sampleRate, channelNum, bitDepthInBytes)}, nil
	}

	class := js.Global().Get("AudioContext")
	if !class.Truthy() {
		class = js.Global().Get("webkitAudioContext")
	}
	if !class.Truthy() {
		return nil, errors.New("audio: AudioContext or webkitAudioContext was not found")
	}
	options := js.Global().Get("Object").New()
	options.Set("sampleRate", context.sampleRate)

	d := &readerDriverImpl{
		context:      context,
		audioContext: class.New(options),
	}

	setCallback := func(event string) js.Func {
		var f js.Func
		f = js.FuncOf(func(this js.Value, arguments []js.Value) interface{} {
			if !d.ready {
				d.audioContext.Call("resume")
				d.ready = true
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

type readerPlayerState int

const (
	readerPlayerPaused readerPlayerState = iota
	readerPlayerPlay
	readerPlayerClosed
)

type readerDriverPlayerImpl struct {
	driver *readerDriverImpl
	src    io.Reader
	eof    bool
	state  readerPlayerState
	gain   js.Value

	nextPos           float64
	bufferSourceNodes []js.Value
	appendBufferFunc  js.Func
}

func (d *readerDriverImpl) NewPlayer(src io.Reader) readerDriverPlayer {
	p := &readerDriverPlayerImpl{
		driver: d,
		src:    src,
		gain:   d.audioContext.Call("createGain"),
	}
	p.appendBufferFunc = js.FuncOf(p.appendBuffer)
	p.gain.Call("connect", d.audioContext.Get("destination"))
	runtime.SetFinalizer(p, (*readerDriverPlayerImpl).Close)
	return p
}

func (d *readerDriverImpl) Close() error {
	// TODO: Implement this
	return nil
}

func (p *readerDriverPlayerImpl) Pause() {
	if p.state != readerPlayerPlay {
		return
	}

	// Change the state first. appendBuffer is called as an 'ended' callback.
	for _, n := range p.bufferSourceNodes {
		n.Set("onended", nil)
		n.Call("stop")
		n.Call("disconnect")
	}
	p.state = readerPlayerPaused
	p.bufferSourceNodes = p.bufferSourceNodes[:0]
	p.nextPos = 0
}

func (p *readerDriverPlayerImpl) appendBuffer(this js.Value, args []js.Value) interface{} {
	// appendBuffer is called as the 'ended' callback of a buffer.
	// 'this' is an AudioBufferSourceNode that already finishes its playing.
	for i, n := range p.bufferSourceNodes {
		if n.Equal(this) {
			p.bufferSourceNodes = append(p.bufferSourceNodes[:i], p.bufferSourceNodes[i+1:]...)
			break
		}
	}

	if p.state != readerPlayerPlay {
		return nil
	}

	if p.eof {
		if len(p.bufferSourceNodes) == 0 {
			p.Pause()
		}
		return nil
	}

	c := p.driver.audioContext.Get("currentTime").Float()
	if p.nextPos < c {
		// The exact current time might be too early. Add some delay on purpose to avoid buffer overlapping.
		p.nextPos = c + 1.0/60.0
	}

	bs := make([]byte, oneBufferSize(p.driver.context.sampleRate))
	n, err := io.ReadFull(p.src, bs)
	if err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			p.driver.context.setError(err)
			return nil
		}
		p.eof = true
	}
	if n == 0 {
		return nil
	}
	bs = bs[:n]
	l, r := toLR(bs)
	tl, tr := float32SliceToTypedArray(l), float32SliceToTypedArray(r)

	buf := p.driver.audioContext.Call("createBuffer", channelNum, len(bs)/channelNum/bitDepthInBytes, p.driver.context.sampleRate)
	if buf.Get("copyToChannel").Truthy() {
		buf.Call("copyToChannel", tl, 0, 0)
		buf.Call("copyToChannel", tr, 1, 0)
	} else {
		// copyToChannel is not defined on Safari 11.
		buf.Call("getChannelData", 0).Call("set", tl)
		buf.Call("getChannelData", 1).Call("set", tr)
	}

	s := p.driver.audioContext.Call("createBufferSource")
	s.Set("buffer", buf)
	s.Set("onended", p.appendBufferFunc)
	s.Call("connect", p.gain)
	s.Call("start", p.nextPos)
	p.nextPos += buf.Get("duration").Float()
	p.bufferSourceNodes = append(p.bufferSourceNodes, s)

	return nil
}

func (p *readerDriverPlayerImpl) Play() {
	if p.state != readerPlayerPaused {
		return
	}
	p.state = readerPlayerPlay
	p.appendBuffer(js.Undefined(), nil)
	p.appendBuffer(js.Undefined(), nil)
}

func (p *readerDriverPlayerImpl) IsPlaying() bool {
	return p.state == readerPlayerPlay
}

func (p *readerDriverPlayerImpl) Reset() {
	if p.state == readerPlayerClosed {
		return
	}

	p.Pause()
	p.eof = false
}

func (p *readerDriverPlayerImpl) Volume() float64 {
	return p.gain.Get("gain").Get("value").Float()
}

func (p *readerDriverPlayerImpl) SetVolume(volume float64) {
	p.gain.Get("gain").Set("value", volume)
}

func (p *readerDriverPlayerImpl) UnplayedBufferSize() int64 {
	// This is not an accurate buffer size as part of the buffers might already be consumed.
	var sec float64
	for _, n := range p.bufferSourceNodes {
		sec += n.Get("buffer").Get("duration").Float()
	}
	return int64(sec * float64(p.driver.context.sampleRate*channelNum*bitDepthInBytes))
}

func (p *readerDriverPlayerImpl) Close() error {
	runtime.SetFinalizer(p, nil)
	p.Reset()
	p.state = readerPlayerClosed
	p.appendBufferFunc.Release()
	return nil
}

type go2cppDriverWrapper struct {
	c *go2cpp.Context
}

func (w *go2cppDriverWrapper) NewPlayer(r io.Reader) readerDriverPlayer {
	return w.c.NewPlayer(r)
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
