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

package gamepad_test

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/sys/unix"

	"github.com/hajimehoshi/ebiten/v2/internal/gamepad"
)

// TestInitWithoutInotify verifies that initialization succeeds when no inotify instance can be
// created, because another program of the same user took them all (#3304).
func TestInitWithoutInotify(t *testing.T) {
	if _, err := os.Stat(gamepad.DirName); err != nil {
		t.Skipf("%s is not available, initialization never reaches inotify", gamepad.DirName)
	}

	// Reading the limit up front bounds the loop below and keeps file descriptors to spare.
	var rlimit unix.Rlimit
	if err := unix.Getrlimit(unix.RLIMIT_NOFILE, &rlimit); err != nil {
		t.Skipf("unix.Getrlimit: %v", err)
	}
	content, err := os.ReadFile("/proc/sys/fs/inotify/max_user_instances")
	if err != nil {
		t.Skipf("os.ReadFile: %v", err)
	}
	limit, err := strconv.Atoi(strings.TrimSpace(string(content)))
	if err != nil {
		t.Skipf("strconv.Atoi: %v", err)
	}
	if uint64(limit)+256 > uint64(rlimit.Cur) {
		t.Skipf("fs.inotify.max_user_instances is %d, too close to the file descriptor limit %d to exhaust", limit, rlimit.Cur)
	}

	var held []int
	defer func() {
		for _, fd := range held {
			_ = unix.Close(fd)
		}
	}()
	for len(held) <= limit {
		fd, err := unix.InotifyInit1(unix.IN_NONBLOCK | unix.IN_CLOEXEC)
		if err != nil {
			break
		}
		held = append(held, fd)
	}
	if len(held) > limit {
		t.Skip("could not exhaust the inotify instances of this user")
	}

	if err := gamepad.InitNativeGamepadsForTest(); err != nil {
		t.Errorf("gamepad.InitNativeGamepadsForTest(): got: %v, want: nil", err)
	}
}
