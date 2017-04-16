// Copyright (c) 2017 Uber Technologies, Inc.
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

package reporter

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/m3db/m3collector/metric"
	"github.com/m3db/m3metrics/policy"
	"github.com/m3db/m3metrics/rules"
	"github.com/m3db/m3x/time"

	"github.com/stretchr/testify/require"
)

var (
	testMappingPolicies = []policy.Policy{
		policy.NewPolicy(20*time.Second, xtime.Second, 6*time.Hour),
		policy.NewPolicy(time.Minute, xtime.Minute, 2*24*time.Hour),
		policy.NewPolicy(10*time.Minute, xtime.Minute, 25*24*time.Hour),
	}
	testRollupPolicies = []policy.Policy{policy.NewPolicy(10*time.Minute, xtime.Minute, 25*24*time.Hour)}
	testMatchResult    = rules.NewMatchResult(
		2,
		0,
		math.MaxInt64,
		testMappingPolicies,
		[]rules.RollupResult{
			{
				ID:       []byte("foo"),
				Policies: []policy.Policy{},
			},
			{
				ID:       []byte("bar"),
				Policies: testRollupPolicies,
			},
		},
	)
	errTestWriteCounterWithPolicies    = errors.New("error writing counter with policies")
	errTestWriteBatchTimerWithPolicies = errors.New("error writing batch timer with policies")
	errTestWriteGaugeWithPolicies      = errors.New("error writing gauge with policies")
)

func TestReporterReportCounterPartialError(t *testing.T) {
	var (
		ids      []string
		vals     []int64
		policies []policy.VersionedPolicies
	)
	reporter := NewReporter(
		&mockMatcher{
			matchFn: func(metric.ID) rules.MatchResult { return testMatchResult },
		},
		&mockServer{
			writeCounterWithPoliciesFn: func(id []byte, val int64, vp policy.VersionedPolicies) error {
				ids = append(ids, string(id))
				vals = append(vals, val)
				policies = append(policies, vp)
				return errTestWriteCounterWithPolicies
			},
		},
	)
	require.Error(t, reporter.ReportCounter(mockID("counter"), 1234))
	require.Equal(t, []string{"counter", "foo", "bar"}, ids)
	require.Equal(t, []int64{1234, 1234, 1234}, vals)
	require.Equal(t, []policy.VersionedPolicies{
		policy.CustomVersionedPolicies(2, time.Unix(0, 0), testMappingPolicies),
		policy.DefaultVersionedPolicies(2, time.Unix(0, 0)),
		policy.CustomVersionedPolicies(2, time.Unix(0, 0), testRollupPolicies),
	}, policies)
}

func TestReporterReportBatchTimerPartialError(t *testing.T) {
	var (
		ids      []string
		vals     [][]float64
		policies []policy.VersionedPolicies
	)
	reporter := NewReporter(
		&mockMatcher{
			matchFn: func(metric.ID) rules.MatchResult { return testMatchResult },
		},
		&mockServer{
			writeBatchTimerWithPoliciesFn: func(id []byte, val []float64, vp policy.VersionedPolicies) error {
				ids = append(ids, string(id))
				vals = append(vals, val)
				policies = append(policies, vp)
				return errTestWriteBatchTimerWithPolicies
			},
		},
	)
	require.Error(t, reporter.ReportBatchTimer(mockID("batchTimer"), []float64{1.3, 2.4}))
	require.Equal(t, []string{"batchTimer", "foo", "bar"}, ids)
	require.Equal(t, [][]float64{{1.3, 2.4}, {1.3, 2.4}, {1.3, 2.4}}, vals)
	require.Equal(t, []policy.VersionedPolicies{
		policy.CustomVersionedPolicies(2, time.Unix(0, 0), testMappingPolicies),
		policy.DefaultVersionedPolicies(2, time.Unix(0, 0)),
		policy.CustomVersionedPolicies(2, time.Unix(0, 0), testRollupPolicies),
	}, policies)
}

func TestReporterReportGaugePartialError(t *testing.T) {
	var (
		ids      []string
		vals     []float64
		policies []policy.VersionedPolicies
	)
	reporter := NewReporter(
		&mockMatcher{
			matchFn: func(metric.ID) rules.MatchResult { return testMatchResult },
		},
		&mockServer{
			writeGaugeWithPoliciesFn: func(id []byte, val float64, vp policy.VersionedPolicies) error {
				ids = append(ids, string(id))
				vals = append(vals, val)
				policies = append(policies, vp)
				return errTestWriteGaugeWithPolicies
			},
		},
	)
	require.Error(t, reporter.ReportGauge(mockID("gauge"), 1.8))
	require.Equal(t, []string{"gauge", "foo", "bar"}, ids)
	require.Equal(t, []float64{1.8, 1.8, 1.8}, vals)
	require.Equal(t, []policy.VersionedPolicies{
		policy.CustomVersionedPolicies(2, time.Unix(0, 0), testMappingPolicies),
		policy.DefaultVersionedPolicies(2, time.Unix(0, 0)),
		policy.CustomVersionedPolicies(2, time.Unix(0, 0), testRollupPolicies),
	}, policies)
}

func TestReporterFlush(t *testing.T) {
	var numFlushes int
	reporter := NewReporter(&mockMatcher{}, &mockServer{
		flushFn: func() error { numFlushes++; return nil },
	})
	require.NoError(t, reporter.Flush())
	require.Equal(t, 1, numFlushes)
}

func TestReporterClose(t *testing.T) {
	reporter := NewReporter(&mockMatcher{}, &mockServer{})
	require.Error(t, reporter.Close())
}

type mockID []byte

func (mid mockID) Bytes() []byte                          { return mid }
func (mid mockID) TagValue(tagName []byte) ([]byte, bool) { return nil, false }

type matchFn func(id metric.ID) rules.MatchResult

type mockMatcher struct {
	matchFn matchFn
}

func (mm *mockMatcher) Match(id metric.ID) rules.MatchResult { return mm.matchFn(id) }
func (mm *mockMatcher) Close() error                         { return errors.New("error closing matcher") }

type writeCounterWithPoliciesFn func(id []byte, val int64, vp policy.VersionedPolicies) error
type writeBatchTimerWithPoliciesFn func(id []byte, val []float64, vp policy.VersionedPolicies) error
type writeGaugeWithPoliciesFn func(id []byte, val float64, vp policy.VersionedPolicies) error
type flushFn func() error

type mockServer struct {
	writeCounterWithPoliciesFn    writeCounterWithPoliciesFn
	writeBatchTimerWithPoliciesFn writeBatchTimerWithPoliciesFn
	writeGaugeWithPoliciesFn      writeGaugeWithPoliciesFn
	flushFn                       flushFn
}

func (ms *mockServer) Open() error  { return nil }
func (ms *mockServer) Flush() error { return ms.flushFn() }
func (ms *mockServer) Close() error { return errors.New("error closing server") }

func (ms *mockServer) WriteCounterWithPolicies(id []byte, val int64, vp policy.VersionedPolicies) error {
	return ms.writeCounterWithPoliciesFn(id, val, vp)
}

func (ms *mockServer) WriteBatchTimerWithPolicies(id []byte, val []float64, vp policy.VersionedPolicies) error {
	return ms.writeBatchTimerWithPoliciesFn(id, val, vp)
}

func (ms *mockServer) WriteGaugeWithPolicies(id []byte, val float64, vp policy.VersionedPolicies) error {
	return ms.writeGaugeWithPoliciesFn(id, val, vp)
}
