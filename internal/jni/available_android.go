// Copyright 2018 The Ebiten Authors
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

package jni

/*
#include <jni.h>
#include <stdbool.h>

JavaVM* current_vm;

static bool isJVMAvailable() {
	return current_vm != NULL;
}
*/
import "C"

// IsJVMAvailable returns a boolean value indicating whether JVM is available or not.
//
// In init functions, JVM is not available.
func IsJVMAvailable() bool {
	return bool(C.isJVMAvailable())
}
