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

// Basically same as:
//
//     WindowService windowService = context.getSystemService(Context.WINDOW_SERVICE);
//     Display display = windowManager.getDefaultDisplay();
//     DisplayMetrics displayMetrics = new DisplayMetrics();
//     display.getRealMetrics(displayMetrics);
//     this.deviceScale = displayMetrics.density;
//
static float deviceScale(uintptr_t java_vm, uintptr_t jni_env, uintptr_t ctx) {
  JavaVM* vm = (JavaVM*)java_vm;
  JNIEnv* env = (JNIEnv*)jni_env;
  jobject context = (jobject)ctx;

  const char* kWindowService = "window";

  const jclass android_content_Context =
      (*env)->FindClass(env, "android/content/Context");
  const jclass android_view_WindowManager =
      (*env)->FindClass(env, "android/view/WindowManager");
  const jclass android_view_Display =
      (*env)->FindClass(env, "android/view/Display");
  const jclass android_util_DisplayMetrics =
      (*env)->FindClass(env, "android/util/DisplayMetrics");

  const jobject android_context_Context_WINDOW_SERVICE =
      (*env)->GetStaticObjectField(
          env, android_content_Context,
          (*env)->GetStaticFieldID(env, android_content_Context, "WINDOW_SERVICE", "Ljava/lang/String;"));

  const jobject windowManager =
      (*env)->CallObjectMethod(
          env, context,
          (*env)->GetMethodID(env, android_content_Context, "getSystemService", "(Ljava/lang/String;)Ljava/lang/Object;"),
          android_context_Context_WINDOW_SERVICE);
  const jobject display =
      (*env)->CallObjectMethod(
          env, windowManager,
          (*env)->GetMethodID(env, android_view_WindowManager, "getDefaultDisplay", "()Landroid/view/Display;"));
  const jobject displayMetrics =
      (*env)->NewObject(
          env, android_util_DisplayMetrics,
          (*env)->GetMethodID(env, android_util_DisplayMetrics, "<init>", "()V"));
  (*env)->CallVoidMethod(
      env, display,
      (*env)->GetMethodID(env, android_view_Display, "getRealMetrics", "(Landroid/util/DisplayMetrics;)V"),
      displayMetrics);
  const float density =
      (*env)->GetFloatField(
          env, displayMetrics,
          (*env)->GetFieldID(env, android_util_DisplayMetrics, "density", "F"));

  (*env)->DeleteLocalRef(env, android_content_Context);
  (*env)->DeleteLocalRef(env, android_view_WindowManager);
  (*env)->DeleteLocalRef(env, android_view_Display);
  (*env)->DeleteLocalRef(env, android_util_DisplayMetrics);

  (*env)->DeleteLocalRef(env, android_context_Context_WINDOW_SERVICE);
  (*env)->DeleteLocalRef(env, windowManager);
  (*env)->DeleteLocalRef(env, display);
  (*env)->DeleteLocalRef(env, displayMetrics);

  return density;
}

*/
import "C"

import (
	"fmt"

	"golang.org/x/mobile/app"
)

func impl(x, y int) float64 {
	s := 0.0
	if err := app.RunOnJVM(func(vm, env, ctx uintptr) error {
		// TODO: This might be crash when this is called from init(). How can we detect this?
		s = float64(C.deviceScale(C.uintptr_t(vm), C.uintptr_t(env), C.uintptr_t(ctx)))
		return nil
	}); err != nil {
		panic(fmt.Sprintf("devicescale: error %v", err))
	}
	return s
}
