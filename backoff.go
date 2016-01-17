package retry

import (
	"math"
	"time"
)

// Backoff is the interface that a backoff strategy needs to implement.
type Backoff interface {
	GetDelay(Context) time.Duration
}

type backoffImpl struct {
	getDelay func(Context) time.Duration
}

// GetDelay is the implementation of the Backoff interface.
func (b *backoffImpl) GetDelay(ctx Context) time.Duration {
	return b.getDelay(ctx)
}

// FixedDelayBackoff returns a backoff strategy that waits for a fixed amount of time.
func FixedDelayBackoff(delay time.Duration) Backoff {
	return &backoffImpl{getDelayFunction(delay)}
}

// DefaultFixedDelayBackoff returns a backoff strategy that waits for 1 second.
func DefaultFixedDelayBackoff() Backoff {
	return FixedDelayBackoff(1 * time.Second)
}

// UniformRandomBackoff returns a backoff strategy that waits for a random amount
// of time between 0 and maxDelay.
func UniformRandomBackoff(maxDelay time.Duration) Backoff {
	randomDelay := getRandomDuration(maxDelay)
	return &backoffImpl{getDelayFunction(randomDelay)}
}

// DefaultUniformRandomBackoff returns a backoff strategy that waits for a random
// amount of time between 0 and 1 second.
func DefaultUniformRandomBackoff() Backoff {
	return UniformRandomBackoff(1 * time.Second)
}

// ExponentialDelayBackoff returns a backoff strategy that increases the delay
// on every retry.
func ExponentialDelayBackoff(initialDelay time.Duration, multiplier float64) Backoff {
	return &backoffImpl{func(ctx Context) time.Duration {
		return time.Duration(float64(initialDelay) * math.Pow(multiplier, float64(ctx.RetryCount-1)))
	}}
}

// BoundedDelayBackoff returns a Backoff where the GetDelay method will return a
// random number between minDelay and maxDelay.
func BoundedDelayBackoff(minDelay, maxDelay time.Duration) Backoff {
	bound := maxDelay - minDelay
	return &backoffImpl{func(ctx Context) time.Duration {
		return minDelay + getRandomDuration(bound)
	}}
}

// boundedMaxDelayBackoff returns a Backoff where the delay will be the maximum
// between maxDelay and the delay of another Backoff.
func boundedMaxDelayBackoff(maxDelay time.Duration, target Backoff) Backoff {
	return &backoffImpl{func(ctx Context) time.Duration {
		return time.Duration(math.Min(float64(target.GetDelay(ctx)), float64(maxDelay)))
	}}
}

// boundedMinDelayBackoff returns a Backoff where the delay will be the minimum
// between minDelay and the delay of another Backoff.
func boundedMinDelayBackoff(minDelay time.Duration, target Backoff) Backoff {
	return &backoffImpl{func(ctx Context) time.Duration {
		return time.Duration(math.Max(float64(target.GetDelay(ctx)), float64(minDelay)))
	}}
}

// firstRetryNoDelayBackoff modifies the target Backoff to not delay in the first
// retry.
func firstRetryNoDelayBackoff(target Backoff) Backoff {
	return &backoffImpl{func(ctx Context) time.Duration {
		if ctx.RetryCount == 1 {
			return 0
		} else {
			return target.GetDelay(ctx.previous())
		}
	}}
}

// fixedJitterBackoff modifies the target Backoff with a fixed delay.
func fixedJitterBackoff(delay time.Duration, target Backoff) Backoff {
	return &backoffImpl{func(ctx Context) time.Duration {
		return delay + target.GetDelay(ctx)
	}}
}

// randomJitterBackoff modifies the target Backoff with a random delay between
// -maxDelay and +maxDelay. Every call to GetDelay will use a different random
// delay.
func randomJitterBackoff(maxDelay time.Duration, target Backoff) Backoff {
	return &backoffImpl{func(ctx Context) time.Duration {
		randomDelay := getRandomDurationMinMax(-maxDelay, maxDelay)
		return randomDelay + target.GetDelay(ctx)
	}}
}

// uniformJitterBackoff modifies the target Backoff with a random delay between
// -maxDelay and +maxDelay.
func uniformJitterBackoff(maxDelay time.Duration, target Backoff) Backoff {
	randomDelay := getRandomDurationMinMax(-maxDelay, maxDelay)
	return &backoffImpl{func(ctx Context) time.Duration {
		return randomDelay + target.GetDelay(ctx)
	}}
}

// proportinalJitterBackoff modifies the target Backoff with a random percentage of the
// target delay.
func proportinalJitterBackoff(multiplier float64, target Backoff) Backoff {
	return &backoffImpl{func(ctx Context) time.Duration {
		delay := target.GetDelay(ctx)
		rangeDelay := time.Duration(float64(delay) * math.Abs(multiplier))
		randomDelay := getRandomDurationMinMax(-rangeDelay, rangeDelay)
		return delay + randomDelay
	}}
}

func getDelayFunction(d time.Duration) func(Context) time.Duration {
	return func(Context) time.Duration {
		return d
	}
}

func getSafeRandom(max int64) int64 {
	if max <= 0 {
		return 0
	}
	return rnd.Int63n(max)
}

func getRandomDuration(maxDelay time.Duration) time.Duration {
	return time.Duration(getSafeRandom(int64(maxDelay + 1)))
}

func getRandomDurationMinMax(minDelay, maxDelay time.Duration) time.Duration {
	min := int64(minDelay)
	max := int64(maxDelay)
	return time.Duration(getSafeRandom(max-min+1) + min)
}
