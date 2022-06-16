package tickler

import (
	"math"
	"sync/atomic"
	"time"
)

// backoff implements the truncated exponential backoff algorithm.
type backoff struct {
	attempt uint64

	// Factor is the multiplicative factor used to calculate the delay between
	// retries.
	Factor float64

	// Minimum value of the counter
	Min time.Duration

	// Maximum value of the counter
	Max time.Duration
}

func (b *backoff) Duration() time.Duration {
	return b.Next(float64(atomic.AddUint64(&b.attempt, 1) - 1))
}

func (b *backoff) Next(attempt float64) time.Duration {
	min := b.Min
	max := b.Max

	if min <= 0 {
		min = 100 * time.Millisecond
	}

	if max <= 0 {
		max = 10 * time.Second
	}

	if min > max {
		return max
	}

	factor := b.Factor
	if factor <= 0 {
		factor = 2
	}

	minf := float64(min)
	durf := minf * math.Pow(factor, attempt)

	//ensure float64 wont overflow int64
	if durf > math.MaxInt64 {
		return max
	}
	dur := time.Duration(durf)
	//keep within bounds
	if dur < min {
		return min
	}
	if dur > max {
		return max
	}
	return dur
}
