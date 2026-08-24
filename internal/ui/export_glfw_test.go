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

//go:build !android && !ios && !js && !nintendosdk && !playstation5

package ui

import "image"

func NewMonitorForTest(boundsInGLFWPixels image.Rectangle, deviceScaleFactor float64) *Monitor {
	return &Monitor{
		boundsInGLFWPixels: boundsInGLFWPixels,
		contentScale:       deviceScaleFactor,
	}
}

func OutsideSizeInDIPForTest(windowWidth, windowHeight int, requestedWidthInDIP, requestedHeightInDIP int, nativeFullscreen bool, deviceScaleFactor float64) (float64, float64) {
	return outsideSizeInDIP(windowWidth, windowHeight, requestedWidthInDIP, requestedHeightInDIP, nativeFullscreen, deviceScaleFactor)
}

func WindowSizeInGLFWPixelsForTest(widthInDIP, heightInDIP int, deviceScaleFactor float64) (int, int) {
	return windowSizeInGLFWPixels(widthInDIP, heightInDIP, deviceScaleFactor)
}

func DIPFromGLFWPixelForTest(x float64, deviceScaleFactor float64) float64 {
	return dipFromGLFWPixel(x, deviceScaleFactor)
}

const InvalidSizeForTest = invalidSize

func WindowSizeToRestoreForTest(origWidth, origHeight int, origMonitor *Monitor, widthInDIP, heightInDIP int, monitor *Monitor) (int, int) {
	return windowSizeToRestore(origWidth, origHeight, origMonitor, widthInDIP, heightInDIP, monitor)
}

func WindowPositionInGLFWPixelsForTest(xInDIP, yInDIP int, monitor *Monitor) (int, int) {
	return windowPositionInGLFWPixels(xInDIP, yInDIP, monitor)
}

func WindowPositionInDIPForTest(windowX, windowY int, xInDIP, yInDIP int, monitor *Monitor) (int, int) {
	return windowPositionInDIP(windowX, windowY, xInDIP, yInDIP, monitor)
}
