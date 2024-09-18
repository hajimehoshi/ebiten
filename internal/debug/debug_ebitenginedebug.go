// Copyright 2020 The Ebiten Authors
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

//go:build ebitenginedebug || ebitendebug

package debug

import (
	"fmt"
	"sync"
)

const IsDebug = true

var theFrameLogger = &frameLogger{}

var flushM sync.Mutex

// FrameLogf calls the current global logger's FrameLogf.
// FrameLogf buffers the arguments and doesn't dump the log immediately.
// You can dump logs by calling SwitchLogger and Flush.
//
// FrameLogf is not concurrent safe.
// FrameLogf and SwitchFrameLogger must be called from the same goroutine.
func FrameLogf(format string, args ...any) {
	theFrameLogger.FrameLogf(format, args...)
}

// SwitchFrameLogger sets a new logger as the current logger and returns the original global logger.
// The new global logger and the returned logger have separate statuses, so you can use them for different goroutines.
//
// SwitchFrameLogger and a returned Logger are not concurrent safe.
// FrameLogf and SwitchFrameLogger must be called from the same goroutine.
func SwitchFrameLogger() FrameLogger {
	current := theFrameLogger
	theFrameLogger = &frameLogger{}
	return current
}

type frameLogger struct {
	items []logItem
}

type logItem struct {
	format string
	args   []any
}

func (l *frameLogger) FrameLogf(format string, args ...any) {
	l.items = append(l.items, logItem{
		format: format,
		args:   args,
	})
}

func (l *frameLogger) Flush() {
	// Flushing is protected by a mutex not to mix another logger's logs.
	flushM.Lock()
	defer flushM.Unlock()

	for i, item := range l.items {
		fmt.Printf(item.format, item.args...)
		l.items[i] = logItem{}
	}
	l.items = l.items[:0]
}
