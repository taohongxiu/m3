// Copyright (c) 2018 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package downsample

// Downsampler is a downsampler.
type Downsampler interface {
	NewMetricsAppender() MetricsAppender
}

// MetricsAppender is a metrics appender that can build a samples
// appender, only valid to use with a single caller at a time.
type MetricsAppender interface {
	AddTag(name, value []byte)
	SamplesAppender() (SamplesAppender, error)
	Reset()
	Finalize()
}

// SamplesAppender is a downsampling samples appender,
// that can only be called by a single caller at a time.
type SamplesAppender interface {
	AppendCounterSample(value int64) error
	AppendGaugeSample(value float64) error
}

type downsampler struct {
	opts DownsamplerOptions
	agg  agg
}

func (d *downsampler) NewMetricsAppender() MetricsAppender {
	return newMetricsAppender(metricsAppenderOptions{
		agg:                    d.agg.aggregator,
		defaultStagedMetadatas: d.agg.defaultStagedMetadatas,
		clockOpts:              d.agg.clockOpts,
		tagEncoder:             d.agg.pools.tagEncoderPool.Get(),
		matcher:                d.agg.matcher,
		metricTagsIteratorPool: d.agg.pools.metricTagsIteratorPool,
	})
}

func newMetricsAppender(opts metricsAppenderOptions) *metricsAppender {
	return &metricsAppender{
		metricsAppenderOptions: opts,
		tags:                   newTags(),
		multiSamplesAppender:   newMultiSamplesAppender(),
	}
}
