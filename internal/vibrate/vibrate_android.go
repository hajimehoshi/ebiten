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

// clearException clears a pending Java exception. A failed lookup like
// FindClass or GetMethodID leaves an exception pending, and any further JNI
// call made with an exception pending is undefined.
static void clearException(JNIEnv* env) {
  if ((*env)->ExceptionCheck(env)) {
    (*env)->ExceptionClear(env);
  }
}

// Basically the following code is equivalent to the following Java code:
//
//     Vibrator v;
//     if (Build.VERSION.SDK_INT >= 31) {
//       v = getSystemService(Context.VIBRATOR_MANAGER_SERVICE).getDefaultVibrator();
//     } else {
//       v = (Vibrator)getSystemService(Context.VIBRATOR_SERVICE);
//     }
//     if (Build.VERSION.SDK_INT >= 26) {
//       VibrationEffect effect = VibrationEffect.createOneShot(milliseconds, amplitude);
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
// A lookup or a call that fails aborts the vibration silently.
//
// Note that this requires a manifest setting:
//
//     <uses-permission android:name="android.permission.VIBRATE"/>
//
#cgo noescape vibrateOneShot
#cgo nocallback vibrateOneShot
static void vibrateOneShot(uintptr_t java_vm, uintptr_t jni_env, uintptr_t ctx, int64_t milliseconds, int amplitude) {
  JavaVM* vm = (JavaVM*)java_vm;
  JNIEnv* env = (JNIEnv*)jni_env;
  jobject context = (jobject)ctx;

  static int apiLevel = 0;
  if (!apiLevel) {
    const jclass android_os_Build_VERSION = (*env)->FindClass(env, "android/os/Build$VERSION");
    if (!android_os_Build_VERSION) {
      clearException(env);
      return;
    }

    const jfieldID sdkIntID = (*env)->GetStaticFieldID(env, android_os_Build_VERSION, "SDK_INT", "I");
    if (!sdkIntID) {
      clearException(env);
      (*env)->DeleteLocalRef(env, android_os_Build_VERSION);
      return;
    }

    apiLevel = (*env)->GetStaticIntField(env, android_os_Build_VERSION, sdkIntID);

    (*env)->DeleteLocalRef(env, android_os_Build_VERSION);
  }

  // API level 26 and newer reject a zero vibration amplitude. Earlier versions ignore amplitude.
  if (apiLevel >= 26 && amplitude == 0) {
    return;
  }

  // Every local reference taken below is released at cleanup.
  jclass android_content_Context = NULL;
  jclass android_os_Vibrator = NULL;
  jclass android_os_VibratorManager = NULL;
  jobject android_context_Context_VIBRATOR_MANAGER_SERVICE = NULL;
  jobject vibratorManager = NULL;
  jobject android_context_Context_VIBRATOR_SERVICE = NULL;
  jobject vibrator = NULL;
  jclass android_os_VibrationEffect = NULL;
  jobject vibrationEffect = NULL;
  jclass android_os_VibrationAttributes = NULL;
  jclass android_os_VibrationAttributes_Builder = NULL;
  jclass android_media_AudioAttributes = NULL;
  jclass android_media_AudioAttributes_Builder = NULL;
  jobject attributesBuilder = NULL;
  jobject vibrationAttributes = NULL;
  jobject audioAttributes = NULL;

  android_content_Context = (*env)->FindClass(env, "android/content/Context");
  if (!android_content_Context) {
    goto cleanup;
  }

  android_os_Vibrator = (*env)->FindClass(env, "android/os/Vibrator");
  if (!android_os_Vibrator) {
    goto cleanup;
  }

  if (apiLevel >= 31) {
    android_os_VibratorManager = (*env)->FindClass(env, "android/os/VibratorManager");
    if (!android_os_VibratorManager) {
      goto cleanup;
    }

    const jfieldID vibratorManagerServiceID =
        (*env)->GetStaticFieldID(env, android_content_Context, "VIBRATOR_MANAGER_SERVICE", "Ljava/lang/String;");
    if (!vibratorManagerServiceID) {
      goto cleanup;
    }

    const jmethodID getSystemServiceID =
        (*env)->GetMethodID(env, android_content_Context, "getSystemService", "(Ljava/lang/String;)Ljava/lang/Object;");
    if (!getSystemServiceID) {
      goto cleanup;
    }

    const jmethodID getDefaultVibratorID =
        (*env)->GetMethodID(env, android_os_VibratorManager, "getDefaultVibrator", "()Landroid/os/Vibrator;");
    if (!getDefaultVibratorID) {
      goto cleanup;
    }

    android_context_Context_VIBRATOR_MANAGER_SERVICE =
        (*env)->GetStaticObjectField(env, android_content_Context, vibratorManagerServiceID);

    // The result of a Java method call is invalid when the call throws, so an
    // exception is checked ahead of the result. getSystemService returns null
    // for a service that is not registered.
    vibratorManager =
        (*env)->CallObjectMethod(env, context, getSystemServiceID, android_context_Context_VIBRATOR_MANAGER_SERVICE);
    if ((*env)->ExceptionCheck(env) || !vibratorManager) {
      goto cleanup;
    }

    vibrator = (*env)->CallObjectMethod(env, vibratorManager, getDefaultVibratorID);
  } else {
    const jfieldID vibratorServiceID =
        (*env)->GetStaticFieldID(env, android_content_Context, "VIBRATOR_SERVICE", "Ljava/lang/String;");
    if (!vibratorServiceID) {
      goto cleanup;
    }

    const jmethodID getSystemServiceID =
        (*env)->GetMethodID(env, android_content_Context, "getSystemService", "(Ljava/lang/String;)Ljava/lang/Object;");
    if (!getSystemServiceID) {
      goto cleanup;
    }

    android_context_Context_VIBRATOR_SERVICE =
        (*env)->GetStaticObjectField(env, android_content_Context, vibratorServiceID);

    vibrator =
        (*env)->CallObjectMethod(env, context, getSystemServiceID, android_context_Context_VIBRATOR_SERVICE);
  }

  // getSystemService returns null for a service that is not registered, and
  // getDefaultVibrator can likewise fail.
  if ((*env)->ExceptionCheck(env) || !vibrator) {
    goto cleanup;
  }

  if (apiLevel >= 26) {
    android_os_VibrationEffect = (*env)->FindClass(env, "android/os/VibrationEffect");
    if (!android_os_VibrationEffect) {
      goto cleanup;
    }

    const jmethodID createOneShotID =
        (*env)->GetStaticMethodID(env, android_os_VibrationEffect, "createOneShot", "(JI)Landroid/os/VibrationEffect;");
    if (!createOneShotID) {
      goto cleanup;
    }

    vibrationEffect =
        (*env)->CallStaticObjectMethod(env, android_os_VibrationEffect, createOneShotID, milliseconds, amplitude);
    if ((*env)->ExceptionCheck(env) || !vibrationEffect) {
      goto cleanup;
    }

    if (apiLevel >= 33) {
      android_os_VibrationAttributes = (*env)->FindClass(env, "android/os/VibrationAttributes");
      if (!android_os_VibrationAttributes) {
        goto cleanup;
      }

      android_os_VibrationAttributes_Builder = (*env)->FindClass(env, "android/os/VibrationAttributes$Builder");
      if (!android_os_VibrationAttributes_Builder) {
        goto cleanup;
      }

      const jmethodID builderInitID =
          (*env)->GetMethodID(env, android_os_VibrationAttributes_Builder, "<init>", "()V");
      if (!builderInitID) {
        goto cleanup;
      }

      const jmethodID setUsageID =
          (*env)->GetMethodID(env, android_os_VibrationAttributes_Builder, "setUsage", "(I)Landroid/os/VibrationAttributes$Builder;");
      if (!setUsageID) {
        goto cleanup;
      }

      const jmethodID buildID =
          (*env)->GetMethodID(env, android_os_VibrationAttributes_Builder, "build", "()Landroid/os/VibrationAttributes;");
      if (!buildID) {
        goto cleanup;
      }

      const jmethodID vibrateID =
          (*env)->GetMethodID(env, android_os_Vibrator, "vibrate", "(Landroid/os/VibrationEffect;Landroid/os/VibrationAttributes;)V");
      if (!vibrateID) {
        goto cleanup;
      }

      attributesBuilder = (*env)->NewObject(env, android_os_VibrationAttributes_Builder, builderInitID);
      if ((*env)->ExceptionCheck(env) || !attributesBuilder) {
        goto cleanup;
      }

      // A purpose for games and media are integrated into VibrationAttributes.USAGE_MEDIA.
      const jint USAGE_MEDIA = 19;
      (*env)->CallObjectMethod(env, attributesBuilder, setUsageID, USAGE_MEDIA);
      if ((*env)->ExceptionCheck(env)) {
        goto cleanup;
      }

      vibrationAttributes = (*env)->CallObjectMethod(env, attributesBuilder, buildID);
      if ((*env)->ExceptionCheck(env) || !vibrationAttributes) {
        goto cleanup;
      }

      (*env)->CallVoidMethod(env, vibrator, vibrateID, vibrationEffect, vibrationAttributes);
    } else {
      android_media_AudioAttributes = (*env)->FindClass(env, "android/media/AudioAttributes");
      if (!android_media_AudioAttributes) {
        goto cleanup;
      }

      android_media_AudioAttributes_Builder = (*env)->FindClass(env, "android/media/AudioAttributes$Builder");
      if (!android_media_AudioAttributes_Builder) {
        goto cleanup;
      }

      const jmethodID builderInitID =
          (*env)->GetMethodID(env, android_media_AudioAttributes_Builder, "<init>", "()V");
      if (!builderInitID) {
        goto cleanup;
      }

      const jmethodID setUsageID =
          (*env)->GetMethodID(env, android_media_AudioAttributes_Builder, "setUsage", "(I)Landroid/media/AudioAttributes$Builder;");
      if (!setUsageID) {
        goto cleanup;
      }

      const jmethodID buildID =
          (*env)->GetMethodID(env, android_media_AudioAttributes_Builder, "build", "()Landroid/media/AudioAttributes;");
      if (!buildID) {
        goto cleanup;
      }

      const jmethodID vibrateID =
          (*env)->GetMethodID(env, android_os_Vibrator, "vibrate", "(Landroid/os/VibrationEffect;Landroid/media/AudioAttributes;)V");
      if (!vibrateID) {
        goto cleanup;
      }

      attributesBuilder = (*env)->NewObject(env, android_media_AudioAttributes_Builder, builderInitID);
      if ((*env)->ExceptionCheck(env) || !attributesBuilder) {
        goto cleanup;
      }

      // Use AudioAttributes.USAGE_GAME as most applications with Ebitengine are games.
      const jint USAGE_GAME = 14;
      (*env)->CallObjectMethod(env, attributesBuilder, setUsageID, USAGE_GAME);
      if ((*env)->ExceptionCheck(env)) {
        goto cleanup;
      }

      audioAttributes = (*env)->CallObjectMethod(env, attributesBuilder, buildID);
      if ((*env)->ExceptionCheck(env) || !audioAttributes) {
        goto cleanup;
      }

      (*env)->CallVoidMethod(env, vibrator, vibrateID, vibrationEffect, audioAttributes);
    }
  } else {
    const jmethodID vibrateID = (*env)->GetMethodID(env, android_os_Vibrator, "vibrate", "(J)V");
    if (!vibrateID) {
      goto cleanup;
    }

    (*env)->CallVoidMethod(env, vibrator, vibrateID, milliseconds);
  }

cleanup:
  // A failed lookup or call leaves an exception pending, and a vibrate call
  // throws e.g. without the VIBRATE permission.
  clearException(env);

  // DeleteLocalRef ignores NULL on Android (ART and Dalvik), which the JNI
  // specification leaves unspecified.
  (*env)->DeleteLocalRef(env, audioAttributes);
  (*env)->DeleteLocalRef(env, vibrationAttributes);
  (*env)->DeleteLocalRef(env, attributesBuilder);
  (*env)->DeleteLocalRef(env, android_media_AudioAttributes_Builder);
  (*env)->DeleteLocalRef(env, android_media_AudioAttributes);
  (*env)->DeleteLocalRef(env, android_os_VibrationAttributes_Builder);
  (*env)->DeleteLocalRef(env, android_os_VibrationAttributes);
  (*env)->DeleteLocalRef(env, vibrationEffect);
  (*env)->DeleteLocalRef(env, android_os_VibrationEffect);
  (*env)->DeleteLocalRef(env, vibrator);
  (*env)->DeleteLocalRef(env, android_context_Context_VIBRATOR_SERVICE);
  (*env)->DeleteLocalRef(env, vibratorManager);
  (*env)->DeleteLocalRef(env, android_context_Context_VIBRATOR_MANAGER_SERVICE);
  (*env)->DeleteLocalRef(env, android_os_VibratorManager);
  (*env)->DeleteLocalRef(env, android_os_Vibrator);
  (*env)->DeleteLocalRef(env, android_content_Context);
}
*/
import "C"

func vibrate(duration time.Duration, magnitude float64) {
	milliseconds := duration / time.Millisecond
	if milliseconds <= 0 {
		return
	}
	amplitude := androidVibrationAmplitude(magnitude)
	go func() {
		_ = app.RunOnJVM(func(vm, env, ctx uintptr) error {
			// TODO: This might be crash when this is called from init(). How can we detect this?
			C.vibrateOneShot(C.uintptr_t(vm), C.uintptr_t(env), C.uintptr_t(ctx), C.int64_t(milliseconds), C.int(amplitude))
			return nil
		})
	}()
}
