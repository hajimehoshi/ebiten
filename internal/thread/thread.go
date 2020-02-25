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

package thread

import (
	"context"
)

type functionType int

const (
	type0ParamsReturnError functionType = iota
	type1ParamsReturnBool
	type2ParamsReturnBool
	typeReturn2
)

const MaxPublicParams = 2
type call struct {
	funcType functionType
	func0ReturnsError func() error
	func1ReturnsBool func(uintptr) bool
	func2ReturnsBool func(uintptr, uintptr) bool
	funcReturns2 func() (uintptr, uintptr)
	params [MaxPublicParams]uintptr
}

type result struct {
	err error
	flag bool
	result1 uintptr
	result2 uintptr
}

// Thread represents an OS thread.
type Thread struct {
	calls   chan call
	results chan result
	closed  chan struct{}
}

// New creates a new thread.
//
// It is assumed that the OS thread is fixed by runtime.LockOSThread when New is called.
func New() *Thread {
	return &Thread{
		calls:   make(chan call),
		results: make(chan result),
		closed:  make(chan struct{}),
	}
}

// Loop starts the thread loop.
//
// Loop must be called on the thread.
func (t *Thread) Loop(context context.Context) {
	defer close(t.closed)
	defer close(t.results)
	defer close(t.calls)
loop:
	for {
		select {
		case c := <-t.calls:
			var callResult result
			switch c.funcType {
			case type0ParamsReturnError:
				callResult.err = c.func0ReturnsError()
			case type1ParamsReturnBool:
				callResult.flag = c.func1ReturnsBool(c.params[0])
			case type2ParamsReturnBool:
				callResult.flag = c.func2ReturnsBool(c.params[0], c.params[1])
			case typeReturn2:
				callResult.result1, callResult.result2 = c.funcReturns2()
			}
			t.results <- callResult
		case <-context.Done():
			break loop
		}
	}
}

// Call calls f on the thread.
//
// Do not call this from the same thread. This would block forever.
//
// Call panics when Loop already ends.
func (t *Thread) Call(f func() error) error {
	thisCall := call{
		funcType: type0ParamsReturnError,
		func0ReturnsError: f,
	}
	select {
	case t.calls <- thisCall:
		result := <-t.results
		return result.err
	case <-t.closed:
		panic("thread: this thread is already terminated")
	}
}

// Call calls f on the thread.
//
// Do not call this from the same thread. This would block forever.
//
// Call panics when Loop already ends.
func (t *Thread) BoolCall1(param1 uintptr, f func(uintptr) bool) bool {
	thisCall := call{
		funcType: type1ParamsReturnBool,
		func1ReturnsBool: f,
		params: [MaxPublicParams]uintptr{param1, 0},
	}
	select {
	case t.calls <- thisCall:
		result := <-t.results
		return result.flag
	case <-t.closed:
		panic("thread: this thread is already terminated")
	}
}

// Call calls f on the thread.
//
// Do not call this from the same thread. This would block forever.
//
// Call panics when Loop already ends.
func (t *Thread) BoolCall2(param1, param2 uintptr, f func(uintptr, uintptr) bool) bool {
	thisCall := call{
		funcType: type2ParamsReturnBool,
		func2ReturnsBool: f,
		params: [MaxPublicParams]uintptr{param1, param2},
	}
	select {
	case t.calls <- thisCall:
		result := <-t.results
		return result.flag
	case <-t.closed:
		panic("thread: this thread is already terminated")
	}
}

// Call calls f on the thread.
//
// Do not call this from the same thread. This would block forever.
//
// Call panics when Loop already ends.
func (t *Thread) CallReturn2(f func() (uintptr, uintptr)) (uintptr, uintptr) {
	thisCall := call{
		funcType: typeReturn2,
		funcReturns2: f,
	}
	select {
	case t.calls <- thisCall:
		result := <-t.results
		return result.result1, result.result2
	case <-t.closed:
		panic("thread: this thread is already terminated")
	}
}
