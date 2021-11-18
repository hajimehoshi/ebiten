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

package mobile

import (
	"time"

	"golang.org/x/mobile/app"
)

/*

#include <jni.h>
#include <stdlib.h>
#include <stdint.h>

// Basically same as:
//
//     Vibrator v = (Vibrator)getSystemService(Context.VIBRATOR_SERVICE);
//     v.vibrate(millisecond)
//
// TODO: As of API Level 26, the new API should be Used instead:
//
//     Vibrator v = (Vibrator)getSystemService(Context.VIBRATOR_SERVICE);
//     v.vibrate(VibrationEffect.createOneShot(millisecond, VibrationEffect.DEFAULT_AMPLITUDE))
//
// Note that this requires a manifest setting:
//
//     <uses-permission android:name="android.permission.VIBRATE"/>
//
static void vibrateOneShot(uintptr_t java_vm, uintptr_t jni_env, uintptr_t ctx, int64_t milliseconds) {
  JavaVM* vm = (JavaVM*)java_vm;
  JNIEnv* env = (JNIEnv*)jni_env;
  jobject context = (jobject)ctx;

  const jclass android_content_Context = (*env)->FindClass(env, "android/content/Context");
  const jclass android_os_Vibrator = (*env)->FindClass(env, "android/os/Vibrator");

  const jobject android_context_Context_VIBRATOR_SERVICE =
      (*env)->GetStaticObjectField(
          env, android_content_Context,
          (*env)->GetStaticFieldID(env, android_content_Context, "VIBRATOR_SERVICE", "Ljava/lang/String;"));

  const jobject vibrator =
      (*env)->CallObjectMethod(
          env, context,
          (*env)->GetMethodID(env, android_content_Context, "getSystemService", "(Ljava/lang/String;)Ljava/lang/Object;"),
          android_context_Context_VIBRATOR_SERVICE);

  (*env)->CallVoidMethod(
      env, vibrator,
      (*env)->GetMethodID(env, android_os_Vibrator, "vibrate", "(J)V"),
      milliseconds);

  (*env)->DeleteLocalRef(env, android_content_Context);
  (*env)->DeleteLocalRef(env, android_os_Vibrator);

  (*env)->DeleteLocalRef(env, android_context_Context_VIBRATOR_SERVICE);
  (*env)->DeleteLocalRef(env, vibrator);
}

*/
import "C"

func (u *UserInterface) Vibrate(duration time.Duration) {
	_ = app.RunOnJVM(func(vm, env, ctx uintptr) error {
		// TODO: This might be crash when this is called from init(). How can we detect this?
		C.vibrateOneShot(C.uintptr_t(vm), C.uintptr_t(env), C.uintptr_t(ctx), C.int64_t(duration/time.Millisecond))
		return nil
	})
}
