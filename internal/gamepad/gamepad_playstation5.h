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

//go:build playstation5

struct Gamepad {
  int id;
  int button_count;
  int axis_count;
  char button_pressed[32];
  float button_values[32];
  float axis_values[16];
};

#ifdef __cplusplus
extern "C" {
#endif

void ebitengine_UpdateGamepads();
int ebitengine_GetGamepadCount();
void ebitengine_GetGamepads(struct Gamepad *gamepads);
void ebitengine_VibrateGamepad(int id, double durationInSeconds,
                               double strongMagnitude, double weakMagnitude);

#ifdef __cplusplus
} // extern "C"
#endif
