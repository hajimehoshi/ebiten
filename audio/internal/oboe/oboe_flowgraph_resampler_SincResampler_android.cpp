/*
 * Copyright 2019 The Android Open Source Project
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

#include <cassert>
#include <math.h>
#include "oboe_flowgraph_resampler_SincResampler_android.h"

using namespace resampler;

SincResampler::SincResampler(const MultiChannelResampler::Builder &builder)
        : MultiChannelResampler(builder)
        , mSingleFrame2(builder.getChannelCount()) {
    assert((getNumTaps() % 4) == 0); // Required for loop unrolling.
    mNumRows = kMaxCoefficients / getNumTaps(); // no guard row needed
//    printf("SincResampler: numRows = %d\n", mNumRows);
    mPhaseScaler = (double) mNumRows / mDenominator;
    double phaseIncrement = 1.0 / mNumRows;
    generateCoefficients(builder.getInputRate(),
                         builder.getOutputRate(),
                         mNumRows,
                         phaseIncrement,
                         builder.getNormalizedCutoff());
}

void SincResampler::readFrame(float *frame) {
    // Clear accumulator for mixing.
    std::fill(mSingleFrame.begin(), mSingleFrame.end(), 0.0);
    std::fill(mSingleFrame2.begin(), mSingleFrame2.end(), 0.0);

    // Determine indices into coefficients table.
    double tablePhase = getIntegerPhase() * mPhaseScaler;
    int index1 = static_cast<int>(floor(tablePhase));
    if (index1 >= mNumRows) { // no guard row needed because we wrap the indices
        tablePhase -= mNumRows;
        index1 -= mNumRows;
    }

    int index2 = index1 + 1;
    if (index2 >= mNumRows) { // no guard row needed because we wrap the indices
        index2 -= mNumRows;
    }

    float *coefficients1 = &mCoefficients[index1 * getNumTaps()];
    float *coefficients2 = &mCoefficients[index2 * getNumTaps()];

    float *xFrame = &mX[mCursor * getChannelCount()];
    for (int i = 0; i < mNumTaps; i++) {
        float coefficient1 = *coefficients1++;
        float coefficient2 = *coefficients2++;
        for (int channel = 0; channel < getChannelCount(); channel++) {
            float sample = *xFrame++;
            mSingleFrame[channel] +=  sample * coefficient1;
            mSingleFrame2[channel] += sample * coefficient2;
        }
    }

    // Interpolate and copy to output.
    float fraction = tablePhase - index1;
    for (int channel = 0; channel < getChannelCount(); channel++) {
        float low = mSingleFrame[channel];
        float high = mSingleFrame2[channel];
        frame[channel] = low + (fraction * (high - low));
    }
}
