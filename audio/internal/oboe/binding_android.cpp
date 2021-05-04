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

namespace {

class Player : public oboe::AudioStreamDataCallback {
public:
  static const char* Suspend() {
    std::lock_guard<std::mutex> lock(GetPlayersMutex());
    for (Player* player : GetPlayers()) {
      // Close should be called rather than Pause for onPause.
      // https://github.com/google/oboe/blob/master/docs/GettingStarted.md
      if (const char* msg = player->Close(); msg) {
        return msg;
      }
    }
    return nullptr;
  }

  static const char* Resume() {
    std::lock_guard<std::mutex> lock(GetPlayersMutex());
    for (Player* player : GetPlayers()) {
      if (const char* msg = player->Play(); msg) {
        return msg;
      }
    }
    return nullptr;
  }

  Player(int sample_rate, int channel_num, int bit_depth_in_bytes, double volume, uintptr_t go_player)
    : sample_rate_{sample_rate},
      channel_num_{channel_num},
      bit_depth_in_bytes_{bit_depth_in_bytes},
      go_player_{go_player} {
    std::atomic_store(&volume_, volume);
    {
      std::lock_guard<std::mutex> lock(GetPlayersMutex());
      GetPlayers().insert(this);
    }
    // Fill zeros with 1/60[s] as the first part to avoid noises (#1632).
    // 1/60[s] is an arbitrary duration and might need to be adjusted.
    size_t mul = channel_num_ * bit_depth_in_bytes_;
    size_t size = (sample_rate_ * channel_num_ * bit_depth_in_bytes_) / 60 / mul * mul;
    buf_.resize(size);
  }

  void SetVolume(double volume) {
    std::atomic_store(&volume_, volume);
  }

  void AppendBuffer(uint8_t* data, int length) {
    // Sync this constants with internal/readerdriver/driver.go
    const size_t bytes_per_sample = channel_num_ * bit_depth_in_bytes_;
    const size_t one_buffer_size = sample_rate_ * channel_num_ * bit_depth_in_bytes_ / 4 / bytes_per_sample * bytes_per_sample;
    const size_t max_buffer_size = one_buffer_size * 2;

    std::lock_guard<std::mutex> lock(mutex_);
    buf_.insert(buf_.end(), data, data + length);
  }

  const char* Play() {
    if (bit_depth_in_bytes_ != 2) {
      return "bit_depth_in_bytes_ must be 2 but not";
    }

    if (!stream_) {
      oboe::AudioStreamBuilder builder;
      oboe::Result result = builder.setDirection(oboe::Direction::Output)
        ->setPerformanceMode(oboe::PerformanceMode::LowLatency)
        ->setSharingMode(oboe::SharingMode::Shared)
        ->setFormat(oboe::AudioFormat::I16)
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

  const char* Pause() {
    if (!stream_) {
      return nullptr;
    }
    if (oboe::Result result = stream_->pause(); result != oboe::Result::OK) {
      return oboe::convertToText(result);
    }
    return nullptr;
  }

  const char* Close() {
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

  const char* CloseAndRemove() {
    // Close and remove self from the players atomically.
    // Otherwise, a removed player might be resumed at Resume unexpectedly.
    std::lock_guard<std::mutex> lock(GetPlayersMutex());
    const char* msg = Close();
    GetPlayers().erase(this);
    return msg;
  }

  int GetUnplayedBufferSize() {
    std::lock_guard<std::mutex> lock(mutex_);
    return buf_.size();
  }

  oboe::DataCallbackResult onAudioReady(oboe::AudioStream *oboe_stream, void *audio_data, int32_t num_frames) override {
    size_t num_bytes = num_frames * channel_num_ * bit_depth_in_bytes_;
    std::vector<uint8_t> buf(num_bytes);
    {
      // TODO: Do not use a lock in onAudioReady.
      // https://google.github.io/oboe/reference/classoboe_1_1_audio_stream_data_callback.html#ad8a3a9f609df5fd3a5d885cbe1b2204d
      std::lock_guard<std::mutex> lock(mutex_);
      size_t copy_bytes = std::min(num_bytes, buf_.size());
      std::copy(buf_.begin(), buf_.begin() + copy_bytes, buf.begin());
      buf_.erase(buf_.begin(), buf_.begin() + copy_bytes);
      onWrittenCallback(go_player_);
    }

    if (const double volume = std::atomic_load(&volume_); volume < 1) {
      for (int i = 0; i < buf.size()/2; i++) {
        int16_t v = static_cast<int16_t>(buf[2*i]) | (static_cast<int16_t>(buf[2*i+1]) << 8);
        v = static_cast<int16_t>(static_cast<double>(v) * volume);
        buf[2*i] = static_cast<uint8_t>(v);
        buf[2*i+1] = static_cast<uint8_t>(v >> 8);
      }
    }

    std::copy(buf.begin(), buf.end(), reinterpret_cast<uint8_t*>(audio_data));
    return oboe::DataCallbackResult::Continue;
  }

private:
  static std::set<Player*>& GetPlayers() {
    static std::set<Player*> players;
    return players;
  }

  static std::mutex& GetPlayersMutex() {
    static std::mutex mutex;
    return mutex;
  }

  const int sample_rate_;
  const int channel_num_;
  const int bit_depth_in_bytes_;
  const uintptr_t go_player_;

  std::atomic<double> volume_{1.0};
  std::vector<uint8_t> buf_;
  std::mutex mutex_;
  std::shared_ptr<oboe::AudioStream> stream_;
};

}  // namespace

extern "C" {

const char* Suspend() {
  return Player::Suspend();
}

const char* Resume() {
  return Player::Resume();
}

PlayerID Player_Create(int sample_rate, int channel_num, int bit_depth_in_bytes, double volume, uintptr_t go_player) {
  Player* p = new Player(sample_rate, channel_num, bit_depth_in_bytes, volume, go_player);
  return reinterpret_cast<PlayerID>(p);
}

void Player_AppendBuffer(PlayerID audio_player, uint8_t* data, int length) {
  Player* p = reinterpret_cast<Player*>(audio_player);
  p->AppendBuffer(data, length);
}

const char* Player_Play(PlayerID audio_player) {
  Player* p = reinterpret_cast<Player*>(audio_player);
  return p->Play();
}

const char* Player_Pause(PlayerID audio_player) {
  Player* p = reinterpret_cast<Player*>(audio_player);
  return p->Pause();
}

void Player_SetVolume(PlayerID audio_player, double volume) {
  Player* p = reinterpret_cast<Player*>(audio_player);
  p->SetVolume(volume);
}

const char* Player_Close(PlayerID audio_player) {
  Player* p = reinterpret_cast<Player*>(audio_player);
  const char* msg = p->CloseAndRemove();
  delete p;
  return msg;
}

int Player_UnplayedBufferSize(PlayerID audio_player) {
  Player* p = reinterpret_cast<Player*>(audio_player);
  return p->GetUnplayedBufferSize();
}

}  // extern "C"
