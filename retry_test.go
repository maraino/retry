package retry

import (
	"errors"
	"sync"
	"testing"
	"time"
)

var errTest = errors.New("test error")

func failTimes(i int) func() error {
	return func() error {
		for i > 0 {
			i--
			return errTest
		}
		return nil
	}
}

func panicTimes(i int) func() error {
	return func() error {
		for i > 0 {
			i--
			panic(errTest)
		}
		return nil
	}
}

func TestRetries(t *testing.T) {
	e := NewExecutor().WithRetries(0).WithDelay(0)
	if err := e.Execute(failTimes(0)); err != nil {
		t.Error("error found and not expected")
	}
	if err := e.Execute(failTimes(1)); err != ErrMaxRetries {
		t.Error("ErrMaxRetries expected and not found")
	}

	e.WithRetries(1)
	if err := e.Execute(failTimes(0)); err != nil {
		t.Error("error found and not expected")
	}
	if err := e.Execute(failTimes(1)); err != nil {
		t.Error("error expected and not found")
	}
	if err := e.Execute(failTimes(2)); err != ErrMaxRetries {
		t.Error("ErrMaxRetries expected and not found")
	}

	e.WithRetries(2)
	if err := e.Execute(failTimes(0)); err != nil {
		t.Error("error found and not expected")
	}
	if err := e.Execute(failTimes(1)); err != nil {
		t.Error("error expected and not found")
	}
	if err := e.Execute(failTimes(2)); err != nil {
		t.Error("error expected and not found")
	}
	if err := e.Execute(failTimes(3)); err != ErrMaxRetries {
		t.Error("ErrMaxRetries expected and not found")
	}
}

func TestWithError(t *testing.T) {
	sameErr := errors.New("test error")
	diffErr := errors.New("different error")
	var testCases = []struct {
		times   int
		delay   time.Duration
		withErr error
		expErr  error
	}{
		{0, 0, errTest, nil},
		{1, 100 * time.Millisecond, errTest, nil},
		{2, 100 * time.Millisecond, errTest, ErrMaxRetries},
		{2, 100 * time.Millisecond, sameErr, ErrMaxRetries},
		{2, 0, diffErr, errTest},
	}
	for _, c := range testCases {
		e := NewExecutor().WithDelay(100 * time.Millisecond).WithError(c.withErr)
		if err := e.Execute(failTimes(c.times)); err != c.expErr {
			t.Error("error found and not expected")
		}
		if e.Context.LastDelay != c.delay {
			t.Error("invalid Context.LastDelay")
		}
	}
}

func TestWithErrorComparator(t *testing.T) {
	var testCases = []struct {
		times       int
		delay       time.Duration
		withErrComp func(error) bool
		expErr      error
	}{
		{0, 0, nil, nil},
		{1, 100 * time.Millisecond, nil, nil},
		{1, 100 * time.Millisecond, func(error) bool { return true }, nil},
		{1, 0, func(error) bool { return false }, errTest},
		{2, 100 * time.Millisecond, nil, ErrMaxRetries},
		{2, 100 * time.Millisecond, func(error) bool { return true }, ErrMaxRetries},
		{2, 0, func(error) bool { return false }, errTest},
	}
	for _, c := range testCases {
		e := NewExecutor().WithDelay(100 * time.Millisecond).WithErrorComparator(c.withErrComp)
		if err := e.Execute(failTimes(c.times)); err != c.expErr {
			t.Error("error found and not expected")
		}
		if e.Context.LastDelay != c.delay {
			t.Error("invalid Context.LastDelay")
		}
	}
}

func TestWithErrorChan(t *testing.T) {
	var wg sync.WaitGroup
	var testCases = []struct {
		times  int
		expErr error
	}{
		{1, nil},
		{5, nil},
		{6, ErrMaxRetries},
	}
	for _, c := range testCases {
		e := NewExecutor().WithRetries(5).WithDelay(100 * time.Millisecond).WithErrorChannel()

		// Read errors
		wg.Add(1)
		go func(times int) {
			var i int
			for range e.ErrorChannel {
				i++
			}
			if i != times {
				t.Error("invalid number of errors")
			}
			wg.Done()
		}(c.times)

		// Execute
		if err := e.Execute(failTimes(c.times)); err != c.expErr {
			t.Error("error found and not expected")
		}
	}
	wg.Wait()
}

func TestWithPanic(t *testing.T) {
	e := NewExecutor().
		WithDelay(100 * time.Millisecond).
		WithPanic()
	if err := e.Execute(panicTimes(1)); err != nil {
		t.Error("error found and not expected")
	}
	if e.Context.LastDelay != 100*time.Millisecond {
		t.Error("invalid Context.LastDelay", e.Context.LastDelay)
	}
}

func TestWithBackoff(t *testing.T) {

}

func TestWithDelay(t *testing.T) {
	var testCases = []struct {
		delay    time.Duration
		expected time.Duration
	}{{0, 0}, {100, 100}, {-100, 0}}
	for _, d := range testCases {
		e := NewExecutor().WithDelay(d.delay)
		if err := e.Execute(failTimes(1)); err != nil {
			t.Error("error found and not expected")
		}
		if e.Context.LastDelay != d.expected {
			t.Error("invalid Context.LastDelay")
		}
	}
}

func TestWithFirstRetryNoDelay(t *testing.T) {
	e := NewExecutor().WithFirstRetryNoDelay()
	now := time.Now()
	if err := e.Execute(failTimes(1)); err != nil {
		t.Error("error found and not expected")
	}
	if time.Now().Sub(now) >= 1*time.Second {
		t.Error("first retry had a delay")
	}
	if e.Context.LastDelay != 0 {
		t.Error("invalid Context.LastDelay")
	}

	e = NewExecutor().WithRetries(2).WithFirstRetryNoDelay()
	if err := e.Execute(failTimes(2)); err != nil {
		t.Error("error found and not expected")
	}
	if e.Context.LastDelay != 1*time.Second {
		t.Error("invalid Context.LastDelay", e.Context.LastDelay)
	}
}

func TestWithMinDelay(t *testing.T) {
	e := NewExecutor().WithMinDelay(100 * time.Millisecond)
	now := time.Now()
	if err := e.Execute(failTimes(1)); err != nil {
		t.Error("error found and not expected")
	}
	if time.Now().Sub(now) <= 1*time.Second {
		t.Error("min delay did not work")
	}
	if e.Context.LastDelay != 1*time.Second {
		t.Error("invalid Context.LastDelay")
	}

	e = NewExecutor().WithMinDelay(2 * time.Second)
	now = time.Now()
	if err := e.Execute(failTimes(1)); err != nil {
		t.Error("error found and not expected")
	}
	if time.Now().Sub(now) < 2*time.Second {
		t.Error("min delay did not work")
	}
	if e.Context.LastDelay != 2*time.Second {
		t.Error("invalid Context.LastDelay")
	}
}

func TestWithMaxDelay(t *testing.T) {
	e := NewExecutor().WithMaxDelay(100 * time.Millisecond)
	now := time.Now()
	if err := e.Execute(failTimes(1)); err != nil {
		t.Error("error found and not expected")
	}
	if time.Now().Sub(now) >= 1*time.Second {
		t.Error("max delay did not work")
	}
	if e.Context.LastDelay != 100*time.Millisecond {
		t.Error("invalid Context.LastDelay")
	}

	e = NewExecutor().WithMaxDelay(2 * time.Second)
	now = time.Now()
	if err := e.Execute(failTimes(1)); err != nil {
		t.Error("error found and not expected")
	}
	if time.Now().Sub(now) >= 2*time.Second {
		t.Error("max delay did not work")
	}
	if e.Context.LastDelay != 1*time.Second {
		t.Error("invalid Context.LastDelay")
	}
}

func TestWithFixedJitter(t *testing.T) {
	e := NewExecutor().
		WithDelay(100 * time.Millisecond).
		WithFixedJitter(1 * time.Millisecond)
	if err := e.Execute(failTimes(1)); err != nil {
		t.Error("error found and not expected")
	}
	if e.Context.LastDelay != 101*time.Millisecond {
		t.Error("invalid Context.LastDelay")
	}
}

func TestWithUniformJitter(t *testing.T) {
	// Force a seed so the random generator does not return 0ms
	// causing the test to fail.
	// Seed(1) causes a negative jitter.
	// Seed(2) causes a positive jitter.
	for _, seed := range []int64{1, 2} {
		Seed(seed)
		e := NewExecutor().
			WithDelay(100 * time.Millisecond).
			WithUniformJitter(100 * time.Millisecond)
		if err := e.Execute(failTimes(1)); err != nil {
			t.Error("error found and not expected")
		}
		if e.Context.LastDelay == 100*time.Millisecond {
			t.Error("invalid Context.LastDelay")
		}
		// Make sure it's uniform
		lastDelay := e.Context.LastDelay
		if err := e.Execute(failTimes(1)); err != nil {
			t.Error("error found and not expected")
		}
		if e.Context.LastDelay != lastDelay {
			t.Error("jitter is not uniform")
		}
	}
}

func TestWithProportionalJitter(t *testing.T) {
	var testCases = []struct {
		multiplier float64
		minDelay   time.Duration
		maxDelay   time.Duration
	}{
		{0, 100 * time.Millisecond, 100 * time.Millisecond},  // +/- 0%
		{0.1, 90 * time.Millisecond, 110 * time.Millisecond}, // +/- 10%
		{0.5, 50 * time.Millisecond, 150 * time.Millisecond}, // +/- 50%
		{1, 0 * time.Millisecond, 200 * time.Millisecond},    // +/- 100%
		{2, 0 * time.Millisecond, 300 * time.Millisecond},    // +/- 200%
	}
	for _, c := range testCases {
		e := NewExecutor().
			WithDelay(100 * time.Millisecond).
			WithProportionalJitter(c.multiplier)
		if err := e.Execute(failTimes(1)); err != nil {
			t.Error("error found and not expected")
		}
		if e.Context.LastDelay < c.minDelay || e.Context.LastDelay > c.maxDelay {
			t.Error("invalid Context.LastDelay")
		}

		// With negative multiplier should have the same behavior
		e = NewExecutor().
			WithDelay(100 * time.Millisecond).
			WithProportionalJitter(-c.multiplier)
		if err := e.Execute(failTimes(1)); err != nil {
			t.Error("error found and not expected")
		}
		if e.Context.LastDelay < c.minDelay || e.Context.LastDelay > c.maxDelay {
			t.Error("invalid Context.LastDelay")
		}
	}
}

func TestWithRandomJitter(t *testing.T) {
	// Force a seed so the random generator does not return 0ms
	// causing the test to fail.
	// Seed(1) causes a negative jitter.
	// Seed(2) causes a positive jitter.
	for _, seed := range []int64{1, 2} {
		Seed(seed)
		e := NewExecutor().
			WithDelay(100 * time.Millisecond).
			WithRandomJitter(100 * time.Millisecond)
		if err := e.Execute(failTimes(1)); err != nil {
			t.Error("error found and not expected")
		}
		if e.Context.LastDelay == 100*time.Millisecond {
			t.Error("invalid Context.LastDelay")
		}
		// Make sure it's not uniform
		lastDelay := e.Context.LastDelay
		if err := e.Execute(failTimes(1)); err != nil {
			t.Error("error found and not expected")
		}
		if e.Context.LastDelay == lastDelay {
			t.Error("jitter is not random")
		}
	}
}
