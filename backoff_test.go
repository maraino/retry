package retry

import (
	"testing"
	"time"
)

func TestFixedDelayBackoff(t *testing.T) {
	var testCases = []struct {
		count int
		delay time.Duration
	}{
		{0, 0 * time.Millisecond},
		{1, 100 * time.Millisecond},
		{2, 200 * time.Millisecond},
	}
	for _, c := range testCases {
		ctx := Context{RetryCount: c.count}
		backoff := FixedDelayBackoff(c.delay)
		if backoff.GetDelay(ctx) != c.delay {
			t.Error("invalid return value for GetDelay(ctx)")
		}
	}
}

func TestDefaultFixedDelayBackoff(t *testing.T) {
	var testCases = []struct {
		count int
		delay time.Duration
	}{
		{0, 1 * time.Second},
		{1, 1 * time.Second},
		{2, 1 * time.Second},
	}
	for _, c := range testCases {
		ctx := Context{RetryCount: c.count}
		backoff := DefaultFixedDelayBackoff()
		if backoff.GetDelay(ctx) != c.delay {
			t.Error("invalid return value for GetDelay(ctx)")
		}
	}
}

func TestUniformRandomBackoff(t *testing.T) {
	var testCases = []struct {
		count int
		delay time.Duration
	}{
		{0, 0 * time.Millisecond},
		{1, 100 * time.Millisecond},
		{2, 200 * time.Millisecond},
	}
	for _, c := range testCases {
		ctx := Context{RetryCount: c.count}
		backoff := UniformRandomBackoff(c.delay)
		if backoff.GetDelay(ctx) > c.delay {
			t.Error("invalid return value for GetDelay(ctx)")
		}
	}

	// Safe bounds
	for i, maxDelay := range []time.Duration{0, -1, -100} {
		ctx := Context{RetryCount: i + 1}
		backoff := UniformRandomBackoff(maxDelay)
		if backoff.GetDelay(ctx) != 0*time.Millisecond {
			t.Error("invalid return value for GetDelay(ctx)")
		}
	}
}

func TestDefaultUniformRandomBackoff(t *testing.T) {
	var testCases = []struct {
		count int
		delay time.Duration
	}{
		{0, 1 * time.Second},
		{1, 1 * time.Second},
		{2, 1 * time.Second},
	}
	for _, c := range testCases {
		ctx := Context{RetryCount: c.count}
		backoff := DefaultUniformRandomBackoff()
		if backoff.GetDelay(ctx) > c.delay {
			t.Error("invalid return value for GetDelay(ctx)")
		}
	}
}

func TestExponentialDelayBackoffBackoff(t *testing.T) {
	var testCases = []struct {
		count      int
		multiplier float64
		delay      time.Duration
		expDelay   time.Duration
	}{
		{1, 0, 0 * time.Millisecond, 0 * time.Millisecond},
		{1, 1, 100 * time.Millisecond, 100 * time.Millisecond},
		{1, 2, 100 * time.Millisecond, 100 * time.Millisecond},
		{1, 2, 200 * time.Millisecond, 200 * time.Millisecond},
		{2, 0, 0 * time.Millisecond, 0 * time.Millisecond},
		{2, 1, 100 * time.Millisecond, 100 * time.Millisecond},
		{2, 2, 100 * time.Millisecond, 200 * time.Millisecond},
		{2, 2, 200 * time.Millisecond, 400 * time.Millisecond},
		{3, 0, 0 * time.Millisecond, 0 * time.Millisecond},
		{3, 1, 100 * time.Millisecond, 100 * time.Millisecond},
		{3, 2, 100 * time.Millisecond, 400 * time.Millisecond},
		{3, 2, 200 * time.Millisecond, 800 * time.Millisecond},
	}
	for _, c := range testCases {
		ctx := Context{RetryCount: c.count}
		backoff := ExponentialDelayBackoff(c.delay, c.multiplier)
		if backoff.GetDelay(ctx) != c.expDelay {
			t.Error("invalid return value for GetDelay(ctx)")
		}
	}

	e := NewExecutor().WithRetries(5).WithBackoff(ExponentialDelayBackoff(10*time.Millisecond, 2))
	if err := e.Execute(failTimes(5)); err != nil {
		t.Error("error found and not expected")
	}
	if e.Context.RetryCount != 5 {
		t.Error("invalid value for RetryCount", e.Context.RetryCount)
	}
}

func TestBoundedDelayBackoff(t *testing.T) {
	var testCases = []struct {
		count    int
		minDelay time.Duration
		maxDelay time.Duration
	}{
		{0, 0 * time.Millisecond, 0 * time.Millisecond},
		{1, 100 * time.Millisecond, 100 * time.Millisecond},
		{2, 100 * time.Millisecond, 200 * time.Millisecond},
		{3, 200 * time.Millisecond, 400 * time.Millisecond},
	}
	for _, c := range testCases {
		ctx := Context{RetryCount: c.count}
		backoff := BoundedDelayBackoff(c.minDelay, c.maxDelay)
		if backoff.GetDelay(ctx) < c.minDelay || backoff.GetDelay(ctx) > c.maxDelay {
			t.Error("invalid return value for GetDelay(ctx)")
		}
	}

	// Safe bounds
	ctx := Context{RetryCount: 1}
	backoff := BoundedDelayBackoff(200*time.Millisecond, 100*time.Millisecond)
	if backoff.GetDelay(ctx) != 200*time.Millisecond {
		t.Error("invalid return value for GetDelay(ctx)")
	}
}
