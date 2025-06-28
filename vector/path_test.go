// Copyright 2024 The Ebitengine Authors
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

package vector_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2/vector"
)

func TestIsPointCloseToSegment(t *testing.T) {
	testCases := []struct {
		p     vector.Point
		p0    vector.Point
		p1    vector.Point
		allow float32
		want  bool
	}{
		{
			p:     vector.Point{0.5, 0.5},
			p0:    vector.Point{0, 0},
			p1:    vector.Point{1, 0},
			allow: 1,
			want:  true,
		},
		{
			p:     vector.Point{0.5, 1.5},
			p0:    vector.Point{0, 0},
			p1:    vector.Point{1, 0},
			allow: 1,
			want:  false,
		},
		{
			p:     vector.Point{0.5, 0.5},
			p0:    vector.Point{0, 0},
			p1:    vector.Point{1, 1},
			allow: 0,
			want:  true,
		},
		{
			p:     vector.Point{0, 1},
			p0:    vector.Point{0, 0},
			p1:    vector.Point{1, 1},
			allow: 0.7,
			want:  false,
		},
		{
			p:     vector.Point{0, 1},
			p0:    vector.Point{0, 0},
			p1:    vector.Point{1, 1},
			allow: 0.8,
			want:  true,
		},
		{
			// p0 and p1 are the same.
			p:     vector.Point{0, 1},
			p0:    vector.Point{0.5, 0.5},
			p1:    vector.Point{0.5, 0.5},
			allow: 0.7,
			want:  false,
		},
		{
			// p0 and p1 are the same.
			p:     vector.Point{0, 1},
			p0:    vector.Point{0.5, 0.5},
			p1:    vector.Point{0.5, 0.5},
			allow: 0.8,
			want:  true,
		},
	}
	for _, tc := range testCases {
		if got := vector.IsPointCloseToSegment(tc.p, tc.p0, tc.p1, tc.allow); got != tc.want {
			t.Errorf("got: %v, want: %v", got, tc.want)
		}
	}
}

func TestMoveToAndClose(t *testing.T) {
	var path vector.Path
	if _, ok := vector.CurrentPosition(&path); ok != false {
		t.Errorf("expected no last position, got one")
	}
	if got, want := vector.SubPathCount(&path), 0; got != want {
		t.Errorf("expected close count to be %d, got %d", want, got)
	}

	path.MoveTo(10, 20)
	if p, ok := vector.CurrentPosition(&path); p != (vector.Point{10, 20}) || !ok {
		t.Errorf("expected last position to be (10, 20), got %v", p)
	}
	if got, want := vector.SubPathCount(&path), 1; got != want {
		t.Errorf("expected close count to be %d, got %d", want, got)
	}

	path.MoveTo(30, 40)
	if p, ok := vector.CurrentPosition(&path); p != (vector.Point{30, 40}) || !ok {
		t.Errorf("expected last position to be (30, 40), got %v", p)
	}
	if got, want := vector.SubPathCount(&path), 1; got != want {
		t.Errorf("expected close count to be %d, got %d", want, got)
	}

	path.LineTo(50, 60)
	if p, ok := vector.CurrentPosition(&path); p != (vector.Point{50, 60}) || !ok {
		t.Errorf("expected last position to be (50, 60), got %v", p)
	}
	if got, want := vector.SubPathCount(&path), 1; got != want {
		t.Errorf("expected close count to be %d, got %d", want, got)
	}

	path.Close()
	if p, ok := vector.CurrentPosition(&path); p != (vector.Point{30, 40}) || !ok {
		t.Errorf("expected last position to be (30, 40) after close, got %v", p)
	}
	if got, want := vector.SubPathCount(&path), 1; got != want {
		t.Errorf("expected close count to be %d, got %d", want, got)
	}

	path.MoveTo(70, 80)
	if p, ok := vector.CurrentPosition(&path); p != (vector.Point{70, 80}) || !ok {
		t.Errorf("expected last position to be (70, 80), got %v", p)
	}
	if got, want := vector.SubPathCount(&path), 2; got != want {
		t.Errorf("expected close count to be %d, got %d", want, got)
	}

	path.LineTo(90, 100)
	if p, ok := vector.CurrentPosition(&path); p != (vector.Point{90, 100}) || !ok {
		t.Errorf("expected last position to be (50, 60), got %v", p)
	}
	if got, want := vector.SubPathCount(&path), 2; got != want {
		t.Errorf("expected close count to be %d, got %d", want, got)
	}

	// MoveTo without closing forces to create a new sub-path.
	// The previous sub-path is left unclosed.
	path.MoveTo(110, 120)
	if p, ok := vector.CurrentPosition(&path); p != (vector.Point{110, 120}) || !ok {
		t.Errorf("expected last position to be (70, 80), got %v", p)
	}
	if got, want := vector.SubPathCount(&path), 3; got != want {
		t.Errorf("expected close count to be %d, got %d", want, got)
	}
}

func TestAddPath(t *testing.T) {
	var path vector.Path
	path.MoveTo(10, 20)
	path.LineTo(30, 40)
	path.Close()

	op := &vector.AddPathOptions{}
	op.GeoM.Translate(100, 100)
	var path2 vector.Path
	path2.AddPath(&path, op)

	if p, ok := vector.CurrentPosition(&path); p != (vector.Point{10, 20}) || !ok {
		t.Errorf("expected last position to be (10, 20), got %v", p)
	}
	if got, want := vector.SubPathCount(&path), 1; got != want {
		t.Errorf("expected close count to be %d, got %d", want, got)
	}
	if p, ok := vector.CurrentPosition(&path2); p != (vector.Point{110, 120}) || !ok {
		t.Errorf("expected last position to be (110, 120), got %v", p)
	}
	if got, want := vector.SubPathCount(&path2), 1; got != want {
		t.Errorf("expected close count to be %d, got %d", want, got)
	}
}

func TestAddPathSelf(t *testing.T) {
	var path vector.Path
	path.MoveTo(10, 20)
	path.LineTo(30, 40)
	path.Close()

	op := &vector.AddPathOptions{}
	op.GeoM.Translate(100, 100)
	path.AddPath(&path, op)

	if p, ok := vector.CurrentPosition(&path); p != (vector.Point{110, 120}) || !ok {
		t.Errorf("expected last position to be (110, 120), got %v", p)
	}
	if got, want := vector.SubPathCount(&path), 2; got != want {
		t.Errorf("expected close count to be %d, got %d", want, got)
	}
}
