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

#ifndef OBOE_ANDROID_H_
#define OBOE_ANDROID_H_

#include <stdbool.h>
#include <stdint.h>
#include <stdlib.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef uintptr_t PlayerID;

const char* Suspend();
const char* Resume();
PlayerID Player_Create(int sample_rate, int channel_num, int bit_depth_in_bytes, double volume, uintptr_t go_player);
void Player_AppendBuffer(PlayerID audio_player, uint8_t* data, int length);
const char* Player_Play(PlayerID audio_player);
const char* Player_Pause(PlayerID audio_player);
const char* Player_Close(PlayerID audio_player);
void Player_SetVolume(PlayerID audio_player, double volume);
int Player_UnplayedBufferSize(PlayerID audio_player);

#ifdef __cplusplus
}
#endif

#endif  // OBOE_ANDROID_H_
