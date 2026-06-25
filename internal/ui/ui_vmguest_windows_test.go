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

package ui_test

import (
	"errors"
	"net"
	"os"
	"syscall"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

func TestIsConnectionReset(t *testing.T) {
	// A host-side connection reset, as net reports it on Windows.
	reset := &net.OpError{
		Op:  "read",
		Net: "unix",
		Err: os.NewSyscallError("wsarecv", syscall.WSAECONNRESET),
	}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "reset",
			err:  reset,
			want: true,
		},
		{
			name: "other",
			err:  errors.New("boom"),
			want: false,
		},
		{
			name: "nil",
			err:  nil,
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ui.IsConnectionReset(tt.err); got != tt.want {
				t.Errorf("IsConnectionReset(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
