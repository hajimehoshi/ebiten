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

package vector_test

import (
	"bytes"
	"fmt"
	"image/color"
	"math"
	"reflect"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func checkRadiusPanic(t *testing.T, want bool, f func()) {
	t.Helper()
	defer func() {
		if got := recover() != nil; got != want {
			t.Errorf("panicked: %v, want: %v", got, want)
		}
	}()
	f()
}

func TestCircleNegativeRadius(t *testing.T) {
	draws := []struct {
		name string
		draw func(*ebiten.Image, float32, float32, float32, bool)
	}{
		{
			name: "FillCircle",
			draw: func(dst *ebiten.Image, x, y, r float32, aa bool) {
				vector.FillCircle(dst, x, y, r, color.White, aa)
			},
		},
		{
			name: "DrawFilledCircle",
			draw: func(dst *ebiten.Image, x, y, r float32, aa bool) {
				vector.DrawFilledCircle(dst, x, y, r, color.White, aa)
			},
		},
	}
	for _, width := range []float32{2, 24, 0, -1, float32(math.NaN()), float32(math.Inf(1))} {
		draws = append(draws, struct {
			name string
			draw func(*ebiten.Image, float32, float32, float32, bool)
		}{
			name: fmt.Sprintf("StrokeCircle/width=%v", width),
			draw: func(dst *ebiten.Image, x, y, r float32, aa bool) {
				vector.StrokeCircle(dst, x, y, r, width, color.White, aa)
			},
		})
	}
	for _, draw := range draws {
		for _, aa := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/AA=%v", draw.name, aa), func(t *testing.T) {
				dst := ebiten.NewImage(64, 64)
				defer dst.Deallocate()
				dst.Fill(color.RGBA{R: 40, G: 80, B: 120, A: 255})
				before := make([]byte, 64*64*4)
				dst.ReadPixels(before)
				checkRadiusPanic(t, true, func() { draw.draw(dst, 32, 32, -10, aa) })
				after := make([]byte, len(before))
				dst.ReadPixels(after)
				if !bytes.Equal(after, before) {
					t.Error("negative radius changed the destination")
				}
				// Non-finite geometry takes precedence over the negative radius.
				for _, nonfinite := range []float32{float32(math.NaN()), float32(math.Inf(1)), float32(math.Inf(-1))} {
					for arg := range 3 {
						args := [3]float32{32, 32, -10}
						args[arg] = nonfinite
						checkRadiusPanic(t, false, func() { draw.draw(dst, args[0], args[1], args[2], aa) })
					}
				}
				dst.ReadPixels(after)
				if !bytes.Equal(after, before) {
					t.Error("non-finite geometry changed the destination")
				}
			})
		}
	}
}

func TestArcNegativeRadius(t *testing.T) {
	arcs := []struct {
		name string
		args [5]float32
		call func(*vector.Path, [5]float32)
	}{
		{
			name: "Arc",
			args: [5]float32{10, 20, -10, 0, math.Pi},
			call: func(p *vector.Path, a [5]float32) {
				p.Arc(a[0], a[1], a[2], a[3], a[4], vector.Clockwise)
			},
		},
		{
			name: "ArcTo",
			args: [5]float32{10, 20, 30, 40, -10},
			call: func(p *vector.Path, a [5]float32) {
				p.ArcTo(a[0], a[1], a[2], a[3], a[4])
			},
		},
	}
	for _, arc := range arcs {
		for _, state := range []string{"empty", "open", "closed"} {
			t.Run(arc.name+"/"+state, func(t *testing.T) {
				var p, want vector.Path
				for _, path := range []*vector.Path{&p, &want} {
					if state != "empty" {
						path.MoveTo(1, 2)
						path.LineTo(10, 20)
						path.LineTo(30, 4)
						if state == "closed" {
							path.Close()
						}
					}
				}
				before := vector.PathOperationsString(&p)
				for _, nonfinite := range []float32{float32(math.NaN()), float32(math.Inf(1)), float32(math.Inf(-1))} {
					for arg := range arc.args {
						args := arc.args
						args[arg] = nonfinite
						checkRadiusPanic(t, false, func() { arc.call(&p, args) })
					}
				}
				if got := vector.PathOperationsString(&p); got != before {
					t.Error("non-finite arc arguments changed the path")
				}
				checkRadiusPanic(t, true, func() { arc.call(&p, arc.args) })
				if arc.name == "ArcTo" && state == "empty" {
					want.MoveTo(arc.args[0], arc.args[1])
				}
				// Continuing the path checks its current position and closure after rejection.
				for _, path := range []*vector.Path{&p, &want} {
					path.LineTo(40, 50)
					path.LineTo(5, 60)
				}
				op := &vector.StrokeOptions{
					Width: 2,
				}
				gotVertices, gotIndices := p.AppendVerticesAndIndicesForStroke(nil, nil, op)
				wantVertices, wantIndices := want.AppendVerticesAndIndicesForStroke(nil, nil, op)
				if !reflect.DeepEqual(gotVertices, wantVertices) || !reflect.DeepEqual(gotIndices, wantIndices) {
					t.Error("unexpected stroke geometry after rejecting the arc")
				}
			})
		}
	}
}
