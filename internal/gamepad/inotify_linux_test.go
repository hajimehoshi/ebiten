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

//go:build !android && !nintendosdk && !playstation5

package gamepad

import (
	"testing"

	"golang.org/x/sys/unix"
)

// Nameless inotify events (Len == 0), e.g. IN_Q_OVERFLOW or IN_IGNORED, must
// be skipped: slicing their absent name as buf[16:16+e.Len-1] panics with
// slice bounds out of range and kills the process while polling gamepads.
func TestUpdateSkipsNamelessInotifyEvents(t *testing.T) {
	dir := t.TempDir()

	fd, err := unix.InotifyInit1(unix.IN_NONBLOCK | unix.IN_CLOEXEC)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(fd)

	wd, err := unix.InotifyAddWatch(fd, dir, unix.IN_CREATE|unix.IN_ATTRIB|unix.IN_DELETE)
	if err != nil {
		t.Fatal(err)
	}

	// Removing the watch queues IN_IGNORED, a nameless (Len == 0) event with
	// the same shape as IN_Q_OVERFLOW.
	if _, err := unix.InotifyRmWatch(fd, uint32(wd)); err != nil {
		t.Fatal(err)
	}

	g := &nativeGamepadsImpl{inotify: fd, watch: wd}
	var gamepads gamepads
	if err := g.update(&gamepads); err != nil {
		t.Fatal(err)
	}
}
