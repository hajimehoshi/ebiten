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

const char *ebiten_oboe_Play(int sample_rate, int channel_num,
                             int bit_depth_in_bytes);
const char *ebiten_oboe_Suspend();
const char *ebiten_oboe_Resume();
PlayerID ebiten_oboe_Player_Create(double volume, uintptr_t go_player);
bool ebiten_oboe_Player_IsPlaying(PlayerID audio_player);
void ebiten_oboe_Player_AppendBuffer(PlayerID audio_player, uint8_t *data,
                                     int length);
void ebiten_oboe_Player_Play(PlayerID audio_player);
void ebiten_oboe_Player_Pause(PlayerID audio_player);
void ebiten_oboe_Player_Close(PlayerID audio_player);
void ebiten_oboe_Player_SetVolume(PlayerID audio_player, double volume);
int ebiten_oboe_Player_UnplayedBufferSize(PlayerID audio_player);

#ifdef __cplusplus
}
#endif

#endif // OBOE_ANDROID_H_
