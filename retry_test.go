package retry

import (
	"errors"
	"sync"
	"testing"
	"time"

	"golang.org/x/net/context"
)

var errTest = errors.New("test error")

func failTimes(i int) Operation {
	return func() error {
		for i > 0 {
			i--
			return errTest
		}
		return nil
	}
}

func panicTimes(i int) Operation {
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
		ctx := NewContext()
		e := NewExecutor().WithDelay(100 * time.Millisecond).WithError(c.withErr)
		if err := e.ExecuteContext(ctx, failTimes(c.times)); err != c.expErr {
			t.Error("error found and not expected")
		}
		if ctx.LastDelay != c.delay {
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
		ctx := NewContext()
		e := NewExecutor().WithDelay(100 * time.Millisecond).WithErrorComparator(c.withErrComp)
		if err := e.ExecuteContext(ctx, failTimes(c.times)); err != c.expErr {
			t.Error("error found and not expected")
		}
		if ctx.LastDelay != c.delay {
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
	ctx := NewContext()
	e := NewExecutor().WithDelay(100 * time.Millisecond).WithPanic()
	if err := e.ExecuteContext(ctx, panicTimes(1)); err != nil {
		t.Error("error found and not expected")
	}
	if ctx.LastDelay != 100*time.Millisecond {
		t.Error("invalid Context.LastDelay", ctx.LastDelay)
	}
}

func TestWithBackoff(t *testing.T) {
	var testCases = []struct {
		delay    time.Duration
		expected time.Duration
	}{{0, 0}, {100, 100}, {-100, 0}}
	for _, d := range testCases {
		ctx := NewContext()
		e := NewExecutor().WithBackoff(FixedDelayBackoff(d.delay))
		if err := e.ExecuteContext(ctx, failTimes(1)); err != nil {
			t.Error("error found and not expected")
		}
		if ctx.LastDelay != d.expected {
			t.Error("invalid Context.LastDelay")
		}
	}
}

func TestWithDelay(t *testing.T) {
	var testCases = []struct {
		delay    time.Duration
		expected time.Duration
	}{{0, 0}, {100, 100}, {-100, 0}}
	for _, d := range testCases {
		ctx := NewContext()
		e := NewExecutor().WithDelay(d.delay)
		if err := e.ExecuteContext(ctx, failTimes(1)); err != nil {
			t.Error("error found and not expected")
		}
		if ctx.LastDelay != d.expected {
			t.Error("invalid Context.LastDelay")
		}
	}
}

func TestWithFirstRetryNoDelay(t *testing.T) {
	ctx := NewContext()
	e := NewExecutor().WithFirstRetryNoDelay()
	now := time.Now()
	if err := e.ExecuteContext(ctx, failTimes(1)); err != nil {
		t.Error("error found and not expected")
	}
	if time.Now().Sub(now) >= 1*time.Second {
		t.Error("first retry had a delay")
	}
	if ctx.LastDelay != 0 {
		t.Error("invalid Context.LastDelay")
	}

	ctx = NewContext()
	e = NewExecutor().WithRetries(2).WithFirstRetryNoDelay()
	if err := e.ExecuteContext(ctx, failTimes(2)); err != nil {
		t.Error("error found and not expected")
	}
	if ctx.LastDelay != 1*time.Second {
		t.Error("invalid Context.LastDelay", ctx.LastDelay)
	}
}

func TestWithMinDelay(t *testing.T) {
	ctx := NewContext()
	e := NewExecutor().WithMinDelay(100 * time.Millisecond)
	now := time.Now()
	if err := e.ExecuteContext(ctx, failTimes(1)); err != nil {
		t.Error("error found and not expected")
	}
	if time.Now().Sub(now) <= 1*time.Second {
		t.Error("min delay did not work")
	}
	if ctx.LastDelay != 1*time.Second {
		t.Error("invalid Context.LastDelay")
	}

	ctx = NewContext()
	e = NewExecutor().WithMinDelay(2 * time.Second)
	now = time.Now()
	if err := e.ExecuteContext(ctx, failTimes(1)); err != nil {
		t.Error("error found and not expected")
	}
	if time.Now().Sub(now) < 2*time.Second {
		t.Error("min delay did not work")
	}
	if ctx.LastDelay != 2*time.Second {
		t.Error("invalid Context.LastDelay")
	}
}

func TestWithMaxDelay(t *testing.T) {
	ctx := NewContext()
	e := NewExecutor().WithMaxDelay(100 * time.Millisecond)
	now := time.Now()
	if err := e.ExecuteContext(ctx, failTimes(1)); err != nil {
		t.Error("error found and not expected")
	}
	if time.Now().Sub(now) >= 1*time.Second {
		t.Error("max delay did not work")
	}
	if ctx.LastDelay != 100*time.Millisecond {
		t.Error("invalid Context.LastDelay")
	}

	ctx = NewContext()
	e = NewExecutor().WithMaxDelay(2 * time.Second)
	now = time.Now()
	if err := e.ExecuteContext(ctx, failTimes(1)); err != nil {
		t.Error("error found and not expected")
	}
	if time.Now().Sub(now) >= 2*time.Second {
		t.Error("max delay did not work")
	}
	if ctx.LastDelay != 1*time.Second {
		t.Error("invalid Context.LastDelay")
	}
}

func TestWithFixedJitter(t *testing.T) {
	ctx := NewContext()
	e := NewExecutor().
		WithDelay(100 * time.Millisecond).
		WithFixedJitter(1 * time.Millisecond)
	if err := e.ExecuteContext(ctx, failTimes(1)); err != nil {
		t.Error("error found and not expected")
	}
	if ctx.LastDelay != 101*time.Millisecond {
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
		ctx := NewContext()
		e := NewExecutor().
			WithDelay(100 * time.Millisecond).
			WithUniformJitter(100 * time.Millisecond)
		if err := e.ExecuteContext(ctx, failTimes(1)); err != nil {
			t.Error("error found and not expected")
		}
		if ctx.LastDelay == 100*time.Millisecond {
			t.Error("invalid Context.LastDelay")
		}

		// Make sure it's uniform
		lastDelay := ctx.LastDelay

		ctx = NewContext()
		if err := e.ExecuteContext(ctx, failTimes(1)); err != nil {
			t.Error("error found and not expected")
		}
		if ctx.LastDelay != lastDelay {
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
		ctx := NewContext()
		e := NewExecutor().
			WithDelay(100 * time.Millisecond).
			WithProportionalJitter(c.multiplier)
		if err := e.ExecuteContext(ctx, failTimes(1)); err != nil {
			t.Error("error found and not expected")
		}
		if ctx.LastDelay < c.minDelay || ctx.LastDelay > c.maxDelay {
			t.Error("invalid Context.LastDelay")
		}

		// With negative multiplier should have the same behavior
		ctx = NewContext()
		e = NewExecutor().
			WithDelay(100 * time.Millisecond).
			WithProportionalJitter(-c.multiplier)
		if err := e.ExecuteContext(ctx, failTimes(1)); err != nil {
			t.Error("error found and not expected")
		}
		if ctx.LastDelay < c.minDelay || ctx.LastDelay > c.maxDelay {
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
		ctx := NewContext()
		e := NewExecutor().
			WithDelay(100 * time.Millisecond).
			WithRandomJitter(100 * time.Millisecond)
		if err := e.ExecuteContext(ctx, failTimes(1)); err != nil {
			t.Error("error found and not expected")
		}
		if ctx.LastDelay == 100*time.Millisecond {
			t.Error("invalid Context.LastDelay")
		}

		// Make sure it's not uniform
		lastDelay := ctx.LastDelay
		ctx = NewContext()
		if err := e.ExecuteContext(ctx, failTimes(1)); err != nil {
			t.Error("error found and not expected")
		}
		if ctx.LastDelay == lastDelay {
			t.Error("jitter is not random")
		}
	}
}

func TestWithCustomContext(t *testing.T) {
	ctx := context.Background()
	e := NewExecutor().WithRetries(0).WithDelay(0)
	if err := e.ExecuteContext(ctx, failTimes(0)); err != nil {
		t.Error("error found and not expected")
	}

	ctx = context.Background()
	if err := e.ExecuteContext(ctx, failTimes(1)); err != ErrMaxRetries {
		t.Error("ErrMaxRetries expected and not found")
	}

	e.WithRetries(1)
	ctx = context.Background()
	if err := e.ExecuteContext(ctx, failTimes(0)); err != nil {
		t.Error("error found and not expected")
	}
	ctx = context.Background()
	if err := e.ExecuteContext(ctx, failTimes(1)); err != nil {
		t.Error("error expected and not found")
	}
	ctx = context.Background()
	if err := e.ExecuteContext(ctx, failTimes(2)); err != ErrMaxRetries {
		t.Error("ErrMaxRetries expected and not found")
	}

	ctx = context.TODO()
	if err := e.ExecuteContext(ctx, failTimes(0)); err != nil {
		t.Error("error found and not expected")
	}
	ctx = context.TODO()
	if err := e.ExecuteContext(ctx, failTimes(1)); err != nil {
		t.Error("error expected and not found")
	}
	ctx = context.TODO()
	if err := e.ExecuteContext(ctx, failTimes(2)); err != ErrMaxRetries {
		t.Error("ErrMaxRetries expected and not found")
	}
}

func TestWithTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(NewContext(), 1*time.Second)
	e := NewExecutor().WithRetries(1).WithDelay(200 * time.Millisecond)
	if err := e.ExecuteContext(ctx, failTimes(1)); err != nil {
		t.Error("error found and not expected")
	}

	ctx, cancel = context.WithTimeout(NewContext(), 100*time.Millisecond)
	if err := e.ExecuteContext(ctx, failTimes(1)); err != context.DeadlineExceeded {
		t.Error("context.DeadlineExceeded expected and not found")
	}

	ctx, cancel = context.WithTimeout(NewContext(), 1*time.Second)
	cancel()
	if err := e.WithRetries(5).ExecuteContext(ctx, failTimes(5)); err != context.Canceled {
		t.Error("context.Canceled expected and not found")
	}

	ctx, cancel = context.WithTimeout(NewContext(), 1*time.Second)
	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()
	if err := e.WithRetries(5).ExecuteContext(ctx, failTimes(5)); err != context.Canceled {
		t.Error("context.Canceled expected and not found")
	}

	ctx, cancel = context.WithTimeout(NewContext(), 2*time.Second)
	go func() {
		time.Sleep(2 * time.Second)
		cancel()
	}()
	if err := e.WithRetries(5).ExecuteContext(ctx, failTimes(5)); err != nil {
		t.Error("error found and not expected")
	}
}
