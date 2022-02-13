// Copyright 2022 The Ebiten Authors
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

package ui

import (
	"sync/atomic"
)

type Context interface {
	UpdateFrame() error
	ForceUpdateFrame() error
	Layout(outsideWidth, outsideHeight float64)

	// AdjustPosition can be called from a different goroutine from Update's or Layout's.
	AdjustPosition(x, y float64, deviceScaleFactor float64) (float64, float64)
}

type contextImpl struct {
	context Context

	err atomic.Value
}

func (c *contextImpl) updateFrame() error {
	if err, ok := c.err.Load().(error); ok && err != nil {
		return err
	}
	return c.context.UpdateFrame()
}

func (c *contextImpl) forceUpdateFrame() error {
	if err, ok := c.err.Load().(error); ok && err != nil {
		return err
	}
	return c.context.ForceUpdateFrame()
}

func (c *contextImpl) layout(outsideWidth, outsideHeight float64) {
	// The given outside size can be 0 e.g. just after restoring from the fullscreen mode on Windows (#1589)
	// Just ignore such cases. Otherwise, creating a zero-sized framebuffer causes a panic.
	if outsideWidth == 0 || outsideHeight == 0 {
		return
	}

	c.context.Layout(outsideWidth, outsideHeight)
}

func (c *contextImpl) adjustPosition(x, y float64, deviceScaleFactor float64) (float64, float64) {
	return c.context.AdjustPosition(x, y, deviceScaleFactor)
}

func (c *contextImpl) setError(err error) {
	c.err.Store(err)
}

func SetError(err error) {
	Get().context.setError(err)
}
