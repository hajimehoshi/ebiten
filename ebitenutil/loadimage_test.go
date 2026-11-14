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
	"io"
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
				if !wantError && (img == nil || original == nil || img.Bounds().Size() != size || original.Bounds().Size() != size) {
					t.Fatal("expected non-nil images with the decoded dimensions")
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
				if !wantError && (img == nil || original == nil || img.Bounds().Size() != size || original.Bounds().Size() != size) {
					t.Fatal("expected non-nil images with the decoded dimensions")
				}
				if wantError && (img != nil || original != nil) {
					t.Fatal("expected nil images on error")
				}
			})
			t.Run("URL", func(t *testing.T) {
				// Browser tests cannot connect to a Go httptest listener. Supply
				// a successful HTTP response while exercising the URL loader.
				oldClient := http.DefaultClient
				t.Cleanup(func() { http.DefaultClient = oldClient })
				http.DefaultClient = &http.Client{Transport: imageRoundTripper{data: buf.Bytes()}}
				img, err := ebitenutil.NewImageFromURL("https://example.com/image.gif")
				if wantError {
					want := fmt.Sprintf("ebitenutil: decoded image has an empty size: %dx%d", size.X, size.Y)
					if err == nil || err.Error() != want {
						t.Fatalf("error: got %v, want %q", err, want)
					}
				}
				if (err != nil) != wantError {
					t.Fatalf("error: got %v, want error: %v", err, wantError)
				}
				if !wantError && (img == nil || img.Bounds().Size() != size) {
					t.Fatal("expected a non-nil image with the decoded dimensions")
				}
				if wantError && img != nil {
					t.Fatal("expected nil image on error")
				}
			})
		})
	}
}

// imageRoundTripper serves encoded image data without requiring a network listener.
type imageRoundTripper struct {
	data []byte
}

func (r imageRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(r.data)),
		Request:    req,
	}, nil
}
