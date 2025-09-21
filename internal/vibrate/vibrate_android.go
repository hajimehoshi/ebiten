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

// Basically the following code is equivalent to the following Java code:
//
//     Vibrator v;
//     if (Build.VERSION.SDK_INT >= 31) {
//       v = getSystemService(Context.VIBRATOR_MANAGER_SERVICE).getDefaultVibrator();
//     } else {
//       v = (Vibrator)getSystemService(Context.VIBRATOR_SERVICE);
//     }
//     if (Build.VERSION.SDK_INT >= 26) {
//       VibrationEffect effect = VibrationEffect.createOneShot(milliseconds, magnitude * 255);
//       if (Build.VERSION.SDK_INT >= 33) {
//         VibrationAttributes attrs = new VibrationAttributes.Builder()
//           .setUsage(VibrationAttributes.USAGE_MEDIA)
//           .build();
//         v.vibrate(effect, attrs);
//       } else {
//         AudioAttributes attrs = new AudioAttributes.Builder()
//           .setUsage(AudioAttributes.USAGE_GAME)
//           .build();
//         v.vibrate(effect, attrs);
//       }
//     } else {
//       v.vibrate(millisecond);
//     }
//
// Note that this requires a manifest setting:
//
//     <uses-permission android:name="android.permission.VIBRATE"/>
//
#cgo noescape vibrateOneShot
#cgo nocallback vibrateOneShot
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

  jobject vibrator = NULL;
  if (apiLevel >= 31) {
    const jclass android_os_VibratorManager = (*env)->FindClass(env, "android/os/VibratorManager");

    const jobject android_context_Context_VIBRATOR_MANAGER_SERVICE =
        (*env)->GetStaticObjectField(
            env, android_content_Context,
            (*env)->GetStaticFieldID(env, android_content_Context, "VIBRATOR_MANAGER_SERVICE", "Ljava/lang/String;"));

    const jobject vibratorManager =
        (*env)->CallObjectMethod(
            env, context,
            (*env)->GetMethodID(env, android_content_Context, "getSystemService", "(Ljava/lang/String;)Ljava/lang/Object;"),
            android_context_Context_VIBRATOR_MANAGER_SERVICE);

    vibrator =
        (*env)->CallObjectMethod(
            env, vibratorManager,
            (*env)->GetMethodID(env, android_os_VibratorManager, "getDefaultVibrator", "()Landroid/os/Vibrator;"));

    (*env)->DeleteLocalRef(env, vibratorManager);
    (*env)->DeleteLocalRef(env, android_context_Context_VIBRATOR_MANAGER_SERVICE);
    (*env)->DeleteLocalRef(env, android_os_VibratorManager);
  } else {
    const jobject android_context_Context_VIBRATOR_SERVICE =
        (*env)->GetStaticObjectField(
            env, android_content_Context,
            (*env)->GetStaticFieldID(env, android_content_Context, "VIBRATOR_SERVICE", "Ljava/lang/String;"));

    vibrator =
        (*env)->CallObjectMethod(
            env, context,
            (*env)->GetMethodID(env, android_content_Context, "getSystemService", "(Ljava/lang/String;)Ljava/lang/Object;"),
            android_context_Context_VIBRATOR_SERVICE);

    (*env)->DeleteLocalRef(env, android_context_Context_VIBRATOR_SERVICE);
  }

  if (apiLevel >= 26) {
    const jclass android_os_VibrationEffect = (*env)->FindClass(env, "android/os/VibrationEffect");

    const jobject vibrationEffect =
        (*env)->CallStaticObjectMethod(
            env, android_os_VibrationEffect,
            (*env)->GetStaticMethodID(env, android_os_VibrationEffect, "createOneShot", "(JI)Landroid/os/VibrationEffect;"),
            milliseconds, (int)(magnitude * 255));

    if (apiLevel >= 33) {
      const jclass android_os_VibrationAttributes = (*env)->FindClass(env, "android/os/VibrationAttributes");
      const jclass android_os_VibrationAttributes_Builder = (*env)->FindClass(env, "android/os/VibrationAttributes$Builder");

      const jobject attributesBuilder =
          (*env)->NewObject(
              env, android_os_VibrationAttributes_Builder,
              (*env)->GetMethodID(env, android_os_VibrationAttributes_Builder, "<init>", "()V"));

      // A purpose for games and media are integrated into VibrationAttributes.USAGE_MEDIA.
      const jint USAGE_MEDIA = 19;
      (*env)->CallObjectMethod(
          env, attributesBuilder,
          (*env)->GetMethodID(env, android_os_VibrationAttributes_Builder, "setUsage", "(I)Landroid/os/VibrationAttributes$Builder;"),
          USAGE_MEDIA);

      const jobject vibrationAttributes =
          (*env)->CallObjectMethod(
              env, attributesBuilder,
              (*env)->GetMethodID(env, android_os_VibrationAttributes_Builder, "build", "()Landroid/os/VibrationAttributes;"));

      (*env)->CallVoidMethod(
          env, vibrator,
          (*env)->GetMethodID(env, android_os_Vibrator, "vibrate", "(Landroid/os/VibrationEffect;Landroid/os/VibrationAttributes;)V"),
          vibrationEffect, vibrationAttributes);

      (*env)->DeleteLocalRef(env, vibrationAttributes);
      (*env)->DeleteLocalRef(env, attributesBuilder);
      (*env)->DeleteLocalRef(env, android_os_VibrationAttributes_Builder);
      (*env)->DeleteLocalRef(env, android_os_VibrationAttributes);
    } else {
      const jclass android_media_AudioAttributes = (*env)->FindClass(env, "android/media/AudioAttributes");
      const jclass android_media_AudioAttributes_Builder = (*env)->FindClass(env, "android/media/AudioAttributes$Builder");

      const jobject attributesBuilder =
          (*env)->NewObject(
              env, android_media_AudioAttributes_Builder,
              (*env)->GetMethodID(env, android_media_AudioAttributes_Builder, "<init>", "()V"));

      // Use AudioAttributes.USAGE_GAME as most applications with Ebitengine are games.
      const jint USAGE_GAME = 14;
      (*env)->CallObjectMethod(
          env, attributesBuilder,
          (*env)->GetMethodID(env, android_media_AudioAttributes_Builder, "setUsage", "(I)Landroid/media/AudioAttributes$Builder;"),
          USAGE_GAME);

      const jobject audioAttributes =
          (*env)->CallObjectMethod(
              env, attributesBuilder,
              (*env)->GetMethodID(env, android_media_AudioAttributes_Builder, "build", "()Landroid/media/AudioAttributes;"));

      (*env)->CallVoidMethod(
          env, vibrator,
          (*env)->GetMethodID(env, android_os_Vibrator, "vibrate", "(Landroid/os/VibrationEffect;Landroid/media/AudioAttributes;)V"),
          vibrationEffect, audioAttributes);

      (*env)->DeleteLocalRef(env, audioAttributes);
      (*env)->DeleteLocalRef(env, attributesBuilder);
      (*env)->DeleteLocalRef(env, android_media_AudioAttributes_Builder);
      (*env)->DeleteLocalRef(env, android_media_AudioAttributes);
    }

    (*env)->DeleteLocalRef(env, vibrationEffect);
    (*env)->DeleteLocalRef(env, android_os_VibrationEffect);
  } else {
    (*env)->CallVoidMethod(
        env, vibrator,
        (*env)->GetMethodID(env, android_os_Vibrator, "vibrate", "(J)V"),
        milliseconds);
  }

  (*env)->DeleteLocalRef(env, vibrator);
  (*env)->DeleteLocalRef(env, android_content_Context);
  (*env)->DeleteLocalRef(env, android_os_Vibrator);
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
