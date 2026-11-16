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
	"log/slog"
	"slices"
	"sync"
	"sync/atomic"
)

const IsDebug = true

var theFrameLogger atomic.Pointer[frameLogger]

func init() {
	theFrameLogger.Store(&frameLogger{})
}

var flushM sync.Mutex

// FrameLog calls the current global logger's FrameLog.
// FrameLog buffers a message and its key-value pairs until Flush is called.
// You can dump logs by calling SwitchFrameLogger and Flush.
// Messages are emitted at INFO level through slog.Default so that a debug build produces logs by default.
//
// FrameLog is concurrent-safe.
func FrameLog(msg string, args ...any) {
	theFrameLogger.Load().FrameLog(msg, args...)
}

// SwitchFrameLogger sets a new logger as the current logger and returns the original global logger.
// The new global logger and the returned logger have separate statuses, so you can use them for different goroutines.
//
// SwitchFrameLogger is concurrent-safe.
func SwitchFrameLogger() FrameLogger {
	return theFrameLogger.Swap(&frameLogger{})
}

type frameLogger struct {
	items []logItem
	mu    sync.Mutex
}

type logItem struct {
	msg  string
	args []any
}

func (l *frameLogger) FrameLog(msg string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.items = append(l.items, logItem{
		msg:  msg,
		args: slices.Clone(args),
	})
}

func (l *frameLogger) Flush() {
	// Flushing is protected by a mutex not to mix another logger's logs.
	flushM.Lock()
	defer flushM.Unlock()

	logger := slog.Default()
	for _, item := range l.takeItems() {
		logger.Info(item.msg, item.args...)
	}
}

// takeItems removes the buffered log items and returns them.
func (l *frameLogger) takeItems() []logItem {
	l.mu.Lock()
	defer l.mu.Unlock()

	items := l.items
	l.items = nil
	return items
}
