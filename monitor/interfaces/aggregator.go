package interfaces

// Aggregator is an interface for implementing an Aggregator plugin.
// the RunningAggregator wraps this interface and guarantees that
// Add, Push, and Reset can not be called concurrently, so locking is not
// required when implementing an Aggregator plugin.
type Aggregator interface {
	// SampleConfig returns the default configuration of the Input.
	SampleConfig() string

	// Description returns a one-sentence description on the Input.
	Description() string

	// Add the metric to the aggregator.
	Add(in Metric)

	// Push pushes the current aggregates to the accumulator.
	Push(acc Accumulator)

	// Reset resets the aggregators caches and aggregates.
	Reset()
}
