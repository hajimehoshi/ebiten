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
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

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

	if _, err := ebitenutil.NewImageFromURL(s.URL); err == nil {
		t.Fatal("NewImageFromURL returned nil error for HTTP 404")
	}
}

func TestLoadImageEmptyBounds(t *testing.T) {
	for _, size := range []image.Point{{0, 1}, {1, 0}, {0, 0}, {1, 1}} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			// The standard GIF encoder and decoder both accept empty images.
			var buf bytes.Buffer
			if err := gif.Encode(&buf, image.NewPaletted(image.Rectangle{Max: size}, color.Palette{color.Black, color.White}), nil); err != nil {
				t.Fatal(err)
			}
			decoded, err := gif.Decode(bytes.NewReader(buf.Bytes()))
			if err != nil {
				t.Fatal(err)
			}
			if got := decoded.Bounds().Size(); got != size {
				t.Fatalf("decoded size: got %v, want %v", got, size)
			}
			wantError := size.X == 0 || size.Y == 0
			t.Run("Reader", func(t *testing.T) {
				img, original, err := ebitenutil.NewImageFromReader(bytes.NewReader(buf.Bytes()))
				if (err != nil) != wantError {
					t.Fatalf("error: got %v, want error: %v", err, wantError)
				}
				if wantError && (img != nil || original != nil) {
					t.Fatal("expected nil images on error")
				}
			})
			t.Run("FileSystem", func(t *testing.T) {
				fs := fstest.MapFS{"image.gif": &fstest.MapFile{Data: buf.Bytes()}}
				img, original, err := ebitenutil.NewImageFromFileSystem(fs, "image.gif")
				if (err != nil) != wantError {
					t.Fatalf("error: got %v, want error: %v", err, wantError)
				}
				if wantError && (img != nil || original != nil) {
					t.Fatal("expected nil images on error")
				}
			})
			t.Run("URL", func(t *testing.T) {
				s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, _ = w.Write(buf.Bytes())
				}))
				defer s.Close()
				img, err := ebitenutil.NewImageFromURL(s.URL)
				if (err != nil) != wantError {
					t.Fatalf("error: got %v, want error: %v", err, wantError)
				}
				if wantError && img != nil {
					t.Fatal("expected nil image on error")
				}
			})
		})
	}
}
