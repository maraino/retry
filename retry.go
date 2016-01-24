package retry

import (
	"errors"
	"math/rand"
	"time"
)

// ErrMaxRetries is the error returned when the Executor has reached the
// maximum number of retries.
var ErrMaxRetries = errors.New("max retries reached")

// rnd is the source of random number used by retry.
var rnd = rand.New(rand.NewSource(time.Now().UnixNano()))

// Context is the context passed to the backoff strategies.
type Context struct {
	RetryCount int
	LastDelay  time.Duration
}

func (c Context) previous() Context {
	var count int
	if c.RetryCount != 0 {
		count = c.RetryCount - 1
	}
	return Context{RetryCount: count}
}

// Executor is the type used to run a method with retries.
type Executor struct {
	Context      Context
	ErrorChannel chan error
	retries      int
	backoff      Backoff
	withErr      error
	withErrComp  func(error) bool
	withPanic    bool
}

// NewExecutor retries a new executor with the number of retries set to 1 and
// the DefaultFixedDelayBackoff as the backoff strategy.
func NewExecutor() *Executor {
	return &Executor{
		backoff: DefaultFixedDelayBackoff(),
		retries: 1,
	}
}

// Seed uses the provided seed value to initialize the retry random source to a
// deterministic state. If Seed is not called a random seed will be provided.
func Seed(seed int64) {
	rnd = rand.New(rand.NewSource(seed))
}

// WithRetries sets the maximum number of retries.
func (r *Executor) WithRetries(n int) *Executor {
	r.retries = n
	return r
}

// WithError enables the retries on a specific error.
func (r *Executor) WithError(err error) *Executor {
	r.withErr = err
	return r
}

// WithError enables the retries on a specific error.
func (r *Executor) WithErrorComparator(f func(error) bool) *Executor {
	r.withErrComp = f
	return r
}

// WithErrorChannel initializes Executor.ErrorChannel a buffered channel used
// to report the errors. The size of the buffer is number of retries + 1, but
// if number of retries has not been set yet and the channel it not read it can
// cause a deadlock.
func (r *Executor) WithErrorChannel() *Executor {
	r.ErrorChannel = make(chan error, r.retries+1)
	return r
}

// WithPanic enables the retries on panics.
func (r *Executor) WithPanic() *Executor {
	r.withPanic = true
	return r
}

// WithBackoff sets a backoff strategy.
func (r *Executor) WithBackoff(backoff Backoff) *Executor {
	r.backoff = backoff
	return r
}

// WithDelay is a shortcut sets a FixedDelayBackoff backoff strategy.
func (r *Executor) WithDelay(delay time.Duration) *Executor {
	r.WithBackoff(FixedDelayBackoff(delay))
	return r
}

// WithFirstRetryNoDelay modifies the backoff strategy to not delay in the
// first retry.
func (r *Executor) WithFirstRetryNoDelay() *Executor {
	return r.WithBackoff(firstRetryNoDelayBackoff(r.backoff))
}

// WithMinDelay modifies the backoff strategy to delay at least minDelay.
func (r *Executor) WithMinDelay(minDelay time.Duration) *Executor {
	return r.WithBackoff(boundedMinDelayBackoff(minDelay, r.backoff))
}

// WithMaxDelay modifies the backoff strategy to delay at most maxDelay.
func (r *Executor) WithMaxDelay(maxDelay time.Duration) *Executor {
	return r.WithBackoff(boundedMaxDelayBackoff(maxDelay, r.backoff))
}

// WithUniformJitter modifies the backoff strategy with an extra fixed delay.
func (r *Executor) WithFixedJitter(delay time.Duration) *Executor {
	return r.WithBackoff(fixedJitterBackoff(delay, r.backoff))
}

// WithUniformJitter modifies the backoff strategy with an extra random delay.
func (r *Executor) WithUniformJitter(rangeDelay time.Duration) *Executor {
	return r.WithBackoff(uniformJitterBackoff(rangeDelay, r.backoff))
}

// WithProportinalJitter modifies the backoff strategy to add
func (r *Executor) WithProportionalJitter(multiplier float64) *Executor {
	return r.WithBackoff(proportinalJitterBackoff(multiplier, r.backoff))
}

// WithRandomJitter modifies the backoff strategy to add or substract a random delay.
// A different random value would be used on every retry.
func (r *Executor) WithRandomJitter(rangeDelay time.Duration) *Executor {
	return r.WithBackoff(randomJitterBackoff(rangeDelay, r.backoff))
}

// Execute runs the function with the retry strategy set. It will only retry
// if it has pending retries and the function returns an error. It returns the
// error ErrMaxRetries if it reaches the maximum number of retries.
func (r *Executor) Execute(f func() error) error {
	var err error
	var rec interface{}

	if r.withPanic {
		ff := f
		f = func() error {
			defer func() {
				rec = recover()
			}()
			// Run original function
			return ff()
		}

	}

	if r.ErrorChannel != nil {
		defer func() {
			close(r.ErrorChannel)
		}()
	}

	for i := 0; i <= r.retries; i++ {
		// Run method
		if err = f(); err == nil && rec == nil {
			return nil
		}

		// Return if not the right error
		if err != nil {
			if r.withErr != nil && r.withErr != err && r.withErr.Error() != err.Error() {
				return err
			}
			if r.withErrComp != nil && r.withErrComp(err) == false {
				return err
			}
			if r.ErrorChannel != nil {
				r.ErrorChannel <- err
			}
		}

		// Backoff.Wait
		if i != r.retries && r.backoff != nil {
			r.Context.RetryCount = i + 1
			r.Context.LastDelay = r.backoff.GetDelay(r.Context)
			if r.Context.LastDelay < 0 {
				r.Context.LastDelay = 0
			}
			r.Wait(r.Context.LastDelay)
		}
	}
	return ErrMaxRetries
}

func (r *Executor) Wait(d time.Duration) {
	time.Sleep(d)
}
