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

package vmprotocol_test

import (
	"runtime"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/vmprotocol"
)

func TestEndpointRoundTrip(t *testing.T) {
	// Use an OS-appropriate socket path so the Windows drive-letter handling is exercised on Windows.
	unixAddr := "/tmp/ebiten-vg/g.sock"
	wantUnixURL := "unix:///tmp/ebiten-vg/g.sock"
	if runtime.GOOS == "windows" {
		unixAddr = `C:\Temp\ebiten-vg\g.sock`
		wantUnixURL = "unix:///C:/Temp/ebiten-vg/g.sock"
	}

	cases := []struct {
		network string
		address string
		wantURL string
	}{
		{"unix", unixAddr, wantUnixURL},
		{"tcp", "127.0.0.1:8123", "tcp://127.0.0.1:8123"},
		{"tcp", "[::1]:8123", "tcp://[::1]:8123"},
		{"tcp", "example.com:7000", "tcp://example.com:7000"},
	}

	for _, c := range cases {
		e := vmprotocol.Endpoint{Network: c.network, Address: c.address}
		url, err := e.URL()
		if err != nil {
			t.Fatalf("(%+v).URL() failed: %v", e, err)
		}
		if url != c.wantURL {
			t.Errorf("(%+v).URL() = %q; want %q", e, url, c.wantURL)
		}

		parsed, err := vmprotocol.ParseEndpoint(url)
		if err != nil {
			t.Fatalf("ParseEndpoint(%q) failed: %v", url, err)
		}
		if parsed != e {
			t.Errorf("ParseEndpoint(%q) = %+v; want %+v", url, parsed, e)
		}
	}
}

func TestParseEndpointUnsupported(t *testing.T) {
	if _, err := vmprotocol.ParseEndpoint("ws://example.com"); err == nil {
		t.Error("ParseEndpoint with an unsupported scheme should fail")
	}
}

func TestEndpointURLUnsupported(t *testing.T) {
	e := vmprotocol.Endpoint{Network: "udp", Address: "127.0.0.1:9000"}
	if _, err := e.URL(); err == nil {
		t.Error("Endpoint.URL with an unsupported network should fail")
	}
}

func TestEndpointURLRelativeUnixPath(t *testing.T) {
	e := vmprotocol.Endpoint{Network: "unix", Address: "rel/g.sock"}
	if _, err := e.URL(); err == nil {
		t.Error("Endpoint.URL with a relative unix path should fail")
	}
}

func TestParseEndpointRelativeUnixPath(t *testing.T) {
	if _, err := vmprotocol.ParseEndpoint("unix://rel/g.sock"); err == nil {
		t.Error("ParseEndpoint with a relative unix path should fail")
	}
}

func TestEndpointURLInvalidTCPAddress(t *testing.T) {
	e := vmprotocol.Endpoint{Network: "tcp", Address: "no-port"}
	if _, err := e.URL(); err == nil {
		t.Error("Endpoint.URL with a tcp address lacking a port should fail")
	}
}

func TestParseEndpointInvalidTCPAddress(t *testing.T) {
	if _, err := vmprotocol.ParseEndpoint("tcp://no-port"); err == nil {
		t.Error("ParseEndpoint with a tcp address lacking a port should fail")
	}
}
