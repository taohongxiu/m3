// // Copyright (c) 2018 Uber Technologies, Inc.
// //
// // Permission is hereby granted, free of charge, to any person obtaining a copy
// // of this software and associated documentation files (the "Software"), to deal
// // in the Software without restriction, including without limitation the rights
// // to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// // copies of the Software, and to permit persons to whom the Software is
// // furnished to do so, subject to the following conditions:
// //
// // The above copyright notice and this permission notice shall be included in
// // all copies or substantial portions of the Software.
// //
// // THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// // IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// // FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// // AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// // LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// // OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// // THE SOFTWARE.

package block

import (
	"math"
	"time"

	"github.com/m3db/m3db/src/coordinator/block"
)

// Meta returns the metadata for the block
func (m MultiSeriesBlock) Meta() block.Metadata {
	return m.Metadata
}

// StepIter creates a new step iterator for a given MultiSeriesBlock
func (m MultiSeriesBlock) StepIter() block.StepIter {
	return &multiSeriesBlockStepIter{
		seriesIters: newConsolidatedSeriesBlockIters(m.Blocks),
		index:       -1,
	}
}

// SeriesIter creates a new series iterator for a given MultiSeriesBlock
func (m MultiSeriesBlock) SeriesIter() block.SeriesIter {
	// todo(braskin): implement SeriesIter()
	return nil
}

// SeriesMeta returns metadata for the individual timeseries
func (m MultiSeriesBlock) SeriesMeta() []block.SeriesMeta {
	metas := make([]block.SeriesMeta, len(m.Blocks))
	for i, s := range m.Blocks {
		metas[i].Tags = s.Metadata.Tags
	}
	return metas
}

// StepCount returns the total steps/columns
func (m MultiSeriesBlock) StepCount() int {
	return m.Blocks[0].Metadata.Bounds.Steps()
}

// Close frees up resources
func (m MultiSeriesBlock) Close() error {
	// todo(braskin): Actually free up resources
	return nil
}

// SeriesCount returns the number of time series in a MultiSeriesBlock
func (m MultiSeriesBlock) SeriesCount() int {
	return len(m.Blocks)
}

func newConsolidatedSeriesBlockIters(blocks ConsolidatedSeriesBlocks) consolidatedSeriesBlockIters {
	consolidatedSeriesBlockIters := make([]*consolidatedSeriesBlockIter, len(blocks))
	for i, seriesBlock := range blocks {
		consolidatedNSBlockIters := make([]*consolidatedNSBlockIter, len(blocks[0].ConsolidatedNSBlocks))
		for j, nsBlock := range seriesBlock.ConsolidatedNSBlocks {
			nsBlockIter := newConsolidatedNSBlockIter(nsBlock)
			consolidatedNSBlockIters[j] = nsBlockIter
		}
		consolidatedSeriesBlockIters[i] = &consolidatedSeriesBlockIter{
			consolidatedNSBlockIters: consolidatedNSBlockIters,
		}
	}

	return consolidatedSeriesBlockIters
}

func newConsolidatedNSBlockIter(nsBlock ConsolidatedNSBlock) *consolidatedNSBlockIter {
	return &consolidatedNSBlockIter{
		consolidatedNSBlockSeriesIters: nsBlock.SeriesIterators.Iters(),
		bounds:    nsBlock.Bounds,
		indexTime: nsBlock.Bounds.Start.Add(-1 * nsBlock.Bounds.StepSize),
	}
}

func (m *multiSeriesBlockStepIter) Next() bool {
	if len(m.seriesIters) == 0 {
		return false
	}

	for _, s := range m.seriesIters {
		if !s.Next() {
			return false
		}
	}

	m.index++
	return true
}

func (m *multiSeriesBlockStepIter) Current() block.Step {
	values := make([]float64, len(m.seriesIters))
	for i, s := range m.seriesIters {
		values[i] = s.Current()
	}

	bounds := m.meta.Bounds
	t := bounds.Start.Add(time.Duration(m.index) * bounds.StepSize)
	return block.NewColStep(t, values)
}

func (m *multiSeriesBlockStepIter) Steps() int {
	return len(m.seriesIters)
}

// TODO: Actually free up resources
func (m *multiSeriesBlockStepIter) Close() {}

func (c *consolidatedSeriesBlockIter) Current() float64 {
	var values []float64
	for _, iter := range c.consolidatedNSBlockIters {
		dp := iter.Current()
		values = append(values, dp)
	}

	// todo(braskin): until we have consolidation
	return values[0]
}

func (c *consolidatedSeriesBlockIter) Next() bool {
	if len(c.consolidatedNSBlockIters) == 0 {
		return false
	}

	for _, nsBlock := range c.consolidatedNSBlockIters {
		if !nsBlock.Next() {
			return false
		}
	}

	return true
}

func (c *consolidatedNSBlockIter) Next() bool {
	c.indexTime = c.indexTime.Add(c.bounds.StepSize)
	lastDP := c.lastDP

	if !c.indexTime.Before(c.bounds.End) {
		return false
	}

	for c.indexTime.After(lastDP.Timestamp) && c.next() {
		lastDP, _, _ = c.consolidatedNSBlockSeriesIters[c.seriesIndex].Current()
		c.lastDP = lastDP
	}

	return true
}

func (c *consolidatedNSBlockIter) next() bool {
	// todo(braskin): check bounds as well
	if len(c.consolidatedNSBlockSeriesIters) == 0 {
		return false
	}

	for c.seriesIndex < len(c.consolidatedNSBlockSeriesIters) {
		if c.consolidatedNSBlockSeriesIters[c.seriesIndex].Next() {
			return true
		}
		c.seriesIndex++
	}

	return false
}

func (c *consolidatedNSBlockIter) Current() float64 {
	if !c.indexTime.After(c.lastDP.Timestamp) && c.indexTime.Add(c.bounds.StepSize).After(c.lastDP.Timestamp) {
		return c.lastDP.Value
	}

	return math.NaN()
}