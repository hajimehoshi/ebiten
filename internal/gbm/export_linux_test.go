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

//go:build linux && (amd64 || arm64) && !android

package gbm

func CRTCForEncoderForTesting(crtcID, possibleCrtcs uint32, crtcs []uint32) uint32 {
	return crtcForEncoder(&drmModeEncoder{crtcID: crtcID, possibleCrtcs: possibleCrtcs}, crtcs)
}

func PreferredModeIndexForTesting(types []uint32) int {
	modes := make([]drmModeModeInfo, len(types))
	for i, typ := range types {
		modes[i].clock = uint32(i + 1)
		modes[i].typ = typ
	}
	return int(preferredMode(modes).clock) - 1
}
