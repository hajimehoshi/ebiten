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

#ifndef OBOE_FIFOPROCESSOR_H
#define OBOE_FIFOPROCESSOR_H

#include <memory>
#include <stdint.h>

#include "oboe_oboe_Definitions_android.h"

#include "oboe_fifo_FifoControllerBase_android.h"

namespace oboe {

class FifoBuffer {
public:
    FifoBuffer(uint32_t bytesPerFrame, uint32_t capacityInFrames);

    FifoBuffer(uint32_t   bytesPerFrame,
               uint32_t   capacityInFrames,
               std::atomic<uint64_t>   *readCounterAddress,
               std::atomic<uint64_t>   *writeCounterAddress,
               uint8_t   *dataStorageAddress);

    ~FifoBuffer();

    int32_t convertFramesToBytes(int32_t frames);

    /**
     * Read framesToRead or, if not enough, then read as many as are available.
     * @param destination
     * @param framesToRead number of frames requested
     * @return number of frames actually read
     */
    int32_t read(void *destination, int32_t framesToRead);

    int32_t write(const void *source, int32_t framesToWrite);

    uint32_t getBufferCapacityInFrames() const;

    /**
     * Calls read(). If all of the frames cannot be read then the remainder of the buffer
     * is set to zero.
     *
     * @param destination
     * @param framesToRead number of frames requested
     * @return number of frames actually read
     */
    int32_t readNow(void *destination, int32_t numFrames);

    uint32_t getFullFramesAvailable() {
        return mFifo->getFullFramesAvailable();
    }

    uint32_t getBytesPerFrame() const {
        return mBytesPerFrame;
    }

    uint64_t getReadCounter() const {
        return mFifo->getReadCounter();
    }

    void setReadCounter(uint64_t n) {
        mFifo->setReadCounter(n);
    }

    uint64_t getWriteCounter() {
        return mFifo->getWriteCounter();
    }
    void setWriteCounter(uint64_t n) {
        mFifo->setWriteCounter(n);
    }

private:
    uint32_t mBytesPerFrame;
    uint8_t* mStorage;
    bool     mStorageOwned; // did this object allocate the storage?
    std::unique_ptr<FifoControllerBase> mFifo;
    uint64_t mFramesReadCount;
    uint64_t mFramesUnderrunCount;
};

} // namespace oboe

#endif //OBOE_FIFOPROCESSOR_H
