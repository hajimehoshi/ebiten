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

#include "binding_android.h"

#include "_cgo_export.h"
#include "oboe_oboe_Oboe_android.h"

#include <memory>
#include <vector>

namespace {

class Player;

class Stream : public oboe::AudioStreamDataCallback {
public:
  // GetInstance returns the instance of Stream. Only one Stream object is used
  // in one process. It is because multiple streams can be problematic in both
  // AAudio and OpenSL (#1656, #1660).
  static Stream &GetInstance();

  const char *Play(int sample_rate, int channel_num, int bit_depth_in_bytes);
  const char *Pause();
  const char *Resume();
  const char *Close();

  oboe::DataCallbackResult onAudioReady(oboe::AudioStream *oboe_stream,
                                        void *audio_data,
                                        int32_t num_frames) override;

private:
  Stream();
  int sample_rate_ = 0;
  int channel_num_ = 0;
  int bit_depth_in_bytes_ = 0;

  std::shared_ptr<oboe::AudioStream> stream_;
};

Stream &Stream::GetInstance() {
  static Stream *stream = new Stream();
  return *stream;
}

const char *Stream::Play(int sample_rate, int channel_num,
                         int bit_depth_in_bytes) {
  sample_rate_ = sample_rate;
  channel_num_ = channel_num;
  bit_depth_in_bytes_ = bit_depth_in_bytes;

  // TODO: Enable bit_depth_in_bytes_ == 1
  if (bit_depth_in_bytes_ != 2) {
    return "bit_depth_in_bytes_ must be 2 but not";
  }

  if (!stream_) {
    oboe::AudioStreamBuilder builder;
    oboe::Result result =
        builder.setDirection(oboe::Direction::Output)
            ->setPerformanceMode(oboe::PerformanceMode::LowLatency)
            ->setSharingMode(oboe::SharingMode::Shared)
            ->setFormat(oboe::AudioFormat::Float)
            ->setChannelCount(channel_num_)
            ->setSampleRate(sample_rate_)
            ->setBufferCapacityInFrames(1024)
            ->setDataCallback(this)
            ->openStream(stream_);
    if (result != oboe::Result::OK) {
      return oboe::convertToText(result);
    }
  }
  if (stream_->getSharingMode() != oboe::SharingMode::Shared) {
    return "oboe::SharingMode::Shared is not available";
  }
  // What if the buffer size is not enough?
  if (oboe::Result result = stream_->start(); result != oboe::Result::OK) {
    return oboe::convertToText(result);
  }
  return nullptr;
}

const char *Stream::Pause() {
  if (!stream_) {
    return nullptr;
  }
  if (oboe::Result result = stream_->pause(); result != oboe::Result::OK) {
    return oboe::convertToText(result);
  }
  return nullptr;
}

const char *Stream::Resume() {
  if (!stream_) {
    return "Play is not called yet at Resume";
  }
  if (oboe::Result result = stream_->start(); result != oboe::Result::OK) {
    return oboe::convertToText(result);
  }
  return nullptr;
}

const char *Stream::Close() {
  // Nobody calls this so far.
  if (!stream_) {
    return nullptr;
  }
  if (oboe::Result result = stream_->stop(); result != oboe::Result::OK) {
    return oboe::convertToText(result);
  }
  if (oboe::Result result = stream_->close(); result != oboe::Result::OK) {
    return oboe::convertToText(result);
  }
  stream_.reset();
  return nullptr;
}

oboe::DataCallbackResult Stream::onAudioReady(oboe::AudioStream *oboe_stream,
                                              void *audio_data,
                                              int32_t num_frames) {
  size_t num = num_frames * channel_num_;
  float *dst = reinterpret_cast<float *>(audio_data);
  // TODO: Do not use a lock in onAudioReady.
  // https://google.github.io/oboe/reference/classoboe_1_1_audio_stream_data_callback.html#ad8a3a9f609df5fd3a5d885cbe1b2204d
  ebiten_oboe_read(dst, num);
  return oboe::DataCallbackResult::Continue;
}

Stream::Stream() = default;

} // namespace

extern "C" {

const char *ebiten_oboe_Play(int sample_rate, int channel_num,
                             int bit_depth_in_bytes) {
  return Stream::GetInstance().Play(sample_rate, channel_num,
                                    bit_depth_in_bytes);
}

const char *ebiten_oboe_Suspend() { return Stream::GetInstance().Pause(); }

const char *ebiten_oboe_Resume() { return Stream::GetInstance().Resume(); }

} // extern "C"
