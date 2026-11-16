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

package debug

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"log/slog"
	"sync"
	"testing"
)

func captureFrameLogs(t *testing.T, level slog.Level) *bytes.Buffer {
	t.Helper()
	var output bytes.Buffer
	old := slog.Default()
	oldWriter, oldFlags := log.Writer(), log.Flags()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&output, &slog.HandlerOptions{Level: level})))
	theFrameLogger.Store(&frameLogger{})
	t.Cleanup(func() {
		slog.SetDefault(old)
		log.SetOutput(oldWriter)
		log.SetFlags(oldFlags)
		theFrameLogger.Store(&frameLogger{})
	})
	return &output
}

func readFrameLogs(t *testing.T, output *bytes.Buffer) []map[string]any {
	t.Helper()
	var records []map[string]any
	decoder := json.NewDecoder(output)
	for {
		var record map[string]any
		err := decoder.Decode(&record)
		if err == io.EOF {
			return records
		}
		if err != nil {
			t.Fatal(err)
		}
		records = append(records, record)
	}
}

func TestFrameLogStructuredAndBuffered(t *testing.T) {
	if !IsDebug {
		t.Fatal("debug logging is disabled")
	}
	output := captureFrameLogs(t, slog.LevelInfo)
	logger := &frameLogger{}
	logger.FrameLog("Internal image", "id", 7, "width", 320, "screen", true)
	if output.Len() != 0 {
		t.Fatal("logged before Flush")
	}
	logger.Flush()
	logger.Flush()
	records := readFrameLogs(t, output)
	if len(records) != 1 {
		t.Fatalf("got %d records, want 1", len(records))
	}
	record := records[0]
	for key, want := range map[string]any{
		"msg": "Internal image", "level": "INFO", "id": float64(7),
		"width": float64(320), "screen": true,
	} {
		if got := record[key]; got != want {
			t.Errorf("%s = %v, want %v", key, got, want)
		}
	}
}

func TestFrameLogCopiesArguments(t *testing.T) {
	output := captureFrameLogs(t, slog.LevelInfo)
	logger := &frameLogger{}
	args := []any{"count", 2}
	logger.FrameLog("Update count per frame", args...)
	args[0] = "changed"
	args[1] = 99
	logger.Flush()
	records := readFrameLogs(t, output)
	if len(records) != 1 || records[0]["count"] != float64(2) {
		t.Fatalf("arguments changed after buffering: %v", records)
	}
}

func TestSwitchFrameLoggerKeepsBatchesSeparate(t *testing.T) {
	output := captureFrameLogs(t, slog.LevelInfo)
	FrameLog("old frame")
	old := SwitchFrameLogger()
	FrameLog("new frame")
	old.FrameLog("old graphics command")
	old.Flush()
	records := readFrameLogs(t, output)
	if len(records) != 2 || records[0]["msg"] != "old frame" || records[1]["msg"] != "old graphics command" {
		t.Fatalf("unexpected old batch: %v", records)
	}
	SwitchFrameLogger().Flush()
	records = readFrameLogs(t, output)
	if len(records) != 1 || records[0]["msg"] != "new frame" {
		t.Fatalf("unexpected new batch: %v", records)
	}
}

func TestFrameLogConcurrentWritesAndFlushes(t *testing.T) {
	output := captureFrameLogs(t, slog.LevelInfo)
	const workers = 4
	const count = 20
	loggers := []*frameLogger{{}, {}}
	var writers sync.WaitGroup
	for batch, logger := range loggers {
		for worker := range workers {
			writers.Go(func() {
				for sequence := range count {
					logger.FrameLog("command", "batch", batch, "worker", worker, "sequence", sequence)
				}
			})
		}
	}
	writers.Wait()
	var flushes sync.WaitGroup
	for _, logger := range loggers {
		flushes.Go(func() { logger.Flush() })
	}
	flushes.Wait()
	records := readFrameLogs(t, output)
	if got, want := len(records), len(loggers)*workers*count; got != want {
		t.Fatalf("got %d records, want %d", got, want)
	}
	seen := map[[3]int]bool{}
	lastBatch := records[0]["batch"]
	batchChanges := 0
	for _, record := range records {
		key := [3]int{int(record["batch"].(float64)), int(record["worker"].(float64)), int(record["sequence"].(float64))}
		if seen[key] {
			t.Fatalf("duplicate record: %v", key)
		}
		seen[key] = true
		if record["batch"] != lastBatch {
			batchChanges++
			lastBatch = record["batch"]
		}
	}
	if batchChanges != 1 {
		t.Fatalf("batches interleaved: %d batch changes", batchChanges)
	}
}

func TestFrameLogRespectsApplicationLevel(t *testing.T) {
	output := captureFrameLogs(t, slog.LevelWarn)
	FrameLog("filtered", "count", 1)
	SwitchFrameLogger().Flush()
	if output.Len() != 0 {
		t.Fatalf("application level ignored: %s", output)
	}
}
