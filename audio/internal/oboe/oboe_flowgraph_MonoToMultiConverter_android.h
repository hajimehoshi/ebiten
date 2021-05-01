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

#ifndef FLOWGRAPH_MONO_TO_MULTI_CONVERTER_H
#define FLOWGRAPH_MONO_TO_MULTI_CONVERTER_H

#include <unistd.h>
#include <sys/types.h>

#include "oboe_flowgraph_FlowGraphNode_android.h"

namespace FLOWGRAPH_OUTER_NAMESPACE {
namespace flowgraph {

/**
 * Convert a monophonic stream to a multi-channel interleaved stream
 * with the same signal on each channel.
 */
class MonoToMultiConverter : public FlowGraphNode {
public:
    explicit MonoToMultiConverter(int32_t outputChannelCount);

    virtual ~MonoToMultiConverter();

    int32_t onProcess(int32_t numFrames) override;

    const char *getName() override {
        return "MonoToMultiConverter";
    }

    FlowGraphPortFloatInput input;
    FlowGraphPortFloatOutput output;
};

} /* namespace flowgraph */
} /* namespace FLOWGRAPH_OUTER_NAMESPACE */

#endif //FLOWGRAPH_MONO_TO_MULTI_CONVERTER_H
