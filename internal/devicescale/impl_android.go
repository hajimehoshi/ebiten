// Copyright 2016 Hajime Hoshi
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

package devicescale

/*

#include <jni.h>
#include <stdlib.h>

// Basically same as `getResources().getDisplayMetrics().density`;
static float deviceScale(uintptr_t java_vm, uintptr_t jni_env, uintptr_t ctx) {
  JavaVM* vm = (JavaVM*)java_vm;
  JNIEnv* env = (JNIEnv*)jni_env;
  jobject context = (jobject)ctx;

  const jclass android_content_ContextWrapper =
      (*env)->FindClass(env, "android/content/ContextWrapper");
  const jclass android_content_res_Resources =
      (*env)->FindClass(env, "android/content/res/Resources");
  const jclass android_util_DisplayMetrics =
      (*env)->FindClass(env, "android/util/DisplayMetrics");

  const jobject resources =
      (*env)->CallObjectMethod(
          env, context,
          (*env)->GetMethodID(env, android_content_ContextWrapper, "getResources", "()Landroid/content/res/Resources;"));
  const jobject displayMetrics =
      (*env)->CallObjectMethod(
          env, resources,
          (*env)->GetMethodID(env, android_content_res_Resources, "getDisplayMetrics", "()Landroid/util/DisplayMetrics;"));
  const float density =
      (*env)->GetFloatField(
          env, displayMetrics,
          (*env)->GetFieldID(env, android_util_DisplayMetrics, "density", "F"));

  (*env)->DeleteLocalRef(env, android_content_ContextWrapper);
  (*env)->DeleteLocalRef(env, android_content_res_Resources);
  (*env)->DeleteLocalRef(env, android_util_DisplayMetrics);
  (*env)->DeleteLocalRef(env, resources);
  (*env)->DeleteLocalRef(env, displayMetrics);

  return density;
}

*/
import "C"

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/internal/jni"
)

func impl() float64 {
	if !jni.IsJVMAvailable() {
		panic("devicescale: JVM is not available yet: is this called from init funcitons?")
	}
	s := 0.0
	if err := jni.RunOnJVM(func(vm, env, ctx uintptr) error {
		s = float64(C.deviceScale(C.uintptr_t(vm), C.uintptr_t(env), C.uintptr_t(ctx)))
		return nil
	}); err != nil {
		panic(fmt.Sprintf("devicescale: error %v", err))
	}
	return s
}
