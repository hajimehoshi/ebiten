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

#ifdef __cplusplus
}
#endif

#endif // OBOE_ANDROID_H_
