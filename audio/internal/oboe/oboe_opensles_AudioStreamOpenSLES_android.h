/*
 * Copyright 2015 The Android Open Source Project
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

#ifndef OBOE_AUDIO_STREAM_OPENSL_ES_H_
#define OBOE_AUDIO_STREAM_OPENSL_ES_H_

#include <memory>

#include <SLES/OpenSLES.h>
#include <SLES/OpenSLES_Android.h>

#include "oboe_oboe_Oboe_android.h"
#include "oboe_common_MonotonicCounter_android.h"
#include "oboe_opensles_AudioStreamBuffered_android.h"
#include "oboe_opensles_EngineOpenSLES_android.h"

namespace oboe {

constexpr int kBitsPerByte = 8;
constexpr int kBufferQueueLength = 2; // double buffered for callbacks

/**
 * INTERNAL USE ONLY
 *
 * A stream that wraps OpenSL ES.
 *
 * Do not instantiate this class directly.
 * Use an OboeStreamBuilder to create one.
 */

class AudioStreamOpenSLES : public AudioStreamBuffered {
public:

    AudioStreamOpenSLES();
    explicit AudioStreamOpenSLES(const AudioStreamBuilder &builder);

    virtual ~AudioStreamOpenSLES() = default;

    virtual Result open() override;

    /**
     * Query the current state, eg. OBOE_STREAM_STATE_PAUSING
     *
     * @return state or a negative error.
     */
    StreamState getState() override { return mState.load(); }

    AudioApi getAudioApi() const override {
        return AudioApi::OpenSLES;
    }

    /**
     * Process next OpenSL ES buffer.
     * Called by by OpenSL ES framework.
     *
     * This is public, but don't call it directly.
     */
    void processBufferCallback(SLAndroidSimpleBufferQueueItf bq);

    Result waitForStateChange(StreamState currentState,
                              StreamState *nextState,
                              int64_t timeoutNanoseconds) override;

protected:

    // This must be called under mLock.
    Result close_l();

    SLuint32 channelCountToChannelMaskDefault(int channelCount) const;

    virtual Result onBeforeDestroy() { return Result::OK; }
    virtual Result onAfterDestroy() { return Result::OK; }

    static SLuint32 getDefaultByteOrder();

    SLresult registerBufferQueueCallback();

    int32_t getBufferDepth(SLAndroidSimpleBufferQueueItf bq);

    SLresult enqueueCallbackBuffer(SLAndroidSimpleBufferQueueItf bq);

    SLresult configurePerformanceMode(SLAndroidConfigurationItf configItf);

    SLresult updateStreamParameters(SLAndroidConfigurationItf configItf);

    PerformanceMode convertPerformanceMode(SLuint32 openslMode) const;
    SLuint32 convertPerformanceMode(PerformanceMode oboeMode) const;

    Result configureBufferSizes(int32_t sampleRate);

    void logUnsupportedAttributes();

    /**
     * Internal use only.
     * Use this instead of directly setting the internal state variable.
     */
    void setState(StreamState state) {
        mState.store(state);
    }

    int64_t getFramesProcessedByServer();

    // OpenSLES stuff
    SLObjectItf                   mObjectInterface = nullptr;
    SLAndroidSimpleBufferQueueItf mSimpleBufferQueueInterface = nullptr;

    int32_t                       mBytesPerCallback = oboe::kUnspecified;
    MonotonicCounter              mPositionMillis; // for tracking OpenSL ES service position

private:
    std::unique_ptr<uint8_t[]>    mCallbackBuffer;
    std::atomic<StreamState>      mState{StreamState::Uninitialized};

};

} // namespace oboe

#endif // OBOE_AUDIO_STREAM_OPENSL_ES_H_
