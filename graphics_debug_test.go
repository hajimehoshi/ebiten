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

//go:build ebitenginedebug || ebitendebug

package ebiten_test

import (
	"context"
	"image/color"
	"io"
	"log"
	"log/slog"
	"sync"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

type debugInfoHandler struct {
	slog.Handler
	mu       sync.Mutex
	commands map[string]map[string]slog.Value
}

func (h *debugInfoHandler) Handle(_ context.Context, record slog.Record) error {
	if record.Message != "Graphics command" {
		return nil
	}
	var info ebiten.DebugInfo
	ebiten.ReadDebugInfo(&info)

	attrs := make(map[string]slog.Value, record.NumAttrs())
	record.Attrs(func(attr slog.Attr) bool {
		attrs[attr.Key] = attr.Value
		return true
	})
	h.mu.Lock()
	defer h.mu.Unlock()
	h.commands[attrs["command"].String()] = attrs
	return nil
}

func TestFrameLogHandlerCanReadDebugInfo(t *testing.T) {
	handler := &debugInfoHandler{
		Handler:  slog.NewTextHandler(io.Discard, nil),
		commands: make(map[string]map[string]slog.Value),
	}
	old := slog.Default()
	oldWriter, oldFlags := log.Writer(), log.Flags()
	slog.SetDefault(slog.New(handler))
	t.Cleanup(func() {
		slog.SetDefault(old)
		log.SetOutput(oldWriter)
		log.SetFlags(oldFlags)
	})

	src := ebiten.NewImage(2, 2)
	defer src.Deallocate()
	src.Fill(color.White)
	dst := ebiten.NewImage(2, 2)
	defer dst.Deallocate()
	dst.DrawImage(src, nil)

	// Reading the pixels waits for queued graphics commands while the atlas is locked.
	// The handler must be able to read that same atlas before ReadPixels returns.
	var pixels [2 * 2 * 4]byte
	dst.ReadPixels(pixels[:])
	if pixels[0] != 0xff {
		t.Fatalf("first pixel's red channel = %d, want 255", pixels[0])
	}

	handler.mu.Lock()
	defer handler.mu.Unlock()
	if _, ok := handler.commands["read-pixels-async"]; !ok {
		t.Error("missing read-pixels-async command")
	}
	draw, ok := handler.commands["draw-triangles"]
	if !ok {
		t.Fatal("missing draw-triangles command")
	}
	if got := draw["indices"]; got.Kind() != slog.KindInt64 || got.Int64() != 6 {
		t.Errorf("draw-triangles indices = %v, want 6", got)
	}
}
