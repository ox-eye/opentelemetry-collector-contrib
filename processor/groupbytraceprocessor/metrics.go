// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package groupbytraceprocessor // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/groupbytraceprocessor"

import (
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/collector/obsreport"
)

var (
	mNumTracesConf       = stats.Int64("processor_groupbytrace_conf_num_traces", "Maximum number of traces to hold in the internal storage", stats.UnitDimensionless)
	mNumEventsInQueue    = stats.Int64("processor_groupbytrace_num_events_in_queue", "Number of events currently in the queue", stats.UnitDimensionless)
	mNumTracesInMemory   = stats.Int64("processor_groupbytrace_num_traces_in_memory", "Number of traces currently in the in-memory storage", stats.UnitDimensionless)
	mTracesEvicted       = stats.Int64("processor_groupbytrace_traces_evicted", "Traces evicted from the internal buffer", stats.UnitDimensionless)
	mReleasedSpans       = stats.Int64("processor_groupbytrace_spans_released", "Spans released to the next consumer", stats.UnitDimensionless)
	mReleasedTraces      = stats.Int64("processor_groupbytrace_traces_released", "Traces released to the next consumer", stats.UnitDimensionless)
	mIncompleteReleases  = stats.Int64("processor_groupbytrace_incomplete_releases", "Releases that are suspected to have been incomplete", stats.UnitDimensionless)
	mDeDuplicatedTraces  = stats.Int64("processor_groupbytrace_deduplicator_deduplicated_traces", "Traces deduplicated by the processor", stats.UnitDimensionless)
	mFailedToHash        = stats.Int64("processor_groupbytrace_deduplicator_failed_to_hash", "Traces that failed to hash", stats.UnitDimensionless)
	mBypassedTraces      = stats.Int64("processor_groupbytrace_deduplicator_bypassed_traces", "Traces that bypassed the deduplication mechanisem", stats.UnitDimensionless)
	mNumOfDistinctTraces = stats.Int64("processor_groupbytrace_deduplicator_distinct_traces", "Number of distinct traces", stats.UnitDimensionless)
	mEventLatency        = stats.Int64("processor_groupbytrace_event_latency", "How long the queue events are taking to be processed", stats.UnitMilliseconds)
)

// MetricViews return the metrics views according to given telemetry level.
func MetricViews() []*view.View {
	return []*view.View{
		{
			Name:        obsreport.BuildProcessorCustomMetricName(string(typeStr), mNumTracesConf.Name()),
			Measure:     mNumTracesConf,
			Description: mNumTracesConf.Description(),
			Aggregation: view.LastValue(),
		},
		{
			Name:        obsreport.BuildProcessorCustomMetricName(string(typeStr), mFailedToHash.Name()),
			Measure:     mFailedToHash,
			Description: mFailedToHash.Description(),
			Aggregation: view.Sum(),
		},
		{
			Name:        obsreport.BuildProcessorCustomMetricName(string(typeStr), mDeDuplicatedTraces.Name()),
			Measure:     mDeDuplicatedTraces,
			Description: mDeDuplicatedTraces.Description(),
			Aggregation: view.Sum(),
		},
		{
			Name:        obsreport.BuildProcessorCustomMetricName(string(typeStr), mBypassedTraces.Name()),
			Measure:     mBypassedTraces,
			Description: mBypassedTraces.Description(),
			Aggregation: view.Sum(),
		},
		{
			Name:        obsreport.BuildProcessorCustomMetricName(string(typeStr), mNumOfDistinctTraces.Name()),
			Measure:     mNumOfDistinctTraces,
			Description: mNumOfDistinctTraces.Description(),
			Aggregation: view.Sum(),
		},
		{
			Name:        obsreport.BuildProcessorCustomMetricName(string(typeStr), mNumEventsInQueue.Name()),
			Measure:     mNumEventsInQueue,
			Description: mNumEventsInQueue.Description(),
			Aggregation: view.LastValue(),
		},
		{
			Name:        obsreport.BuildProcessorCustomMetricName(string(typeStr), mNumTracesInMemory.Name()),
			Measure:     mNumTracesInMemory,
			Description: mNumTracesInMemory.Description(),
			Aggregation: view.LastValue(),
		},
		{
			Name:        obsreport.BuildProcessorCustomMetricName(string(typeStr), mTracesEvicted.Name()),
			Measure:     mTracesEvicted,
			Description: mTracesEvicted.Description(),
			// sum allows us to start from 0, count will only show up if there's at least one eviction, which might take a while to happen (if ever!)
			Aggregation: view.Sum(),
		},
		{
			Name:        obsreport.BuildProcessorCustomMetricName(string(typeStr), mReleasedSpans.Name()),
			Measure:     mReleasedSpans,
			Description: mReleasedSpans.Description(),
			Aggregation: view.Sum(),
		},
		{
			Name:        obsreport.BuildProcessorCustomMetricName(string(typeStr), mReleasedTraces.Name()),
			Measure:     mReleasedTraces,
			Description: mReleasedTraces.Description(),
			Aggregation: view.Sum(),
		},
		{
			Name:        obsreport.BuildProcessorCustomMetricName(string(typeStr), mIncompleteReleases.Name()),
			Measure:     mIncompleteReleases,
			Description: mIncompleteReleases.Description(),
			Aggregation: view.Sum(),
		},
		{
			Name:        obsreport.BuildProcessorCustomMetricName(string(typeStr), mEventLatency.Name()),
			Measure:     mEventLatency,
			Description: mEventLatency.Description(),
			TagKeys: []tag.Key{
				tag.MustNewKey("event"),
			},
			Aggregation: view.Distribution(0, 5, 10, 20, 50, 100, 200, 500, 1000),
		},
	}
}
