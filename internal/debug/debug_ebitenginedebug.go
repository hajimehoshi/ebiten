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

var theLogger = &logger{}

var flushM sync.Mutex

// Logf calls the current global logger's Logf.
// Logf buffers the arguments and doesn't dump the log immediately.
// You can dump logs by calling SwitchLogger and Flush.
//
// Logf is not concurrent safe.
func Logf(format string, args ...any) {
	theLogger.Logf(format, args...)
}

// SwitchLogger sets a new logger as the current logger and returns the original global logger.
// The new global logger and the returned logger have separate statuses, so you can use them for different goroutines.
//
// SwitchLogger and a returned Logger are not concurrent safe.
func SwitchLogger() Logger {
	current := theLogger
	theLogger = &logger{}
	return current
}

type logger struct {
	items []logItem
}

type logItem struct {
	format string
	args   []any
}

func (l *logger) Logf(format string, args ...any) {
	l.items = append(l.items, logItem{
		format: format,
		args:   args,
	})
}

func (l *logger) Flush() {
	// Flushing is protected by a mutex not to mix another logger's logs.
	flushM.Lock()
	defer flushM.Unlock()

	for i, item := range l.items {
		fmt.Printf(item.format, item.args...)
		l.items[i] = logItem{}
	}
	l.items = l.items[:0]
}
