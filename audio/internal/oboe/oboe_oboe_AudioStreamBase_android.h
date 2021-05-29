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

#ifndef OBOE_STREAM_BASE_H_
#define OBOE_STREAM_BASE_H_

#include <memory>
#include "oboe_oboe_AudioStreamCallback_android.h"
#include "oboe_oboe_Definitions_android.h"

namespace oboe {

/**
 * Base class containing parameters for audio streams and builders.
 **/
class AudioStreamBase {

public:

    AudioStreamBase() {}

    virtual ~AudioStreamBase() = default;

    // This class only contains primitives so we can use default constructor and copy methods.

    /**
     * Default copy constructor
     */
    AudioStreamBase(const AudioStreamBase&) = default;

    /**
     * Default assignment operator
     */
    AudioStreamBase& operator=(const AudioStreamBase&) = default;

    /**
     * @return number of channels, for example 2 for stereo, or kUnspecified
     */
    int32_t getChannelCount() const { return mChannelCount; }

    /**
     * @return Direction::Input or Direction::Output
     */
    Direction getDirection() const { return mDirection; }

    /**
     * @return sample rate for the stream or kUnspecified
     */
    int32_t getSampleRate() const { return mSampleRate; }

    /**
     * @deprecated use `getFramesPerDataCallback` instead.
     */
    int32_t getFramesPerCallback() const { return getFramesPerDataCallback(); }

    /**
     * @return the number of frames in each data callback or kUnspecified.
     */
    int32_t getFramesPerDataCallback() const { return mFramesPerCallback; }

    /**
     * @return the audio sample format (e.g. Float or I16)
     */
    AudioFormat getFormat() const { return mFormat; }

    /**
     * Query the maximum number of frames that can be filled without blocking.
     * If the stream has been closed the last known value will be returned.
     *
     * @return buffer size
     */
    virtual int32_t getBufferSizeInFrames() { return mBufferSizeInFrames; }

    /**
     * @return capacityInFrames or kUnspecified
     */
    virtual int32_t getBufferCapacityInFrames() const { return mBufferCapacityInFrames; }

    /**
     * @return the sharing mode of the stream.
     */
    SharingMode getSharingMode() const { return mSharingMode; }

    /**
     * @return the performance mode of the stream.
     */
    PerformanceMode getPerformanceMode() const { return mPerformanceMode; }

    /**
     * @return the device ID of the stream.
     */
    int32_t getDeviceId() const { return mDeviceId; }

    /**
     * For internal use only.
     * @return the data callback object for this stream, if set.
     */
    AudioStreamDataCallback *getDataCallback() const {
        return mDataCallback;
    }

    /**
     * For internal use only.
     * @return the error callback object for this stream, if set.
     */
    AudioStreamErrorCallback *getErrorCallback() const {
        return mErrorCallback;
    }

    /**
     * @return true if a data callback was set for this stream
     */
    bool isDataCallbackSpecified() const {
        return mDataCallback != nullptr;
    }

    /**
     * Note that if the app does not set an error callback then a
     * default one may be provided.
     * @return true if an error callback was set for this stream
     */
    bool isErrorCallbackSpecified() const {
        return mErrorCallback != nullptr;
    }

    /**
     * @return the usage for this stream.
     */
    Usage getUsage() const { return mUsage; }

    /**
     * @return the stream's content type.
     */
    ContentType getContentType() const { return mContentType; }

    /**
     * @return the stream's input preset.
     */
    InputPreset getInputPreset() const { return mInputPreset; }

    /**
     * @return the stream's session ID allocation strategy (None or Allocate).
     */
    SessionId getSessionId() const { return mSessionId; }

    /**
     * @return true if Oboe can convert channel counts to achieve optimal results.
     */
    bool isChannelConversionAllowed() const {
        return mChannelConversionAllowed;
    }

    /**
     * @return true if  Oboe can convert data formats to achieve optimal results.
     */
    bool  isFormatConversionAllowed() const {
        return mFormatConversionAllowed;
    }

    /**
     * @return whether and how Oboe can convert sample rates to achieve optimal results.
     */
    SampleRateConversionQuality getSampleRateConversionQuality() const {
        return mSampleRateConversionQuality;
    }

protected:
    /** The callback which will be fired when new data is ready to be read/written. **/
    AudioStreamDataCallback        *mDataCallback = nullptr;

    /** The callback which will be fired when an error or a disconnect occurs. **/
    AudioStreamErrorCallback       *mErrorCallback = nullptr;

    /** Number of audio frames which will be requested in each callback */
    int32_t                         mFramesPerCallback = kUnspecified;
    /** Stream channel count */
    int32_t                         mChannelCount = kUnspecified;
    /** Stream sample rate */
    int32_t                         mSampleRate = kUnspecified;
    /** Stream audio device ID */
    int32_t                         mDeviceId = kUnspecified;
    /** Stream buffer capacity specified as a number of audio frames */
    int32_t                         mBufferCapacityInFrames = kUnspecified;
    /** Stream buffer size specified as a number of audio frames */
    int32_t                         mBufferSizeInFrames = kUnspecified;

    /** Stream sharing mode */
    SharingMode                     mSharingMode = SharingMode::Shared;
    /** Format of audio frames */
    AudioFormat                     mFormat = AudioFormat::Unspecified;
    /** Stream direction */
    Direction                       mDirection = Direction::Output;
    /** Stream performance mode */
    PerformanceMode                 mPerformanceMode = PerformanceMode::None;

    /** Stream usage. Only active on Android 28+ */
    Usage                           mUsage = Usage::Media;
    /** Stream content type. Only active on Android 28+ */
    ContentType                     mContentType = ContentType::Music;
    /** Stream input preset. Only active on Android 28+
     * TODO InputPreset::Unspecified should be considered as a possible default alternative.
    */
    InputPreset                     mInputPreset = InputPreset::VoiceRecognition;
    /** Stream session ID allocation strategy. Only active on Android 28+ */
    SessionId                       mSessionId = SessionId::None;

    // Control whether Oboe can convert channel counts to achieve optimal results.
    bool                            mChannelConversionAllowed = false;
    // Control whether Oboe can convert data formats to achieve optimal results.
    bool                            mFormatConversionAllowed = false;
    // Control whether and how Oboe can convert sample rates to achieve optimal results.
    SampleRateConversionQuality     mSampleRateConversionQuality = SampleRateConversionQuality::None;

    /** Validate stream parameters that might not be checked in lower layers */
    virtual Result isValidConfig() {
        switch (mFormat) {
            case AudioFormat::Unspecified:
            case AudioFormat::I16:
            case AudioFormat::Float:
            case AudioFormat::I24:
            case AudioFormat::I32:
                break;

            default:
                return Result::ErrorInvalidFormat;
        }

        switch (mSampleRateConversionQuality) {
            case SampleRateConversionQuality::None:
            case SampleRateConversionQuality::Fastest:
            case SampleRateConversionQuality::Low:
            case SampleRateConversionQuality::Medium:
            case SampleRateConversionQuality::High:
            case SampleRateConversionQuality::Best:
                return Result::OK;
            default:
                return Result::ErrorIllegalArgument;
        }
    }
};

} // namespace oboe

#endif /* OBOE_STREAM_BASE_H_ */
