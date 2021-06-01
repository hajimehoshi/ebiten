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

#include <atomic>
#include <set>
#include <vector>

#include <android/log.h>

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

  void AddPlayer(Player *player);
  void RemovePlayer(Player *player);

  oboe::DataCallbackResult onAudioReady(oboe::AudioStream *oboe_stream,
                                        void *audio_data,
                                        int32_t num_frames) override;

private:
  Stream();
  int sample_rate_ = 0;
  int channel_num_ = 0;
  int bit_depth_in_bytes_ = 0;

  std::mutex mutex_;
  std::set<Player *> players_;
  std::shared_ptr<oboe::AudioStream> stream_;
};

class Player {
public:
  Player(double volume, uintptr_t go_player) : go_player_{go_player} {
    std::atomic_store(&volume_, volume);
    Stream::GetInstance().AddPlayer(this);
  }

  ~Player() { Stream::GetInstance().RemovePlayer(this); }

  void SetVolume(double volume) { std::atomic_store(&volume_, volume); }

  void Play() { std::atomic_store(&playing_, true); }

  void Pause() { std::atomic_store(&playing_, false); }

  bool IsPlaying() { return std::atomic_load(&playing_); }

  void AppendBuffer(uint8_t *data, int length) {
    std::lock_guard<std::mutex> lock(mutex_);
    buf_.insert(buf_.end(), data, data + length);
  }

  size_t GetUnplayedBufferSize() {
    std::lock_guard<std::mutex> lock(mutex_);
    return buf_.size();
  }

  size_t Read(std::vector<float> &buf) {
    if (!std::atomic_load(&playing_)) {
      return 0;
    }

    const double volume = std::atomic_load(&volume_);
    size_t copy_num = 0;
    {
      // TODO: Do not use a lock in onAudioReady.
      // https://google.github.io/oboe/reference/classoboe_1_1_audio_stream_data_callback.html#ad8a3a9f609df5fd3a5d885cbe1b2204d
      std::lock_guard<std::mutex> lock(mutex_);
      copy_num = std::min(buf.size(), buf_.size() / 2);
      for (size_t i = 0; i < copy_num; i++) {
        int16_t v = static_cast<int16_t>(buf_[2 * i]) |
                    (static_cast<int16_t>(buf_[2 * i + 1]) << 8);
        buf[i] = static_cast<float>(v) / (1 << 15) * volume;
      }
      buf_.erase(buf_.begin(), buf_.begin() + copy_num * 2);
    }

    if (copy_num) {
      ebiten_oboe_onWrittenCallback(go_player_);
    }
    return copy_num;
  }

private:
  const uintptr_t go_player_;

  std::atomic<double> volume_{1.0};
  std::atomic<bool> playing_;
  std::vector<uint8_t> buf_;
  std::mutex mutex_;
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

void Stream::AddPlayer(Player *player) {
  std::lock_guard<std::mutex> lock(mutex_);
  players_.insert(player);
}

void Stream::RemovePlayer(Player *player) {
  std::lock_guard<std::mutex> lock(mutex_);
  players_.erase(player);
}

oboe::DataCallbackResult Stream::onAudioReady(oboe::AudioStream *oboe_stream,
                                              void *audio_data,
                                              int32_t num_frames) {
  size_t num = num_frames * channel_num_;
  std::vector<std::vector<float>> bufs;
  {
    // TODO: Do not use a lock in onAudioReady.
    // https://google.github.io/oboe/reference/classoboe_1_1_audio_stream_data_callback.html#ad8a3a9f609df5fd3a5d885cbe1b2204d
    std::lock_guard<std::mutex> lock(mutex_);
    bufs.resize(players_.size());
    size_t i = 0;
    for (Player *player : players_) {
      bufs[i].resize(num);
      player->Read(bufs[i]);
      i++;
    }
  }

  float *dst = reinterpret_cast<float *>(audio_data);
  for (int i = 0; i < num; i++) {
    dst[i] = 0;
    for (const std::vector<float> buf : bufs) {
      dst[i] += buf[i];
    }
  }
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

PlayerID ebiten_oboe_Player_Create(double volume, uintptr_t go_player) {
  Player *p = new Player(volume, go_player);
  return reinterpret_cast<PlayerID>(p);
}

bool ebiten_oboe_Player_IsPlaying(PlayerID audio_player) {
  Player *p = reinterpret_cast<Player *>(audio_player);
  return p->IsPlaying();
}

void ebiten_oboe_Player_AppendBuffer(PlayerID audio_player, uint8_t *data,
                                     int length) {
  Player *p = reinterpret_cast<Player *>(audio_player);
  p->AppendBuffer(data, length);
}

void ebiten_oboe_Player_Play(PlayerID audio_player) {
  Player *p = reinterpret_cast<Player *>(audio_player);
  p->Play();
}

void ebiten_oboe_Player_Pause(PlayerID audio_player) {
  Player *p = reinterpret_cast<Player *>(audio_player);
  return p->Pause();
}

void ebiten_oboe_Player_SetVolume(PlayerID audio_player, double volume) {
  Player *p = reinterpret_cast<Player *>(audio_player);
  p->SetVolume(volume);
}

void ebiten_oboe_Player_Close(PlayerID audio_player) {
  Player *p = reinterpret_cast<Player *>(audio_player);
  delete p;
}

int ebiten_oboe_Player_UnplayedBufferSize(PlayerID audio_player) {
  Player *p = reinterpret_cast<Player *>(audio_player);
  return p->GetUnplayedBufferSize();
}

} // extern "C"
