/*
 * Copyright 2018 The Android Open Source Project
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

#include "oboe_common_OboeDebug_android.h"
#include <algorithm>
#include <unistd.h>
#include "oboe_flowgraph_FlowGraphNode_android.h"
#include "oboe_flowgraph_SourceFloat_android.h"

using namespace FLOWGRAPH_OUTER_NAMESPACE::flowgraph;

SourceFloat::SourceFloat(int32_t channelCount)
        : FlowGraphSourceBuffered(channelCount) {
}

int32_t SourceFloat::onProcess(int32_t numFrames) {
    float *outputBuffer = output.getBuffer();
    int32_t channelCount = output.getSamplesPerFrame();

    int32_t framesLeft = mSizeInFrames - mFrameIndex;
    int32_t framesToProcess = std::min(numFrames, framesLeft);
    int32_t numSamples = framesToProcess * channelCount;

    const float *floatBase = (float *) mData;
    const float *floatData = &floatBase[mFrameIndex * channelCount];
    memcpy(outputBuffer, floatData, numSamples * sizeof(float));
    mFrameIndex += framesToProcess;
    return framesToProcess;
}

