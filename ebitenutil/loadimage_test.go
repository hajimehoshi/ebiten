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
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

func TestNewImageFromURLHTTPError(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		if err := png.Encode(w, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
			t.Errorf("png.Encode failed: %v", err)
		}
	}))
	defer s.Close()

	_, err := ebitenutil.NewImageFromURL(s.URL)
	if err == nil {
		t.Fatal("NewImageFromURL returned nil error for HTTP 404")
	}
}
