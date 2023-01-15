// Copyright 2023 The Ebitengine Authors
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

struct Touch {
  int id;
  int x;
  int y;
};

extern const int kScreenWidth;
extern const int kScreenHeight;

#ifdef __cplusplus
extern "C" {
#endif

void ebitengine_UpdateTouches();
int ebitengine_GetTouchCount();
void ebitengine_GetTouches(struct Touch* touches);

#ifdef __cplusplus
} // extern "C"
#endif
