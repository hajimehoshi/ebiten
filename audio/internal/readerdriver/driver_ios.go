// Copyright 2021 The Ebiten Authors
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

//go:build darwin && ios
// +build darwin,ios

package readerdriver

// 12288 seems necessary at least on iPod touch (7th).
// With 48000[Hz] stereo, the maximum delay is (12288 / 4 / 2 [samples]) / 48000 [Hz] = 0.032 [sec].
// '4' is float32 size in bytes. '2' is a number of channels for stereo.

const bufferSizeInBytes = 12288
