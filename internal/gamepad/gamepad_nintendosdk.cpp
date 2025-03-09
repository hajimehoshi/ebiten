// Copyright 2022 The Ebitengine Authors
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

//go:build nintendosdk

// The actual implementation will be provided by github.com/hajimehoshi/uwagaki.

#include "gamepad_nintendosdk.h"

extern "C" void ebitengine_UpdateGamepads() {}

extern "C" int ebitengine_GetGamepadCount() { return 0; }

extern "C" void ebitengine_GetGamepads(struct Gamepad *gamepads) {}

extern "C" void ebitengine_VibrateGamepad(int id, double durationInSeconds,
                                          double strongMagnitude,
                                          double weakMagnitude) {}
