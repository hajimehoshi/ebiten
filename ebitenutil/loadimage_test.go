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

package ebitenutil_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

func TestNewImageFromURLHTTPError(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer s.Close()

	_, err := ebitenutil.NewImageFromURL(s.URL)
	if err == nil {
		t.Fatal("NewImageFromURL returned nil error for HTTP 404")
	}
	if got, want := err.Error(), "404 Not Found"; !strings.Contains(got, want) {
		t.Fatalf("NewImageFromURL error = %q, want to contain %q", got, want)
	}
}
