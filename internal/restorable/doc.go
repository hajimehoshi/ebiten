// Copyright 2017 The Ebiten Authors
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

// Package restorable offers an Image struct that stores image commands
// and restores its pixel data from the commands when context lost happens.
//
// When a function like DrawImage or Fill is called, an Image tries to record
// the information for restoration.
//
// * Context lost
//
// Contest lost is a process that information on GPU memory are removed by OS
// to make more room on GPU memory.
// This can happen e.g. when GPU memory usage is high, or just switching applications
// might cause context lost on mobiles.
// As Ebitengine's image data is on GPU memory, the game can't continue when context lost happens
// without restoring image information.
// The package restorable is the package to record information for such restoration.
//
// * DrawImage
//
// DrawImage function tries to record an item of 'draw image history' in the target image.
// If a target image is stale or volatile, no item is created.
// If an item of the history is created,
// it can be said that the target image depends on the source image.
// In other words, If A.DrawImage(B, ...) is called,
// it can be said that the image A depends on the image B.
//
// * Fill, WritePixels and Dispose
//
// These functions are also drawing functions and the target image stores the pixel data
// instead of draw image history items. There is no dependency here.
//
// * Making images stale
//
// After any of the drawing functions is called, the target image can't be depended on by
// any other images. For example, if an image A depends on an image B, and B is changed
// by a Fill call after that, the image A can't depend on the image B anymore.
// In this case, as the image B can no longer be used to restore the image A,
// the image A becomes 'stale'.
// As all the stale images are resolved before context lost happens,
// draw image history items are kept as they are
// (even if an image C depends on the stale image A, it is still fine).
//
// * Stale image
//
// A stale image is an image that can't be restored from the recorded information.
// All stale images must be resolved by reading pixels from GPU before the frame ends.
// If a source image of DrawImage is a stale image, the target always becomes stale.
//
// * Volatile image
//
// A volatile image is a special image that is always cleared when a frame starts.
// For instance, the game screen passed via the update function is a volatile image.
// A volatile image doesn't have to record the drawing history.
// If a source image of DrawImage is a volatile image, the target always becomes stale.
package restorable
