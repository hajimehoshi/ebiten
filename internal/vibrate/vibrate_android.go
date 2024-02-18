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

package vibrate

import (
	"time"

	"github.com/ebitengine/gomobile/app"
)

/*
#include <jni.h>
#include <stdlib.h>
#include <stdint.h>

#include <android/log.h>

// Basically same as:
//
//     Vibrator v = (Vibrator)getSystemService(Context.VIBRATOR_SERVICE);
//     if (Build.VERSION.SDK_INT >= 26) {
//       v.vibrate(VibrationEffect.createOneShot(milliseconds, magnitude * 255))
//     } else {
//       v.vibrate(millisecond)
//     }
//
// Note that this requires a manifest setting:
//
//     <uses-permission android:name="android.permission.VIBRATE"/>
//
static void vibrateOneShot(uintptr_t java_vm, uintptr_t jni_env, uintptr_t ctx, int64_t milliseconds, double magnitude) {
  JavaVM* vm = (JavaVM*)java_vm;
  JNIEnv* env = (JNIEnv*)jni_env;
  jobject context = (jobject)ctx;

  static int apiLevel = 0;
  if (!apiLevel) {
    const jclass android_os_Build_VERSION = (*env)->FindClass(env, "android/os/Build$VERSION");

    apiLevel = (*env)->GetStaticIntField(
        env, android_os_Build_VERSION,
        (*env)->GetStaticFieldID(env, android_os_Build_VERSION, "SDK_INT", "I"));

    (*env)->DeleteLocalRef(env, android_os_Build_VERSION);
  }

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

  if (apiLevel >= 26) {
    const jclass android_os_VibrationEffect = (*env)->FindClass(env, "android/os/VibrationEffect");

    const jobject vibrationEffect =
        (*env)->CallStaticObjectMethod(
            env, android_os_VibrationEffect,
            (*env)->GetStaticMethodID(env, android_os_VibrationEffect, "createOneShot", "(JI)Landroid/os/VibrationEffect;"),
            milliseconds, (int)(magnitude * 255));

    (*env)->CallVoidMethod(
        env, vibrator,
        (*env)->GetMethodID(env, android_os_Vibrator, "vibrate", "(Landroid/os/VibrationEffect;)V"),
        vibrationEffect);

    (*env)->DeleteLocalRef(env, android_os_VibrationEffect);

    (*env)->DeleteLocalRef(env, vibrationEffect);
  } else {
    (*env)->CallVoidMethod(
        env, vibrator,
        (*env)->GetMethodID(env, android_os_Vibrator, "vibrate", "(J)V"),
        milliseconds);
  }

  (*env)->DeleteLocalRef(env, android_content_Context);
  (*env)->DeleteLocalRef(env, android_os_Vibrator);

  (*env)->DeleteLocalRef(env, android_context_Context_VIBRATOR_SERVICE);
  (*env)->DeleteLocalRef(env, vibrator);
}

*/
import "C"

func Vibrate(duration time.Duration, magnitude float64) {
	go func() {
		_ = app.RunOnJVM(func(vm, env, ctx uintptr) error {
			// TODO: This might be crash when this is called from init(). How can we detect this?
			C.vibrateOneShot(C.uintptr_t(vm), C.uintptr_t(env), C.uintptr_t(ctx), C.int64_t(duration/time.Millisecond), C.double(magnitude))
			return nil
		})
	}()
}
