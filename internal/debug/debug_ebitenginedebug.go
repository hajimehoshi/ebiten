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

var completedFrameLogs struct {
	mu       sync.Mutex
	batches  [][]logItem
	flushing bool
}

// FrameLog calls the current global logger's FrameLog.
// FrameLog buffers a message and its key-value pairs for the next graphics flush.
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

// Queue hands off a completed batch from the render thread without calling the log handler.
func (l *frameLogger) Queue() {
	items := l.takeItems()
	if len(items) == 0 {
		return
	}
	completedFrameLogs.mu.Lock()
	completedFrameLogs.batches = append(completedFrameLogs.batches, items)
	completedFrameLogs.mu.Unlock()
}

// takeItems removes the buffered log items and returns them.
func (l *frameLogger) takeItems() []logItem {
	l.mu.Lock()
	defer l.mu.Unlock()

	items := l.items
	l.items = nil
	return items
}

// FlushFrameLogs invokes the application logger after rendering has handed
// off its batches. The caller must not hold the atlas lock, which a handler
// can acquire through ebiten.ReadDebugInfo.
func FlushFrameLogs() {
	logs := &completedFrameLogs
	logs.mu.Lock()
	if logs.flushing {
		// The active flush will pick up any new batches, including when a
		// handler calls back into the engine.
		logs.mu.Unlock()
		return
	}

	logs.flushing = true
	for len(logs.batches) > 0 {
		batches := logs.batches
		logs.batches = nil
		logs.mu.Unlock()

		for _, batch := range batches {
			logger := slog.Default()
			for _, item := range batch {
				logger.Info(item.msg, item.args...)
			}
		}

		logs.mu.Lock()
	}
	logs.flushing = false
	logs.mu.Unlock()
}
