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

#include <EGL/egl.h>

#ifdef __cplusplus
extern "C" {
#endif

NativeWindowType ebitengine_Initialize();
void ebitengine_InitializeProfiler();
void ebitengine_RecordProfilerHeartbeat();

#ifdef __cplusplus
} // extern "C"
#endif
