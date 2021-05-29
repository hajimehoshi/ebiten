/*
 * Copyright 2016 The Android Open Source Project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

#include <dlfcn.h>
#include "oboe_oboe_Utilities_android.h"
#include "oboe_common_OboeDebug_android.h"
#include "oboe_aaudio_AAudioLoader_android.h"

#define LIB_AAUDIO_NAME "libaaudio.so"

namespace oboe {

AAudioLoader::~AAudioLoader() {
    // Issue 360: thread_local variables with non-trivial destructors
    // will cause segfaults if the containing library is dlclose()ed on
    // devices running M or newer, or devices before M when using a static STL.
    // The simple workaround is to not call dlclose.
    // https://github.com/android/ndk/wiki/Changelog-r22#known-issues
    //
    // The libaaudio and libaaudioclient do not use thread_local.
    // But, to be safe, we should avoid dlclose() if possible.
    // Because AAudioLoader is a static Singleton, we can safely skip
    // calling dlclose() without causing a resource leak.
    LOGI("%s() dlclose(%s) not called, OK", __func__, LIB_AAUDIO_NAME);
}

AAudioLoader* AAudioLoader::getInstance() {
    static AAudioLoader instance;
    return &instance;
}

int AAudioLoader::open() {
    if (mLibHandle != nullptr) {
        return 0;
    }

    // Use RTLD_NOW to avoid the unpredictable behavior that RTLD_LAZY can cause.
    // Also resolving all the links now will prevent a run-time penalty later.
    mLibHandle = dlopen(LIB_AAUDIO_NAME, RTLD_NOW);
    if (mLibHandle == nullptr) {
        LOGI("AAudioLoader::open() could not find " LIB_AAUDIO_NAME);
        return -1; // TODO review return code
    } else {
        LOGD("AAudioLoader():  dlopen(%s) returned %p", LIB_AAUDIO_NAME, mLibHandle);
    }

    // Load all the function pointers.
    createStreamBuilder        = load_I_PPB("AAudio_createStreamBuilder");
    builder_openStream         = load_I_PBPPS("AAudioStreamBuilder_openStream");

    builder_setChannelCount    = load_V_PBI("AAudioStreamBuilder_setChannelCount");
    if (builder_setChannelCount == nullptr) {
        // Use old deprecated alias if needed.
        builder_setChannelCount = load_V_PBI("AAudioStreamBuilder_setSamplesPerFrame");
    }

    builder_setBufferCapacityInFrames = load_V_PBI("AAudioStreamBuilder_setBufferCapacityInFrames");
    builder_setDeviceId        = load_V_PBI("AAudioStreamBuilder_setDeviceId");
    builder_setDirection       = load_V_PBI("AAudioStreamBuilder_setDirection");
    builder_setFormat          = load_V_PBI("AAudioStreamBuilder_setFormat");
    builder_setFramesPerDataCallback = load_V_PBI("AAudioStreamBuilder_setFramesPerDataCallback");
    builder_setSharingMode     = load_V_PBI("AAudioStreamBuilder_setSharingMode");
    builder_setPerformanceMode = load_V_PBI("AAudioStreamBuilder_setPerformanceMode");
    builder_setSampleRate      = load_V_PBI("AAudioStreamBuilder_setSampleRate");

    if (getSdkVersion() >= __ANDROID_API_P__){
        builder_setUsage       = load_V_PBI("AAudioStreamBuilder_setUsage");
        builder_setContentType = load_V_PBI("AAudioStreamBuilder_setContentType");
        builder_setInputPreset = load_V_PBI("AAudioStreamBuilder_setInputPreset");
        builder_setSessionId   = load_V_PBI("AAudioStreamBuilder_setSessionId");
    }

    builder_delete             = load_I_PB("AAudioStreamBuilder_delete");


    builder_setDataCallback    = load_V_PBPDPV("AAudioStreamBuilder_setDataCallback");
    builder_setErrorCallback   = load_V_PBPEPV("AAudioStreamBuilder_setErrorCallback");

    stream_read                = load_I_PSPVIL("AAudioStream_read");

    stream_write               = load_I_PSCPVIL("AAudioStream_write");

    stream_waitForStateChange  = load_I_PSTPTL("AAudioStream_waitForStateChange");

    stream_getTimestamp        = load_I_PSKPLPL("AAudioStream_getTimestamp");

    stream_getChannelCount     = load_I_PS("AAudioStream_getChannelCount");
    if (stream_getChannelCount == nullptr) {
        // Use old alias if needed.
        stream_getChannelCount    = load_I_PS("AAudioStream_getSamplesPerFrame");
    }

    stream_close               = load_I_PS("AAudioStream_close");

    stream_getBufferSize       = load_I_PS("AAudioStream_getBufferSizeInFrames");
    stream_getDeviceId         = load_I_PS("AAudioStream_getDeviceId");
    stream_getBufferCapacity   = load_I_PS("AAudioStream_getBufferCapacityInFrames");
    stream_getFormat           = load_F_PS("AAudioStream_getFormat");
    stream_getFramesPerBurst   = load_I_PS("AAudioStream_getFramesPerBurst");
    stream_getFramesRead       = load_L_PS("AAudioStream_getFramesRead");
    stream_getFramesWritten    = load_L_PS("AAudioStream_getFramesWritten");
    stream_getPerformanceMode  = load_I_PS("AAudioStream_getPerformanceMode");
    stream_getSampleRate       = load_I_PS("AAudioStream_getSampleRate");
    stream_getSharingMode      = load_I_PS("AAudioStream_getSharingMode");
    stream_getState            = load_I_PS("AAudioStream_getState");
    stream_getXRunCount        = load_I_PS("AAudioStream_getXRunCount");

    stream_requestStart        = load_I_PS("AAudioStream_requestStart");
    stream_requestPause        = load_I_PS("AAudioStream_requestPause");
    stream_requestFlush        = load_I_PS("AAudioStream_requestFlush");
    stream_requestStop         = load_I_PS("AAudioStream_requestStop");

    stream_setBufferSize       = load_I_PSI("AAudioStream_setBufferSizeInFrames");

    convertResultToText        = load_CPH_I("AAudio_convertResultToText");

    if (getSdkVersion() >= __ANDROID_API_P__){
        stream_getUsage        = load_I_PS("AAudioStream_getUsage");
        stream_getContentType  = load_I_PS("AAudioStream_getContentType");
        stream_getInputPreset  = load_I_PS("AAudioStream_getInputPreset");
        stream_getSessionId    = load_I_PS("AAudioStream_getSessionId");
    }
    return 0;
}

static void AAudioLoader_check(void *proc, const char *functionName) {
    if (proc == nullptr) {
        LOGW("AAudioLoader could not find %s", functionName);
    }
}

AAudioLoader::signature_I_PPB AAudioLoader::load_I_PPB(const char *functionName) {
    void *proc = dlsym(mLibHandle, functionName);
    AAudioLoader_check(proc, functionName);
    return reinterpret_cast<signature_I_PPB>(proc);
}

AAudioLoader::signature_CPH_I AAudioLoader::load_CPH_I(const char *functionName) {
    void *proc = dlsym(mLibHandle, functionName);
    AAudioLoader_check(proc, functionName);
    return reinterpret_cast<signature_CPH_I>(proc);
}

AAudioLoader::signature_V_PBI AAudioLoader::load_V_PBI(const char *functionName) {
    void *proc = dlsym(mLibHandle, functionName);
    AAudioLoader_check(proc, functionName);
    return reinterpret_cast<signature_V_PBI>(proc);
}

AAudioLoader::signature_V_PBPDPV AAudioLoader::load_V_PBPDPV(const char *functionName) {
    void *proc = dlsym(mLibHandle, functionName);
    AAudioLoader_check(proc, functionName);
    return reinterpret_cast<signature_V_PBPDPV>(proc);
}

AAudioLoader::signature_V_PBPEPV AAudioLoader::load_V_PBPEPV(const char *functionName) {
    void *proc = dlsym(mLibHandle, functionName);
    AAudioLoader_check(proc, functionName);
    return reinterpret_cast<signature_V_PBPEPV>(proc);
}

AAudioLoader::signature_I_PSI AAudioLoader::load_I_PSI(const char *functionName) {
    void *proc = dlsym(mLibHandle, functionName);
    AAudioLoader_check(proc, functionName);
    return reinterpret_cast<signature_I_PSI>(proc);
}

AAudioLoader::signature_I_PS AAudioLoader::load_I_PS(const char *functionName) {
    void *proc = dlsym(mLibHandle, functionName);
    AAudioLoader_check(proc, functionName);
    return reinterpret_cast<signature_I_PS>(proc);
}

AAudioLoader::signature_L_PS AAudioLoader::load_L_PS(const char *functionName) {
    void *proc = dlsym(mLibHandle, functionName);
    AAudioLoader_check(proc, functionName);
    return reinterpret_cast<signature_L_PS>(proc);
}

AAudioLoader::signature_F_PS AAudioLoader::load_F_PS(const char *functionName) {
    void *proc = dlsym(mLibHandle, functionName);
    AAudioLoader_check(proc, functionName);
    return reinterpret_cast<signature_F_PS>(proc);
}

AAudioLoader::signature_B_PS AAudioLoader::load_B_PS(const char *functionName) {
    void *proc = dlsym(mLibHandle, functionName);
    AAudioLoader_check(proc, functionName);
    return reinterpret_cast<signature_B_PS>(proc);
}

AAudioLoader::signature_I_PB AAudioLoader::load_I_PB(const char *functionName) {
    void *proc = dlsym(mLibHandle, functionName);
    AAudioLoader_check(proc, functionName);
    return reinterpret_cast<signature_I_PB>(proc);
}

AAudioLoader::signature_I_PBPPS AAudioLoader::load_I_PBPPS(const char *functionName) {
    void *proc = dlsym(mLibHandle, functionName);
    AAudioLoader_check(proc, functionName);
    return reinterpret_cast<signature_I_PBPPS>(proc);
}

AAudioLoader::signature_I_PSCPVIL AAudioLoader::load_I_PSCPVIL(const char *functionName) {
    void *proc = dlsym(mLibHandle, functionName);
    AAudioLoader_check(proc, functionName);
    return reinterpret_cast<signature_I_PSCPVIL>(proc);
}

AAudioLoader::signature_I_PSPVIL AAudioLoader::load_I_PSPVIL(const char *functionName) {
    void *proc = dlsym(mLibHandle, functionName);
    AAudioLoader_check(proc, functionName);
    return reinterpret_cast<signature_I_PSPVIL>(proc);
}

AAudioLoader::signature_I_PSTPTL AAudioLoader::load_I_PSTPTL(const char *functionName) {
    void *proc = dlsym(mLibHandle, functionName);
    AAudioLoader_check(proc, functionName);
    return reinterpret_cast<signature_I_PSTPTL>(proc);
}

AAudioLoader::signature_I_PSKPLPL AAudioLoader::load_I_PSKPLPL(const char *functionName) {
    void *proc = dlsym(mLibHandle, functionName);
    AAudioLoader_check(proc, functionName);
    return reinterpret_cast<signature_I_PSKPLPL>(proc);
}

// Ensure that all AAudio primitive data types are int32_t
#define ASSERT_INT32(type) static_assert(std::is_same<int32_t, type>::value, \
#type" must be int32_t")

#define ERRMSG "Oboe constants must match AAudio constants."

// These asserts help verify that the Oboe definitions match the equivalent AAudio definitions.
// This code is in this .cpp file so it only gets tested once.
#ifdef AAUDIO_AAUDIO_H

    ASSERT_INT32(aaudio_stream_state_t);
    ASSERT_INT32(aaudio_direction_t);
    ASSERT_INT32(aaudio_format_t);
    ASSERT_INT32(aaudio_data_callback_result_t);
    ASSERT_INT32(aaudio_result_t);
    ASSERT_INT32(aaudio_sharing_mode_t);
    ASSERT_INT32(aaudio_performance_mode_t);

    static_assert((int32_t)StreamState::Uninitialized == AAUDIO_STREAM_STATE_UNINITIALIZED, ERRMSG);
    static_assert((int32_t)StreamState::Unknown == AAUDIO_STREAM_STATE_UNKNOWN, ERRMSG);
    static_assert((int32_t)StreamState::Open == AAUDIO_STREAM_STATE_OPEN, ERRMSG);
    static_assert((int32_t)StreamState::Starting == AAUDIO_STREAM_STATE_STARTING, ERRMSG);
    static_assert((int32_t)StreamState::Started == AAUDIO_STREAM_STATE_STARTED, ERRMSG);
    static_assert((int32_t)StreamState::Pausing == AAUDIO_STREAM_STATE_PAUSING, ERRMSG);
    static_assert((int32_t)StreamState::Paused == AAUDIO_STREAM_STATE_PAUSED, ERRMSG);
    static_assert((int32_t)StreamState::Flushing == AAUDIO_STREAM_STATE_FLUSHING, ERRMSG);
    static_assert((int32_t)StreamState::Flushed == AAUDIO_STREAM_STATE_FLUSHED, ERRMSG);
    static_assert((int32_t)StreamState::Stopping == AAUDIO_STREAM_STATE_STOPPING, ERRMSG);
    static_assert((int32_t)StreamState::Stopped == AAUDIO_STREAM_STATE_STOPPED, ERRMSG);
    static_assert((int32_t)StreamState::Closing == AAUDIO_STREAM_STATE_CLOSING, ERRMSG);
    static_assert((int32_t)StreamState::Closed == AAUDIO_STREAM_STATE_CLOSED, ERRMSG);
    static_assert((int32_t)StreamState::Disconnected == AAUDIO_STREAM_STATE_DISCONNECTED, ERRMSG);

    static_assert((int32_t)Direction::Output == AAUDIO_DIRECTION_OUTPUT, ERRMSG);
    static_assert((int32_t)Direction::Input == AAUDIO_DIRECTION_INPUT, ERRMSG);

    static_assert((int32_t)AudioFormat::Invalid == AAUDIO_FORMAT_INVALID, ERRMSG);
    static_assert((int32_t)AudioFormat::Unspecified == AAUDIO_FORMAT_UNSPECIFIED, ERRMSG);
    static_assert((int32_t)AudioFormat::I16 == AAUDIO_FORMAT_PCM_I16, ERRMSG);
    static_assert((int32_t)AudioFormat::Float == AAUDIO_FORMAT_PCM_FLOAT, ERRMSG);

    static_assert((int32_t)DataCallbackResult::Continue == AAUDIO_CALLBACK_RESULT_CONTINUE, ERRMSG);
    static_assert((int32_t)DataCallbackResult::Stop == AAUDIO_CALLBACK_RESULT_STOP, ERRMSG);

    static_assert((int32_t)Result::OK == AAUDIO_OK, ERRMSG);
    static_assert((int32_t)Result::ErrorBase == AAUDIO_ERROR_BASE, ERRMSG);
    static_assert((int32_t)Result::ErrorDisconnected == AAUDIO_ERROR_DISCONNECTED, ERRMSG);
    static_assert((int32_t)Result::ErrorIllegalArgument == AAUDIO_ERROR_ILLEGAL_ARGUMENT, ERRMSG);
    static_assert((int32_t)Result::ErrorInternal == AAUDIO_ERROR_INTERNAL, ERRMSG);
    static_assert((int32_t)Result::ErrorInvalidState == AAUDIO_ERROR_INVALID_STATE, ERRMSG);
    static_assert((int32_t)Result::ErrorInvalidHandle == AAUDIO_ERROR_INVALID_HANDLE, ERRMSG);
    static_assert((int32_t)Result::ErrorUnimplemented == AAUDIO_ERROR_UNIMPLEMENTED, ERRMSG);
    static_assert((int32_t)Result::ErrorUnavailable == AAUDIO_ERROR_UNAVAILABLE, ERRMSG);
    static_assert((int32_t)Result::ErrorNoFreeHandles == AAUDIO_ERROR_NO_FREE_HANDLES, ERRMSG);
    static_assert((int32_t)Result::ErrorNoMemory == AAUDIO_ERROR_NO_MEMORY, ERRMSG);
    static_assert((int32_t)Result::ErrorNull == AAUDIO_ERROR_NULL, ERRMSG);
    static_assert((int32_t)Result::ErrorTimeout == AAUDIO_ERROR_TIMEOUT, ERRMSG);
    static_assert((int32_t)Result::ErrorWouldBlock == AAUDIO_ERROR_WOULD_BLOCK, ERRMSG);
    static_assert((int32_t)Result::ErrorInvalidFormat == AAUDIO_ERROR_INVALID_FORMAT, ERRMSG);
    static_assert((int32_t)Result::ErrorOutOfRange == AAUDIO_ERROR_OUT_OF_RANGE, ERRMSG);
    static_assert((int32_t)Result::ErrorNoService == AAUDIO_ERROR_NO_SERVICE, ERRMSG);
    static_assert((int32_t)Result::ErrorInvalidRate == AAUDIO_ERROR_INVALID_RATE, ERRMSG);

    static_assert((int32_t)SharingMode::Exclusive == AAUDIO_SHARING_MODE_EXCLUSIVE, ERRMSG);
    static_assert((int32_t)SharingMode::Shared == AAUDIO_SHARING_MODE_SHARED, ERRMSG);

    static_assert((int32_t)PerformanceMode::None == AAUDIO_PERFORMANCE_MODE_NONE, ERRMSG);
    static_assert((int32_t)PerformanceMode::PowerSaving
            == AAUDIO_PERFORMANCE_MODE_POWER_SAVING, ERRMSG);
    static_assert((int32_t)PerformanceMode::LowLatency
            == AAUDIO_PERFORMANCE_MODE_LOW_LATENCY, ERRMSG);

// The aaudio_ usage, content and input_preset types were added in NDK 17,
// which is the first version to support Android Pie (API 28).
#if __NDK_MAJOR__ >= 17

    ASSERT_INT32(aaudio_usage_t);
    ASSERT_INT32(aaudio_content_type_t);
    ASSERT_INT32(aaudio_input_preset_t);

    static_assert((int32_t)Usage::Media == AAUDIO_USAGE_MEDIA, ERRMSG);
    static_assert((int32_t)Usage::VoiceCommunication == AAUDIO_USAGE_VOICE_COMMUNICATION, ERRMSG);
    static_assert((int32_t)Usage::VoiceCommunicationSignalling
            == AAUDIO_USAGE_VOICE_COMMUNICATION_SIGNALLING, ERRMSG);
    static_assert((int32_t)Usage::Alarm == AAUDIO_USAGE_ALARM, ERRMSG);
    static_assert((int32_t)Usage::Notification == AAUDIO_USAGE_NOTIFICATION, ERRMSG);
    static_assert((int32_t)Usage::NotificationRingtone == AAUDIO_USAGE_NOTIFICATION_RINGTONE, ERRMSG);
    static_assert((int32_t)Usage::NotificationEvent == AAUDIO_USAGE_NOTIFICATION_EVENT, ERRMSG);
    static_assert((int32_t)Usage::AssistanceAccessibility == AAUDIO_USAGE_ASSISTANCE_ACCESSIBILITY, ERRMSG);
    static_assert((int32_t)Usage::AssistanceNavigationGuidance
            == AAUDIO_USAGE_ASSISTANCE_NAVIGATION_GUIDANCE, ERRMSG);
    static_assert((int32_t)Usage::AssistanceSonification == AAUDIO_USAGE_ASSISTANCE_SONIFICATION, ERRMSG);
    static_assert((int32_t)Usage::Game == AAUDIO_USAGE_GAME, ERRMSG);
    static_assert((int32_t)Usage::Assistant == AAUDIO_USAGE_ASSISTANT, ERRMSG);

    static_assert((int32_t)ContentType::Speech == AAUDIO_CONTENT_TYPE_SPEECH, ERRMSG);
    static_assert((int32_t)ContentType::Music == AAUDIO_CONTENT_TYPE_MUSIC, ERRMSG);
    static_assert((int32_t)ContentType::Movie == AAUDIO_CONTENT_TYPE_MOVIE, ERRMSG);
    static_assert((int32_t)ContentType::Sonification == AAUDIO_CONTENT_TYPE_SONIFICATION, ERRMSG);

    static_assert((int32_t)InputPreset::Generic == AAUDIO_INPUT_PRESET_GENERIC, ERRMSG);
    static_assert((int32_t)InputPreset::Camcorder == AAUDIO_INPUT_PRESET_CAMCORDER, ERRMSG);
    static_assert((int32_t)InputPreset::VoiceRecognition == AAUDIO_INPUT_PRESET_VOICE_RECOGNITION, ERRMSG);
    static_assert((int32_t)InputPreset::VoiceCommunication
            == AAUDIO_INPUT_PRESET_VOICE_COMMUNICATION, ERRMSG);
    static_assert((int32_t)InputPreset::Unprocessed == AAUDIO_INPUT_PRESET_UNPROCESSED, ERRMSG);

    static_assert((int32_t)SessionId::None == AAUDIO_SESSION_ID_NONE, ERRMSG);
    static_assert((int32_t)SessionId::Allocate == AAUDIO_SESSION_ID_ALLOCATE, ERRMSG);

#endif // __NDK_MAJOR__ >= 17

#endif // AAUDIO_AAUDIO_H

} // namespace oboe
