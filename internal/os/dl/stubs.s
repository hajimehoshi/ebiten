// Copyright 2022 The Ebiten Authors
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

#include "textflag.h"

//func dlopen(path *byte, mode int) (ret uintptr)
GLOBL ·dlopenABI0(SB), NOPTR|RODATA, $8
DATA ·dlopenABI0(SB)/8, $·dlopen(SB)
TEXT ·dlopen(SB), NOSPLIT, $0-0
   	JMP	_dlopen(SB)
   	RET

//func dlerror() (ret uintptr)
GLOBL ·dlerrorABI0(SB), NOPTR|RODATA, $8
DATA ·dlerrorABI0(SB)/8, $·dlerror(SB)
TEXT ·dlerror(SB), NOSPLIT, $0-0
	JMP _dlerror(SB)
	RET

//func dlclose(handle uintptr) (ret int)
GLOBL ·dlcloseABI0(SB), NOPTR|RODATA, $8
DATA ·dlcloseABI0(SB)/8, $·dlclose(SB)
TEXT ·dlclose(SB), NOSPLIT, $0-0
	JMP _dlclose(SB)
	RET

//func dlsym(handle uintptr, symbol *byte) (ret uintptr)
GLOBL ·dlsymABI0(SB), NOPTR|RODATA, $8
DATA ·dlsymABI0(SB)/8, $·dlsym(SB)
TEXT ·dlsym(SB), NOSPLIT, $0-0
	JMP _dlsym(SB)
	RET

