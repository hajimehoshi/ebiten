// Copyright 2018 The Ebiten Authors
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

// +build android ios
// +build gomobilebuild

package ui

import (
	"golang.org/x/mobile/app"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
	"golang.org/x/mobile/event/touch"
	"golang.org/x/mobile/gl"

	"github.com/hajimehoshi/ebiten/internal/devicescale"
	"github.com/hajimehoshi/ebiten/internal/input"
	"github.com/hajimehoshi/ebiten/internal/opengl"
)

var (
	glContextCh chan gl.Context
)

func appMain(a app.App) {
	var glctx gl.Context
	touches := map[touch.Sequence]*input.Touch{}
	for e := range a.Events() {
		switch e := a.Filter(e).(type) {
		case lifecycle.Event:
			switch e.Crosses(lifecycle.StageVisible) {
			case lifecycle.CrossOn:
				glctx, _ = e.DrawContext.(gl.Context)
				// Assume that glctx is always a same instance.
				// Then, only once initializing should be enough.
				if glContextCh != nil {
					glContextCh <- glctx
					glContextCh = nil
				}
				a.Send(paint.Event{})
			case lifecycle.CrossOff:
				glctx = nil
			}
		case size.Event:
			setFullscreen(e.WidthPx, e.HeightPx)
		case paint.Event:
			if glctx == nil || e.External {
				continue
			}
			chRender <- struct{}{}
			<-chRenderEnd
			a.Publish()
			a.Send(paint.Event{})
		case touch.Event:
			switch e.Type {
			case touch.TypeBegin, touch.TypeMove:
				s := devicescale.DeviceScale()
				x, y := float64(e.X)/s, float64(e.Y)/s
				// TODO: Is it ok to cast from int64 to int here?
				t := input.NewTouch(int(e.Sequence), int(x), int(y))
				touches[e.Sequence] = t
			case touch.TypeEnd:
				delete(touches, e.Sequence)
			}
			ts := []*input.Touch{}
			for _, t := range touches {
				ts = append(ts, t)
			}
			UpdateTouches(ts)
		}
	}
}

func RunMainThreadLoop(ch <-chan error) error {
	glContextCh = make(chan gl.Context)
	app.Main(appMain)
	return nil
}

func initOpenGL() {
	ctx := <-glContextCh
	opengl.InitWithContext(ctx)
}
